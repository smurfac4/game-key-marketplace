import { useState, type FormEvent } from "react";
import { Link, Navigate, useNavigate, useSearchParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api, json, safeNext } from "../api/client";
import type { AuthMethods, Profile, User } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { Button, ErrorState, Skeleton } from "../components/ui";
import { Icon } from "../components/Icon";
import { copy } from "../content/copy";
export function AuthPage({ register = false }: { register?: boolean }) {
  const auth = useAuth();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const next = safeNext(params.get("next"));
  const [validation, setValidation] = useState("");
  const [passwordMode, setPasswordMode] = useState(false);
  const [visible, setVisible] = useState(false);
  const [marketing, setMarketing] = useState(false);
  const [email, setEmail] = useState("");
  const [challenge, setChallenge] = useState("");
  const [code, setCode] = useState("");
  const methods = useQuery({
    queryKey: ["auth-methods"],
    queryFn: ({ signal }) =>
      api<{ data: AuthMethods }>("/auth/methods", { signal }),
    retry: false,
    staleTime: 60000,
  });
  const emailMode = !!methods.data?.data.email_code && !passwordMode;
  async function savePreference() {
    if (register && marketing) {
      await api("/account", {
        method: "PATCH",
        body: json({ marketing_opt_in: true } satisfies Partial<Profile>),
      });
    }
  }
  const action = useMutation({
    mutationFn: async ({
      email,
      password,
    }: {
      email: string;
      password: string;
    }) => {
      if (register)
        await api("/auth/register", {
          method: "POST",
          body: json({ email, password }),
        });
      await auth.signIn(email, password);
      await savePreference();
    },
    onSuccess: () => navigate(next, { replace: true }),
  });
  const request = useMutation({
    mutationFn: () =>
      api<{ data: { challenge_id: string } }>("/auth/email/request", {
        method: "POST",
        body: json({ email }),
      }),
    onSuccess: (result) => {
      setChallenge(result.data.challenge_id);
      setCode("");
    },
  });
  const verify = useMutation({
    mutationFn: async () => {
      const result = await api<{ data: User }>("/auth/email/verify", {
        method: "POST",
        body: json({ challenge_id: challenge, code }),
      });
      auth.acceptSession(result.data);
      await savePreference();
    },
    onSuccess: () => navigate(next, { replace: true }),
  });
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setValidation("");
    if (emailMode) {
      if (challenge) {
        if (!/^\d{6}$/.test(code)) {
          setValidation(copy.codeInvalid);
          return;
        }
        verify.mutate();
      } else request.mutate();
      return;
    }
    const f = new FormData(e.currentTarget);
    const password = String(f.get("password"));
    if (
      [...password].length < 12 ||
      new TextEncoder().encode(password).length > 72
    ) {
      setValidation(copy.invalidPassword);
      return;
    }
    if (register && password !== f.get("confirm")) {
      setValidation(copy.passwordMismatch);
      return;
    }
    action.mutate({ email: email.trim(), password });
  }
  if (auth.user && !action.isPending && !verify.isPending)
    return <Navigate to={next} replace />;
  const pending = action.isPending || request.isPending || verify.isPending;
  return (
    <div className="bg-[#1c0d43] px-4 py-10 sm:py-16">
      <section className="auth-panel">
        <div className="mb-6 flex items-center gap-2 text-xs font-bold uppercase tracking-[.2em] text-accent">
          <Icon name="key" width="18" />
          {copy.brand}
        </div>
        <h1 className="mb-3 text-3xl font-extrabold tracking-tight sm:text-4xl">
          {register ? copy.authTitleRegister : copy.authTitleLogin}
        </h1>
        <p className="mb-8 text-sm text-muted">
          {register ? copy.haveAccount : copy.newAccount}{" "}
          <Link
            className="font-bold text-accent hover:underline"
            to={`${register ? "/login" : "/register"}?next=${encodeURIComponent(next)}`}
          >
            {register ? copy.login : copy.register}
          </Link>
        </p>
        {params.has("identity_error") && (
          <p className="error mb-5" role="alert">
            {copy.authFailed}
          </p>
        )}
        {methods.isPending ? (
          <Skeleton className="h-40" />
        ) : (
          <>
            {!!methods.data?.data.providers.length && (
              <>
                <div className="space-y-3">
                  {methods.data.data.providers.map((provider) => (
                    <a
                      key={provider.start_url}
                      href={`${provider.start_url}?next=${encodeURIComponent(next)}`}
                      className="button min-h-14 w-full bg-white text-[#251047] hover:bg-slate-100"
                    >
                      <Icon name="shield" width="20" />
                      {copy.continueProvider} {provider.name}
                    </a>
                  ))}
                </div>
                <div className="my-7 flex items-center gap-3 text-sm text-muted">
                  <span className="h-px flex-1 bg-white/20" />
                  {copy.or}
                  <span className="h-px flex-1 bg-white/20" />
                </div>
              </>
            )}
            <form onSubmit={submit} className="space-y-5">
              <label className="block">
                <span className="label">{copy.email}</span>
                <input
                  name="email"
                  type="email"
                  autoComplete="email"
                  className="auth-field"
                  maxLength={254}
                  value={email}
                  readOnly={emailMode && !!challenge}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                />
              </label>
              {emailMode ? (
                challenge ? (
                  <>
                    <p className="text-sm leading-relaxed text-muted">
                      {copy.codeSent}
                    </p>
                    <label className="block">
                      <span className="label">{copy.verificationCode}</span>
                      <input
                        className="auth-field text-center text-2xl tracking-[.3em]"
                        name="code"
                        autoComplete="one-time-code"
                        inputMode="numeric"
                        pattern="[0-9]{6}"
                        minLength={6}
                        maxLength={6}
                        required
                        value={code}
                        onChange={(e) =>
                          setCode(e.target.value.replace(/\D/g, ""))
                        }
                      />
                    </label>
                  </>
                ) : (
                  <p className="text-sm leading-relaxed text-muted">
                    {copy.emailCodeHelp}
                  </p>
                )
              ) : (
                <>
                  <div className="block">
                    <label className="label" htmlFor="auth-password">
                      {copy.password}
                    </label>
                    <span className="relative block">
                      <input
                        id="auth-password"
                        name="password"
                        type={visible ? "text" : "password"}
                        autoComplete={
                          register ? "new-password" : "current-password"
                        }
                        className="auth-field pr-32"
                        required
                        aria-describedby="password-help"
                      />
                      <button
                        type="button"
                        className="absolute inset-y-0 right-0 min-w-16 px-3 text-xs text-accent"
                        aria-label={
                          visible ? copy.hidePassword : copy.showPassword
                        }
                        onClick={() => setVisible(!visible)}
                      >
                        {visible ? copy.hidePassword : copy.showPassword}
                      </button>
                    </span>
                  </div>
                  <p id="password-help" className="text-xs text-muted">
                    {copy.passwordHelp}
                  </p>
                  {register && (
                    <label className="block">
                      <span className="label">{copy.confirmPassword}</span>
                      <input
                        name="confirm"
                        type={visible ? "text" : "password"}
                        autoComplete="new-password"
                        className="auth-field"
                        required
                      />
                    </label>
                  )}
                </>
              )}
              {register && !challenge && (
                <label className="flex items-start gap-3 text-sm leading-relaxed">
                  <input
                    type="checkbox"
                    className="mt-1 h-5 w-5 shrink-0 accent-[#ffe600]"
                    checked={marketing}
                    onChange={(e) => setMarketing(e.target.checked)}
                  />
                  <span>{copy.marketing}</span>
                </label>
              )}
              {validation && (
                <p role="alert" className="error">
                  {validation}
                </p>
              )}
              {action.isError && <ErrorState error={action.error} />}
              {request.isError && <ErrorState error={request.error} />}{" "}
              {verify.isError && <ErrorState error={verify.error} />}
              <Button className="min-h-14 w-full text-base" disabled={pending}>
                {pending
                  ? copy.processing
                  : emailMode
                    ? challenge
                      ? copy.verifyCode
                      : copy.continueEmail
                    : register
                      ? copy.register
                      : copy.login}
              </Button>
            </form>
            {methods.data?.data.email_code && (
              <div className="mt-5 flex flex-col items-start gap-4 text-sm">
                <button
                  type="button"
                  className="text-accent hover:underline"
                  disabled={pending}
                  onClick={() => {
                    setPasswordMode(!passwordMode);
                    setChallenge("");
                    setValidation("");
                    request.reset();
                    verify.reset();
                    action.reset();
                  }}
                >
                  {passwordMode ? copy.useCode : copy.usePassword}
                </button>
                {challenge && (
                  <button
                    type="button"
                    className="text-muted underline"
                    disabled={pending}
                    onClick={() => {
                      setChallenge("");
                      setCode("");
                      verify.reset();
                      request.reset();
                    }}
                  >
                    {copy.changeEmail}
                  </button>
                )}
              </div>
            )}
          </>
        )}
        <p className="mt-8 border-t border-white/15 pt-5 text-xs leading-relaxed text-muted">
          {copy.authDemoNotice}
        </p>
      </section>
    </div>
  );
}

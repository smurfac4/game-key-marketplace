import { useState, type FormEvent } from "react";
import { Link, Navigate, useNavigate, useSearchParams } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";
import { api, json, safeNext } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { Button, ErrorState } from "../components/ui";
import { copy } from "../content/copy";
export function AuthPage({ register = false }: { register?: boolean }) {
  const auth = useAuth();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [validation, setValidation] = useState("");
  const next = safeNext(params.get("next"));
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
    },
    onSuccess: () => navigate(next, { replace: true }),
  });
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setValidation("");
    const f = new FormData(e.currentTarget);
    const email = String(f.get("email")).trim();
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
    action.mutate({ email, password });
  }
  if (auth.user) return <Navigate to={next} replace />;
  return (
    <div className="container-page">
      <div className="panel mx-auto my-6 max-w-md sm:my-12">
        <h1 className="page-title text-3xl">
          {register ? copy.register : copy.login}
        </h1>
        <p className="mb-6 text-sm leading-relaxed text-muted">
          {copy.authHelp}
        </p>
        <form onSubmit={submit} className="space-y-5">
          <label className="block">
            <span className="label">{copy.email}</span>
            <input
              name="email"
              type="email"
              autoComplete="email"
              className="field"
              maxLength={254}
              required
            />
          </label>
          <label className="block">
            <span className="label">{copy.password}</span>
            <input
              name="password"
              type="password"
              autoComplete={register ? "new-password" : "current-password"}
              className="field"
              required
              aria-describedby="password-help"
            />
          </label>
          <p id="password-help" className="text-xs text-muted">
            {copy.passwordHelp}
          </p>
          {register && (
            <label className="block">
              <span className="label">{copy.confirmPassword}</span>
              <input
                name="confirm"
                type="password"
                autoComplete="new-password"
                className="field"
                required
              />
            </label>
          )}
          {validation && (
            <p role="alert" className="error">
              {validation}
            </p>
          )}
          {action.isError && <ErrorState error={action.error} />}
          <Button className="w-full" disabled={action.isPending}>
            {action.isPending
              ? copy.processing
              : register
                ? copy.register
                : copy.login}
          </Button>
        </form>
        <p className="mt-6 text-sm text-muted">
          {register ? copy.haveAccount : copy.newAccount}{" "}
          <Link
            className="font-semibold text-accent underline"
            to={`${register ? "/login" : "/register"}?next=${encodeURIComponent(next)}`}
          >
            {register ? copy.login : copy.register}
          </Link>
        </p>
      </div>
    </div>
  );
}

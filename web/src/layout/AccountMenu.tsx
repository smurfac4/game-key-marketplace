import { useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useAccount } from "../hooks/account";
import { Avatar } from "../components/Avatar";
import { Icon } from "../components/Icon";
import { ErrorState, PriceTag, Skeleton } from "../components/ui";
import { copy } from "../content/copy";
export function AccountMenu() {
  const auth = useAuth();
  const account = useAccount();
  const navigate = useNavigate();
  const ref = useRef<HTMLDetailsElement>(null);
  const [error, setError] = useState<unknown>();
  const data = account.data?.data;
  if (auth.pending) return <Skeleton className="h-11 w-40" />;
  if (!auth.user)
    return (
      <div className="flex items-center gap-2">
        <Link to="/login" className="button button-secondary px-4">
          {copy.login}
        </Link>
        <Link to="/register" className="button px-4">
          {copy.signUp}
        </Link>
      </div>
    );
  const entries = [
    {
      to: "/wishlist",
      icon: "heart",
      label: copy.favorites,
      value: data?.wishlist_count,
    },
    { to: "/account", icon: "user", label: copy.myAccount },
    { to: "/account/balance", icon: "wallet", label: copy.balance },
    {
      to: "/account/rewards",
      icon: "gem",
      label: copy.rewards,
      value: data?.reward_points,
    },
    { to: "/account/cashback", icon: "bolt", label: copy.cashback },
    { to: "/orders", icon: "orders", label: copy.orders },
    { to: "/library", icon: "key", label: copy.library },
  ];
  return (
    <details
      ref={ref}
      className="account-dropdown"
      onKeyDown={(e) => {
        if (e.key === "Escape") {
          e.currentTarget.open = false;
          ref.current?.querySelector("summary")?.focus();
        }
      }}
      onClick={(e) => {
        if ((e.target as HTMLElement).closest("a,button"))
          e.currentTarget.open = false;
      }}
    >
      <summary
        className="button button-ghost list-none px-2"
        aria-label={`${copy.account} ▾`}
      >
        <Avatar kind={data?.profile.avatar} />
        <span className="hidden sm:inline">{copy.account} ▾</span>
      </summary>
      <div className="account-popover">
        <>
          <Link to="/account" className="flex min-w-0 items-center gap-3 p-5">
            <Avatar kind={data?.profile.avatar} className="h-12 w-12" />
            <span className="min-w-0">
              <strong className="block truncate text-base">
                {data?.profile.display_name || auth.user.email.split("@")[0]}
              </strong>
              <span className="block truncate text-sm text-muted">
                {auth.user.email}
              </span>
            </span>
          </Link>
          {account.isPending ? (
            <div className="px-5 pb-5">
              <Skeleton className="h-16" />
            </div>
          ) : account.isError ? (
            <div className="px-4 pb-3">
              <ErrorState
                error={account.error}
                retry={() => void account.refetch()}
              />
            </div>
          ) : (
            data && (
              <Link
                to="/account"
                className="block border-y border-white/15 p-5 hover:bg-white/5"
              >
                <span className="mb-3 block text-sm font-bold">
                  {copy.completeProfile} ({data.completion.percent}%)
                </span>
                <progress
                  className="account-progress"
                  value={data.completion.completed}
                  max={data.completion.total}
                  aria-label={copy.completeProfile}
                />
                <span className="mt-2 block text-sm text-muted">
                  {data.completion.completed} / {data.completion.total}{" "}
                  {copy.tasksCompleted} →
                </span>
              </Link>
            )
          )}
          <div className="py-2">
            {entries.map((entry) => (
              <Link key={entry.to} to={entry.to} className="account-menu-link">
                <Icon name={entry.icon} width="22" />
                <span>{entry.label}</span>
                {entry.to === "/account/balance" && data ? (
                  <span className="ml-auto text-success">
                    <PriceTag
                      price={data.wallet_balance}
                      currency={data.currency}
                    />
                  </span>
                ) : entry.value !== undefined ? (
                  <span className="ml-auto text-sm text-accent">
                    {entry.value}
                  </span>
                ) : null}
              </Link>
            ))}
            <button
              type="button"
              className="account-menu-link w-full text-left"
              onClick={() =>
                void auth
                  .signOut()
                  .then(() => navigate("/", { replace: true, flushSync: true }))
                  .catch(setError)
              }
            >
              <Icon name="logout" width="22" />
              {copy.logout}
            </button>
          </div>
          <p className="border-t border-white/15 px-5 py-3 text-xs text-muted">
            {copy.demoAccountHelp}
          </p>
        </>
        {!!error && <ErrorState error={error} />}
      </div>
    </details>
  );
}

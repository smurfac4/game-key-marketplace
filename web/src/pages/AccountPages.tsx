import { useState, type FormEvent, type ReactNode } from "react";
import { Link, NavLink, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import type { AccountSummary, OrderList, Profile } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import {
  useAccount,
  useProfileAction,
  useWishlist,
  useFavoriteAction,
} from "../hooks/account";
import { Avatar, avatarChoices } from "../components/Avatar";
import { Icon } from "../components/Icon";
import {
  Button,
  CoverImage,
  EmptyState,
  ErrorState,
  Modal,
  PriceTag,
  Skeleton,
} from "../components/ui";
import { DigitalKey } from "./Orders";
import { copy, valueLabel } from "../content/copy";
import { preferences } from "../content/preferences";
const accountLinks = () => [
  ["/account", copy.myAccount],
  ["/wishlist", copy.favorites],
  ["/orders", copy.orders],
  ["/library", copy.library],
  ["/account/balance", copy.balance],
  ["/account/rewards", copy.rewards],
  ["/account/cashback", copy.cashback],
];
export function AccountShell({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <div className="container-page">
      <h1 className="page-title">{title}</h1>
      <nav
        aria-label={copy.profileNavigation}
        className="mb-8 flex flex-wrap gap-2"
      >
        {accountLinks().map(([to, label]) => (
          <NavLink
            key={to}
            to={to}
            end
            className={({ isActive }) =>
              `button ${isActive ? "" : "button-secondary"}`
            }
          >
            {label}
          </NavLink>
        ))}
      </nav>
      {children}
    </div>
  );
}
function ProfileForm({ data }: { data: AccountSummary }) {
  const { user } = useAuth();
  const [profile, setProfile] = useState<Profile>(data.profile);
  const save = useProfileAction();
  const countryNames = new Intl.DisplayNames([preferences().language], {
    type: "region",
  });
  function submit(e: FormEvent) {
    e.preventDefault();
    save.mutate(profile);
  }
  return (
    <form onSubmit={submit} className="panel max-w-3xl space-y-6">
      <div className="flex items-center gap-4">
        <Avatar kind={profile.avatar} className="h-16 w-16" />
        <div className="min-w-0">
          <p className="text-xl font-bold">
            {profile.display_name || user?.email.split("@")[0]}
          </p>
          <p className="break-all text-sm text-muted">{user?.email}</p>
        </div>
      </div>
      <p className="text-sm text-muted">{copy.profileHelp}</p>
      <div className="rounded-control border border-line/50 bg-page/50 p-4">
        <span className="mb-3 block text-sm font-semibold">
          {copy.completeProfile} ({data.completion.percent}%)
        </span>
        <progress
          className="account-progress"
          value={data.completion.completed}
          max={data.completion.total}
          aria-label={copy.completeProfile}
        />
        <p className="mt-2 text-xs text-muted">{copy.profileTasks}</p>
      </div>
      <div className="block">
        <label className="label" htmlFor="profile-display-name">
          {copy.displayName}
        </label>
        <input
          id="profile-display-name"
          name="display_name"
          className="field"
          minLength={2}
          maxLength={32}
          value={profile.display_name ?? ""}
          aria-describedby="name-help"
          autoComplete="nickname"
          onChange={(e) =>
            setProfile({ ...profile, display_name: e.target.value })
          }
        />
        <span id="name-help" className="mt-2 block text-xs text-muted">
          {copy.displayNameHelp}
        </span>
      </div>
      <div className="grid gap-5 sm:grid-cols-2">
        <div>
          <label className="label" htmlFor="profile-country">
            {copy.country}
          </label>
          <select
            id="profile-country"
            className="field"
            value={profile.country_code ?? ""}
            autoComplete="country"
            onChange={(e) =>
              setProfile({ ...profile, country_code: e.target.value })
            }
          >
            <option value="">{copy.chooseCountry}</option>
            {data.countries.map((c) => (
              <option key={c} value={c}>
                {countryNames.of(c) ?? c}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="label" htmlFor="profile-platform">
            {copy.preferredPlatform}
          </label>
          <select
            id="profile-platform"
            className="field"
            value={profile.preferred_platform ?? ""}
            onChange={(e) =>
              setProfile({ ...profile, preferred_platform: e.target.value })
            }
          >
            <option value="">{copy.choosePlatform}</option>
            {data.platforms.map((p) => (
              <option key={p}>{p}</option>
            ))}
          </select>
        </div>
      </div>
      <fieldset>
        <legend className="label">{copy.avatar}</legend>
        <div className="flex flex-wrap gap-3">
          {avatarChoices().map((a) => (
            <label
              key={a.value}
              className={`cursor-pointer rounded-control border p-3 ${profile.avatar === a.value ? "border-accent bg-accent/10" : "border-line"}`}
            >
              <input
                className="mr-2 accent-[#f8d653]"
                type="radio"
                name="avatar"
                value={a.value}
                checked={profile.avatar === a.value}
                onChange={() => setProfile({ ...profile, avatar: a.value })}
              />
              <span className="inline-flex items-center gap-2 text-sm">
                <Avatar kind={a.value} className="h-9 w-9" />
                {a.label}
              </span>
            </label>
          ))}
        </div>
      </fieldset>
      <div className="space-y-3">
        <label className="flex items-start gap-3">
          <input
            type="checkbox"
            className="mt-1 h-5 w-5 shrink-0 accent-[#f8d653]"
            checked={profile.marketing_opt_in ?? false}
            onChange={(e) =>
              setProfile({ ...profile, marketing_opt_in: e.target.checked })
            }
          />
          <span>{copy.marketing}</span>
        </label>
        <label className="flex items-start gap-3">
          <input
            type="checkbox"
            className="mt-1 h-5 w-5 shrink-0 accent-[#f8d653]"
            checked={profile.wishlist_notifications ?? false}
            onChange={(e) =>
              setProfile({
                ...profile,
                wishlist_notifications: e.target.checked,
              })
            }
          />
          <span>{copy.wishlistAlerts}</span>
        </label>
        <p className="text-xs leading-relaxed text-muted">
          {copy.notificationHelp}
        </p>
      </div>
      {save.isError && <ErrorState error={save.error} />}
      {save.isSuccess && (
        <p role="status" className="text-success">
          {copy.saved}
        </p>
      )}
      <Button disabled={save.isPending}>
        {save.isPending ? copy.processing : copy.saveProfile}
      </Button>
    </form>
  );
}
export function AccountPage() {
  const q = useAccount();
  return (
    <AccountShell title={copy.myAccount}>
      {q.isPending ? (
        <Skeleton className="h-96" />
      ) : q.isError ? (
        <ErrorState error={q.error} retry={() => void q.refetch()} />
      ) : (
        <ProfileForm data={q.data.data} />
      )}
    </AccountShell>
  );
}
export function AccountValuePage({
  kind,
}: {
  kind: "balance" | "rewards" | "cashback";
}) {
  const q = useAccount();
  const title =
    kind === "balance"
      ? copy.balance
      : kind === "rewards"
        ? copy.rewards
        : copy.cashback;
  return (
    <AccountShell title={title}>
      {q.isPending ? (
        <Skeleton className="h-64" />
      ) : q.isError ? (
        <ErrorState error={q.error} retry={() => void q.refetch()} />
      ) : (
        <section className="panel max-w-2xl py-10">
          <Icon
            name={
              kind === "balance"
                ? "wallet"
                : kind === "rewards"
                  ? "gem"
                  : "bolt"
            }
            className="mb-6 h-12 w-12 text-accent"
          />
          <h2 className="mb-5 text-3xl font-bold">
            {kind === "rewards" ? (
              q.data.data.reward_points
            ) : (
              <PriceTag
                price={
                  kind === "balance"
                    ? q.data.data.wallet_balance
                    : q.data.data.cashback_earned
                }
                currency={q.data.data.currency}
              />
            )}
          </h2>
          <p className="mb-4 leading-relaxed">
            {kind === "balance"
              ? copy.balanceHelp
              : kind === "rewards"
                ? copy.rewardsHelp
                : copy.cashbackHelp}
          </p>
          <p className="text-sm leading-relaxed text-muted">
            {copy.demoAccountHelp}
          </p>
          <Link to="/orders" className="button button-secondary mt-6">
            {copy.orders} →
          </Link>
        </section>
      )}
    </AccountShell>
  );
}
export function WishlistPage() {
  const q = useWishlist();
  const account = useAccount();
  const remove = useFavoriteAction();
  const save = useProfileAction();
  const [search, setSearch] = useState("");
  const [modal, setModal] = useState(false);
  const [alerts, setAlerts] = useState(false);
  const filtered =
    q.data?.data.filter((p) =>
      p.name.toLocaleLowerCase().includes(search.toLocaleLowerCase()),
    ) ?? [];
  return (
    <AccountShell title={copy.favorites}>
      <div className="mb-8 flex flex-wrap gap-4">
        <label className="min-w-0 flex-1">
          <span className="sr-only">{copy.wishlistSearch}</span>
          <input
            className="field"
            type="search"
            value={search}
            placeholder={copy.wishlistSearch}
            onChange={(e) => setSearch(e.target.value)}
          />
        </label>
        <Button
          variant="secondary"
          disabled={!account.data}
          onClick={() => {
            setAlerts(
              account.data?.data.profile.wishlist_notifications ?? false,
            );
            setModal(true);
          }}
        >
          <Icon name="settings" width="20" />
          {copy.notifications}
        </Button>
      </div>
      {remove.isError && <ErrorState error={remove.error} />}
      {q.isPending ? (
        <Skeleton className="h-80" />
      ) : q.isError ? (
        <ErrorState error={q.error} retry={() => void q.refetch()} />
      ) : q.data.data.length === 0 ? (
        <EmptyState title={copy.wishlistEmpty}>
          <Icon name="heart" className="mx-auto mb-5 h-16 w-16 text-brand" />
          <p className="mb-6">{copy.wishlistHelp}</p>
          <Link to="/store" className="button">
            {copy.browse}
          </Link>
        </EmptyState>
      ) : filtered.length === 0 ? (
        <EmptyState title={copy.noFavoriteMatch} />
      ) : (
        <div className="grid gap-4">
          {filtered.map((p) => (
            <article
              key={p.id}
              className="panel flex flex-wrap items-center gap-5"
            >
              <Link
                to={`/products/${p.slug}`}
                className="flex min-w-0 flex-1 items-center gap-4"
              >
                <CoverImage
                  src={p.image_url}
                  alt=""
                  className="aspect-[2/3] w-16 shrink-0 rounded object-cover"
                />
                <span>
                  <h2 className="font-bold">{p.name}</h2>
                  <span className="mt-2 block text-xs text-muted">
                    {p.platform} · {valueLabel(p.region)}
                  </span>
                </span>
              </Link>
              {p.price ? (
                <PriceTag price={p.price} currency={p.currency} />
              ) : (
                <span className="text-muted">{copy.soldOut}</span>
              )}
              <Button
                variant="ghost"
                aria-label={`${copy.favoriteRemove}: ${p.name}`}
                disabled={remove.isPending}
                onClick={() => remove.mutate({ id: p.id, remove: true })}
              >
                <Icon
                  name="heart"
                  fill="currentColor"
                  className="text-accent"
                />
                {copy.remove}
              </Button>
            </article>
          ))}
        </div>
      )}
      <Modal
        open={modal}
        onClose={() => setModal(false)}
        title={copy.notifications}
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (account.data)
              save.mutate(
                {
                  ...account.data.data.profile,
                  wishlist_notifications: alerts,
                },
                { onSuccess: () => setModal(false) },
              );
          }}
        >
          <label className="flex items-start gap-3">
            <input
              className="mt-1 h-5 w-5 shrink-0 accent-[#f8d653]"
              type="checkbox"
              checked={alerts}
              onChange={(e) => setAlerts(e.target.checked)}
            />
            <span>{copy.wishlistAlerts}</span>
          </label>
          <p className="my-5 text-sm leading-relaxed text-muted">
            {copy.notificationHelp}
          </p>
          {save.isError && <ErrorState error={save.error} />}
          <Button disabled={save.isPending}>{copy.saveProfile}</Button>
        </form>
      </Modal>
    </AccountShell>
  );
}
export function KeysLibrary() {
  const { user } = useAuth();
  const [params, setParams] = useSearchParams();
  const page = params.get("page") ?? "1";
  const q = useQuery({
    queryKey: ["orders", user?.id, page],
    queryFn: ({ signal }) =>
      api<OrderList>(`/orders?page=${encodeURIComponent(page)}`, { signal }),
    retry: false,
  });
  return (
    <AccountShell title={copy.library}>
      <p className="mb-6 text-sm text-muted">{copy.libraryHelp}</p>
      {q.isPending ? (
        <Skeleton className="h-80" />
      ) : q.isError ? (
        <ErrorState error={q.error} retry={() => void q.refetch()} />
      ) : q.data.data.length === 0 ? (
        <EmptyState title={copy.orderEmpty}>
          <Link className="button" to="/store">
            {copy.browse}
          </Link>
        </EmptyState>
      ) : (
        <>
          <div className="space-y-5">
            {q.data.data.map((o) => (
              <section className="panel" key={o.id}>
                <Link
                  to={`/orders/${o.id}`}
                  className="mb-4 inline-block text-sm text-accent"
                >
                  {copy.order} #{o.id} →
                </Link>
                {o.items.map((item) => (
                  <div key={item.id} className="mb-5 last:mb-0">
                    <h2 className="mb-3 font-bold">{item.product_name}</h2>
                    <div className="space-y-3">
                      {item.keys.map((k) => (
                        <DigitalKey key={k} value={k} />
                      ))}
                    </div>
                  </div>
                ))}
              </section>
            ))}
          </div>
          {q.data.total_pages > 1 && (
            <nav
              aria-label={copy.page}
              className="mt-6 flex items-center justify-center gap-4"
            >
              <Button
                disabled={q.data.page <= 1}
                onClick={() => setParams({ page: String(q.data.page - 1) })}
              >
                {copy.previous}
              </Button>
              <span>
                {q.data.page}/{q.data.total_pages}
              </span>
              <Button
                disabled={q.data.page >= q.data.total_pages}
                onClick={() => setParams({ page: String(q.data.page + 1) })}
              >
                {copy.next}
              </Button>
            </nav>
          )}
        </>
      )}
    </AccountShell>
  );
}

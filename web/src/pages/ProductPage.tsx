import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate, useParams, useLocation } from "react-router-dom";
import { api, ApiError } from "../api/client";
import type { Details } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { useCartAction } from "../hooks/cart";
import {
  CoverImage,
  Badge,
  Button,
  EmptyState,
  ErrorState,
  PriceTag,
  Skeleton,
} from "../components/ui";
import { Icon } from "../components/Icon";
import { copy, valueLabel } from "../content/copy";
export function ProductPage() {
  const { slug } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const { user } = useAuth();
  const add = useCartAction();
  const query = useQuery({
    queryKey: ["product", slug],
    queryFn: ({ signal }) =>
      api<Details>(`/products/${encodeURIComponent(slug!)}`, { signal }),
    retry: false,
  });
  if (query.isPending)
    return (
      <div className="container-page grid gap-8 md:grid-cols-3">
        <Skeleton className="aspect-[2/3]" />
        <Skeleton className="h-96 md:col-span-2" />
      </div>
    );
  if (query.isError)
    return (
      <div className="container-page">
        {query.error instanceof ApiError && query.error.status === 404 ? (
          <EmptyState title={copy.productMissing}>
            <Link className="button" to="/store">
              {copy.browse}
            </Link>
          </EmptyState>
        ) : (
          <ErrorState error={query.error} retry={() => void query.refetch()} />
        )}
      </div>
    );
  const p = query.data.data;
  const available = p.offers.filter((o) => o.stock > 0);
  const featured = available.find((o) => o.currency === "EUR") ?? available[0];
  const best = new Map<string, number>();
  available.forEach((o) =>
    best.set(
      o.currency,
      Math.min(best.get(o.currency) ?? Infinity, Number(o.effective_price)),
    ),
  );
  return (
    <div className="container-page">
      <Link
        to="/store"
        className="mb-6 inline-block text-sm text-muted hover:text-accent"
      >
        ← {copy.store}
      </Link>
      <div className="grid gap-8 md:grid-cols-[220px_1fr] xl:grid-cols-[220px_1fr_280px]">
        <CoverImage
          src={p.image_url}
          alt={p.name}
          className="mx-auto aspect-[2/3] w-full max-w-80 rounded-card object-cover shadow-2xl"
        />
        <div>
          <div className="mb-4 flex flex-wrap gap-2">
            <Badge>{p.platform}</Badge>
            <Badge>{valueLabel(p.region)}</Badge>
            {p.genre && <Badge>{valueLabel(p.genre)}</Badge>}
          </div>
          <h1 className="page-title text-2xl sm:text-3xl">
            {p.name}{" "}
            <span className="block mt-2 text-lg font-semibold text-muted">
              {p.platform} {copy.digitalKey} · {valueLabel(p.region)}
            </span>
          </h1>
          <h2 className="mb-3 text-lg font-bold">{copy.description}</h2>
          <p className="max-w-3xl whitespace-pre-line leading-relaxed text-muted">
            {p.description}
          </p>
          {p.release_date && (
            <p className="mt-6 text-sm text-muted">
              {copy.releaseDate}:{" "}
              <time dateTime={p.release_date}>{p.release_date}</time>
            </p>
          )}
          {available.length === 0 && (
            <p className="mt-6">
              <Badge tone="accent">{copy.soldOut}</Badge>
            </p>
          )}
        </div>
        {featured && (
          <aside className="panel self-start md:col-span-2 xl:col-span-1">
            <div className="mb-5 flex items-center justify-between gap-3">
              <Badge tone="accent">{copy.best}</Badge>
              <span className="text-xs text-success">
                {featured.stock} {copy.stock.toLowerCase()}
              </span>
            </div>
            <PriceTag
              price={featured.effective_price}
              currency={featured.currency}
              original={featured.discount_price ? featured.price : null}
            />
            <p className="my-5 text-sm">
              {featured.seller.name}
              <span className="mt-1 block text-accent">
                ★ {featured.seller.rating.toFixed(2)} / 5
              </span>
            </p>
            <Button
              className="w-full"
              disabled={add.isPending}
              onClick={() => {
                if (!user) {
                  navigate(
                    `/login?next=${encodeURIComponent(location.pathname)}`,
                  );
                  return;
                }
                add.mutate(
                  { method: "POST", offer: featured.id, quantity: 1 },
                  { onSuccess: () => navigate("/cart") },
                );
              }}
            >
              {copy.purchase}
            </Button>
            <p className="mt-5 flex items-center gap-2 text-xs text-muted">
              <Icon name="bolt" className="shrink-0 text-success" width="18" />
              {copy.demoDelivery}
            </p>
            <p className="mt-3 text-xs text-muted">{copy.footer}</p>
          </aside>
        )}
      </div>
      <section className="mt-section">
        <h2 className="section-title mb-5">{copy.offers}</h2>
        {add.isError && (
          <div className="mb-4">
            <ErrorState error={add.error} />
          </div>
        )}
        {add.isSuccess && (
          <p
            role="status"
            className="mb-4 rounded-control bg-success/10 p-4 text-success"
          >
            {copy.added}.{" "}
            <Link className="underline" to="/cart">
              {copy.cart} →
            </Link>
          </p>
        )}
        {p.offers.length === 0 ? (
          <EmptyState title={copy.noOffers} />
        ) : (
          <div className="overflow-x-auto rounded-card border border-line">
            <table className="offers-table w-full min-w-[640px] text-left text-sm">
              <thead className="bg-raised text-muted">
                <tr>
                  {[
                    copy.seller,
                    copy.rating,
                    copy.stock,
                    copy.price,
                    copy.cart,
                  ].map((h) => (
                    <th key={h} scope="col" className="px-5 py-4">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {p.offers.map((o) => (
                  <tr
                    key={o.id}
                    className={`border-t border-line/60 ${o.stock > 0 && best.get(o.currency) === Number(o.effective_price) ? "bg-brand/10" : "bg-surface"}`}
                  >
                    <td className="px-5 py-5">
                      <span className="font-bold">{o.seller.name}</span>
                      {o.stock > 0 &&
                        best.get(o.currency) === Number(o.effective_price) && (
                          <div className="mt-2">
                            <Badge tone="accent">
                              {copy.best} · {o.currency}
                            </Badge>
                          </div>
                        )}
                    </td>
                    <td data-label={copy.rating} className="px-5 py-5">
                      <span className="text-accent">★</span>{" "}
                      {o.seller.rating.toFixed(2)} / 5
                    </td>
                    <td data-label={copy.stock} className="px-5 py-5">
                      {o.stock > 0 ? o.stock : copy.soldOut}
                    </td>
                    <td className="px-5 py-5">
                      <PriceTag
                        price={o.effective_price}
                        currency={o.currency}
                        original={o.discount_price ? o.price : null}
                      />
                    </td>
                    <td className="px-5 py-5">
                      <Button
                        disabled={o.stock === 0 || add.isPending}
                        onClick={() => {
                          if (!user) {
                            navigate(
                              `/login?next=${encodeURIComponent(location.pathname)}`,
                            );
                            return;
                          }
                          add.mutate({
                            method: "POST",
                            offer: o.id,
                            quantity: 1,
                          });
                        }}
                      >
                        {o.stock === 0 ? copy.soldOut : copy.add}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
      {query.data.similar_products.length > 0 && (
        <section className="mt-section">
          <h2 className="section-title mb-5">{copy.similar}</h2>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-8">
            {query.data.similar_products.map((s) => (
              <Link
                key={s.id}
                to={`/products/${s.slug}`}
                className="overflow-hidden rounded-card border border-line bg-surface transition hover:border-brand"
              >
                <CoverImage
                  src={s.image_url}
                  alt=""
                  loading="lazy"
                  className="aspect-[2/3] w-full object-cover"
                />
                <div className="p-3">
                  <h3 className="text-sm font-bold">{s.name}</h3>
                  <p className="mt-2 text-xs text-muted">
                    {s.platform} · {s.region}
                  </p>
                </div>
              </Link>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

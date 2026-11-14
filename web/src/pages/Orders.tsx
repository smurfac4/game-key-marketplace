import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { api, ApiError } from "../api/client";
import type { Order, OrderList } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import {
  Badge,
  Button,
  EmptyState,
  ErrorState,
  PriceTag,
  Skeleton,
} from "../components/ui";
import { preferences } from "../content/preferences";
import { copy } from "../content/copy";
export function DigitalKey({ value }: { value: string }) {
  const [state, setState] = useState<"idle" | "copied" | "failed">("idle");
  return (
    <div className="rounded-control border border-line bg-page p-3">
      <div className="flex flex-wrap items-center gap-3">
        <code
          className="min-w-0 flex-1 break-all text-sm text-success"
          data-testid="digital-key"
        >
          {value}
        </code>
        <Button
          variant="secondary"
          onClick={() => {
            if (!navigator.clipboard) {
              setState("failed");
              return;
            }
            void navigator.clipboard
              .writeText(value)
              .then(() => setState("copied"))
              .catch(() => setState("failed"));
          }}
        >
          {state === "copied" ? copy.copied : copy.copy}
        </Button>
      </div>
      <span role="status" className="text-xs text-muted">
        {state === "failed"
          ? copy.copyFailed
          : state === "copied"
            ? copy.copied
            : ""}
      </span>
    </div>
  );
}
export function OrderPage() {
  const { id } = useParams();
  const [params] = useSearchParams();
  const { user } = useAuth();
  const q = useQuery({
    queryKey: ["order", user?.id, id],
    queryFn: ({ signal }) =>
      api<{
        data: Order;
      }>(`/orders/${encodeURIComponent(id!)}`, { signal }),
    retry: false,
  });
  if (q.isPending)
    return (
      <div className="container-page">
        <Skeleton className="h-96" />
      </div>
    );
  if (q.isError)
    return (
      <div className="container-page">
        {q.error instanceof ApiError && q.error.status === 404 ? (
          <EmptyState title={copy.orderMissing}>
            <Link className="button" to="/orders">
              {copy.backOrders}
            </Link>
          </EmptyState>
        ) : (
          <ErrorState error={q.error} retry={() => void q.refetch()} />
        )}
      </div>
    );
  const order = q.data.data;
  return (
    <div className="container-page">
      <Link className="mb-6 inline-block text-sm text-muted" to="/orders">
        ← {copy.backOrders}
      </Link>
      <div className="mb-7 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="page-title mb-3">
            {params.has("confirmed")
              ? copy.confirmation
              : `${copy.order} #${order.id}`}
          </h1>
          <p className="text-sm text-muted">
            {copy.order} #{order.id} ·{" "}
            {new Date(order.created_at).toLocaleString(
              preferences().language === "ru" ? "ru-RU" : "en-GB",
            )}
          </p>
        </div>
        <Badge tone="success">
          {order.status === "paid" ? copy.paid : order.status}
        </Badge>
      </div>
      <div className="space-y-5">
        {order.items.map((item) => (
          <article key={item.id} className="panel">
            <div className="mb-5 flex flex-wrap justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold">{item.product_name}</h2>
                <p className="mt-2 text-sm text-muted">
                  {item.seller_name} · {copy.quantity}: {item.quantity}
                </p>
              </div>
              <span className="mb-1 block text-xs text-muted">
                {copy.unitPrice}
              </span>
              <PriceTag price={item.unit_price} currency={item.currency} />
            </div>
            <h3 className="mb-3 font-semibold">{copy.keys}</h3>
            <div className="space-y-3">
              {item.keys.map((key) => (
                <DigitalKey key={key} value={key} />
              ))}
            </div>
          </article>
        ))}
      </div>
      <div className="mt-7 flex items-center justify-end gap-4">
        <strong>{copy.total}</strong>
        <PriceTag price={order.total} currency={order.currency} />
      </div>
    </div>
  );
}
export function OrderHistory() {
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
    <div className="container-page">
      <h1 className="page-title">{copy.orders}</h1>
      {q.isPending ? (
        <Skeleton className="h-96" />
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
          <div className="space-y-4">
            {q.data.data.map((order) => (
              <article
                key={order.id}
                className="panel flex flex-wrap items-center justify-between gap-5"
              >
                <div>
                  <h2 className="text-lg font-bold">
                    {copy.order} #{order.id}
                  </h2>
                  <p className="mt-1 text-sm text-muted">
                    {new Date(order.created_at).toLocaleDateString(
                      preferences().language === "ru" ? "ru-RU" : "en-GB",
                    )}{" "}
                    · {order.status}
                  </p>
                  <p className="mt-2 max-w-xl text-sm">
                    {order.items.map((i) => i.product_name).join(", ")}
                  </p>
                </div>
                <PriceTag price={order.total} currency={order.currency} />
                <Link
                  className="button button-secondary"
                  to={`/orders/${order.id}`}
                >
                  {copy.viewOrder}
                </Link>
              </article>
            ))}
          </div>
          {q.data.total_pages > 1 && (
            <nav
              aria-label={copy.page}
              className="mt-6 flex justify-center gap-4"
            >
              <Button
                disabled={q.data.page <= 1}
                onClick={() => setParams({ page: String(q.data.page - 1) })}
              >
                {copy.previous}
              </Button>
              <span className="self-center">
                {q.data.page} / {q.data.total_pages}
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
    </div>
  );
}

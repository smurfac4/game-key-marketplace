import { useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";
import { api, json } from "../api/client";
import type { Cart, Order } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { useCart } from "../hooks/cart";
import {
  Button,
  EmptyState,
  ErrorState,
  PriceTag,
  Skeleton,
} from "../components/ui";
import { copy } from "../content/copy";
function paymentToken(fingerprint: string) {
  const key = `checkout:${fingerprint}`;
  try {
    const old = sessionStorage.getItem(key);
    if (old) return old;
    const token = crypto.randomUUID();
    sessionStorage.setItem(key, token);
    return token;
  } catch {
    return crypto.randomUUID();
  }
}
function PaymentForm({
  cart,
  fingerprint,
}: {
  cart: Cart;
  fingerprint: string;
}) {
  const client = useQueryClient();
  const navigate = useNavigate();
  const [token] = useState(() => paymentToken(fingerprint));
  const action = useMutation({
    mutationFn: (card: string) =>
      api<{
        data: Order;
      }>("/checkout", {
        method: "POST",
        headers: { "Idempotency-Key": token },
        body: json({ currency: cart.currency, card_number: card }),
      }),
    onSuccess: (result) => {
      void client.invalidateQueries({ queryKey: ["cart"] });
      void client.invalidateQueries({ queryKey: ["orders"] });
      void client.invalidateQueries({ queryKey: ["products"] });
      void client.invalidateQueries({ queryKey: ["home"] });
      try {
        sessionStorage.removeItem(`checkout:${fingerprint}`);
      } catch {}
      navigate(`/orders/${result.data.id}?confirmed=1`, { replace: true });
    },
    onError: () => {
      void client.invalidateQueries({ queryKey: ["cart"] });
    },
  });
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    action.mutate(String(new FormData(e.currentTarget).get("card")));
  }
  return (
    <div className="grid gap-7 lg:grid-cols-[1fr_360px]">
      <form onSubmit={submit} className="panel space-y-5">
        <h2 className="section-title">{copy.payment}</h2>
        <p className="rounded-control bg-accent/10 p-3 text-sm font-semibold text-accent">
          {copy.testOnly}
        </p>
        <label className="block">
          <span className="label">{copy.card}</span>
          <input
            className="field"
            name="card"
            inputMode="numeric"
            autoComplete="off"
            maxLength={32}
            required
            placeholder="4242 4242 4242 4242"
          />
        </label>
        <div className="space-y-2 text-sm text-muted">
          <p>{copy.successCard}</p>
          <p>{copy.failureCard}</p>
        </div>
        {action.isError && <ErrorState error={action.error} />}
        <Button
          type="submit"
          disabled={
            action.isPending || cart.items.some((i) => i.quantity > i.stock)
          }
          className="w-full"
        >
          {action.isPending ? copy.processing : copy.pay}
        </Button>
        <p className="text-xs leading-relaxed text-muted">{copy.paymentHelp}</p>
        <Link className="inline-block text-sm text-accent underline" to="/cart">
          ← {copy.cart}
        </Link>
      </form>
      <aside className="panel h-fit">
        <h2 className="mb-5 text-xl font-bold">{copy.total}</h2>
        <ul className="mb-6 space-y-4 border-b border-line pb-5">
          {cart.items.map((i) => (
            <li key={i.offer_id} className="flex justify-between gap-4 text-sm">
              <span>
                {i.name} × {i.quantity}
              </span>
              <span className="whitespace-nowrap">
                {i.total} {i.currency}
              </span>
            </li>
          ))}
        </ul>
        <PriceTag price={cart.total} currency={cart.currency} />
        {cart.items.some((i) => i.quantity > i.stock) && (
          <p role="alert" className="error mt-5">
            {copy.cartUnavailable}
          </p>
        )}
      </aside>
    </div>
  );
}
export function CheckoutPage() {
  const q = useCart();
  const { user } = useAuth();
  if (q.isPending)
    return (
      <div className="container-page">
        <Skeleton className="h-96" />
      </div>
    );
  if (q.isError)
    return (
      <div className="container-page">
        <ErrorState error={q.error} retry={() => void q.refetch()} />
      </div>
    );
  const cart = q.data!.data;
  const fingerprint = `${user!.id}:${cart.currency}:${cart.items.map((i) => `${i.offer_id}-${i.quantity}-${i.unit_price}`).join(",")}`;
  return (
    <div className="container-page">
      <h1 className="page-title">{copy.checkout}</h1>
      {cart.items.length ? (
        <PaymentForm key={fingerprint} cart={cart} fingerprint={fingerprint} />
      ) : (
        <EmptyState title={copy.cartEmpty}>
          <Link className="button" to="/store">
            {copy.browse}
          </Link>
        </EmptyState>
      )}
    </div>
  );
}

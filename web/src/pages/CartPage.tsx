import { Link } from "react-router-dom";
import { useCart, useCartAction } from "../hooks/cart";
import {
  CoverImage,
  Button,
  EmptyState,
  ErrorState,
  PriceTag,
  Skeleton,
} from "../components/ui";
import { copy } from "../content/copy";
export function CartPage() {
  const query = useCart();
  const action = useCartAction();
  if (query.isPending)
    return (
      <div className="container-page">
        <Skeleton className="h-96" />
      </div>
    );
  if (query.isError)
    return (
      <div className="container-page">
        <ErrorState error={query.error} retry={() => void query.refetch()} />
      </div>
    );
  const cart = query.data!.data;
  const invalid = cart.items.some((i) => i.stock < i.quantity);
  return (
    <div className="container-page">
      <h1 className="page-title">{copy.cart}</h1>
      {cart.items.length === 0 ? (
        <EmptyState title={copy.cartEmpty}>
          <p className="mb-5">{copy.cartHelp}</p>
          <Link className="button" to="/store">
            {copy.browse}
          </Link>
        </EmptyState>
      ) : (
        <div className="grid gap-7 lg:grid-cols-[1fr_320px]">
          <div className="space-y-4">
            {action.isError && <ErrorState error={action.error} />}{" "}
            {invalid && (
              <p role="alert" className="error">
                {copy.cartUnavailable}
              </p>
            )}
            {cart.items.map((i) => (
              <article
                key={i.offer_id}
                className="panel flex flex-wrap items-center gap-4"
              >
                <Link to={`/products/${i.slug}`} className="shrink-0">
                  <CoverImage
                    src={i.image_url}
                    alt={i.name}
                    className="h-28 w-20 rounded-control object-cover"
                  />
                </Link>
                <div className="min-w-0 flex-1">
                  <Link
                    className="font-bold hover:text-accent"
                    to={`/products/${i.slug}`}
                  >
                    {i.name}
                  </Link>
                  <p className="mt-2 text-sm text-muted">{i.seller}</p>
                  <div className="mt-3">
                    <span className="mb-1 block text-xs text-muted">
                      {copy.unitPrice}
                    </span>
                    <PriceTag price={i.unit_price} currency={i.currency} />
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <label className="w-24">
                    <span className="label">{copy.quantity}</span>
                    <select
                      aria-label={`${copy.quantity}: ${i.name}`}
                      className="field"
                      value={i.quantity}
                      disabled={action.isPending}
                      onChange={(e) =>
                        action.mutate({
                          method: "PATCH",
                          offer: i.offer_id,
                          quantity: Number(e.target.value),
                        })
                      }
                    >
                      {Array.from({ length: 10 }, (_, n) => (
                        <option
                          key={n}
                          value={n + 1}
                          disabled={n + 1 > i.stock && n + 1 !== i.quantity}
                        >
                          {n + 1}
                        </option>
                      ))}
                    </select>
                  </label>
                  <Button
                    variant="ghost"
                    disabled={action.isPending}
                    onClick={() =>
                      action.mutate({ method: "DELETE", offer: i.offer_id })
                    }
                  >
                    {copy.remove}
                  </Button>
                </div>
              </article>
            ))}
          </div>
          <aside className="panel h-fit">
            <h2 className="mb-5 text-xl font-bold">{copy.total}</h2>
            <PriceTag price={cart.total} currency={cart.currency} />
            {invalid ? (
              <Button disabled className="mt-6 w-full">
                {copy.checkout}
              </Button>
            ) : (
              <Link
                className={`button mt-6 w-full ${action.isPending ? "pointer-events-none opacity-50" : ""}`}
                to="/checkout"
              >
                {copy.checkout}
              </Link>
            )}
            <Link
              className="mt-4 block text-center text-sm text-muted hover:text-ink"
              to="/store"
            >
              {copy.continueShopping}
            </Link>
          </aside>
        </div>
      )}
    </div>
  );
}

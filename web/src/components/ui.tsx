import {
  useEffect,
  type ImgHTMLAttributes,
  useId,
  useRef,
  type ButtonHTMLAttributes,
  type ReactNode,
} from "react";
import { Link } from "react-router-dom";
import type { Product } from "../api/types";
import { errorText } from "../api/client";
import { Icon } from "./Icon";
import { FavoriteButton } from "./FavoriteButton";
import { preferences } from "../content/preferences";
import { copy, valueLabel } from "../content/copy";
export function CoverImage({
  src,
  ...props
}: ImgHTMLAttributes<HTMLImageElement>) {
  return (
    <img
      {...props}
      src={src}
      onError={(e) => {
        if (!e.currentTarget.src.endsWith("/media/image-unavailable.svg"))
          e.currentTarget.src = "/media/image-unavailable.svg";
      }}
    />
  );
}
export function Button({
  variant = "primary",
  className = "",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost";
}) {
  return (
    <button
      className={`button ${variant === "primary" ? "" : `button-${variant}`} ${className}`}
      {...props}
    />
  );
}
export function Badge({
  children,
  tone = "default",
}: {
  children: ReactNode;
  tone?: "default" | "success" | "accent";
}) {
  return (
    <span
      className={`inline-flex rounded px-2 py-1 text-xs font-semibold ${tone === "accent" ? "bg-accent text-page" : tone === "success" ? "bg-success/15 text-success" : "bg-raised text-muted"}`}
    >
      {children}
    </span>
  );
}
export function Skeleton({ className = "" }: { className?: string }) {
  return (
    <div
      aria-label={copy.loading}
      role="status"
      className={`animate-pulse rounded-card bg-raised ${className}`}
    />
  );
}
export function GridSkeleton({ count = 8 }: { count?: number }) {
  return (
    <div className="product-grid">
      {Array.from({ length: count }, (_, i) => (
        <Skeleton key={i} className="aspect-[3/5]" />
      ))}
    </div>
  );
}
export function PriceTag({
  price,
  currency,
  original,
}: {
  price: string;
  currency: string;
  original?: string | null;
}) {
  const format = (v: string) =>
    new Intl.NumberFormat(preferences().language === "ru" ? "ru-RU" : "en-GB", {
      style: "currency",
      currency,
    }).format(Number(v));
  return (
    <span className="inline-flex flex-wrap items-baseline gap-x-2">
      <strong className="text-2xl font-extrabold text-ink">
        {format(price)}
      </strong>
      {original && Number(original) > Number(price) && (
        <del className="text-sm text-muted">{format(original)}</del>
      )}
    </span>
  );
}
export function ProductCard({ product }: { product: Product }) {
  return (
    <div className="relative h-full">
      <FavoriteButton id={product.id} name={product.name} compact />
      <Link
        to={`/products/${product.slug}`}
        className="group flex h-full flex-col overflow-hidden rounded-card border border-line/35 bg-surface transition duration-200 hover:-translate-y-1 hover:border-brand hover:shadow-xl motion-reduce:transform-none"
      >
        <div className="relative overflow-hidden">
          <CoverImage
            src={product.image_url}
            alt=""
            loading="lazy"
            className="aspect-[2/3] w-full object-cover transition duration-300 group-hover:scale-105 motion-reduce:transform-none"
          />
          {product.original_price && (
            <span className="absolute left-2 top-2">
              <Badge tone="accent">
                −
                {Math.round(
                  (1 - Number(product.price) / Number(product.original_price)) *
                    100,
                )}
                %
              </Badge>
            </span>
          )}
        </div>
        <div className="flex items-center justify-center gap-1.5 border-t border-white/10 bg-black/25 px-2 py-1.5 text-[11px] font-semibold">
          <Icon name="desktop" width="13" height="13" />
          {product.platform}
        </div>
        <div className="flex flex-1 flex-col gap-2.5 p-3">
          <h3 className="line-clamp-2 min-h-10 text-sm font-bold leading-5 ">
            {product.name} {copy.digitalKey}
          </h3>
          <p className="text-xs text-muted">
            <span className="font-bold uppercase text-success">
              {valueLabel(product.region)}
            </span>
          </p>
          <div className="mt-auto">
            <span className="mb-1 block text-xs text-muted">{copy.from}</span>
            <PriceTag
              price={product.price}
              currency={product.currency}
              original={product.original_price}
            />
          </div>
        </div>
      </Link>
    </div>
  );
}
export function ErrorState({
  error,
  retry,
}: {
  error: unknown;
  retry?: () => void;
}) {
  return (
    <div role="alert" className="error">
      <p>{errorText(error)}</p>
      {retry && (
        <Button className="mt-3" variant="secondary" onClick={retry}>
          {copy.retry}
        </Button>
      )}
    </div>
  );
}
export function EmptyState({
  title,
  children,
}: {
  title: string;
  children?: ReactNode;
}) {
  return (
    <div className="panel py-14 text-center">
      <h2 className="text-xl font-bold">{title}</h2>
      <div className="mt-4 text-muted">{children}</div>
    </div>
  );
}
export function Modal({
  open,
  onClose,
  title,
  children,
  drawer = false,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  drawer?: boolean;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const id = useId();
  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open) {
      d.showModal();
      const old = document.body.style.overflow;
      document.body.style.overflow = "hidden";
      return () => {
        d.close();
        document.body.style.overflow = old;
      };
    }
    d.close();
  }, [open]);
  return (
    <dialog
      ref={ref}
      aria-labelledby={id}
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      className={`fixed overflow-y-auto border border-line bg-surface p-6 shadow-2xl ${drawer ? "inset-y-0 right-0 left-auto m-0 h-dvh max-h-dvh w-[min(90vw,24rem)]" : "inset-0 m-auto max-h-[90dvh] w-[calc(100%-2rem)] max-w-lg rounded-card"}`}
    >
      <div className="mb-6 flex items-center justify-between gap-4">
        <h2 id={id} className="text-2xl font-bold">
          {title}
        </h2>
        <Button variant="ghost" aria-label={copy.close} onClick={onClose}>
          ✕
        </Button>
      </div>
      {children}
    </dialog>
  );
}

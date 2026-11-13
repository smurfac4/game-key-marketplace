import { Link } from "react-router-dom";
import { Categories } from "../components/Categories";
import { Icon } from "../components/Icon";
import { useState, useRef } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import type { Block, Product } from "../api/types";
import {
  Button,
  ErrorState,
  GridSkeleton,
  ProductCard,
  Skeleton,
} from "../components/ui";
import { copy } from "../content/copy";
import { useBanners, useBestSellers, useDeals, useNewReleases } from "./hooks";
function Banners() {
  const q = useBanners();
  const [active, setActive] = useState(0);
  if (q.isPending) return <Skeleton className="h-72 sm:h-96" />;
  if (q.isError)
    return <ErrorState error={q.error} retry={() => void q.refetch()} />;
  if (!q.data.title || q.data.data.length === 0) return null;
  const index = Math.min(active, q.data.data.length - 1);
  const banner = q.data.data[index];
  return (
    <section
      aria-roledescription="carousel"
      aria-label={q.data.title}
      className="hero-carousel"
    >
      <div className="hero-track">
        <div className="hero-main relative overflow-hidden bg-raised">
          <img
            src={banner.image_url}
            alt=""
            className="h-full w-full object-cover"
          />
          <div className="absolute inset-0 bg-gradient-to-r from-black/85 via-black/45 to-transparent" />
          <div className="absolute inset-y-0 left-0 flex max-w-xl flex-col justify-center p-7 sm:p-12">
            <p className="mb-4 text-sm font-semibold tracking-wide text-accent">
              {q.data.title}
            </p>
            <h1 className="text-3xl font-extrabold leading-tight sm:text-[2.65rem]">
              {banner.title}
            </h1>
            <a className="button mt-7 self-start" href={banner.link_url}>
              {copy.browse} →
            </a>
          </div>
        </div>
        <div className="hero-side">
          {q.data.data
            .filter((b) => b.id !== banner.id)
            .slice(0, 2)
            .map((b) => (
              <a className="hero-tile" key={b.id} href={b.link_url}>
                <img src={b.image_url} alt="" />
                <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/30 to-transparent" />
                <span className="relative z-10 mt-auto p-5 text-lg font-extrabold leading-tight">
                  {b.title}
                  <Icon name="arrow" className="mt-3 text-accent" />
                </span>
              </a>
            ))}
        </div>
      </div>
      <div className="mt-2 flex items-center justify-center gap-3">
        <Button
          variant="secondary"
          aria-label={copy.bannerPrevious}
          onClick={() =>
            setActive((index - 1 + q.data.data.length) % q.data.data.length)
          }
        >
          ←
        </Button>
        {q.data.data.map((b, i) => (
          <button
            key={b.id}
            aria-label={`${copy.banner} ${i + 1}`}
            aria-current={i === index ? "true" : undefined}
            onClick={() => setActive(i)}
            className="grid h-11 w-8 place-items-center"
          >
            <span
              className={`h-2 rounded-full ${i === index ? "w-7 bg-accent" : "w-2 bg-line"}`}
            />
          </button>
        ))}
        <Button
          variant="secondary"
          aria-label={copy.bannerNext}
          onClick={() => setActive((index + 1) % q.data.data.length)}
        >
          →
        </Button>
      </div>
    </section>
  );
}
function ProductBlock({
  id,
  query: q,
}: {
  id: string;
  query: UseQueryResult<Block<Product>, Error>;
}) {
  const row = useRef<HTMLDivElement>(null);
  if (q.isPending)
    return (
      <section aria-busy="true">
        <Skeleton className="mb-5 h-9 w-60" />
        <GridSkeleton count={6} />
      </section>
    );
  if (q.isError)
    return <ErrorState error={q.error} retry={() => void q.refetch()} />;
  if (!q.data.title || q.data.data.length === 0) return null;
  return (
    <section id={id} aria-labelledby={`${id}-title`}>
      <div className="mb-5 flex flex-wrap items-center justify-between gap-4">
        <h2 id={`${id}-title`} className="section-title">
          {q.data.title}
        </h2>
        <div className="flex gap-2">
          <button
            className="view-all min-h-11"
            aria-label={`${copy.previous}: ${q.data.title}`}
            onClick={() =>
              row.current?.scrollBy({
                left: -row.current.clientWidth,
                behavior: "smooth",
              })
            }
          >
            ←
          </button>
          <button
            className="view-all min-h-11"
            aria-label={`${copy.next}: ${q.data.title}`}
            onClick={() =>
              row.current?.scrollBy({
                left: row.current.clientWidth,
                behavior: "smooth",
              })
            }
          >
            →
          </button>
          <Link
            className="view-all"
            to={`/store?sort=${id === "new-releases" ? "newest" : id === "deals" ? "price_asc" : "popularity"}`}
          >
            {id === "deals" ? copy.browse : copy.showAll}
            <Icon name="arrow" width="18" />
          </Link>
        </div>
      </div>
      <div ref={row} className="home-product-row">
        {q.data.data.map((p) => (
          <ProductCard key={p.id} product={p} />
        ))}
      </div>
    </section>
  );
}
function BestSellers() {
  const query = useBestSellers();
  return <ProductBlock id="best-sellers" query={query} />;
}
function Deals() {
  const query = useDeals();
  return <ProductBlock id="deals" query={query} />;
}
function NewReleases() {
  const query = useNewReleases();
  return <ProductBlock id="new-releases" query={query} />;
}
export function Homepage() {
  return (
    <div className="container-page space-y-10 pt-0">
      <Banners />
      <Categories />
      <BestSellers />
      <Deals />
      <NewReleases />
      <div className="benefits-strip">
        {[
          ["bolt", copy.delivery, copy.deliveryHelp],
          ["shield", copy.secure, copy.secureHelp],
          ["game", copy.selection, copy.selectionHelp],
        ].map(([icon, title, description]) => (
          <div key={icon} className="flex gap-4">
            <Icon
              name={icon}
              className="shrink-0 text-accent"
              width="32"
              height="32"
            />
            <div>
              <h2 className="font-bold">{title}</h2>
              <p className="mt-2 text-sm leading-relaxed text-muted">
                {description}
              </p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

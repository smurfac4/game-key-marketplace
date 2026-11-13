import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useSearchParams } from "react-router-dom";
import { api } from "../api/client";
import { useCategories } from "../components/Categories";
import { preferences } from "../content/preferences";
import type { Facet, Listing } from "../api/types";
import {
  Button,
  EmptyState,
  ErrorState,
  GridSkeleton,
  Modal,
  ProductCard,
} from "../components/ui";
import { copy, valueLabel } from "../content/copy";
type FacetField = "platform" | "region" | "genre";
function FacetSelect({
  field,
  selected,
  counts,
  all,
}: {
  field: FacetField;
  selected: string;
  counts?: Facet[];
  all?: Facet[];
}) {
  const [value, setValue] = useState(selected);
  const choices = [
    ...new Set([
      ...(all ?? []).map((f) => f.value),
      ...(counts ?? []).map((f) => f.value),
      ...(selected ? [selected] : []),
    ]),
  ].sort();
  return (
    <label className="block">
      <span className="label">{copy[field]}</span>
      <select
        aria-label={copy[field]}
        name={field}
        className="field"
        value={value}
        onChange={(e) => setValue(e.target.value)}
      >
        <option value="">{copy.all}</option>
        {choices.map((v) => (
          <option key={v} value={v}>
            {valueLabel(v)}
            {counts
              ? ` (${counts.find((f) => f.value === v)?.count ?? 0})`
              : ""}
          </option>
        ))}
      </select>
    </label>
  );
}
export function Store() {
  const [params, setParams] = useSearchParams();
  const [drawer, setDrawer] = useState(false);
  const queryString = params.toString();
  const taxonomy = useCategories();
  const query = useQuery({
    queryKey: ["products", queryString],
    queryFn: ({ signal }) =>
      api<Listing>(`/products?${queryString}`, { signal }),
    retry: false,
  });
  function change(values: Record<string, string>) {
    const next = new URLSearchParams(params);
    Object.entries(values).forEach(([k, v]) =>
      v ? next.set(k, v) : next.delete(k),
    );
    if (!("page" in values)) next.delete("page");
    if ("q" in values && !values.q && next.get("sort") === "relevance")
      next.delete("sort");
    setParams(next, { flushSync: true });
  }
  function apply(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = new FormData(e.currentTarget);
    change(
      Object.fromEntries(
        ["platform", "region", "genre", "min_price", "max_price"].map((k) => [
          k,
          String(form.get(k) ?? ""),
        ]),
      ),
    );
    setDrawer(false);
  }
  function filters() {
    return (
      <form
        key={queryString}
        autoComplete="off"
        onSubmit={apply}
        className="space-y-5"
      >
        {(["platform", "region", "genre"] as const).map((field) => (
          <FacetSelect
            key={field}
            field={field}
            selected={params.get(field) ?? ""}
            counts={query.data?.facets[field]}
            all={taxonomy.data?.facets[field]}
          />
        ))}
        <fieldset>
          <legend className="label">
            {copy.priceRange} · {preferences().currency}
          </legend>
          <div className="grid grid-cols-2 gap-3">
            <label>
              <span className="sr-only">{copy.minPrice}</span>
              <input
                className="field"
                type="number"
                name="min_price"
                aria-label={copy.minPrice}
                min="0"
                step="0.01"
                defaultValue={params.get("min_price") ?? ""}
                placeholder="0.00"
              />
            </label>
            <label>
              <span className="sr-only">{copy.maxPrice}</span>
              <input
                className="field"
                type="number"
                name="max_price"
                aria-label={copy.maxPrice}
                min="0"
                step="0.01"
                defaultValue={params.get("max_price") ?? ""}
              />
            </label>
          </div>
        </fieldset>
        <Button type="submit" className="w-full">
          {copy.apply}
        </Button>
        <Button
          type="button"
          variant="ghost"
          className="w-full"
          onClick={() => {
            change({
              platform: "",
              region: "",
              genre: "",
              min_price: "",
              max_price: "",
            });
            setDrawer(false);
          }}
        >
          {copy.clear}
        </Button>
      </form>
    );
  }
  const total = query.data?.pagination.total;
  return (
    <div className="container-page">
      <div className="mb-7 flex flex-wrap items-center justify-between gap-4">
        <h1 className="page-title mb-0">
          {params.get("q")
            ? `${copy.search}: “${params.get("q")}”`
            : copy.store}
        </h1>
        {total !== undefined && (
          <p className="text-muted" aria-live="polite">
            {total} {copy.results}
          </p>
        )}
      </div>
      <div className="mb-6 flex flex-wrap gap-3">
        <form
          className="flex min-w-0 flex-1 gap-2"
          key={params.get("q")}
          onSubmit={(e) => {
            e.preventDefault();
            change({
              q: String(new FormData(e.currentTarget).get("q") ?? "").trim(),
            });
          }}
        >
          <input
            name="q"
            aria-label={copy.search}
            className="field"
            maxLength={200}
            defaultValue={params.get("q") ?? ""}
          />
          <Button className="shrink-0 whitespace-nowrap">{copy.search}</Button>
        </form>
        <label className="flex items-center gap-2 text-sm">
          <span className="whitespace-nowrap">{copy.sort}</span>
          <select
            aria-label={copy.sort}
            className="field"
            value={
              params.get("sort") ??
              (params.get("q") ? "relevance" : "popularity")
            }
            onChange={(e) => change({ sort: e.target.value })}
          >
            {[
              ...(params.get("q") || params.get("sort") === "relevance"
                ? [["relevance", copy.relevance]]
                : []),
              ["price_asc", copy.priceAsc],
              ["price_desc", copy.priceDesc],
              ["popularity", copy.popular],
              ["newest", copy.newest],
              ["name", copy.name],
            ].map(([v, label]) => (
              <option key={v} value={v}>
                {label}
              </option>
            ))}
          </select>
        </label>
        <Button
          variant="secondary"
          className="lg:hidden"
          onClick={() => setDrawer(true)}
        >
          {copy.filters}
        </Button>
      </div>
      <div className="grid gap-7 lg:grid-cols-[240px_1fr]">
        <aside className="panel hidden self-start lg:block">
          <h2 className="mb-5 text-lg font-bold">{copy.filters}</h2>
          {filters()}
        </aside>
        <div className="min-w-0">
          {query.isPending ? (
            <GridSkeleton />
          ) : query.isError ? (
            <ErrorState
              error={query.error}
              retry={() => void query.refetch()}
            />
          ) : (
            <>
              {query.data.data.some((p) => p.fuzzy_match) && (
                <p className="mb-5 rounded-control border border-brand/60 bg-brand/10 p-4 text-sm">
                  {copy.fuzzy}{" "}
                  <Link
                    className="font-bold text-accent underline"
                    to={`/search?q=${encodeURIComponent(query.data.data.find((p) => p.fuzzy_match)!.name)}`}
                  >
                    {query.data.data.find((p) => p.fuzzy_match)!.name}
                  </Link>
                </p>
              )}
              {query.data.data.length === 0 ? (
                <EmptyState title={copy.empty}>
                  <p className="mb-4">{copy.emptyHelp}</p>
                  <Button onClick={() => setParams({})}>{copy.resetAll}</Button>
                </EmptyState>
              ) : (
                <div className="product-grid">
                  {query.data.data.map((p) => (
                    <ProductCard key={p.id} product={p} />
                  ))}
                </div>
              )}
              {query.data.pagination.total_pages > 1 && (
                <nav
                  aria-label={copy.page}
                  className="mt-8 flex flex-wrap items-center justify-center gap-4"
                >
                  <Button
                    variant="secondary"
                    disabled={query.data.pagination.page <= 1}
                    onClick={() =>
                      change({ page: String(query.data.pagination.page - 1) })
                    }
                  >
                    {copy.previous}
                  </Button>
                  <span>
                    {copy.page} {query.data.pagination.page} /{" "}
                    {query.data.pagination.total_pages}
                  </span>
                  <Button
                    variant="secondary"
                    disabled={
                      query.data.pagination.page >=
                      query.data.pagination.total_pages
                    }
                    onClick={() =>
                      change({ page: String(query.data.pagination.page + 1) })
                    }
                  >
                    {copy.next}
                  </Button>
                </nav>
              )}
            </>
          )}
        </div>
      </div>
      {drawer && (
        <Modal
          drawer
          open={drawer}
          onClose={() => setDrawer(false)}
          title={copy.filters}
        >
          {filters()}
        </Modal>
      )}
    </div>
  );
}

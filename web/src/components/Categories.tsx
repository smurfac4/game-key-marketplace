import { useQuery } from "@tanstack/react-query";
import { Link, useLocation } from "react-router-dom";
import { api } from "../api/client";
import type { Listing } from "../api/types";
import { Icon } from "./Icon";
import { copy, valueLabel } from "../content/copy";
export function useCategories() {
  return useQuery({
    queryKey: ["catalog-categories"],
    queryFn: ({ signal }) => api<Listing>("/products?page_size=1", { signal }),
    staleTime: 300000,
  });
}
export function Categories({ compact = false }: { compact?: boolean }) {
  const q = useCategories();
  const location = useLocation();
  function destination(field: string, value: string) {
    const params = new URLSearchParams(
      ["/store", "/search"].includes(location.pathname) ? location.search : "",
    );
    params.set(field, value);
    params.delete("page");
    return "/store?" + params.toString();
  }
  const entries = [
    ...(q.data?.facets.platform.map((f) => ({
      ...f,
      field: "platform",
      icon: "desktop",
    })) ?? []),
    ...(q.data?.facets.genre.map((f) => ({
      ...f,
      field: "genre",
      icon:
        f.value === "Racing"
          ? "bolt"
          : f.value === "Strategy"
            ? "grid"
            : "game",
    })) ?? []),
  ];
  if (!entries.length) return null;
  return (
    <nav
      aria-label={copy.categories}
      className={compact ? "category-menu" : "category-rail"}
    >
      {(compact ? ["platform", "genre"] : [""]).map((group) => (
        <div key={group} className={compact ? "" : "contents"}>
          {compact && (
            <p className="px-3 pt-3 pb-1 text-xs font-semibold uppercase tracking-wide text-muted">
              {copy[group as "platform" | "genre"]}
            </p>
          )}
          {entries
            .filter((e) => !group || e.field === group)
            .map((e) => (
              <Link
                key={`${e.field}-${e.value}`}
                to={destination(e.field, e.value)}
                aria-current={
                  ["/store", "/search"].includes(location.pathname) &&
                  new URLSearchParams(location.search).get(e.field) === e.value
                    ? "true"
                    : undefined
                }
                className="category-link aria-[current=true]:bg-raised aria-[current=true]:text-accent"
              >
                <Icon name={e.icon} />
                <span>{valueLabel(e.value)}</span>
              </Link>
            ))}
        </div>
      ))}
    </nav>
  );
}

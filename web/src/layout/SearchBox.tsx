import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { api } from "../api/client";
import type { Suggestion } from "../api/types";
import { copy } from "../content/copy";
const storageKey = "keyspace.recent-searches";
function recentSearches(): string[] {
  try {
    const v: unknown = JSON.parse(localStorage.getItem(storageKey) ?? "[]");
    return Array.isArray(v)
      ? v
          .filter((x): x is string => typeof x === "string" && x.length <= 200)
          .slice(0, 5)
      : [];
  } catch {
    return [];
  }
}
export function SearchBox() {
  const navigate = useNavigate();
  const [text, setText] = useState("");
  const [debounced, setDebounced] = useState("");
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const [recent, setRecent] = useState(recentSearches);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(text.trim()), 250);
    return () => clearTimeout(id);
  }, [text]);
  useEffect(() => {
    function close(e: PointerEvent) {
      if (!ref.current?.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, []);
  const query = useQuery({
    queryKey: ["suggest", debounced],
    queryFn: ({ signal }) =>
      api<{
        data: Suggestion[];
      }>(`/search/suggest?q=${encodeURIComponent(debounced)}`, { signal }),
    enabled: debounced.length >= 2,
    staleTime: 30000,
    retry: false,
  });
  const suggestions = debounced === text.trim() ? (query.data?.data ?? []) : [];
  const options = text.trim()
    ? suggestions.map((p) => ({ label: p.name, path: `/products/${p.slug}` }))
    : recent.map((q) => ({
        label: q,
        path: `/search?q=${encodeURIComponent(q)}`,
      }));
  const expanded = open && (options.length > 0 || text.trim().length >= 2);
  function remember(value: string) {
    const next = [value, ...recent.filter((q) => q !== value)].slice(0, 5);
    setRecent(next);
    try {
      localStorage.setItem(storageKey, JSON.stringify(next));
    } catch {}
  }
  function choose(index: number) {
    const option = options[index];
    const term = text.trim();
    if (option) {
      remember(option.label);
      navigate(option.path);
    } else if (term) {
      remember(term);
      navigate(`/search?q=${encodeURIComponent(term)}`);
    }
    setOpen(false);
    setActive(-1);
  }
  return (
    <div ref={ref} className="relative w-full">
      <form
        role="search"
        onSubmit={(e) => {
          e.preventDefault();
          choose(active);
        }}
      >
        <label className="sr-only" htmlFor="header-search">
          {copy.search}
        </label>
        <div className="flex rounded-control border-2 border-brand bg-page focus-within:border-accent">
          <input
            id="header-search"
            type="search"
            autoComplete="off"
            maxLength={200}
            role="combobox"
            aria-autocomplete="list"
            aria-expanded={expanded}
            aria-controls={expanded ? "search-options" : undefined}
            aria-activedescendant={
              open && active >= 0 ? `search-option-${active}` : undefined
            }
            placeholder={copy.searchPlaceholder}
            value={text}
            onFocus={() => setOpen(true)}
            onChange={(e) => {
              setText(e.target.value);
              setActive(-1);
              setOpen(true);
            }}
            onKeyDown={(e) => {
              if (e.key === "Escape") {
                setOpen(false);
                setActive(-1);
              }
              if (e.key === "ArrowDown") {
                e.preventDefault();
                setOpen(true);
                setActive((i) =>
                  options.length ? (i + 1) % options.length : -1,
                );
              }
              if (e.key === "ArrowUp") {
                e.preventDefault();
                setOpen(true);
                setActive((i) =>
                  options.length ? (i <= 0 ? options.length - 1 : i - 1) : -1,
                );
              }
            }}
            className="min-h-12 min-w-0 flex-1 bg-transparent px-4 text-sm outline-none"
          />
          <button
            type="submit"
            aria-label={copy.search}
            className="px-4 text-xl text-accent"
          >
            ⌕
          </button>
        </div>
      </form>
      {open && (options.length > 0 || text.trim().length >= 2) && (
        <div className="absolute inset-x-0 top-full z-40 mt-2 overflow-hidden rounded-control border border-line bg-surface shadow-2xl">
          <div className="flex justify-between px-4 py-2 text-xs text-muted">
            <span>{text ? copy.suggestions : copy.recent}</span>
            {!text && (
              <button
                onClick={() => {
                  setRecent([]);
                  try {
                    localStorage.removeItem(storageKey);
                  } catch {}
                }}
              >
                {copy.clearRecent}
              </button>
            )}
          </div>
          <ul id="search-options" role="listbox" aria-label={copy.suggestions}>
            {options.map((o, i) => (
              <li
                id={`search-option-${i}`}
                key={o.path}
                role="option"
                aria-selected={active === i}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => choose(i)}
                className={`cursor-pointer px-4 py-3 text-sm hover:bg-raised ${i === active ? "bg-raised text-accent" : ""}`}
              >
                {o.label}
              </li>
            ))}
          </ul>
          {text.trim() && (
            <button
              onClick={() => choose(-1)}
              className="w-full border-t border-line px-4 py-3 text-left text-sm text-accent"
            >
              {copy.allResults} →
            </button>
          )}
        </div>
      )}
    </div>
  );
}

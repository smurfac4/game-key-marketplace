import { preferences, setPreferences } from "../content/preferences";
import { useQueryClient } from "@tanstack/react-query";
import { LanguageFlag } from "../components/LanguageFlag";
import { Categories } from "../components/Categories";
import { Icon } from "../components/Icon";
import {
  Link,
  NavLink,
  Outlet,
  useNavigate,
  useLocation,
} from "react-router-dom";
import { AccountMenu } from "./AccountMenu";
import { useCart } from "../hooks/cart";
import { copy } from "../content/copy";
import { SearchBox } from "./SearchBox";
export function Layout() {
  const settings = preferences();
  const client = useQueryClient();
  function update(
    language: typeof settings.language,
    currency: typeof settings.currency,
  ) {
    client.removeQueries({ predicate: (q) => q.queryKey[0] !== "me" });
    setPreferences({ language, currency });
    if (location.pathname === "/store" || location.pathname === "/search") {
      const p = new URLSearchParams(location.search);
      p.set("currency", currency);
      navigate(location.pathname + "?" + p.toString(), { replace: true });
    }
  }
  const cart = useCart();
  const navigate = useNavigate();
  const location = useLocation();
  const count = cart.data?.data.items.reduce((n, i) => n + i.quantity, 0) ?? 0;
  return (
    <div className="flex min-h-screen flex-col">
      <a
        className="fixed left-4 top-2 z-50 -translate-y-24 rounded bg-accent p-3 text-page focus:translate-y-0"
        href="#main-content"
      >
        {copy.skip}
      </a>
      <div className="demo-strip">
        <span className="mr-2 rounded-sm bg-accent px-1.5 py-0.5 text-[10px] font-black text-[#221047]">
          {copy.demo}
        </span>
        {copy.demoBar}
        <Link to="/help" className="ml-3 underline underline-offset-4">
          {copy.help}
        </Link>
      </div>
      <header className="site-header">
        <div className="mx-auto grid max-w-content grid-cols-[1fr_auto] items-center gap-4 px-4 py-5 sm:px-6 lg:grid-cols-[auto_1fr_auto_auto] lg:gap-8 lg:px-8">
          <Link
            to="/"
            aria-label={copy.home}
            className="flex items-center gap-2 text-2xl font-black tracking-tight sm:text-3xl"
          >
            <span
              aria-hidden
              className="grid h-9 w-9 rotate-[-8deg] place-items-center rounded-lg bg-accent text-[#221047]"
            >
              K
            </span>
            {copy.brand}
            <span className="sr-only">{copy.brandNote}</span>
          </Link>
          <div className="order-3 col-span-2 lg:order-none lg:col-span-1">
            <SearchBox />
          </div>
          <div className="order-4 col-span-2 flex items-center justify-end gap-3 text-xs lg:order-none lg:col-span-1">
            <LanguageFlag language={settings.language} />
            <label className="sr-only" htmlFor="language">
              Language
            </label>
            <select
              id="language"
              className="max-w-28 rounded border border-line bg-surface p-2"
              value={settings.language}
              onChange={(e) =>
                update(
                  e.target.value as typeof settings.language,
                  settings.currency,
                )
              }
            >
              <option value="en">English</option>
              <option value="ru">Русский</option>
            </select>
            <label className="sr-only" htmlFor="currency">
              Currency
            </label>
            <select
              title={copy.fxHelp}
              id="currency"
              className="rounded border border-line bg-surface p-2"
              value={settings.currency}
              onChange={(e) =>
                update(
                  settings.language,
                  e.target.value as typeof settings.currency,
                )
              }
            >
              <option>EUR</option>
              <option>USD</option>
              <option>GBP</option>
            </select>
            <span className="text-[10px] text-muted">{copy.demoFx}</span>
          </div>
          <nav
            aria-label={copy.account}
            className="flex items-center gap-1 sm:gap-2"
          >
            <Link
              to="/wishlist"
              className="button button-ghost px-2"
              aria-label={copy.favorites}
            >
              <Icon name="heart" width="23" />
            </Link>
            <Link
              to="/cart"
              className="button button-ghost relative"
              aria-label={`${copy.cart} (${count})`}
            >
              <svg
                aria-hidden
                width="23"
                height="23"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
              >
                <path
                  d="M2 3h3l3 13h11l3-10H6M9 20h.01M18 20h.01"
                  strokeLinecap="round"
                />
              </svg>
              {count > 0 && (
                <span className="absolute -right-1 -top-1 rounded-full bg-accent px-1.5 text-xs text-page">
                  {count}
                </span>
              )}
            </Link>
            <AccountMenu />
          </nav>
        </div>
        <nav aria-label={copy.menu} className="main-nav">
          <details
            className="categories-dropdown"
            onClick={(e) => {
              if ((e.target as HTMLElement).closest("a"))
                e.currentTarget.open = false;
            }}
          >
            <summary className="flex cursor-pointer list-none items-center gap-2 whitespace-nowrap">
              <Icon name="grid" width="18" />
              {copy.categories}
            </summary>
            <div className="category-popover">
              <Categories compact />
            </div>
          </details>
          {[
            ["/", copy.home],
            ["/store", copy.store],
            ["/store?max_price=10&sort=price_asc", copy.cheapGames],
            ["/store?sort=newest", copy.newest],
            ["/orders", copy.orders],
          ].map(([to, label]) => (
            <NavLink
              key={to}
              to={to}
              end
              className={({ isActive }) =>
                isActive &&
                (!to.includes("?") ||
                  [...new URLSearchParams(to.split("?")[1])].every(
                    ([key, value]) =>
                      new URLSearchParams(location.search).get(key) === value,
                  ))
                  ? "text-accent"
                  : "text-muted hover:text-ink"
              }
            >
              {label}
            </NavLink>
          ))}
        </nav>
      </header>
      <main id="main-content" className="flex-1">
        <Outlet />
      </main>
      <footer className="mt-12 border-t border-line/30 bg-surface">
        <div className="container-page flex flex-col justify-between gap-5 text-sm text-muted sm:flex-row">
          <div>
            <p className="mb-2 font-bold text-ink">{copy.brand}</p>
            <p>{copy.footer}</p>
          </div>
          <nav aria-label={copy.shop} className="footer-column">
            <h2>{copy.shop}</h2>
            <Link to="/store">{copy.browse}</Link>
            <Link to="/store?sort=newest">{copy.newest}</Link>
            <Link to="/store?max_price=10&sort=price_asc">
              {copy.cheapGames}
            </Link>
          </nav>
          <nav aria-label={copy.yourAccount} className="footer-column">
            <h2>{copy.yourAccount}</h2>
            <Link to="/login">{copy.login}</Link>
            <Link to="/cart">{copy.cart}</Link>
            <Link to="/orders">{copy.orders}</Link>
          </nav>
          <nav aria-label={copy.help} className="footer-column">
            <h2>{copy.help}</h2>
            <Link to="/help">{copy.helpIntro}</Link>
            <span className="flex items-center gap-2">
              <Icon name="shield" width="18" />
              {copy.demo}
            </span>
          </nav>
        </div>
      </footer>
    </div>
  );
}

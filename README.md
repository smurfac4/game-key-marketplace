# Game-key marketplace

Public repository: [smurfac4/game-key-marketplace](https://github.com/smurfac4/game-key-marketplace).

An intern-assignment marketplace with a PostgreSQL catalog, independent homepage blocks, typo-tolerant search, cookie authentication, carts, transactional mock checkout, and private digital-key orders.

- `web/`: React, TypeScript, Vite, Tailwind CSS, React Router, TanStack Query.
- `api/`: Go REST server using chi and a pgx connection pool.
- `ai-logs/`: sanitized AI assistance records.
- `CLAUDE.md`: contribution rules. `DECISIONS.md`: scope trade-offs.

## Run with Docker

Requires Docker Engine and Compose v2.

```sh
cp .env.example .env
docker compose up --build -d
docker compose run --rm --no-deps api migrate up
docker compose run --rm --no-deps api seed
```

Web: http://localhost:5173. API: http://localhost:8080 by default. If a port is already in use, change `WEB_PORT`, `API_PORT`, or `POSTGRES_PORT` in `.env` before starting Compose. The Vite proxy uses the internal API port and needs no change.

```sh
curl --fail http://localhost:8080/api/health/live
curl --fail http://localhost:5173/api/health/ready
docker compose logs -f
```

The readiness request through Vite verifies web → API → PostgreSQL connectivity. Homepage blocks require the seed data described below. Rebuild with `docker compose up --build -d` after edits. Stop with `docker compose down`; data remains in the named PostgreSQL volume. `docker compose down -v` also deletes that local database.

## Run applications locally

Requires Node.js 24+, npm, Go 1.26+, and Docker for PostgreSQL.

```sh
cp .env.example .env # only if .env does not already exist
npm ci
docker compose up -d db
```

In one terminal, using credentials matching `.env`:

```sh
cd api
export DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/marketplace?sslmode=disable'
go run ./cmd/server
```

In another terminal, from the repository root:

```sh
npm run dev
```

Go does not automatically read `.env`. `DATABASE_URL` is required; `HTTP_ADDR` defaults to `:8080`. Vite's optional `API_PROXY_TARGET` defaults to `http://localhost:8080`. Root `.env` configures Compose's database credentials and published ports; see `.env.example`. Local example credentials are for development only.

## Validate

```sh
npm ci
npm run build
cd api
go test ./...
go vet ./...
go build -o bin/server ./cmd/server
```

`npm run typecheck` checks TypeScript separately. Seed generation has unit tests; database integration tests are opt-in (see below). Dependency resolutions are recorded in `package-lock.json` and `api/go.sum`.

Tailwind uses the official [Vite plugin setup](https://tailwindcss.com/docs/installation/using-vite); Node requirements follow [Vite's guide](https://vite.dev/guide/).

## Database migrations and seed

Migrations use [golang-migrate](https://github.com/golang-migrate/migrate) and are embedded in the `migrate` command. The API image includes `server`, `migrate`, and `seed`. Start PostgreSQL and build the updated image first:

```sh
docker compose up -d db
docker compose build api
docker compose run --rm --no-deps api migrate up
docker compose run --rm --no-deps api seed
```

The seed inserts exactly 150 demo product listings (50 game titles in regional/store variants) and 10 simulated sellers, with 2–8 offers per product and 5–30 fake keys per offer. Stock matches the available keys. It is deterministic and atomic, and refuses to run if products or sellers already exist. The seed also includes homepage banners and a disabled demo user with paid sales; no carts are seeded. Game covers and hero images are served from the local media directory.

For local Go commands, set `DATABASE_URL` as above and run from `api/`:

```sh
go run ./cmd/migrate up
go run ./cmd/seed
```

To roll back **one migration and delete its tables and data**, use `go run ./cmd/migrate down` or `docker compose run --rm --no-deps api migrate down`. Do this only on a disposable database or after a deliberate backup. The shared `pg_trgm` extension remains installed. A migration failure may leave golang-migrate's version marked dirty: inspect and fix the cause before changing migration state; do not blindly force a version.

`go test ./...` runs unit tests. To include database integration tests, create a separate empty PostgreSQL database with a name ending in `_test` and pass its connection URL:

```sh
TEST_DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/marketplace_test?sslmode=disable' go test ./... -count=1
```

The test refuses a nonempty database, creates and rolls back the schema, and leaves golang-migrate's metadata table behind. Drop and recreate that disposable test database before another integration run. `DATABASE_URL` is for runtime commands; `TEST_DATABASE_URL` is exclusively for these tests. Schema rationale and deferred checkout invariants are in `DECISIONS.md`.

## Product listing API

```sh
curl 'http://localhost:8081/api/products?q=astral&platform=Steam&min_price=5&max_price=50&sort=price_asc&page=1&page_size=24'
```

Use the API port from your `.env` (8080 by default), or `/api/products` through the web proxy.

| Parameter | Meaning / default |
| --- | --- |
| `platform`, `region`, `genre` | Exact, case-sensitive single value; optional, up to 80 characters |
| `q` | Case-insensitive partial/typo-tolerant name search, up to 200 characters |
| `min_price`, `max_price` | Inclusive bounds on the lowest available price; nonnegative decimal, at most 2 decimal places and 10 whole digits |
| `currency` | Three uppercase letters; default `EUR`; no exchange conversion |
| `sort` | `relevance` (default with `q`), `price_asc`, `price_desc`, `popularity` (default without `q`), `newest`, `name` |
| `page` | 1–1,000,000; default 1 |
| `page_size` | 1–100; default 24 |

Prices use positive-stock offers and apply discounts. Products without available offers in the selected currency are omitted. Popularity counts units in paid orders; newest uses release date with unknown dates last. Each sort breaks ties by product ID.

Successful response shape (illustrative):

```json
{
  "data": [{
    "id": 1, "name": "Astral Frontier", "slug": "demo-game-001",
    "description": "Fictional game", "platform": "Steam", "region": "Europe",
    "genre": "RPG", "image_url": "https://placehold.co/600x800/png",
    "release_date": "2024-01-01", "price": "9.99", "currency": "EUR", "popularity": 0
  }],
  "pagination": {"page": 1, "page_size": 24, "total": 1, "total_pages": 1},
  "facets": {
    "platform": [{"value": "Steam", "count": 1}],
    "region": [{"value": "Europe", "count": 1}],
    "genre": [{"value": "RPG", "count": 1}]
  }
}
```

Facet counts count products before pagination, with search, currency and price bounds applied. Each facet ignores its own selected value while respecting the other selected facets, so alternative choices remain available. Zero-result combinations can therefore still return useful alternative facet values; a page beyond the end retains the overall total and facets. `release_date` may be null. Prices are decimal strings.

Invalid queries return HTTP 400, for example:

```json
{"error":{"code":"invalid_query","message":"must be less than or equal to max_price","field":"min_price"}}
```

Other errors use the same envelope with codes `not_found` (404), `method_not_allowed` (405), `internal_error` (500), `unavailable` (503 readiness), and `timeout` (504). Unknown/repeated/empty parameters are rejected; unknown valid facet values simply match no products.

Product integration tests use their own empty database ending in `_test`, separate from seed integration tests:

```sh
TEST_PRODUCTS_DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/products_test?sslmode=disable' go test ./internal/products -count=1
```

As with seed integration tests, migration metadata remains; recreate the disposable database before rerunning. Normal `go test ./...` runs all parser, handler, and seed unit tests without a database.


## Search suggestions

```sh
curl 'http://localhost:8081/api/search/suggest?q=cyberpnk'
curl 'http://localhost:8081/api/products?q=witcer'
```

Suggestions accept only required `q` and optional `currency` (default EUR). They return at most eight available products:

```json
{"data":[{"id":123,"name":"Cyberpunk 2077","slug":"cyberpunk-2077","image_url":"https://placehold.co/600x800/png"}]}
```

This is an illustrative response. The demo catalog contains real titles with stable demo slugs; dedicated tests also insert Cyberpunk/FIFA/Witcher search fixtures into a disposable database.

Both endpoints combine literal partial-name matches with pg_trgm typo tolerance. Ranking prefers exact names, then prefixes, substrings, and fuzzy matches; similarity scores and product ID break ties. Explicit listing sorts override relevance. Search text treats `%`, `_`, and backslash literally.

Fuzzy matching requires at least four characters. One-character suggestion queries return `{"data":[]}` without querying PostgreSQL; two/three-character suggestions use literal matching. Missing/blank, duplicate, or oversized `q` is a 400. Thresholds (0.30 whole-name, 0.50 word similarity), ranking, and performance trade-offs are explained in `DECISIONS.md`. Suggestions share existing JSON errors and have a one-second database deadline. Run the product integration tests described above to exercise the typo fixtures and suggestion cap.

## Product details

```sh
curl 'http://localhost:8081/api/products/demo-game-001'
```

`GET /api/products/{slug}` returns `data` with all product fields (`id`, `name`, `slug`, `description`, `platform`, `region`, `genre`, `image_url`, nullable `release_date`) and an `offers` array. Each offer has this shape:

```json
{
  "id": 12,
  "seller": {"id": 2, "name": "Demo Seller 02", "rating": 4.75},
  "price": "20.00",
  "discount_price": "15.00",
  "effective_price": "15.00",
  "currency": "EUR",
  "stock": 10
}
```

All offers are returned, including stock-zero offers. They sort by effective price ascending (discounted price when present), then currency and offer ID. `discount_price` is null when there is no discount. All currencies are included, so ordering compares nominal amounts without currency conversion; always display each offer's currency.

The top-level `similar_products` array contains up to eight available same-genre products, preferring matching platform and region, with ID as the final tie-breaker. Each includes `id`, `name`, `slug`, `image_url`, `platform`, `region`, and `genre`. The current product and unavailable products are excluded.

An existing product with no offers returns 200 with `offers: []`; no recommendations produces `similar_products: []`. Unknown slugs return 404:

```json
{"error":{"code":"not_found","message":"product not found"}}
```

Product detail cases run with the same `TEST_PRODUCTS_DATABASE_URL` integration command documented above.

## Authentication

Generate a signing secret into the ignored `.env` before Compose starts (do this once; changing it invalidates existing logins):

```sh
openssl rand -hex 32
```

Set `JWT_SECRET` to that value, `COOKIE_SECURE=false` for local HTTP, `COOKIE_SAMESITE=lax`, and `AUTH_ALLOWED_ORIGINS` to your actual frontend origin, e.g. `http://localhost:5174` for this workspace. Apply the new session migration and restart:

```sh
docker compose build api
docker compose run --rm --no-deps api migrate up
docker compose up -d api
```

Local `go run ./cmd/server` requires these values exported in addition to `DATABASE_URL`; it does not load `.env`. Go defaults to Secure cookies, while Compose's development default is false. Keep `.env` out of version control.

| Endpoint | Request | Success |
| --- | --- | --- |
| `POST /api/auth/register` | JSON `{ "email": "you@example.com", "password": "a-long-unique-password" }` | 201 `{ "data": { "id", "email", "created_at" } }`; then log in |
| `POST /api/auth/login` | Same JSON shape | 200 public user plus HttpOnly JWT cookie |
| `POST /api/auth/logout` | No body needed | 204; revoke session and clear cookie |
| `GET /api/me` | Session cookie | 200 public user, or 401 |

All write requests require `X-CSRF-Protection: 1`. Registration/login require `Content-Type: application/json`. Passwords must be at least 12 characters and no more than 72 UTF-8 bytes. Registration returns 409 for an existing email. Failed login uses a generic 401. Authentication attempts share a limit of 10 per peer IP per 15 minutes; exhausted limits return 429 and Retry-After. Standard JSON error envelopes apply.

Example login (use your actual local API port and registered credentials):

```sh
curl -c /tmp/marketplace-cookies.txt \
  -H 'Content-Type: application/json' -H 'X-CSRF-Protection: 1' \
  -d '{"email":"you@example.com","password":"a-long-unique-password"}' \
  http://localhost:8081/api/auth/login
curl -b /tmp/marketplace-cookies.txt http://localhost:8081/api/me
curl -b /tmp/marketplace-cookies.txt -c /tmp/marketplace-cookies.txt \
  -H 'X-CSRF-Protection: 1' -X POST http://localhost:8081/api/auth/logout
```

For cross-domain deployment use HTTPS, `COOKIE_SECURE=true`, `COOKIE_SAMESITE=none`, and `AUTH_ALLOWED_ORIGINS=https://your-frontend.example` (exact origins, comma-separated; no trailing slash/wildcard). Browser requests must use `credentials: 'include'` and the custom CSRF header for writes. Third-party-cookie blocking may require same-site hosting/proxying. Cookie deletion uses the same attributes as login.

JWTs last 24 hours and contain no password. PostgreSQL session checks make logout revoke replayed tokens. `/api/checkout` and `/api/orders`, including nested paths, require authentication; their handlers enforce ownership using the authenticated user ID. Registration, login and persistent-session frontend pages are implemented. Email verification/reset/MFA remain outside the assignment.

The limiter uses the socket peer IP, never arbitrary forwarded headers. Behind Vite/a reverse proxy users may share a budget. Multi-instance deployments need a shared limiter and trusted-proxy configuration; see `DECISIONS.md`.

Run real HTTP/database auth tests against a separate empty disposable database:

```sh
TEST_AUTH_DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/auth_test?sslmode=disable' go test ./internal/httpapi -count=1
go test -race ./internal/auth ./internal/httpapi
```

Recreate the test database before each integration run. Ordinary unit tests require no PostgreSQL. Migration `000002` adds `auth_sessions`; rolling it back logs out every session while preserving users/catalog data.

## Modular homepage

The React homepage independently loads these public endpoints:

| Endpoint | Data |
| --- | --- |
| `/api/home/banners` | Up to 8 active, currently scheduled banners, ordered by position |
| `/api/home/best-sellers` | Up to 12 available products ranked by units in paid orders |
| `/api/home/deals` | Up to 12 products with available discounted offers, ranked by percentage saved |
| `/api/home/new-releases` | Up to 12 latest released products (no future/unknown dates) |

Each returns `{ "title": "...", "data": [...] }`. Titles and enabled state come from `homepage_blocks`; no active configuration means an empty block. Product cards include ID/name/slug/image/platform/region, decimal-string price, nullable original price, currency, and release date. Best sellers also return `units_sold`. Use `?currency=USD` to select available prices in another currency; default EUR. Banners return ID/title/image URL/link URL. Responses are independent and use standard JSON errors.

The frontend renders only returned titles, banners, and products. Empty blocks disappear; loading/error/retry states affect only their own section. Product cards link to product details and seller comparisons. Banner links scroll to the corresponding homepage sections.

Fresh `seed` now includes homepage fixtures. For an existing demo catalog, run the additive, repeat-safe command:

```sh
docker compose build api web
docker compose run --rm --no-deps api seed -homepage
docker compose up -d api web
```

This adds 3 banners, 4 block definitions, and 12 paid orders (24 allocated fake keys), with stock kept consistent. The demo sales user cannot log in. It refuses conflicting homepage configuration or a non-demo catalog. It does not reset your database.

Each block has its own Go package, repository, and handler under `api/internal/home/`. `DECISIONS.md` explains why the deployment is a modular monolith and how to extract a block into an independent service with an event-fed read model.

```sh
TEST_HOME_DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/home_test?sslmode=disable' go test ./internal/home/... -count=1
```

## Cart, checkout and orders

All these endpoints require a session cookie; writes also require the CSRF header described above.

| Endpoint | Request / behavior |
| --- | --- |
| `GET /api/cart` | Current items and server-calculated totals |
| `POST /api/cart/items` | `{ "offer_id": 1, "quantity": 1 }`, adds to existing quantity |
| `PATCH /api/cart/items/:id` | `{ "quantity": 2 }`, replaces quantity |
| `DELETE /api/cart/items/:id` | Removes an item owned by the current user |
| `POST /api/checkout` | `{ "card_number": "4242424242424242" }`, plus `Idempotency-Key` header (16–80 letters, digits, hyphens or underscores) |
| `GET /api/orders?page=1` | Own orders, 20 per page |
| `GET /api/orders/:id` | Own order snapshots and purchased keys; another user's order returns 404 |

Mock cards: **4242424242424242 succeeds**, **4000000000000002 declines**. All other numbers are rejected. Do not use real payment information. A decline rolls back orders, key allocation and stock changes, preserving the cart. Prices always come from the database; client-supplied prices/owners are rejected. Retrying a successful checkout with the same idempotency key returns the same order. Carts allow up to 50 offers, 10 keys per offer and one currency. Adding an item does not reserve stock.

Migration `000003` adds checkout idempotency. Existing installations should run `docker compose run --rm --no-deps api migrate up` after rebuilding the API.

## Frontend

Routes: `/`, `/store`, `/search`, `/products/:slug`, `/register`, `/login`, `/cart`, `/checkout`, `/orders`, `/orders/:id`. Protected pages redirect to login and return to the requested page. Sessions restore through `/api/me`; logout removes private cached data.

Store filters, sorting, search and pagination live in URL query parameters. Mobile filters use a focus-trapping drawer. Header search debounces requests, supports arrows/Enter/Escape, and stores the last five searches locally. Fuzzy results show a server-derived hint. Homepage blocks have separate query hooks, skeletons and error states. Catalog content, banner text and section titles come exclusively from the API; operational interface copy is centralized in `web/src/content/copy.ts`.

Design tokens live in `web/tailwind.config.js` and `web/src/styles.css`; reusable Button, ProductCard, PriceTag, Badge, Skeleton and Modal components live in `web/src/components/ui.tsx`. The original Keyspace branding and project-authored illustrations do not copy Eneba assets.

## Full verification

```sh
npm ci
npm run lint
npm run format:check
npm run build
npm audit
(cd api && go test -race ./... && go vet ./...)
./scripts/test-e2e.sh
```

PostgreSQL integration tests run when `TEST_DATABASE_URL`, `TEST_AUTH_DATABASE_URL`, `TEST_PRODUCTS_DATABASE_URL`, `TEST_HOME_DATABASE_URL`, and `TEST_COMMERCE_DATABASE_URL` point to **five separate empty disposable databases whose names end in `_test`**. Each suite migrates its database and rolls back its application schema; migration metadata remains, so recreate these disposable databases before rerunning. Without these variables the database-dependent tests skip; unit tests still run. Never use the development or production database for these tests.

The Docker E2E script builds the application and creates a separate `eneba-e2e` Compose project, database volume, credentials and seeded catalog. It runs Chromium Playwright tests for search/URL navigation, registration/session restoration, cart quantities, declined/successful checkout, key copying, order ownership, logout, independent homepage failures, and mobile filters/empty/404 states. Axe checks cover the principal pages. Test containers and their database volume are removed on exit. Screenshots and the HTML report remain under `web/test-results` and `web/playwright-report`. Docker and npm dependencies are required; no local browser installation is needed.

## Showcase artwork and visual demo

The catalog now presents 50 recognizable PC game titles as 150 regional/store listings, with locally served original generated genre covers and a cinematic portal hero. All offers, sellers, prices and keys are simulated. Current artwork provenance is listed in `ai-logs/ASSETS.md`; original PNG artwork and neutral SVG fallbacks are included in the source. The checked-in images are included in the web build, so browsing does not require external image hosts.

Upgrade an existing seeded demo without resetting orders, carts, stock or keys:

```sh
docker compose build api web
docker compose run --rm --no-deps api migrate up
docker compose run --rm --no-deps api seed -showcase
docker compose up -d --wait api web
```

A fresh `seed` already includes the showcase. Stable `demo-game-001`-style URLs remain compatible with previous demo links. The `/help` page explains the complete mock purchase journey. `DECISIONS.md` records the direct Eneba comparison, changes and remaining differences.

## Backup and restore

Use a custom-format archive and keep backups outside the checkout with restricted permissions. These commands use container environment credentials without printing them. Replace `marketplace_restore` with a new, empty destination database; never restore over a live database.

```sh
umask 077
docker compose exec -T db sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' > /private/tmp/marketplace-backup.dump
docker compose exec -T db sh -c 'createdb -U "$POSTGRES_USER" marketplace_restore'
docker compose exec -T db sh -c 'pg_restore -U "$POSTGRES_USER" -d marketplace_restore --exit-on-error' < /private/tmp/marketplace-backup.dump
```

`bash scripts/final-audit-db.sh` reproduces unit/integration tests, parallel migrations on seeded data, and dump/restore on disposable `eneba_final_*_test` databases, removed on exit. It requires local Go and this project's running development PostgreSQL container with the documented public development credentials. It never changes the development catalog. `bash scripts/final-audit-performance.sh` reproduces the 51x SQL-plan experiment and removes its database. Ensure those reserved disposable names do not already contain unrelated data.

Current submission uses original generated PNG artwork and neutral SVG fallbacks; the earlier Steam JPEG artwork and downloader were removed. The closeout pass tested the refresh on a disposable copy, then refreshed the local demo and rebuilt/restarted its containers. The runtime now serves neutral art. Historical Docker build-cache layers may still contain earlier files; they are not part of the submitted tree or current runtime. See `ai-logs/FINAL-AUDIT.md` for exact checks and omissions.

## Coursework and closeout reproduction

This repository is non-commercial coursework, with simulated inventory, payments and non-redeemable keys. It is not affiliated with Eneba or game publishers. Current artwork is original generated PNG with neutral SVG fallbacks; see `ai-logs/ASSETS.md`.

`python3 scripts/closeout-repro.py` copies Git-filtered source into a fresh directory (excluding ignored environment/dependency files), supplies isolated configuration in process memory, follows Compose build/up → migrations → seed, smoke-checks HTTP and real-browser catalog/details, and removes the temporary project, volume and source directory. No commit is required. It requires Docker and the existing Playwright runner/dependencies in the working checkout.

`bash scripts/closeout-copy-timeout.sh` streams the local development DB directly into a disposable copy, prints aggregate counts only, tests showcase refresh, and reproduces a real twelve-second HTTP pool timeout on another disposable database; both are removed on exit. It assumes the public development credentials used by the local audit database, not arbitrary production credentials.

The full `./scripts/test-e2e.sh` runs six general tests, restarts only the disposable API to reset its intentionally shared proxy-IP auth budget, then runs two complete keyboard purchase tests. Each group retains the production rate-limit policy. Playwright traces are disabled to avoid saving session cookies/request credentials to trace artifacts.

## Language and order currency

English is the default language. The header offers English/Russian switching and EUR/USD/GBP selection on desktop and mobile; both preferences persist after reload. The demo uses fixed server-side rates: 1 EUR = 1.10 USD = 0.85 GBP. Cart prices, mock payment amount and persisted order currency follow the selection; existing orders keep their original recorded amounts. This is demonstration FX, not live exchange-rate data. Catalog game names/descriptions remain their original source text.

The UI sends `X-Display-Currency` for catalog/cart reads; direct API consumers can use that header to request conversion, or omit it to retain original inventory-currency filtering. Checkout JSON now also accepts `currency` (EUR/USD/GBP, default EUR). Client prices are still rejected. Original vector illustrations can be regenerated with `python3 scripts/generate-demo-art.py`.

The latest visual revision uses three original generated genre cover illustrations without DEMO markings and an original portal promotion background. These are author illustrations, not official publisher covers. Promotion headlines follow the selected English/Russian language; select chevrons have a consistent inset. `scripts/generate-demo-art.py` regenerates only the earlier SVG fallback illustrations, not the primary generated PNG files (which are included in source).

## Repository history

The project existed as workspace files before Git commits were recorded. The initial repository history is an honest grouped import of that existing source: configuration, schema/seed, catalog/home modules, auth/commerce API, frontend, then tests/audit documentation. At the owner's request, commit titles and dates were reconstructed as a project-stage timeline for 13 November 2026 under the configured owner's Git identity. The assigned timestamps are not an original development log; reconstructed commit messages explicitly record that provenance. Subsequent development can be committed normally on `main`.

Clone the public repository with `git clone https://github.com/smurfac4/game-key-marketplace.git`, enter the directory, then follow Run with Docker above. Local `.env`, dependencies, builds and test artifacts remain ignored and are not published.

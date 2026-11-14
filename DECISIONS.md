# Architecture and decisions

This document describes the current implementation. A concise implementation history is in [WORKLOG.md](ai-logs/WORKLOG.md); complete earlier scope entries remain in the [pre-cleanup Git snapshot](https://github.com/smurfac4/game-key-marketplace/blob/50ff218/ai-logs/DECISIONS-HISTORY.md). Setup and API usage belong in [README](README.md) and [docs](docs/DEVELOPMENT.md).

## Scope

KEYSPACE is an independent, non-commercial Eneba-inspired marketplace demo. It demonstrates catalog discovery, account management and a complete purchase flow using simulated sellers, offers, payments, stock, keys and rewards. It does not sell or activate games. Game names are referential catalog labels; project artwork is original and is not official game cover art.

The frontend defaults to English and supports Russian and Lithuanian. EUR/USD/GBP selection changes quotes and new order currency, not just a currency symbol. Existing orders retain their recorded values. Real payments, seller administration, refunds, taxes, guest carts and production operations remain outside scope.

## Monorepo and modular monolith

An npm workspace contains `web/`; `api/` is a separate Go module. React Router owns navigation, TanStack Query owns server state, chi owns HTTP routing and pgx provides PostgreSQL access. A monorepo task framework and generated shared types are unnecessary for the project's current size.

The API is one deployed process with one database, organized into catalog, authentication, account, commerce and homepage packages. This is a modular monolith, not a microservice deployment. Homepage `banners`, `bestsellers`, `deals` and `newreleases` packages each own their repository and handler without calling another block. Shared transport types and infrastructure do not remove their shared-database coupling.

A block could be extracted behind a separate HTTP process while preserving `/api/home/...` through a gateway. Read-only access to the same database would be an intermediate step, not independent data ownership. A complete extraction needs its own read model fed by catalog/offer/paid-order events, reliable outbox delivery, idempotent consumers, backfills and operational monitoring. Best-seller totals are a suitable projection. Separate ownership or scaling needs should justify that extra operational cost.

## Data model and migrations

| Area | Tables and role |
| --- | --- |
| Identity | `users`, `auth_sessions`, `account_profiles`, `email_challenges`, `external_identities`, `oauth_flows` |
| Catalog | `products`, `sellers`, `offers` |
| Inventory | `game_keys`, with available/sold state and an order-item reference when sold |
| Cart | One `carts` row per user and `cart_items` per offer |
| Purchase history | `orders` and `order_items`, including name/price snapshots and checkout idempotency metadata |
| Homepage | `banners` and `homepage_blocks` |
| Favorites | Owner/product pairs in `wishlists` |

Products represent a platform/region listing with a unique slug, one genre and an optional release date. Platform, region and genre are text fields, avoiding a speculative ingestion/taxonomy system. Seller ratings are stored demo values rather than computed reviews.

Prices use PostgreSQL `numeric(12,2)`, JSON decimal strings and integer cents in Go. Discounts must be positive and lower than their base price. Orders store the actual chosen currency and unit-price snapshots. Composite foreign keys keep item/order currencies consistent and sold keys attached to the purchased offer. History references restrict deletion; mutable profile/cart/favorite data can cascade with its owner.

`game_keys` represents individual inventory; `offers.stock` is its cached available count. Checkout updates both transactionally. Constraints distinguish available keys without an order item from sold keys with one. The seed establishes consistent quantities; there is no general inventory-repair or seller-ingestion service.

Six embedded golang-migrate versions cover the base marketplace, revocable sessions, checkout idempotency, constrained local media paths, checkout request fingerprints and account/identity tables. Migrations run explicitly, with driver locking, instead of racing at every API startup. Down rolls back one version and can delete data. `pg_trgm` remains installed on base rollback because other consumers may use the extension.

The deterministic seed creates 150 listings for 50 titles, ten sellers, two–eight offers per listing and five–thirty fake keys per offer before sample sales. It also adds three banners, four block definitions and twelve paid demo orders representing twenty-four keys. Full seed is atomic and refuses a populated catalog. Additive homepage/showcase modes are limited to the recognized demo and preserve purchases and inventory; they are not arbitrary import tools.

## Catalog, search and facets

Listing prices are the lowest effective in-stock offer prices. Bounds apply to that product-level minimum. Listing rows, totals and facets come from one SQL statement/snapshot; multiple offers do not inflate product counts. Each facet excludes its own selection while respecting other filters. The frontend combines current counts with API-supplied taxonomy choices so users can recover from zero-result combinations.

Sort expressions are allowlisted and input values are bound SQL parameters. Product ID breaks ties. Popularity counts paid units; newest uses release dates. Offset pagination is adequate for this dataset but can shift under concurrent catalog changes and gets more expensive at deep offsets.

`pg_trgm` GIN on product names supports escaped literal `ILIKE` plus whole-name and word-similarity operators. Exact/prefix/substring matches rank ahead of fuzzy-only results. Whole-name similarity uses **0.30**; word similarity uses **0.50** to admit fixtures such as `witcer` inside a longer title without broadly lowering whole-title matching. The fixtures also cover `cyberpnk` and `fifa 2`; these are tested heuristics, not a universal search-quality guarantee.

Fuzzy matching starts at four characters. Suggestions return at most eight products, skip a DB call for one character and use literal matching for two/three characters. Search thresholds use SET LOCAL in the query transaction so pooled connections do not retain altered defaults. Suggestions omit full facets/popularity/price aggregation and have a one-second deadline. Filters and release-date lookups also have B-tree indexes; optimize further only with workload evidence.

Product details include sold-out offers and up to eight available same-genre recommendations, preferring matching platform/region. They omit key values. Without a conversion header, different offer currencies are displayed explicitly and nominal sorting is not an FX comparison.

## Currency and language

Fixed demo rates are 1 EUR = 1.10 USD = 0.85 GBP. `X-Display-Currency` converts supported source offers before minimum-price selection, bounds and sorting. Direct catalog requests without that header retain original-currency filtering. Checkout independently rereads prices and converts/rounds each unit before multiplication, then stores the selected currency. Mixing source currencies in one cart remains unsupported. Card/currency both participate in idempotency fingerprints.

English/Russian/Lithuanian preferences persist in browser storage. Shared catalog links retain query state and can initialize currency. Changing preferences removes non-session query caches and remounts the UI; unfinished form state resets. Interface copy and seed-owned promotional headings translate; catalog names/descriptions keep their source text. Backend-derived prices, products, sellers, banners and block titles are not hardcoded into components.

## Authentication and account data

Password login uses bcrypt cost 10, generic credential failures and dummy hash work for unknown/passwordless-only accounts. Passwords require twelve Unicode characters and at most seventy-two UTF-8 bytes. Registration's duplicate-email 409 explicitly reveals account existence; the demo accepts that trade-off. Password registration does not verify email ownership.

JWTs use HS256 with fixed issuer/audience, required expiry and a random session ID backed by PostgreSQL. The host-only cookie is httpOnly, lasts twenty-four hours and follows configurable Secure/SameSite settings. Every authenticated request checks revocable session state. Logout deletes the current session; it does not revoke all devices. Tokens are not stored in browser localStorage.

Exact-origin credentialed CORS and a required custom header protect mutations. Supplied foreign origins are rejected; SameSite=None requires Secure. The limiter uses the socket peer, not untrusted forwarded headers. Auth attempts share ten per IP/fifteen minutes; checkout permits ten per user/minute, suggestions sixty per IP/minute and OIDC starts ten per IP/minute. Proxied clients may share an IP budget.

Profiles and favorites always derive the owner from the session. Profile updates replace all editable fields; client owner/email changes are not supported. Countries/platforms come from the account API. Favorites are idempotent and capped at 200 by serializing additions on the user's row. Profile completion counts account creation, supplied email, name, country, avatar and preferred platform; it does not claim that an entered email has been verified.

The private keys library reuses owner-scoped paginated orders. Order/key/account responses use no-store. Wallet balance is zero; one point per whole EUR of paid demo orders and 1% illustrative cashback are server-derived, non-spendable summaries. Marketing/alert preferences default off and persist, but no automatic marketing/price-alert delivery system exists.

Optional SMTP sign-in uses authenticated STARTTLS, certificate checking, TLS 1.2 minimum and bounded network deadlines. Six-digit random codes have HMAC hashes, ten-minute expiry, five guesses, row locking and one-time consumption. Codes are emailed only. Verification signs into or creates the matching email account; password login remains available.

One configured OIDC issuer supports authorization code with a GET callback, PKCE S256, nonce, hashed state and a separate browser-binding httpOnly Lax cookie. The library validates token signature/issuer/audience/expiry, and the application requires a verified-email claim. Existing subjects remain authoritative; new identities may link to the same verified email. Only trusted issuers should be configured. Other protocols/callback modes need adapters. Provider discovery runs at startup; only configured/initialized methods appear in the UI. Actual SMTP/provider credentials and a live login are not included in automated tests.

Access logs record route patterns, method, status, timing and request ID rather than URLs/query strings, avoiding OAuth code/state leakage. Error responses use a common JSON envelope without SQL/internal details.

## Transactional checkout

Cart mutations and checkout serialize on the owner's cart. Checkout locks offers in stable ID order, reads server prices, locks available keys using `FOR UPDATE SKIP LOCKED`, writes order/item snapshots, marks allocated keys sold, decrements stock and clears the cart in one transaction. At most fifty lines and ten units per offer bound the work. Adding to a cart reserves nothing.

The payment interface currently has a deterministic mock implementation inside the transaction. A decline or insufficient inventory rolls back the changes. Real external payments cannot be rolled back by PostgreSQL; they would require authorization/capture, reservation expiry, durable reconciliation and provider idempotency instead.

A unique user-scoped idempotency key plus a request fingerprint returns the original order on a same-body retry and 409 on changed card/currency. Old tokenized orders without a fingerprint also return 409 because their bodies cannot be reconstructed safely. No raw card body is persisted. Tests cover competing buyers, a held key lock, rollback, owner isolation and lost HTTP success replies after commit; they do not inject a TCP failure inside PostgreSQL COMMIT.

## Homepage, assets and interface

Homepage endpoints return their own enabled title/content, queried separately by React. A block failure does not hide the others; empty blocks disappear. Banners respect activation/schedules. Best sellers use paid units, deals pair a discounted offer with its own original price, and new releases exclude future/unknown dates. There is no scheduled-discount model: active means a valid discount on an in-stock offer. Block order is fixed; editorial administration is deferred.

The purple palette, yellow actions, compact grid and seller comparison are marketplace-style design references. KEYSPACE branding, avatars, icons and artwork are original. Three generated genre illustrations and one portal hero are bundled PNG files; project SVG scenes provide alternatives/fallbacks. These are not per-title official covers. Provenance and historical publisher-art removal are in [ASSETS.md](ai-logs/ASSETS.md).

URL state drives store/search filters, sorting and pagination. Search debounces autocomplete with keyboard navigation; native dialog supplies mobile filter focus containment. Manual carousel controls, visible focus, labels, reduced-motion behavior and clipboard failure feedback support accessibility. Chromium/Axe coverage is useful evidence, not exhaustive screen-reader or cross-browser certification.

## Known limitations

- Compose runs Vite and local HTTP with database TLS disabled; production hosting, HTTPS/HSTS, CSP, security headers and operational hardening are not supplied.
- Keys are plaintext in PostgreSQL and must remain non-redeemable fixtures. Real key encryption and key-management infrastructure are not implemented.
- Payment is a mock inside a DB transaction; there are no real charges, refunds, wallet funds or reward redemptions.
- Rate limiting is process-local. Multiple instances require shared limits and an explicit trusted-proxy policy.
- Expired sessions are cleaned on that user's next login. Expired email challenges/OIDC flows are cleaned by later requests. There is no global periodic cleanup; order idempotency metadata is retained indefinitely.
- Readiness checks connectivity, not migration compatibility. Apply migrations before starting updated application code.
- Router wiring detects pgx pool capabilities. Explicit dependency construction would make production composition and unit mocking clearer; a broad refactor is deferred.
- Offset pages and correlated price/popularity calculations are acceptable at demo scale. Historical EXPLAIN samples are not throughput or production-capacity guarantees.
- Live SMTP/OIDC operation depends on externally configured services. Password reset/change, MFA, all-device logout, automatic alerts and non-OIDC social adapters are not implemented.
- Chromium is the automated browser target. Missing scanner, Lighthouse and load-test runs remain explicitly identified in the historical reports; E2E success does not substitute for them.

## Repository provenance

Early commits group files that already existed in the workspace. Their dates were reassigned at the owner's request and do not represent the original development times; commit messages record this. Later changes use normal commits.

## Documentation consolidation — 2026-11-14

README, configuration, API and development instructions were separated from historical audit/work records. Superseded decisions were archived with their original wording; asset provenance now describes the current shipped files. The fresh-source reproduction helper was aligned with the current bundled PNG/SVG artwork and startup order. This is documentation and validation maintenance, with no change to marketplace behavior or development catalog data.

## Visible guest actions after logout — 2026-11-14

The account dropdown previously rendered an avatar for guests and hid login/registration links inside a closed menu. It now renders direct Log in and Sign up links once the session resolves to a guest; the avatar/private menu is rendered only for signed-in users. A loading skeleton avoids presenting guest actions while `/api/me` is pending. Header registration uses a separate localized Sign up label, retaining Create account for the form action. On small screens, guest controls occupy a wrapping full-width row so both actions remain available alongside cart/favorites without horizontal overflow. The logout navigation commits synchronously with replacement history so the protected page cannot race it with an intervening login redirect. Session revocation remains server-backed.

`bash scripts/test-e2e.sh` passed all fourteen Chromium cases with Axe checks after adding assertions for guest buttons on first load, immediate logout, reload, English/Russian labels and keyboard navigation to both forms. The keyboard mobile case checks no horizontal overflow at 320px. A regression caught the protected-page redirect race before the synchronous logout navigation fix. Node 24 lint/format/build and Go unit tests/vet passed; unchanged backend integration suites were not separately rerun. Read-only Chromium smoke checks against the updated local web container confirmed visible links at 1280/390/320px and working login/register navigation, without changing development users or catalog data.

## Project positioning — 2026-11-14

The current project is described as an independent Eneba-inspired game-key marketplace. README, current asset documentation, English/Russian interface copy and repository metadata follow that positioning. Original KEYSPACE branding and the lack of Eneba/publisher affiliation remain explicit; payment, inventory, key and reward simulation disclosures remain accurate. Archived work records retain their historical wording and are not current product descriptions.

## One catalog search and Lithuanian — 2026-11-14

The duplicate catalog/search-page form was removed; the header is the single search entry point. Its draft initializes from the catalog URL and remounts when that query changes, keeping direct links and Back/Forward consistent. Submitting a catalog query preserves other filters/sorting and resets the page; submitting an empty query removes q and an obsolete relevance sort. Searching from other pages still opens the global results route, and autocomplete/recent-search navigation remains available.

Lithuanian joins English/Russian as a persisted language, with a project-authored flag and a complete 202-entry interface dictionary checked against the English key set by TypeScript. Country names, money, order dates, known region/genre labels, error messages and seed-owned homepage/banner titles follow the locale. API-owned translations preserve unknown/custom titles as source content, and catalog game names/descriptions stay in their original language. Homepage localization shares a small title helper across blocks rather than introducing a translation service or rewriting catalog rows.

Validation: the full Go race/PostgreSQL/vet runner passed, including locale fallback/cache-variation tests; all fifteen Chromium scenarios passed with Axe checks. Search regression checks direct URLs, preservation of platform filters, pagination reset, Back/Forward and an empty query removing relevance. The Lithuanian scenario covers language persistence, country labels, translated API content, mobile filters, decimal-comma prices, declined/successful checkout, key display, orders/library and logout. Build/lint/format checks passed, screenshots were inspected, and read-only smoke checks passed on the updated local application. No schema or development catalog refresh was needed.

## Submission screenshots — 2026-11-14

Six documentation screenshots are real Playwright captures at 1280px width, generated by `bash scripts/capture-screenshots.sh` in a dedicated `keyspace-screenshots` Compose project with its own fresh database and volume. The runner ignores the development environment file, disables SMTP/OIDC, keeps generated account/session credentials in memory, and removes its disposable resources on exit. Public pages are captured as a guest; checkout/library use only a synthetic account and visibly non-redeemable seed keys, without opening the account menu. The product screenshot is full-page so every seller row is visible; the others use a 1280×1000 viewport. Pillow optimization checks exact decoded-pixel equality and retains the smaller PNG encoding; actual sizes/dimensions are recorded in docs/screenshots/manifest.json. Capture/optimization completed and the six images were visually inspected; development data was not used.

## README gallery and concise setup — 2026-11-14

The screenshot gallery sits immediately below the introduction and links all six full PNGs. Quick start keeps the required configuration and five startup commands; the complete previous section, including credentials/origins guidance, default addresses, smoke requests and seed behavior, is preserved verbatim under docs/DEVELOPMENT.md#docker-setup-details. No live-demo section or hosted-site claim is added. A content-preservation assertion confirmed that the previous setup body remains in the development guide.

## Provenance presentation — 2026-11-14

The history explanation is kept briefly and neutrally in Repository provenance, with one README link to it. This changes documentation only; no commit dates, authors or history are rewritten.

## Project license — 2026-11-14

Added the requested MIT license with copyright attributed to the GitHub handle smurfac4 for 2026; no full legal name is needed or inferred. README links the license. Third-party dependency licenses and rights in referenced game names remain their respective owners' rights; asset provenance remains documented separately.

## E2E override location — 2026-11-14

Moved the image-only override to scripts/docker-compose.e2e.yml and updated the runner and development guide. The root application Compose file remains first in the -f list, preserving relative build/volume resolution; the override contains no relative paths. Shell syntax and quiet Compose configuration validation passed, followed by all fifteen Chromium/Axe scenarios through the relocated runner. Its disposable database/volume were removed on exit; the development catalog was not used.

## Chat export placeholder — 2026-11-14

A complete verbatim session transcript is not accessible in the current environment. ai-logs/CHAT-LOG.md therefore reserves space for the owner's export and explicitly contains no reconstructed prompts/responses. Existing work summaries remain summaries, and the AI records index links the placeholder. Any eventual export must exclude secrets, personal data and real keys before it is committed.

## Repository topics — 2026-11-14

Set the requested GitHub topics with gh repo edit: react, typescript, golang, postgresql, marketplace, game-keys and rest-api. gh repo view returned all seven names afterward. This is an explicitly requested repository-metadata update; the seven submission-pass commits are kept local and no Git push is performed pending owner confirmation.

## AI record consolidation — 2026-11-14

Reduced ai-logs to five purpose-specific files: README (guide), WORKLOG (implementation summary), VERIFICATION (checks/results/omissions), ASSETS (provenance) and CHAT-LOG (export placeholder). Replaced overlapping audit/closeout/backend reports and raw SQL output with a clearly dated consolidated verification record. Complete original reports and earlier scope decisions remain in the pinned pre-cleanup Git snapshot, preserving evidence without treating superseded states as current documentation. No transcript or previously unrun verification is invented.

## Obsolete helper and generated output cleanup — 2026-11-14

Removed `scripts/closeout-copy-timeout.sh`, a one-off audit helper with no remaining callers in the current repository. The real timeout test and the reusable database, performance, screenshot and source-reproduction runners remain available. Its original version is preserved in Git history. Removed only ignored generated directories (`web/dist`, `web/test-results`, `web/playwright-report`), totaling 31,429,769 bytes after the build check; normal build/browser commands recreate them.

Retained source, migrations, tests, installed dependencies, documentation screenshots and compatibility SVG media. The SVG paths are still used by browser tests and can be stored in existing catalog rows; deleting them would break supported URLs. No development database or environment file was accessed or changed by this cleanup.

## Recorded chat history and language cleanup — 2026-11-14

The earlier export-pending decision is superseded: local primary conversation records matching this repository were found and exported to ai-logs/CHAT-LOG.md. The file includes all 77 user and 184 assistant text messages in those two records through its stated cutoff; it does not claim recovery of messages absent from the local recorder. Assistant commentary and final answers are included; internal reasoning, system/developer instructions, tool traffic and background review-agent records are excluded from the human-facing chat history. Image attachments have explicit markers rather than copied binary data.

At the owner's request, profanity is explicitly replaced with redaction markers. Personal home-directory names and credential/key-like values are sanitized before writing; no raw session export is added to Git. A standard-library checker reports only file/line/category for configured profanity and credential/key patterns across all AI-log Markdown files. This is a reproducible limited pattern check, not a complete secrets or language audit. The dated transcript is a fixed snapshot; future chat messages require a later export rather than invented updates.

## Free-tier deployment packaging — 2026-11-14

Added a dedicated Render Docker image and Blueprint rather than deploying the Vite development server or changing the Go/PostgreSQL architecture. The single public web origin serves the built React app and REST API; a private PostgreSQL 17 instance stores the hosted demo data independently of local development. Render-generated credentials and its external URL supply runtime configuration, with Secure/HttpOnly/SameSite=Lax cookies and an exact origin allowlist. Missing assets remain 404, SPA deep links serve the frontend, and all /api paths retain their API/auth behavior.

Render Free does not provide the paid pre-deploy command, so the container startup wrapper runs the existing advisory-locked migrations and a new explicit seed -if-empty mode before the server. The empty check runs inside the seed transaction/advisory lock; repeated starts preserve existing rows and purchases. The normal seed refusal remains unchanged. The web runtime is optional through WEB_DIST and does not alter the local API-only Compose setup. Auto-deploy is off so subsequent documentation commits do not trigger unreviewed deployments.

This is a free demo deployment: web instances can sleep, the free PostgreSQL database expires after 30 days, and SMTP/OIDC are not configured by this change. Real payments, key encryption, shared rate limiting and full production hardening remain deferred. Current deployment status and actual checks are recorded in docs/DEPLOYMENT.md; adding configuration alone is not claimed as a successful public deployment.

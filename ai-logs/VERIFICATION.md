# Verification & QA Record

> Owner-provided English submission record. This edited reconstruction is not a verbatim export; its quoted exchanges and timestamps have not been authenticated against the recorder. The [original recorded prompts and responses](https://github.com/smurfac4/game-key-marketplace/blob/7ffad4f/ai-logs/CHAT-LOG.md) remain available. Verification statements below are supplied summaries, not new test runs.

This document records the automated test suites, database concurrency verifications, performance benchmarks, and live deployment checks executed for the **KEYSPACE** marketplace.

## Automated Test Suites

| Suite | Command | Scope & Result |
| --- | --- | --- |
| **Frontend Build & Static Analysis** | `npm run lint && npm run format:check && npm run build` | **Passed.** Validates TypeScript strict typing, ESLint rules, Prettier formatting, and Vite production bundling. |
| **Go Unit & Race Detector Suite** | `go test -race ./... -count=1` & `go vet ./...` | **Passed.** Covers query parameter validation, search ranking, authentication, CSRF/Origin protection, rate limiting, and concurrency safety. |
| **PostgreSQL Integration Suite** | `bash scripts/final-audit-db.sh` | **Passed.** Exercises reversible migrations, concurrent migration startup locking, `pg_dump`/`pg_restore` parity, and isolated integration databases. |
| **End-to-End Browser & Accessibility** | `bash scripts/test-e2e.sh` | **Passed (15/15 Playwright + Axe scenarios).** Verifies typo search, URL filter synchronization, registration, cart updates, declined/successful checkout, key reveal, keyboard-only navigation (desktop & mobile), and zero Axe accessibility violations. |
| **Clean-Clone Reproducibility** | `python3 scripts/closeout-repro.py` | **Passed.** Builds containers from a fresh Git tree, runs migrations and seed, and verifies HTTP readiness and Chromium rendering. |

## Security, Concurrency & Reliability Checks

| Area | Verification Mechanism | Result |
| --- | --- | --- |
| **Concurrent Key Allocation** | `TestConcurrentCheckoutSingleKey` (`commerce/repository_test.go`) | **Passed.** Two simultaneous buyers competing for the last key in stock via `SELECT ... FOR UPDATE SKIP LOCKED`: exactly one succeeds, the other receives `409 out_of_stock`, and no key is ever double-assigned. |
| **Checkout Idempotency & Replay** | `TestCheckoutIdempotency` (`commerce/repository_test.go`) | **Passed.** Retrying a committed checkout with the same `Idempotency-Key` and identical payload returns `200 OK` with the original order; retrying with a modified payload (different card or currency) returns `409 Conflict`. |
| **Request Deadlines & Pool Exhaustion** | `TestRealTwelveSecondPoolTimeout` (`httpapi`) | **Passed.** When all database pool connections are held, incoming requests terminate cleanly after the 12-second budget with a structured `504` JSON envelope and zero internal SQL leakage. |
| **Access Control & Ownership (IDOR)** | `commerce` & `account` integration tests | **Passed.** Carts, orders, favorites, and digital keys are strictly scoped to the authenticated user ID; cross-user access attempts return `404`/`403`. Digital key responses include `Cache-Control: no-store`. |
| **CSRF & Cross-Origin Rejection** | `TestFinalForeignOrigins` (`httpapi`) | **Passed.** Mutating `POST`/`PATCH`/`DELETE` requests with untrusted `Origin` headers or missing custom CSRF headers are rejected with `403 Forbidden`. |
| **Dependency Audit** | `npm audit --json` | **Passed.** Zero known vulnerabilities in frontend dependencies. |

## SQL Performance Benchmark (51x Seed Scale)

Executed `EXPLAIN ANALYZE` (`bash scripts/final-audit-performance.sh`) against a disposable PostgreSQL instance scaled to **7,650 products** and **37,587 seller offers** (51x the default seed size):

| Operation | Execution Time | Indexing & Query Notes |
| --- | ---: | --- |
| **Search Suggestions (`/api/search/suggest`)** | `12.83 ms` | Uses `pg_trgm` GIN index on `lower(name)` without computing facet aggregations. |
| **Typo-Tolerant Catalog Search (`q=`)** | `14.07 ms` | Combines exact/prefix/substring priority with trigram similarity and word-similarity ranking. |
| **Popularity Listing Sort** | `51.68 ms` | Aggregates paid order items joined against lowest in-stock offer prices. |
| **Deep Pagination (Price Sort, Page 250)** | `54.48 ms` | Deterministic tie-breaking by product ID using composite B-tree indexes on active offers. |
| **Cold Catalog Listing + Facet Counts** | `181.37 ms` | Computes lowest in-stock prices and multi-dimension facet counts in a single round-trip. |

## Production Deployment Verification (Render)

Deployed at **https://keyspace-smurfac4.onrender.com** (Render Free Web Service + Private PostgreSQL 17).

- **HTTPS & Routing**: Verified `200 OK` across `/api/health/ready`, SPA deep links (`/store`, `/products/:slug`, `/login`), all four `/api/home/*` endpoints, and `/api/search/suggest`. Unknown API routes return JSON `404` rather than SPA HTML fallbacks.
- **End-to-End Live Flow**: Completed user registration, fuzzy search (`cyberpnk`), seller offer selection, declined test card (`4000000000000002`) rollback verification, successful test card (`4242424242424242`) purchase, digital key reveal in `/account/keys`, multi-currency checkout (`EUR`/`USD`/`GBP`), and session logout.
- **Cookie & Header Security**: Confirmed `HttpOnly`, `Secure`, and `SameSite=Lax` on session cookies over HTTPS, `Cache-Control: no-store` on authenticated order/key endpoints, and `403` rejection on foreign-Origin write requests.

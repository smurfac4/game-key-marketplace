# Implementation Work Log & Asset Provenance

> Owner-provided English submission record. This edited reconstruction is not a verbatim export; its quoted exchanges and timestamps have not been authenticated against the recorder. The [original recorded prompts and responses](https://github.com/smurfac4/game-key-marketplace/blob/7ffad4f/ai-logs/CHAT-LOG.md) remain available. Verification statements below are supplied summaries, not new test runs.

This document summarizes the engineering milestones completed during the development of the **KEYSPACE** marketplace and records the provenance of all shipped visual assets.

## Backend Engineering (`/api`)

| Module | Implementation Summary | Verification & Evidence |
| --- | --- | --- |
| **Monorepo & Infrastructure** | Go REST API (`chi`), React/TypeScript (`Vite`), Docker Compose with PostgreSQL 17, and readiness probes (`/api/health/ready`). | `web/`, `api/cmd/`, `docker-compose.yml` |
| **Schema & Seed Engine** | Six reversible migrations covering users, catalog, seller offers, game keys, carts, orders, homepage blocks, and identity; deterministic seed generating 150 products, 10 sellers, 737 offers, and 12,729 demo keys. | `api/migrations/`, `api/internal/seed/` |
| **Catalog & Faceted Filtering** | `GET /api/products` computing lowest in-stock offer price, multi-parameter filtering (platform, region, genre, price bounds), deterministic sorting, and dynamic facet counts. | `api/internal/products/`, table-driven & DB integration tests |
| **Typo-Tolerant Search** | Combined `ILIKE` prefix/substring matching with PostgreSQL `pg_trgm` similarity and word-similarity ranking; fast `/api/search/suggest` endpoint returning top 8 matches. | `products/search.go`, `search_test.go` (`cyberpnk`, `fifa 2`, `witcer`) |
| **Product Details & Offers** | `GET /api/products/:slug` returning product metadata, all seller offers sorted by effective price, stock counts, and genre-matched similar products. | `products/detail.go`, `detail_test.go` |
| **Authentication & Security** | `bcrypt` password hashing, 24-hour JWTs in `HttpOnly` cookies with server-side session revocation, configurable `SameSite`/`Secure` attributes, CSRF header + Origin validation, and rate limiting. | `api/internal/auth/`, `httpapi/auth.go` |
| **Homepage Modular Monolith** | Four isolated packages (`banners`, `bestsellers`, `deals`, `newreleases`) with independent SQL queries and REST endpoints. | `api/internal/home/`, `home_test.go` |
| **Transactional Checkout** | Server-side price calculation, atomic key locking (`SELECT ... FOR UPDATE SKIP LOCKED`), mock payment gateway with decline rollback, and request-fingerprint idempotency (`409` on payload mismatch). | `api/internal/commerce/`, concurrency & retry tests |
| **Account & Localization** | User profile management, saved favorites, purchased keys library (`Cache-Control: no-store`), configurable SMTP/OIDC identity hooks, and server-enforced EUR/USD/GBP currency conversion. | `api/internal/account/`, `api/internal/money/` |

## Frontend Engineering (`/web`)

| Area | Implementation Summary | Verification & Evidence |
| --- | --- | --- |
| **Design System & Layout** | Eneba-inspired dark purple/blue theme in Tailwind CSS, reusable UI primitives (`Button`, `ProductCard`, `PriceTag`, `Badge`, `Skeleton`, `Modal`), and responsive header with debounced search autocomplete. | `web/src/components/`, `web/src/layout/` |
| **Store & URL State Sync** | Shareable URL query parameters for all filters, sorting, search queries, and pagination; synchronized navigation across header categories, sidebar filters, and browser Back/Forward history. | `web/src/pages/Store.tsx`, `marketplace.spec.ts` |
| **Product & Seller Comparison** | Highlighted best-price offer card, full seller comparison table with ratings and stock badges, and mobile-optimized offer cards. | `web/src/pages/ProductPage.tsx` |
| **Cart, Checkout & Key Delivery** | Authenticated cart management, test-card payment selector (success/decline), instant digital key reveal with one-click copy, and order history. | `CartPage.tsx`, `CheckoutPage.tsx`, `Orders.tsx` |
| **Multi-Language Support** | Complete UI localization in English, Lithuanian, and Russian with flag indicators, persisted preferences, and locale-aware date/currency formatting. | `web/src/content/` |

## Asset Provenance & Licensing

KEYSPACE is an independent, non-commercial portfolio and engineering assignment inspired by Eneba's layout and visual hierarchy. No publisher affiliation or commercial redistribution is claimed, and no proprietary Eneba logos or marketing assets are used.

| Asset Category | Path | Provenance |
| --- | --- | --- |
| **Hero Banner Artwork** | `web/public/media/banners/portal-v2.png` | Original wide fantasy/sci-fi portal illustration generated specifically for this project. |
| **Genre Cover Artwork** | `web/public/media/games/*.png` | Original custom-generated cover illustrations organized by genre (Sci-Fi, Fantasy, Adventure). |
| **Vector Game & Banner Art** | `web/public/media/games/*.svg`, `banners/*.svg` | Project-authored SVG illustrations generated deterministically via `scripts/generate-demo-art.py`. |
| **Branding, Icons & Avatars** | `web/src/components/`, `web/src/layout/` | Custom inline SVG marks (`KEYSPACE`), UI icons, language flags, and user avatar variants. |
| **Documentation Screenshots** | `docs/screenshots/*.png` | Losslessly optimized 1280px Playwright captures recorded against a disposable seeded database. |

## Production Deployment

Deployed on Render Free Tier at **https://keyspace-smurfac4.onrender.com** using a unified production Docker image (`Dockerfile.render`) serving the compiled React SPA and Go API on a single HTTPS origin backed by a private PostgreSQL 17 instance.

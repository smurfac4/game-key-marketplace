# API reference

[Back to README](../README.md)

All routes below are relative to `/api`. For the default local setup use `http://localhost:5173/api` through the web proxy, or `http://localhost:8080/api` directly.

JSON bodies require `Content-Type: application/json`. Write requests also require `X-CSRF-Protection: 1`. Protected routes use the httpOnly session cookie; ownership comes from that session, not a request-supplied user ID. Prices are decimal strings, not floating-point JSON numbers.

## Catalog and search

| Method | Route | Result |
| --- | --- | --- |
| GET | `/products` | Product list, pagination and facets |
| GET | `/products/{slug}` | Product details, seller offers and similar products; unknown slug returns 404 |
| GET | `/search/suggest?q=...` | Up to eight available product suggestions |

Listing parameters:

| Parameter | Meaning |
| --- | --- |
| `q` | Partial/typo-tolerant product-name search, up to 200 characters |
| `platform`, `region`, `genre` | One exact, case-sensitive value per filter |
| `min_price`, `max_price` | Inclusive bounds on the cheapest available effective price; nonnegative decimals with at most two fractional digits |
| `sort` | `relevance`, `price_asc`, `price_desc`, `popularity`, `newest` or `name` |
| `page` | Default 1; range 1–1,000,000 |
| `page_size` | Default 24; range 1–100 |
| `currency` | Original offer currency filter, three uppercase letters; default `EUR` |

With search, the default sort is relevance; otherwise it is popularity. Popularity counts units in paid orders. Newest sorts release dates with unknown dates last. All sorts use product ID to break ties. Products without an available offer in the requested currency are excluded from listings.

The response contains `data`, `pagination` (`page`, `page_size`, `total`, `total_pages`) and `facets` (`platform`, `region`, `genre`, each with `value`/`count`). Facet counts apply search, price, currency and the other selected filters while excluding the facet's own selection. Counts are calculated before pagination.

Suggestions accept only `q` and optional `currency`. One-character queries return an empty array; two/three-character queries use literal matching. Fuzzy matching starts at four characters. Exact names, prefixes and substrings rank ahead of fuzzy-only results. `%`, `_` and backslash in searches are treated literally.

Details include all offers, including sold-out offers, sorted by effective price, currency and offer ID. Offer fields include nested seller ID/name/rating, `price`, nullable `discount_price`, `effective_price`, `currency` and `stock`. `similar_products` contains up to eight available same-genre products, preferring the same platform and region.

```sh
curl --fail 'http://localhost:5173/api/products?q=cyberpnk&page_size=12'
curl --fail 'http://localhost:5173/api/search/suggest?q=witcer'
curl --fail http://localhost:5173/api/products/demo-game-001
```

## Currency conversion

For EUR/USD/GBP conversion, send `X-Display-Currency` with the target currency. It takes precedence over the currency query parameter and converts supported source offers before catalog price selection and filtering. Details also convert and re-sort supported offers. Without the header, catalog queries retain original-currency filtering and details can include offers in different currencies; those raw nominal amounts are not an FX comparison.

Product details and cart reads also accept a supported `currency` query parameter. Account and wishlist reads use the display-currency header and default to EUR. Historical order amounts are never repriced.

```sh
curl --fail -H 'X-Display-Currency: GBP' 'http://localhost:5173/api/products?sort=price_asc&page_size=12'
```

## Homepage

| Method | Route | Content |
| --- | --- | --- |
| GET | `/home/banners` | Up to eight active, currently scheduled banners |
| GET | `/home/best-sellers` | Up to twelve available products ranked by paid units sold |
| GET | `/home/deals` | Up to twelve available discounted products ranked by percentage saved |
| GET | `/home/new-releases` | Up to twelve latest released products; future/unknown dates excluded |

Each endpoint returns `{ "title": "...", "data": [...] }`. Titles and activation come from `homepage_blocks`. Missing/disabled configuration returns empty content. A deal uses the cheapest discounted offer and its matching original price, which can differ from a product's cheapest undiscounted listing offer. Empty sections disappear in the frontend; failures affect only the individual block. Product blocks support the currency behavior above.

## Authentication

| Method | Route | Request / result |
| --- | --- | --- |
| POST | `/auth/register` | `{ "email": "...", "password": "..." }`; creates a user (201), then login is required |
| POST | `/auth/login` | Same fields; returns public user data and sets a session cookie |
| POST | `/auth/logout` | Revokes the current session and clears its cookie (204) |
| GET | `/me` | Current public user or 401 |
| GET | `/auth/methods` | Enabled password/email-code/provider methods |
| POST | `/auth/email/request` | `{ "email": "..." }`; sends a code and returns a challenge ID (202) |
| POST | `/auth/email/verify` | `{ "challenge_id": "...", "code": "..." }`; consumes the code and sets a session cookie |
| GET | `/auth/oidc/start?next=/account` | Starts the configured provider redirect flow |
| GET | `/auth/oidc/callback` | Provider's authorization-code callback |

Passwords require at least 12 Unicode characters and at most 72 UTF-8 bytes. Duplicate password registration returns 409. Login failures return a generic 401. Optional email/OIDC methods require [service configuration](CONFIGURATION.md).

## Account and favorites

All routes in this section require authentication and use `Cache-Control: no-store`.

| Method | Route | Request / result |
| --- | --- | --- |
| GET | `/account` | Profile, completion, own-order/favorite/key counts, choices and demo reward summaries |
| PATCH | `/account` | Replaces the editable profile fields listed below |
| GET | `/wishlist` | Up to 200 own favorites, with available price or null |
| PUT | `/wishlist/{product_id}` | Adds a favorite; repeat additions are harmless (204) |
| DELETE | `/wishlist/{product_id}` | Removes an own favorite; repeat removals are harmless (204) |

Editable fields: `display_name`, `country_code`, `avatar`, `preferred_platform`, `marketing_opt_in`, `wishlist_notifications`. Omitted fields reset to their zero values: PATCH is a full editable-profile replacement, not a partial merge. Available country/platform choices come from GET `/account`; avatar values are `nebula`, `pixel`, `orbit`, `ember` or empty. Unknown ownership fields are rejected.

## Cart, checkout and orders

All routes in this section require authentication. Cart item IDs in these routes are **offer IDs**.

| Method | Route | Request / result |
| --- | --- | --- |
| GET | `/cart` | Current cart and server-calculated totals |
| POST | `/cart/items` | `{ "offer_id": 1, "quantity": 1 }`; adds to existing quantity |
| PATCH | `/cart/items/{offer_id}` | `{ "quantity": 2 }`; replaces quantity |
| DELETE | `/cart/items/{offer_id}` | Removes the item from the current user's cart |
| POST | `/checkout` | `{ "card_number": "...", "currency": "EUR" }` and an `Idempotency-Key` header |
| GET | `/orders?page=1` | Own orders, twenty per page |
| GET | `/orders/{id}` | Own order snapshots and purchased keys; another user's order returns 404 |

Checkout currency is EUR/USD/GBP, default EUR. The idempotency key must contain 16–80 letters, digits, hyphens or underscores. A successful same-key/same-body retry returns the same order (200); changed card or currency returns 409 `idempotency_conflict`. Legacy orders without a stored request fingerprint also conflict rather than guessing the original request. Mock card numbers are in [README](../README.md#try-the-purchase-flow).

Prices are read from the database. Orders, key allocation, stock and cart changes commit together; payment failure rolls them back. Carts allow 50 offers, ten units per offer and one source currency. Cart additions do not reserve stock. Order/key responses use `Cache-Control: no-store`.

## Health, errors and limits

GET `/health/live` checks the process. GET `/health/ready` checks database connectivity, not migration compatibility.

Errors use a consistent envelope:

```json
{"error":{"code":"invalid_query","message":"...","field":"page"}}
```

`field` is optional. Common statuses are 400 validation, 401 authentication, 402 mock payment decline, 403 browser-origin/CSRF rejection, 404 missing resource, 409 stock/idempotency/conflict, 429 rate limit, 500 internal error, 503 unavailable service and 504 timeout. Invalid query parameters are rejected by catalog endpoints. Unexpected internal/SQL details are not included in the response.

Authentication attempts share ten requests per peer IP per fifteen minutes. Checkout permits ten per authenticated user per minute; suggestions permit sixty per peer IP per minute; OIDC start permits ten per peer IP per minute. These limiters are process-local. Proxy users can share the same peer-IP budget. The global request budget is twelve seconds, with shorter feature deadlines where applicable.

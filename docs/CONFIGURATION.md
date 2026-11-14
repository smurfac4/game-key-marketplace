# Configuration

[Back to README](../README.md)

Compose reads the ignored root `.env`. Start from `.env.example`; never commit passwords, signing secrets, provider credentials or session cookies. The Go commands do **not** load `.env` automatically: export their configuration into the process environment when running outside Docker.

## Database and ports

| Variable | Purpose |
| --- | --- |
| `POSTGRES_USER` | Compose database user; template: `marketplace` |
| `POSTGRES_PASSWORD` | Database password; replace the template placeholder |
| `POSTGRES_DB` | Compose database name; template: `marketplace` |
| `POSTGRES_PORT` | Published database port; default `5432` |
| `API_PORT` | Published API port; default `8080` |
| `WEB_PORT` | Published web port; default `5173` |
| `DATABASE_URL` | Required by host Go commands; Compose supplies its internal database address |
| `PGUSER`, `PGPASSWORD`, `PGDATABASE`, `PGSSLMODE` | PostgreSQL connection settings; Compose supplies these from the database configuration |
| `HTTP_ADDR` | Go listen address; default `:8080` outside Compose |
| `WEB_DIST` | Optional built frontend directory served by Go; the Render image sets `/app/web`. Leave unset for API-only local Compose |
| `PORT` | Render-supplied public service port; the deployment wrapper maps it to `HTTP_ADDR` |
| `RENDER_EXTERNAL_URL` | Render-supplied HTTPS origin; the deployment wrapper uses it for browser and optional identity configuration |
| `API_PROXY_TARGET` | Vite's Go API target; host default `http://localhost:8080`, Compose target `http://api:8080` |

Changing a database password in `.env` does not change the password already stored in an initialized PostgreSQL volume. Keep existing credentials when reusing a volume, or deliberately update PostgreSQL credentials. Do not delete a volume merely to resolve a port conflict.

## Session cookies and browser access

| Variable | Purpose |
| --- | --- |
| `JWT_SECRET` | Required random signing secret, at least 32 bytes; changing it invalidates existing JWTs |
| `COOKIE_SECURE` | `false` for local HTTP; use `true` with HTTPS. Go defaults to `true`; development Compose defaults to `false` |
| `COOKIE_SAMESITE` | `lax` by default; also accepts `strict` or `none` |
| `AUTH_ALLOWED_ORIGINS` | Comma-separated exact frontend origins, with no paths, trailing slash or wildcard |
| `AUTH_WEB_URL` | Frontend origin for OIDC return redirects; must be one of the allowed origins |

For the default local setup, use `COOKIE_SECURE=false`, `COOKIE_SAMESITE=lax` and `AUTH_ALLOWED_ORIGINS=http://localhost:5173`. If you use another web port, update the allowed origin too.

Cross-site deployment requires HTTPS, `COOKIE_SECURE=true`, `COOKIE_SAMESITE=none`, explicit allowed origins and browser requests with credentials. `SameSite=None` without Secure is rejected. Third-party-cookie restrictions may still require a same-site API proxy. Compose itself provides local HTTP development, not a production HTTPS deployment.

Sessions use a host-only httpOnly cookie and expire after 24 hours. Logout revokes the corresponding database session. Mutating API requests require `X-CSRF-Protection: 1`; a supplied foreign Origin is rejected.

## Language and currency

The header persists English/Russian/Lithuanian and EUR/USD/GBP preferences in browser storage. English is the default. Interface copy and seed-owned promotional headings follow the language; catalog game names and descriptions keep their source text.

Fixed demonstration rates are **1 EUR = 1.10 USD = 0.85 GBP**. They are not live exchange-rate data. The selected currency changes catalog quotes, cart totals and the currency stored for new orders. Existing orders retain their recorded amounts. Unit prices are converted and rounded on the server before totals are calculated.

The frontend sends `X-Display-Currency` for supported price reads and `currency` in checkout JSON. For direct API requests, the header enables conversion; a catalog `currency` query parameter on its own filters original offer currency. See [API reference](API.md).

## Email-code sign-in

Leave these fields empty to keep email-code sign-in disabled:

| Variable | Purpose |
| --- | --- |
| `SMTP_ADDR` | SMTP host and port, typically `mail.example:587` |
| `SMTP_USER` | SMTP username |
| `SMTP_PASSWORD` | SMTP authentication password |
| `SMTP_FROM` | Plain sender email address without a display name |

The server requires authenticated STARTTLS, a valid server certificate and TLS 1.2 or newer. Implicit TLS on port 465 and unauthenticated local mail relays are not supported.

A sign-in code has six digits, expires after ten minutes and permits five incorrect guesses. Only its HMAC hash is stored. A successful code signs into the matching account or creates an account for that email. Codes are sent by email, never displayed by the application or returned by the API. Password registration/login remains available.

## External OpenID Connect sign-in

One OIDC provider can be configured:

| Variable | Purpose |
| --- | --- |
| `OIDC_ISSUER` | Trusted issuer/discovery URL; HTTPS except for local loopback tests |
| `OIDC_CLIENT_ID` | Provider application client ID |
| `OIDC_CLIENT_SECRET` | Provider application secret |
| `OIDC_REDIRECT_URL` | Exact callback registered in the provider console |
| `OIDC_PROVIDER_NAME` | Optional button label; default `OpenID Connect` |
| `AUTH_WEB_URL` | Exact frontend origin already included in `AUTH_ALLOWED_ORIGINS` |

For the default web port, the callback is `http://localhost:5173/api/auth/oidc/callback`. Use your actual web origin so the browser's flow cookie and API proxy share a host. The supported flow is authorization code with a GET callback, PKCE S256 and a verified email claim. Providers requiring other protocols or form-post callbacks need separate adapters; this is not a ready-made integration for every social network.

Only configure an issuer you trust to assert ownership of email addresses. A new external identity may link to an existing account with the same verified email; subsequent logins use the stored issuer/subject mapping.

## Apply optional settings

After changing settings, recreate the API:

```sh
docker compose up -d --force-recreate --wait api
```

OIDC discovery runs at startup. Invalid SMTP configuration or failed OIDC initialization produces a generic warning and disables that method. `/api/auth/methods` reports available methods, and the frontend displays only those enabled. Do not share raw environment configuration while troubleshooting.

External credentials are not included. Automated tests use a mailer in memory, a local OIDC issuer and mocked browser responses; actual delivery and a live provider login require your configured services.

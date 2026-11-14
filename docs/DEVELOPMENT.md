# Development and testing

[Back to README](../README.md)

The default application setup uses Docker only. Host development requires Node.js 24+, npm, Go 1.26+ and PostgreSQL 17 (or the provided database container).

## Host development

Create and configure `.env` as described in README, then install frontend dependencies and start PostgreSQL:

```sh
npm ci
docker compose up -d --wait db
```

Go does not load `.env`. Export `PGUSER`, `PGPASSWORD` and `JWT_SECRET` from your local configuration before running the API. `PGDATABASE` and the port in `DATABASE_URL` must match your database. For the default database name and port:

```sh
cd api
export DATABASE_URL='postgres://localhost:5432/marketplace?sslmode=disable'
export COOKIE_SECURE=false
export COOKIE_SAMESITE=lax
export AUTH_ALLOWED_ORIGINS=http://localhost:5173
go run ./cmd/migrate up
go run ./cmd/seed
go run ./cmd/server
```

Skip `seed` when the catalog is already populated. In another terminal, from the repository root:

```sh
npm run dev
```

Vite listens on port 5173 with strict port selection and proxies `/api` to `http://localhost:8080`. Set `API_PROXY_TARGET` in Vite's process environment if Go uses another address. Compose's `WEB_PORT` and `API_PORT` configure Docker port publishing, not these host processes.

## Unit checks and build

From the repository root:

```sh
npm run lint
npm run format:check
npm run build
(cd api && go test ./... && go vet ./...)
```

The build includes TypeScript checking. `npm run typecheck` runs that separately. Database-dependent Go tests skip when their test URLs are absent; a passing unit run alone does not establish database behavior.

## PostgreSQL integration tests

Each suite requires its own empty database, with a name ending in `_test`:

| Environment variable | Suite |
| --- | --- |
| `TEST_DATABASE_URL` | Seed, schema and inventory |
| `TEST_PRODUCTS_DATABASE_URL` | Listing, facets, search and details |
| `TEST_AUTH_DATABASE_URL` | Authentication and browser security |
| `TEST_HOME_DATABASE_URL` | Homepage modules |
| `TEST_COMMERCE_DATABASE_URL` | Checkout, rollback, ownership and concurrency |
| `TEST_ACCOUNT_DATABASE_URL` | Profiles, favorites, email codes and local OIDC |

Provision six disposable databases and export their URLs, then run from `api/`:

```sh
go test -race ./... -count=1
go vet ./...
```

Connection URLs may omit passwords when `PGUSER` and `PGPASSWORD` are exported. Never use the development catalog. Tests apply migrations; some leave migration metadata, so drop and recreate the disposable databases before rerunning. Remove them when finished.

`bash scripts/final-audit-db.sh` automates six integration databases plus migration-concurrency and backup/restore checks. This existing audit runner assumes the database user `marketplace`, password `local-development-only` and host port 5432; it does not read arbitrary `.env` credentials. Use it only with that local audit setup. Its reserved `eneba_final_*_test` names must not contain unrelated data, because the cleanup removes those databases. For other credentials, use the explicit suite variables above.

## Browser tests

Install npm dependencies, then run:

```sh
bash scripts/test-e2e.sh
```

The runner builds Docker images and uses a separate `eneba-e2e` Compose project, PostgreSQL volume and configuration. It runs fifteen Chromium scenarios in three batches: general marketplace, keyboard purchases, and account/email forms. The disposable API restarts between batches to reset the shared proxy-IP authentication budget; application rate limits remain unchanged.

Coverage includes search, filters/history, registration/session restoration, cart changes, declined/successful checkout, ownership, keys, language/currency, favorites, saved profile and mobile layouts. Axe checks are included. Email-form HTTP responses are mocked; the Go integration tests cover real code-consumption and local OIDC protocol behavior. This does not test a live SMTP account or external identity provider.

Docker supplies the browser; no host browser installation is required. The first run may download images. The test project and volume are removed on exit. Screenshots and reports remain in ignored `web/test-results/` and `web/playwright-report/`. Trace capture is disabled. Optional external mail/provider settings are cleared by the runner.

A focused run accepts normal Playwright arguments, for example:

```sh
bash scripts/test-e2e.sh --grep 'account profile'
```

## Fresh-source reproduction

```sh
python3 scripts/closeout-repro.py
```

This copies Git-listed/non-ignored source to a temporary directory, supplies isolated configuration in process memory, builds the application, applies migrations and seeds a fresh database. It smoke-checks readiness, catalog and details over HTTP and opens catalog/details in Chromium, including locally served artwork. It then removes its temporary Compose project, database volume and source copy. It is a filtered-copy check, not a literal remote `git clone`.

Requires Python 3, Git, Docker and installed project npm dependencies for the browser runner. It does not read or copy the development `.env`, and does not change the development catalog. The reserved `eneba-closeout-clean` Compose project must be unused.

## Migrations and seed modes

| Command inside the API image | Purpose |
| --- | --- |
| `migrate up` | Apply pending migrations |
| `migrate down` | Roll back one migration; its down SQL may delete data |
| `seed` | Populate an empty catalog with the complete demo |
| `seed -homepage` | Add seed-owned homepage fixtures/sales to the recognized demo; repeat-safe |
| `seed -showcase` | Refresh recognized demo catalog content/artwork without resetting purchases/inventory |

Invoke these with `docker compose run --rm --no-deps api ...` after building the API and starting PostgreSQL. Do not force a dirty migration version without investigating the failure. A full seed refuses an existing catalog instead of truncating it.

## Backup and restore

Keep backups outside the repository, with restricted permissions. A backup contains account and order data as well as catalog data. Restore into a new empty database, not over a live catalog. These commands use the database container's environment without printing its credentials:

```sh
umask 077
docker compose exec -T db sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' > /tmp/keyspace-backup.dump
docker compose exec -T db sh -c 'createdb -U "$POSTGRES_USER" keyspace_restore'
docker compose exec -T db sh -c 'pg_restore -U "$POSTGRES_USER" -d keyspace_restore --exit-on-error' < /tmp/keyspace-backup.dump
```

Use an unused destination name. Application updates do not require a database reset.

## Historical audit tools

Earlier audit commands and results are retained in [the AI records](../ai-logs/README.md). The SQL-plan runner is a single-dataset performance investigation, not a capacity claim. Security scanners, load tools or Lighthouse not run in a recorded pass are not implied to have passed by a build or E2E result.

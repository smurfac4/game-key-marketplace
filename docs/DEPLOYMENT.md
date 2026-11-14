# Render deployment

The repository includes a free-tier Render Blueprint for the complete React/Go/PostgreSQL demo. A configuration file alone is not proof of deployment; a public URL and the checks actually completed are recorded here after the service is live.

## Architecture

- `Dockerfile.render` builds React with Node 24 and the API with Go 1.26. Its runtime contains only the Go binaries, CA certificates and static frontend assets; it does not run the Vite development server.
- A single web service serves the frontend and `/api` on the same HTTPS origin. React Router deep links fall back to `index.html`; missing assets return 404 and API requests retain their JSON/auth behavior.
- Render supplies `PORT` and `RENDER_EXTERNAL_URL`. `scripts/start-render.sh` derives the listener and exact browser origin from them. Session cookies remain HttpOnly, Secure and SameSite=Lax.
- A separate PostgreSQL 17 instance is connected through Render's private network. `ipAllowList: []` blocks external database connections. Render generates the session signing secret; credentials are not committed.
- Startup runs `migrate up` followed by `seed -if-empty`, then starts the API. The seed checks under its existing transaction/advisory lock and leaves populated catalogs and purchases intact. The ordinary `seed` command still refuses a populated catalog.

## Deploy from GitHub

1. In Render, choose **New → Blueprint** and select the public repository `https://github.com/smurfac4/game-key-marketplace` on `main`, using `render.yaml`.
2. Confirm both the web service and database use **Free** and the estimate is zero. The Blueprint uses Frankfurt for both resources. Do not choose a paid plan to bypass an account-verification prompt.
3. Create the Blueprint, wait for the web service to become live, and open its returned HTTPS URL. Check `/api/health/ready`, `/api/products?q=cyberpnk` and `/products/demo-game-001`, then try the purchase flow described in the root README.

Automatic deploys are disabled to avoid replacing a reviewed demo when unrelated documentation commits land. Deploy a chosen commit manually from Render after running the project checks. A redeploy reruns migrations and the non-destructive empty-catalog check; it does not reset purchases.

## Demo boundaries

Use email/password registration. External email delivery and OIDC remain disabled until separately configured; they are not required for the demo purchase flow. Use only the documented mock card numbers and synthetic, non-redeemable inventory.

Free web services sleep when idle, so a first visit may need additional startup time. Free Render Postgres expires 30 days after creation, and the workspace has free compute/bandwidth/build limits. This is a submission/demo environment, not a permanent production service. See [Render's free-plan limits](https://render.com/docs/free) and [Blueprint specification](https://render.com/docs/blueprint-spec).

The hosted database is separate from the local development catalog. Never put local `.env` values or database credentials into screenshots, AI logs or Git commits. Use backups before an intentional database replacement; expiry does not preserve the demo indefinitely.

## Local deployment checks — 2026-11-14

`docker build -f Dockerfile.render -t keyspace-render-check .` completed, including `npm run build`. `go test ./...` and `go vet ./...` passed from `api/`; the new webui table tests cover deep links, HEAD, static caching, API isolation and hidden/missing files. The seed database integration test ran on a separate PostgreSQL 17 database and passed, including repeated RunIfEmpty calls.

An in-task Python smoke runner started the built image against another disposable PostgreSQL database, checked registration/login, typo search, mock checkout, idempotency replay/conflict, purchased-key retrieval and order/key preservation after restarting the image. The existing Playwright suite then ran against this production image in three batches (9 + 2 + 4 cases), with all 15 cases and their Axe checks passing. The SQL check found no sold keys without the matching offer/order item. All test containers, database volumes and the dedicated network were removed; no development catalog was used. These are local checks, not a claim that the public Render instance has already passed them.

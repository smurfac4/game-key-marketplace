#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export AUTH_WEB_URL= SMTP_ADDR= SMTP_USER= SMTP_PASSWORD= SMTP_FROM= OIDC_ISSUER= OIDC_CLIENT_ID= OIDC_CLIENT_SECRET= OIDC_REDIRECT_URL= OIDC_PROVIDER_NAME=
export POSTGRES_USER=marketplace POSTGRES_PASSWORD=e2e-local-only POSTGRES_DB=marketplace
export POSTGRES_PORT=15432 API_PORT=18080 WEB_PORT=15173
export COOKIE_SECURE=false COOKIE_SAMESITE=lax AUTH_ALLOWED_ORIGINS=http://localhost:5173
export JWT_SECRET
JWT_SECRET="$(openssl rand -hex 32)"
compose=(docker compose -p eneba-e2e -f docker-compose.yml -f scripts/docker-compose.e2e.yml)
cleanup() { "${compose[@]}" down -v --remove-orphans; }
trap cleanup EXIT
docker compose build api web
"${compose[@]}" up -d --wait db
"${compose[@]}" run --rm --no-deps api migrate up
"${compose[@]}" run --rm --no-deps api seed
"${compose[@]}" up -d --wait api web
web_container="$("${compose[@]}" ps -q web)"
run_browser() {
  docker run --rm --ipc=host --network "container:$web_container" \
    -v "$PWD:/app" -w /app/web -e E2E_BASE_URL=http://localhost:5173 \
    mcr.microsoft.com/playwright:v1.58.2-noble npx playwright test "$@"
}
if [[ $# -eq 0 ]]; then
  run_browser --grep-invert 'complete keyboard purchase|account profile|email code form' --output=test-results/main
  "${compose[@]}" restart api
  "${compose[@]}" up -d --wait api
  run_browser --grep 'complete keyboard purchase' --output=test-results/keyboard
  "${compose[@]}" restart api
  "${compose[@]}" up -d --wait api
  run_browser --grep 'account profile|email code form' --output=test-results/account
else
  run_browser "$@"
fi

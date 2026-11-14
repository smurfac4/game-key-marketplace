#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export AUTH_WEB_URL= SMTP_ADDR= SMTP_USER= SMTP_PASSWORD= SMTP_FROM= OIDC_ISSUER= OIDC_CLIENT_ID= OIDC_CLIENT_SECRET= OIDC_REDIRECT_URL= OIDC_PROVIDER_NAME=
export POSTGRES_USER=marketplace POSTGRES_PASSWORD=screenshots-local-only POSTGRES_DB=marketplace
export POSTGRES_PORT=16432 API_PORT=18880 WEB_PORT=15673
export COOKIE_SECURE=false COOKIE_SAMESITE=lax AUTH_ALLOWED_ORIGINS=http://localhost:5173
export JWT_SECRET
JWT_SECRET="$(openssl rand -hex 32)"
compose=(docker compose --env-file /dev/null -p keyspace-screenshots -f docker-compose.yml)
cleanup() { "${compose[@]}" down -v --remove-orphans; }
trap cleanup EXIT
"${compose[@]}" build api web
"${compose[@]}" up -d --wait db
"${compose[@]}" run --rm --no-deps api migrate up
"${compose[@]}" run --rm --no-deps api seed
"${compose[@]}" up -d --wait api web
screenshot_web_container="$("${compose[@]}" ps -q web)"
docker run --rm --ipc=host --network "container:$screenshot_web_container" \
  -v "$PWD:/app" -w /app -e SCREENSHOT_DISPOSABLE=1 \
  mcr.microsoft.com/playwright:v1.58.2-noble node scripts/capture-screenshots.mjs
python3 scripts/optimize-screenshots.py

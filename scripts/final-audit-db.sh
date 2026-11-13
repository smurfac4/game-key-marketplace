#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
areas='seed products auth home commerce meta restore'
cleanup(){ for area in $areas;do docker compose exec -T db dropdb -U marketplace --if-exists "eneba_final_${area}_test" >/dev/null;done; }
trap cleanup EXIT
for area in $areas;do docker compose exec -T db createdb -U marketplace "eneba_final_${area}_test";done
export GOMODCACHE=/private/tmp/eneba-go-mod GOCACHE=/private/tmp/eneba-go-cache
export TEST_DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/eneba_final_seed_test?sslmode=disable'
export TEST_PRODUCTS_DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/eneba_final_products_test?sslmode=disable'
export TEST_AUTH_DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/eneba_final_auth_test?sslmode=disable'
export TEST_HOME_DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/eneba_final_home_test?sslmode=disable'
export TEST_COMMERCE_DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/eneba_final_commerce_test?sslmode=disable'
(cd api && go test -race ./... -count=1 && go vet ./... && go build -o /private/tmp/eneba-audit-migrate ./cmd/migrate && go build -o /private/tmp/eneba-audit-seed ./cmd/seed)
export DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/eneba_final_meta_test?sslmode=disable'
/private/tmp/eneba-audit-migrate up
/private/tmp/eneba-audit-seed
/private/tmp/eneba-audit-migrate down
/private/tmp/eneba-audit-migrate up & a=$!
/private/tmp/eneba-audit-migrate up & b=$!
wait "$a";wait "$b"
docker compose exec -T db pg_dump -U marketplace -Fc eneba_final_meta_test > /private/tmp/eneba-audit-backup.dump
docker compose exec -T db pg_restore -U marketplace -d eneba_final_restore_test --exit-on-error < /private/tmp/eneba-audit-backup.dump
docker compose exec -T db psql -U marketplace -d eneba_final_restore_test -Atc 'SELECT (SELECT count(*) FROM products),(SELECT count(*) FROM offers),(SELECT count(*) FROM orders),(SELECT count(*) FROM game_keys);'
rm /private/tmp/eneba-audit-backup.dump

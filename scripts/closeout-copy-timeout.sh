#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
cleanup(){ docker compose exec -T db dropdb -U marketplace --if-exists eneba_closeout_copy_test >/dev/null;docker compose exec -T db dropdb -U marketplace --if-exists eneba_closeout_timeout_test >/dev/null; }
trap cleanup EXIT
docker compose exec -T db createdb -U marketplace eneba_closeout_copy_test
docker compose exec -T db createdb -U marketplace eneba_closeout_timeout_test
docker compose exec -T db pg_dump -U marketplace -d marketplace -Fc | docker compose exec -T db pg_restore -U marketplace -d eneba_closeout_copy_test --exit-on-error
export DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/eneba_closeout_copy_test?sslmode=disable'
export GOMODCACHE=/private/tmp/eneba-go-mod GOCACHE=/private/tmp/eneba-go-cache
counts="SELECT json_build_object('products',(SELECT count(*) FROM products),'orders',(SELECT count(*) FROM orders),'items',(SELECT count(*) FROM order_items),'sold',(SELECT count(*) FROM game_keys WHERE status='sold'),'keys',(SELECT count(*) FROM game_keys),'stock',(SELECT sum(stock) FROM offers),'legacy_images',(SELECT count(*) FROM products WHERE image_url LIKE '%.jpg'),'neutral_images',(SELECT count(*) FROM products WHERE image_url LIKE '%.svg'));"
docker compose exec -T db psql -U marketplace -d eneba_closeout_copy_test -Atc "$counts"
(cd api && go run ./cmd/migrate up && go run ./cmd/seed -showcase)
docker compose exec -T db psql -U marketplace -d eneba_closeout_copy_test -Atc "$counts"
(cd api && TEST_TIMEOUT_DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/eneba_closeout_timeout_test?sslmode=disable' go test ./internal/httpapi -run TestRealTwelveSecondPoolTimeout -v -count=1)

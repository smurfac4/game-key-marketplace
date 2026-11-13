#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
trap 'docker compose exec -T db dropdb -U marketplace --if-exists eneba_final_perf_test >/dev/null' EXIT
docker compose exec -T db createdb -U marketplace eneba_final_perf_test
export DATABASE_URL='postgres://marketplace:local-development-only@localhost:5432/eneba_final_perf_test?sslmode=disable'
export GOMODCACHE=/private/tmp/eneba-go-mod GOCACHE=/private/tmp/eneba-go-cache
(cd api && go run ./cmd/migrate up && go run ./cmd/seed)
docker compose exec -T db psql -U marketplace -d eneba_final_perf_test -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO products(name,slug,description,platform,region,genre,image_url,release_date)
 SELECT p.name,p.slug||'-scale-'||g,p.description,p.platform,p.region,p.genre,p.image_url,p.release_date FROM products p CROSS JOIN generate_series(1,50) g;
INSERT INTO offers(product_id,seller_id,price,discount_price,currency,stock)
 SELECT p.id,o.seller_id,o.price,o.discount_price,o.currency,o.stock FROM products p JOIN products original ON split_part(p.slug,'-scale-',1)=original.slug JOIN offers o ON o.product_id=original.id WHERE p.slug LIKE '%-scale-%';
ANALYZE;
SELECT count(*) AS products FROM products;
SELECT count(*) AS offers FROM offers;
SQL
(cd api && TEST_PERF_DATABASE_URL="$DATABASE_URL" go test ./internal/products -run TestFinalPerformance -v -count=1)

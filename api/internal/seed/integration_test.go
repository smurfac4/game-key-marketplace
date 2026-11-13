package seed

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/game-key-marketplace/api/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This test owns a disposable database, never the developer's application data.
func TestDatabaseIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to an empty database ending in _test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var db string
	if err = pool.QueryRow(ctx, "SELECT current_database()").Scan(&db); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(db, "_test") {
		t.Fatal("integration database name must end in _test")
	}
	var existing int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'`).Scan(&existing); err != nil {
		t.Fatal(err)
	}
	if existing != 0 {
		t.Fatal("integration database must be empty")
	}
	if err = migrations.Run(url, "up"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := migrations.Run(url, "down"); err != nil {
			t.Fatal(err)
		}
		if err := migrations.Run(url, "down"); err != nil {
			t.Fatal(err)
		}
		if err := migrations.Run(url, "down"); err != nil {
			t.Fatal(err)
		}
		if err := migrations.Run(url, "down"); err != nil {
			t.Fatal(err)
		}
		if err := migrations.Run(url, "down"); err != nil {
			t.Error(err)
		}
	}()
	if err = migrations.Run(url, "up"); err != nil {
		t.Fatal("second up:", err)
	}
	// Fail after sellers, products, and an offer have been inserted to prove atomicity.
	if _, err = pool.Exec(ctx, `CREATE FUNCTION reject_seed_key() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'intentional seed failure'; END $$; CREATE TRIGGER reject_seed_key BEFORE INSERT ON game_keys FOR EACH ROW EXECUTE FUNCTION reject_seed_key()`); err != nil {
		t.Fatal(err)
	}
	if _, err = Run(ctx, pool); err == nil {
		t.Fatal("injected failure should abort seed")
	}
	var rolledBack bool
	if err = pool.QueryRow(ctx, "SELECT NOT EXISTS(SELECT 1 FROM products) AND NOT EXISTS(SELECT 1 FROM sellers) AND NOT EXISTS(SELECT 1 FROM offers) AND NOT EXISTS(SELECT 1 FROM game_keys)").Scan(&rolledBack); err != nil || !rolledBack {
		t.Fatal("failed seed left partial rows", err)
	}
	if _, err = pool.Exec(ctx, "DROP TRIGGER reject_seed_key ON game_keys; DROP FUNCTION reject_seed_key()"); err != nil {
		t.Fatal(err)
	}
	counts, err := Run(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Products != 150 || counts.Sellers != 10 || counts.Offers < 300 || counts.Offers > 1200 {
		t.Fatalf("unexpected counts: %+v", counts)
	}
	if _, err = Run(ctx, pool); err == nil {
		t.Fatal("second seed should refuse populated catalog")
	}
	if again, err := RunHomepage(ctx, pool); err != nil || again.Banners != 0 || again.Orders != 0 {
		t.Fatal("homepage seed must be idempotent", again, err)
	}
	// Presentation refresh must be repeat-safe and must not change purchases or stock.
	for i := 0; i < 2; i++ {
		if _, err = RunShowcase(ctx, pool); err != nil {
			t.Fatal("showcase refresh", err)
		}
	}
	checks := map[string]string{
		"local artwork":   "SELECT count(*)=150 AND bool_and(image_url LIKE '/media/games/%') FROM products",
		"real title demo": "SELECT name='Cyberpunk 2077' AND description LIKE '%non-redeemable%' FROM products WHERE slug='demo-game-001'",

		"paid sales":           "SELECT count(*)=12 AND sum(total_amount)>0 FROM orders WHERE status='paid'",
		"allocated keys":       "SELECT count(*)=24 FROM game_keys WHERE status='sold'",
		"sale quantities":      "SELECT bool_and(quantity=n) FROM (SELECT oi.id,oi.quantity,count(k.id) n FROM order_items oi LEFT JOIN game_keys k ON k.order_item_id=oi.id GROUP BY oi.id) s",
		"banner count":         "SELECT count(*)=3 FROM banners",
		"block count":          "SELECT count(*)=4 FROM homepage_blocks",
		"product count":        "SELECT count(*) = 150 FROM products",
		"seller count":         "SELECT count(*) = 10 FROM sellers",
		"offer bounds":         "SELECT bool_and(n BETWEEN 2 AND 8) FROM (SELECT count(*) n FROM offers GROUP BY product_id) s",
		"key bounds and stock": "SELECT bool_and(n BETWEEN 5 AND 30 AND stock = available) FROM (SELECT o.id,o.stock,count(k.id) n,count(k.id) FILTER (WHERE k.status='available') available FROM offers o LEFT JOIN game_keys k ON k.offer_id=o.id GROUP BY o.id) s",
		"fake keys only":       "SELECT bool_and(key_value LIKE 'FAKE-NOT-REDEEMABLE-%' AND ((status='available' AND order_item_id IS NULL) OR (status='sold' AND order_item_id IS NOT NULL))) FROM game_keys",
		"trigram index":        "SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE tablename='products' AND indexdef LIKE '%USING gin (name gin_trgm_ops)%')",
	}
	for name, query := range checks {
		var ok bool
		if err := pool.QueryRow(ctx, query).Scan(&ok); err != nil || !ok {
			t.Errorf("%s: ok=%v err=%v", name, ok, err)
		}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	var user, order, item, offer, other int64
	mustID := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := tx.QueryRow(ctx, query, args...).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	user = mustID("INSERT INTO users(email,password_hash) VALUES ('test@example.invalid','fake-test-hash') RETURNING id")
	offer = mustID("SELECT id FROM offers ORDER BY id LIMIT 1")
	other = mustID("SELECT id FROM offers WHERE id <> $1 LIMIT 1", offer)
	order = mustID("INSERT INTO orders(user_id,currency,total_amount) VALUES ($1,'EUR',10) RETURNING id", user)
	item = mustID("INSERT INTO order_items(order_id,offer_id,product_name,seller_name,quantity,unit_price,currency) VALUES ($1,$2,'Snapshot','Seller',1,10,'EUR') RETURNING id", order, offer)
	invalid := []struct {
		name, query string
		args        []any
	}{
		{"negative stock", "UPDATE offers SET stock=-1 WHERE id=$1", []any{offer}},
		{"invalid discount", "UPDATE offers SET discount_price=price WHERE id=$1", []any{offer}},
		{"sold without order", "UPDATE game_keys SET status='sold' WHERE id=(SELECT min(id) FROM game_keys WHERE status='available')", nil},
		{"wrong offer key", "INSERT INTO game_keys(offer_id,key_value,status,order_item_id) VALUES ($1,'FAKE-invalid','sold',$2)", []any{other, item}},
		{"available assigned key", "INSERT INTO game_keys(offer_id,key_value,order_item_id) VALUES ($1,'FAKE-invalid',$2)", []any{offer, item}},
		{"mixed order currency", "INSERT INTO order_items(order_id,offer_id,product_name,seller_name,quantity,unit_price,currency) VALUES ($1,$2,'Game','Seller',1,10,'USD')", []any{order, offer}},
		{"zero quantity", "INSERT INTO order_items(order_id,offer_id,product_name,seller_name,quantity,unit_price,currency) VALUES ($1,$2,'Game','Seller',0,10,'EUR')", []any{order, offer}},
		{"duplicate email", "INSERT INTO users(email,password_hash) VALUES ('TEST@example.invalid','hash')", nil},
		{"invalid rating", "INSERT INTO sellers(name,rating) VALUES ('Invalid',5.1)", nil},
		{"unknown key status", "UPDATE game_keys SET status='reserved' WHERE id=(SELECT min(id) FROM game_keys)", nil},
	}
	for _, tc := range invalid {
		if _, err := tx.Exec(ctx, "SAVEPOINT invalid_case"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, tc.query, tc.args...); err == nil {
			t.Errorf("%s should fail", tc.name)
		}
		if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT invalid_case"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE game_keys SET status='sold',order_item_id=$1 WHERE id=(SELECT min(id) FROM game_keys WHERE offer_id=$2)", item, offer); err != nil {
		t.Fatal("valid allocation:", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Run(url, "down"); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Run(url, "down"); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Run(url, "down"); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Run(url, "down"); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Run(url, "down"); err != nil {
		t.Fatal(err)
	}
	var gone bool
	if err := pool.QueryRow(ctx, "SELECT to_regclass('public.products') IS NULL AND to_regclass('public.users') IS NULL").Scan(&gone); err != nil || !gone {
		t.Fatal("down did not remove tables", err)
	}
	if err := migrations.Run(url, "up"); err != nil {
		t.Fatal("up after down:", err)
	}
	if _, err := Run(ctx, pool); err != nil {
		t.Fatal("seed after rollback:", err)
	}
}

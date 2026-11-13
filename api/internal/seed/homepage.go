package seed

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunHomepage adds demo homepage fixtures to the existing demo catalog without resetting it.
func RunHomepage(ctx context.Context, pool *pgxpool.Pool) (Counts, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Counts{}, err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(741530021)"); err != nil {
		return Counts{}, err
	}
	counts, err := addHomepage(ctx, tx)
	if err != nil {
		return Counts{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Counts{}, err
	}
	return counts, nil
}
func addHomepage(ctx context.Context, tx pgx.Tx) (Counts, error) {
	var count, owned int
	if err := tx.QueryRow(ctx, `SELECT count(*),count(*) FILTER (WHERE config->>'seed'='homepage-v1') FROM homepage_blocks WHERE slug IN ('banners','best-sellers','deals','new-releases')`).Scan(&count, &owned); err != nil {
		return Counts{}, err
	}
	if count == 4 && owned == 4 {
		return Counts{}, nil
	}
	if count != 0 {
		return Counts{}, errors.New("homepage seed refused: existing/partial homepage configuration")
	}
	var demoCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM products WHERE slug ~ '^demo-game-[0-9]{3}$' AND (description='Fictional game created for marketplace development and testing. No redeemable content.' OR description LIKE '%Demo inventory: fictional prices, sellers and non-redeemable keys. No publisher affiliation.')`).Scan(&demoCount); err != nil {
		return Counts{}, err
	}
	if demoCount != 150 {
		return Counts{}, errors.New("homepage seed requires the 150-product fictional demo catalog")
	}
	var firstBanner int64
	for i, b := range []struct{ title, link, color string }{
		{"Discover your next adventure", "/#new-releases", "312e81"},
		{"Explore this week's deals", "/#deals", "065f46"},
		{"Favorites from our players", "/#best-sellers", "7c2d12"},
	} {
		var id int64
		err := tx.QueryRow(ctx, `INSERT INTO banners(title,image_url,link_url,position) VALUES ($1,$2,$3,$4) RETURNING id`, b.title, "/media/banners/portal-v2.png", b.link, i).Scan(&id)
		if err != nil {
			return Counts{}, err
		}
		if i == 0 {
			firstBanner = id
		}
	}
	for i, b := range []struct{ slug, title string }{{"banners", "Explore the marketplace"}, {"best-sellers", "Best sellers"}, {"deals", "Deals worth discovering"}, {"new-releases", "New releases"}} {
		kind := "product_collection"
		var banner any
		if i == 0 {
			kind = "banner"
			banner = firstBanner
		}
		if _, err := tx.Exec(ctx, `INSERT INTO homepage_blocks(slug,title,kind,banner_id,position,config) VALUES ($1,$2,$3,$4,$5,'{"seed":"homepage-v1"}')`, b.slug, b.title, kind, banner, i); err != nil {
			return Counts{}, err
		}
	}
	// This account cannot authenticate: deliberately not a valid bcrypt hash.
	var userID int64
	if err := tx.QueryRow(ctx, `INSERT INTO users(email,password_hash) VALUES ('homepage-sales@example.invalid','disabled-demo-account') RETURNING id`).Scan(&userID); err != nil {
		return Counts{}, err
	}
	rows, err := tx.Query(ctx, `SELECT o.id,p.name,s.name,coalesce(o.discount_price,o.price)::text
 FROM products p JOIN LATERAL (SELECT * FROM offers WHERE product_id=p.id AND currency='EUR' AND stock>=3 ORDER BY coalesce(discount_price,price),id LIMIT 1) o ON true
 JOIN sellers s ON s.id=o.seller_id WHERE p.slug ~ '^demo-game-[0-9]{3}$' ORDER BY p.slug LIMIT 12`)
	if err != nil {
		return Counts{}, err
	}
	type sale struct {
		id                     int64
		product, seller, price string
	}
	sales := []sale{}
	for rows.Next() {
		var s sale
		if err = rows.Scan(&s.id, &s.product, &s.seller, &s.price); err != nil {
			rows.Close()
			return Counts{}, err
		}
		sales = append(sales, s)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return Counts{}, err
	}
	if len(sales) != 12 {
		return Counts{}, errors.New("homepage seed requires 12 demo products with available EUR inventory")
	}
	for i, s := range sales {
		qty := i%3 + 1
		var orderID, itemID int64
		if err = tx.QueryRow(ctx, `INSERT INTO orders(user_id,status,currency,total_amount) VALUES ($1,'paid','EUR',$2::numeric*$3::integer) RETURNING id`, userID, s.price, qty).Scan(&orderID); err != nil {
			return Counts{}, err
		}
		if err = tx.QueryRow(ctx, `INSERT INTO order_items(order_id,offer_id,product_name,seller_name,quantity,unit_price,currency) VALUES ($1,$2,$3,$4,$5,$6,'EUR') RETURNING id`, orderID, s.id, s.product, s.seller, qty, s.price).Scan(&itemID); err != nil {
			return Counts{}, err
		}
		tag, err := tx.Exec(ctx, `UPDATE game_keys SET status='sold',order_item_id=$1 WHERE id IN (SELECT id FROM game_keys WHERE offer_id=$2 AND status='available' ORDER BY id LIMIT $3 FOR UPDATE)`, itemID, s.id, qty)
		if err != nil {
			return Counts{}, err
		}
		if tag.RowsAffected() != int64(qty) {
			return Counts{}, errors.New("insufficient fake keys for demo sales")
		}
		tag, err = tx.Exec(ctx, `UPDATE offers SET stock=stock-$2 WHERE id=$1 AND stock>=$2`, s.id, qty)
		if err != nil {
			return Counts{}, err
		}
		if tag.RowsAffected() != 1 {
			return Counts{}, errors.New("insufficient stock for demo sales")
		}
	}
	return Counts{Banners: 3, Orders: 12}, nil
}

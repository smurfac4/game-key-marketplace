package home_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"example.com/game-key-marketplace/api/internal/home/banners"
	"example.com/game-key-marketplace/api/internal/home/bestsellers"
	"example.com/game-key-marketplace/api/internal/home/block"
	"example.com/game-key-marketplace/api/internal/home/deals"
	"example.com/game-key-marketplace/api/internal/home/newreleases"
	"example.com/game-key-marketplace/api/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHomeIntegration(t *testing.T) {
	url := os.Getenv("TEST_HOME_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_HOME_DATABASE_URL to empty *_test database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var db string
	var count int
	if err = pool.QueryRow(ctx, "SELECT current_database()").Scan(&db); err != nil || !strings.HasSuffix(db, "_test") {
		t.Fatal("test database required", err)
	}
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'").Scan(&count); err != nil || count != 0 {
		t.Fatal("empty database required", err)
	}
	if err = migrations.Run(url, "up"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		for i := 0; i < 6; i++ {
			if err := migrations.Run(url, "down"); err != nil {
				t.Error(err)
			}
		}
	}()
	_, err = pool.Exec(ctx, `
 INSERT INTO sellers(name) VALUES ('Seller One'),('Seller Two');
 INSERT INTO products(name,slug,platform,region,genre,image_url,release_date) VALUES
 ('Popular','popular','Steam','Global','RPG','https://example.invalid/1',CURRENT_DATE-1),
 ('Newest','newest','Steam','Global','RPG','https://example.invalid/2',CURRENT_DATE),
 ('Future','future','Steam','Global','RPG','https://example.invalid/3',CURRENT_DATE+1),
 ('Sold','sold','Steam','Global','RPG','https://example.invalid/4',CURRENT_DATE-1),
 ('Regular','regular','Steam','Global','RPG','https://example.invalid/5',CURRENT_DATE-1),
 ('Unknown','unknown','Steam','Global','RPG','https://example.invalid/6',NULL);
 INSERT INTO offers(product_id,seller_id,price,discount_price,currency,stock) VALUES
 (1,1,20,10,'EUR',2),(1,2,9,NULL,'EUR',1),(2,1,30,24,'EUR',2),(3,1,50,25,'EUR',2),
 (4,1,1,0.5,'EUR',0),(5,1,5,NULL,'EUR',2),(6,1,10,NULL,'EUR',2),(1,1,1,0.8,'USD',5);
 INSERT INTO users(email,password_hash) VALUES ('home@example.invalid','disabled');
 INSERT INTO orders(user_id,status,currency,total_amount) VALUES (1,'paid','EUR',100),(1,'pending','EUR',100),(1,'cancelled','EUR',100);
 INSERT INTO order_items(order_id,offer_id,product_name,seller_name,quantity,unit_price,currency) VALUES
 (1,1,'Popular','Seller One',2,10,'EUR'),(1,2,'Popular','Seller Two',3,9,'EUR'),(1,3,'Newest','Seller One',1,24,'EUR'),
 (2,3,'Newest','Seller One',100,24,'EUR'),(3,4,'Future','Seller One',100,25,'EUR'),(1,5,'Sold','Seller One',9,0.5,'EUR');
 INSERT INTO banners(title,image_url,link_url,position,active,starts_at,ends_at) VALUES
 ('Live banner','https://example.invalid/live','/#deals',2,true,now()-interval '1 day',now()+interval '1 day'),
 ('Inactive','https://example.invalid/no','/#deals',0,false,NULL,NULL),
 ('Future','https://example.invalid/no','/#deals',0,true,now()+interval '1 day',NULL),
 ('Expired','https://example.invalid/no','/#deals',0,true,NULL,now()-interval '1 day'),
 ('First banner','https://example.invalid/first','/#deals',1,true,NULL,NULL);
 INSERT INTO homepage_blocks(slug,title,kind,banner_id) VALUES ('banners','Editorial banners','banner',1),
 ('best-sellers','Editorial best sellers','product_collection',NULL),('deals','Editorial deals','product_collection',NULL),('new-releases','Editorial new releases','product_collection',NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	br := banners.Repository{DB: pool}
	bs := bestsellers.Repository{DB: pool}
	dr := deals.Repository{DB: pool}
	nr := newreleases.Repository{DB: pool}
	b, err := br.List(ctx, "EUR")
	if err != nil || b.Title != "Editorial banners" || len(b.Data) != 2 || b.Data[0].Title != "First banner" || b.Data[1].Title != "Live banner" {
		t.Fatal(b, err)
	}
	for _, tc := range []struct {
		name  string
		fetch func(context.Context, string) (block.Response[block.Card], error)
		ids   []int64
	}{
		{"best", bs.List, []int64{1, 2}}, {"deals", dr.List, []int64{1, 3, 2}}, {"new", nr.List, []int64{2, 1, 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.fetch(ctx, "EUR")
			if err != nil {
				t.Fatal(err)
			}
			ids := []int64{}
			for _, c := range got.Data {
				ids = append(ids, c.ID)
				if c.Currency != "EUR" {
					t.Fatal("mixed currency")
				}
			}
			if !reflect.DeepEqual(ids, tc.ids) {
				t.Fatal(ids)
			}
			if tc.name == "best" && (got.Data[0].UnitsSold != 5 || got.Data[0].Price != "9.00") {
				t.Fatal("sales aggregation multiplied by offers", got)
			}
			if tc.name == "deals" && (got.Data[0].Price != "10.00" || *got.Data[0].OriginalPrice != "20.00") {
				t.Fatal("discounted offer mismatch", got)
			}
			empty, err := tc.fetch(ctx, "JPY")
			if err != nil || empty.Data == nil || len(empty.Data) != 0 {
				t.Fatal("empty currency", empty, err)
			}
			usd, err := tc.fetch(ctx, "USD")
			if err != nil || len(usd.Data) != 1 || usd.Data[0].ID != 1 {
				t.Fatal("currency filter", usd, err)
			}
		})
	}
	for _, h := range []http.HandlerFunc{banners.Handler(br), bestsellers.Handler(bs), deals.Handler(dr), newreleases.Handler(nr)} {
		for _, tc := range []struct {
			query  string
			status int
		}{{"", 200}, {"?currency=eur", 400}, {"?currency=EUR&currency=USD", 400}, {"?limit=1", 400}} {
			rr := httptest.NewRecorder()
			h(rr, httptest.NewRequest("GET", "/api/home/test"+tc.query, nil))
			if rr.Code != tc.status || !json.Valid(rr.Body.Bytes()) {
				t.Fatal(rr.Code, rr.Body.String())
			}
		}
	}
	if _, err = pool.Exec(ctx, "UPDATE homepage_blocks SET active=false WHERE slug='deals'"); err != nil {
		t.Fatal(err)
	}
	d, err := dr.List(ctx, "EUR")
	if err != nil || d.Title != "" || len(d.Data) != 0 {
		t.Fatal("disabled block", d, err)
	}
	best, err := bs.List(ctx, "EUR")
	if err != nil || len(best.Data) != 2 {
		t.Fatal("one disabled block affected another", err)
	}
	if _, err = pool.Exec(ctx, "DELETE FROM homepage_blocks WHERE slug='new-releases'"); err != nil {
		t.Fatal(err)
	}
	n, err := nr.List(ctx, "EUR")
	if err != nil || n.Title != "" || n.Data == nil {
		t.Fatal("missing block", n, err)
	}
}

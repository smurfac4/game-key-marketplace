package products

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"example.com/game-key-marketplace/api/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestListIntegration(t *testing.T) {
	url := os.Getenv("TEST_PRODUCTS_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_PRODUCTS_DATABASE_URL to an empty database ending in _test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var db string
	var count int
	if err = pool.QueryRow(ctx, "SELECT current_database()").Scan(&db); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(db, "_test") {
		t.Fatal("database must end in _test")
	}
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'").Scan(&count); err != nil || count != 0 {
		t.Fatal("test requires empty database", err)
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
			t.Fatal(err)
		}
		if err := migrations.Run(url, "down"); err != nil {
			t.Error(err)
		}
	}()
	_, err = pool.Exec(ctx, `
 INSERT INTO sellers(name) VALUES ('One'),('Two'),('Three');
 INSERT INTO products(name,slug,platform,region,genre,image_url,release_date) VALUES
 ('Alpha','alpha','Steam','Europe','RPG','https://example.invalid/1','2020-01-01'),
 ('Beta','beta','GOG','Global','Action','https://example.invalid/2','2024-01-01'),
 ('Gamma','gamma','Steam','Global','RPG','https://example.invalid/3',NULL),
 ('Sold Out','sold-out','Steam','Europe','RPG','https://example.invalid/4','2025-01-01'),
 ('100%_Fun','literal','Steam','Europe','Action','https://example.invalid/5','2022-01-01'),
 ('No offers','no-offers','GOG','Global','RPG','https://example.invalid/6',NULL);
 INSERT INTO offers(product_id,seller_id,price,discount_price,currency,stock) VALUES
 (1,1,20,10,'EUR',2),(1,2,15,NULL,'EUR',1),(1,3,1,NULL,'EUR',0),
 (2,1,8,NULL,'EUR',1),(3,1,10,NULL,'EUR',1),(4,1,1,NULL,'EUR',0),
 (5,1,12,NULL,'EUR',1),(1,1,2,NULL,'USD',1);
 INSERT INTO users(email,password_hash) VALUES ('test@example.invalid','fake-hash');
 INSERT INTO orders(user_id,status,currency,total_amount) VALUES (1,'paid','EUR',100),(1,'pending','EUR',100),(1,'cancelled','EUR',100);
 INSERT INTO order_items(order_id,offer_id,product_name,seller_name,quantity,unit_price,currency) VALUES
 (1,1,'Alpha','One',2,10,'EUR'),(1,2,'Alpha','Two',3,15,'EUR'),(1,4,'Beta','One',1,8,'EUR'),
 (2,5,'Gamma','One',100,10,'EUR'),(3,5,'Gamma','One',100,10,'EUR');`)
	if err != nil {
		t.Fatal(err)
	}
	repo := Repository{DB: pool}
	cases := []struct {
		name, raw string
		ids       []int64
		total     int64
	}{
		{"lowest effective available price", "sort=price_asc", []int64{2, 1, 3, 5}, 4},
		{"price descending", "sort=price_desc", []int64{5, 1, 3, 2}, 4},
		{"paid sales popularity", "sort=popularity", []int64{1, 2, 3, 5}, 4},
		{"newest null last", "sort=newest", []int64{2, 5, 1, 3}, 4},
		{"name", "sort=name", []int64{5, 1, 2, 3}, 4},
		{"pagination", "sort=price_asc&page=2&page_size=2", []int64{3, 5}, 4},
		{"beyond last", "page=9", []int64{}, 4},
		{"combined filters", "platform=Steam&region=Europe&genre=RPG", []int64{1}, 1},
		{"price boundaries", "min_price=10&max_price=10", []int64{1, 3}, 2},
		{"min applies after cheapest offer", "min_price=11", []int64{5}, 1},
		{"case insensitive substring", "q=LPH", []int64{1}, 1},
		{"literal wildcards", "q=%25_", []int64{5}, 1},
		{"injection text", "q=%27%20OR%201%3D1--", []int64{}, 0},
		{"unknown facet value", "platform=Unknown", []int64{}, 0},
		{"currency isolation", "currency=USD", []int64{1}, 1},
		{"no offers in currency", "currency=JPY", []int64{}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := ParseQuery(tc.raw)
			if err != nil {
				t.Fatal(err)
			}
			got, err := repo.List(ctx, q)
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]int64, 0, len(got.Data))
			for _, p := range got.Data {
				ids = append(ids, p.ID)
			}
			if !reflect.DeepEqual(ids, tc.ids) || got.Pagination.Total != tc.total {
				t.Fatalf("got IDs %v total %d", ids, got.Pagination.Total)
			}
			expectedPages := (tc.total + int64(q.PageSize) - 1) / int64(q.PageSize)
			if got.Pagination.Page != q.Page || got.Pagination.PageSize != q.PageSize || got.Pagination.TotalPages != expectedPages {
				t.Fatalf("pagination: %+v", got.Pagination)
			}
			for _, facets := range [][]Facet{got.Facets.Platform, got.Facets.Region, got.Facets.Genre} {
				var sum int64
				for _, f := range facets {
					sum += f.Count
				}
				if sum < tc.total || (q.Platform == "" && q.Region == "" && q.Genre == "" && sum != tc.total) {
					t.Fatalf("facets count offers or pages: %+v", got.Facets)
				}
			}
			if tc.name == "lowest effective available price" && (got.Data[1].Price != "10.00" || got.Data[1].Popularity != 5) {
				t.Fatalf("price/popularity: %+v", got.Data[1])
			}
			if tc.name == "pagination" && !reflect.DeepEqual(got.Facets.Platform, []Facet{{"GOG", 1}, {"Steam", 3}}) {
				t.Fatalf("facets: %+v", got.Facets)
			}
			if tc.name == "currency isolation" && (got.Data[0].Price != "2.00" || got.Data[0].Currency != "USD") {
				t.Fatal("mixed currency prices")
			}
		})
	}
	for _, tc := range []struct {
		raw             string
		platform, genre []Facet
	}{
		{"platform=Steam", []Facet{{"GOG", 1}, {"Steam", 3}}, []Facet{{"Action", 1}, {"RPG", 2}}},
		{"platform=GOG&genre=RPG", []Facet{{"Steam", 2}}, []Facet{{"Action", 1}}},
		{"platform=Steam&max_price=10", []Facet{{"GOG", 1}, {"Steam", 2}}, []Facet{{"RPG", 2}}},
	} {
		q, _ := ParseQuery(tc.raw)
		got, err := repo.List(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Facets.Platform, tc.platform) || !reflect.DeepEqual(got.Facets.Genre, tc.genre) {
			t.Fatalf("disjunctive facets %s: %+v", tc.raw, got.Facets)
		}
	}
	testDetail(t, ctx, pool)
	testSearch(t, ctx, pool)
	testSimilarLimit(t, ctx, pool)
}

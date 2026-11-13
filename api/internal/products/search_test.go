package products

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSuggestQuery(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		valid bool
	}{
		{"q=cyberpnk", true}, {"q=fifa+2", true}, {"q=witcer&currency=USD", true},
		{"", false}, {"q=", false}, {"q=a&q=b", false}, {"q=x&sort=name", false}, {"q=%FF", false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			_, err := ParseSuggest(tc.raw)
			if (err == nil) != tc.valid {
				t.Fatalf("err=%v", err)
			}
		})
	}
	q, _ := ParseSuggest("q=a")
	got, err := (Repository{}).Suggest(context.Background(), q)
	if err != nil || len(got.Data) != 0 {
		t.Fatal("one-character suggestion should skip DB", got, err)
	}
}
func testSearch(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	names := []string{"Cyberpunk 2077", "The Witcher 3: Wild Hunt", "FIFA 23", "FIFA 24", "Cyberpnk", "FIFA 20", "FIFA 21", "FIFA 22", "FIFA 25", "FIFA 26", "FIFA 27", "FIFA 28", "FIFA 29", "FIFA 2", "Cyberpunk unavailable"}
	for i, name := range names {
		var id int64
		err := pool.QueryRow(ctx, `INSERT INTO products(name,slug,platform,region,genre,image_url) VALUES ($1,$2,'Steam','Global','RPG','https://example.invalid/search') RETURNING id`, name, fmt.Sprintf("search-%d", i)).Scan(&id)
		// Slugs are independent from names and must satisfy the schema's URL-safe constraint.
		if err != nil {
			t.Fatal(err)
		}
		stock := 1
		if i == len(names)-1 {
			stock = 0
		}
		if _, err := pool.Exec(ctx, `INSERT INTO offers(product_id,seller_id,price,currency,stock) VALUES ($1,1,$2,'EUR',$3)`, id, 10+i, stock); err != nil {
			t.Fatal(err)
		}
	}
	repo := Repository{DB: pool}
	for _, tc := range []struct{ query, first, contains string }{
		{"cyberpnk", "Cyberpnk", "Cyberpunk 2077"},
		{"fifa 2", "FIFA 2", "FIFA 23"},
		{"witcer", "The Witcher 3: Wild Hunt", "The Witcher 3: Wild Hunt"},
		{"CYBERPUNK", "Cyberpunk 2077", "Cyberpunk 2077"},
		{"witcher 3", "The Witcher 3: Wild Hunt", "The Witcher 3: Wild Hunt"},
		{"zzzzqqqqxxxx", "", ""}, {"%_", "100%_Fun", "100%_Fun"},
	} {
		t.Run("search "+tc.query, func(t *testing.T) {
			q, err := ParseQuery("q=" + url.QueryEscape(tc.query))
			if err != nil {
				t.Fatal(err)
			}
			if q.Sort != "relevance" {
				t.Fatal("search should default to relevance")
			}
			got, err := repo.List(ctx, q)
			if err != nil {
				t.Fatal(err)
			}
			suggestions, err := repo.Suggest(ctx, q)
			if err != nil {
				t.Fatal(err)
			}
			if tc.first == "" {
				if len(got.Data) != 0 || len(suggestions.Data) != 0 {
					t.Fatal("unrelated query matched")
				}
				return
			}
			if len(got.Data) == 0 || got.Data[0].Name != tc.first || len(suggestions.Data) == 0 || suggestions.Data[0].Name != tc.first {
				t.Fatalf("ranking: list=%+v suggestions=%+v", got.Data, suggestions)
			}
			found := false
			for _, p := range got.Data {
				found = found || p.Name == tc.contains
				if p.Name == "Cyberpunk unavailable" {
					t.Fatal("unavailable product included")
				}
			}
			if !found {
				t.Fatalf("missing typo/partial match %s", tc.contains)
			}
			n := min(8, len(got.Data))
			if len(suggestions.Data) != n {
				t.Fatal("suggestion cap/count")
			}
			for i, s := range suggestions.Data {
				if s.ID != got.Data[i].ID {
					t.Fatal("listing and suggest ranking differ")
				}
			}
			if tc.query == "fifa 2" && len(suggestions.Data) != 8 {
				t.Fatal("expected top 8")
			}
			again, err := repo.Suggest(ctx, q)
			if err != nil || !reflect.DeepEqual(again, suggestions) {
				t.Fatal("unstable suggestions", err)
			}
		})
	}
	q, _ := ParseQuery("q=fifa+2&sort=price_desc")
	got, err := repo.List(ctx, q)
	if err != nil || got.Data[0].Name != "FIFA 2" {
		t.Fatal("explicit sort should override relevance", got, err)
	}
	q, _ = ParseQuery("q=witcer&platform=GOG")
	got, err = repo.List(ctx, q)
	if err != nil || got.Pagination.Total != 0 {
		t.Fatal("fuzzy search must preserve filters", err)
	}
	var threshold string
	if err := pool.QueryRow(ctx, "SHOW pg_trgm.word_similarity_threshold").Scan(&threshold); err != nil || threshold != "0.6" {
		t.Fatal("threshold leaked from transaction", threshold, err)
	}
}

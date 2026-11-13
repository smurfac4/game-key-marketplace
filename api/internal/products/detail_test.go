package products

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testDetail(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	repo := Repository{DB: pool}
	for _, tc := range []struct {
		slug            string
		offers, similar []int64
	}{
		{"alpha", []int64{3, 8, 1, 2}, []int64{3}},
		{"sold-out", []int64{6}, []int64{1, 3}},
		{"no-offers", []int64{}, []int64{3, 1}},
		{"beta", []int64{4}, []int64{5}},
	} {
		t.Run("detail "+tc.slug, func(t *testing.T) {
			got, err := repo.Detail(ctx, tc.slug)
			if err != nil {
				t.Fatal(err)
			}
			if got.Data.Slug != tc.slug || got.Data.Name == "" {
				t.Fatalf("missing details: %+v", got.Data)
			}
			offers, similar := []int64{}, []int64{}
			for _, o := range got.Data.Offers {
				offers = append(offers, o.ID)
				if o.Seller.ID == 0 || o.Seller.Name == "" {
					t.Fatal("missing seller")
				}
			}
			for _, p := range got.SimilarProducts {
				similar = append(similar, p.ID)
				if p.ID == got.Data.ID || p.Genre != got.Data.Genre {
					t.Fatal("invalid recommendation")
				}
			}
			if !reflect.DeepEqual(offers, tc.offers) || !reflect.DeepEqual(similar, tc.similar) {
				t.Fatalf("offers=%v similar=%v", offers, similar)
			}
			if tc.slug == "alpha" {
				discounted := got.Data.Offers[2]
				if discounted.Price != "20.00" || discounted.DiscountPrice == nil || *discounted.DiscountPrice != "10.00" || discounted.EffectivePrice != "10.00" {
					t.Fatalf("discount: %+v", discounted)
				}
				if got.Data.Offers[0].Stock != 0 || got.Data.Offers[1].Currency != "USD" || got.Data.Offers[3].DiscountPrice != nil {
					t.Fatal("all offers including sold-out and foreign currency must be present")
				}
			}
			if tc.slug == "no-offers" && got.Data.ReleaseDate != nil {
				t.Fatal("unknown date should remain null")
			}
			raw, _ := json.Marshal(got)
			if strings.Contains(string(raw), "key_value") || strings.Contains(string(raw), "password_hash") {
				t.Fatal("secret fields leaked")
			}
		})
	}
	for _, slug := range []string{"unknown", "ALPHA", "' OR 1=1 --"} {
		if _, err := repo.Detail(ctx, slug); !errors.Is(err, ErrNotFound) {
			t.Fatalf("%q expected not found, got %v", slug, err)
		}
	}
	// Seller values must come from the database, not static response defaults.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "UPDATE sellers SET name='Updated Seller',rating=4.75 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	got, err := (Repository{DB: tx}).Detail(ctx, "beta")
	if err != nil || got.Data.Offers[0].Seller.Name != "Updated Seller" || got.Data.Offers[0].Seller.Rating != 4.75 {
		t.Fatal("seller data", got, err)
	}
}
func testSimilarLimit(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	repo := Repository{DB: pool}
	a, err := repo.Detail(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	b, err := repo.Detail(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if len(a.SimilarProducts) != 8 || !reflect.DeepEqual(a.SimilarProducts, b.SimilarProducts) {
		t.Fatal("expected stable top eight")
	}
	for _, p := range a.SimilarProducts {
		if p.Slug == "sold-out" || p.Slug == "no-offers" || p.Slug == "alpha" {
			t.Fatal("unavailable/self recommendation")
		}
	}
	// No same-genre candidates must produce an empty array, not null.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "UPDATE products SET genre='UniqueGenre' WHERE slug='alpha'"); err != nil {
		t.Fatal(err)
	}
	got, err := (Repository{DB: tx}).Detail(ctx, "alpha")
	if err != nil || got.SimilarProducts == nil || len(got.SimilarProducts) != 0 {
		t.Fatal("empty recommendations", got, err)
	}
}

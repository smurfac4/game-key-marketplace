package seed

import (
	"reflect"
	"testing"
)

func TestGenerate(t *testing.T) {
	products := Generate()
	if len(products) != 150 || !reflect.DeepEqual(products, Generate()) {
		t.Fatal("expected 150 deterministic products")
	}
	slugs, names := map[string]bool{}, map[string]bool{}
	minOffers, maxOffers, minKeys, maxKeys := 8, 2, 30, 5
	discounted, fullPrice := false, false
	for _, p := range products {
		if slugs[p.Slug] || names[p.Name+"|"+p.Platform+"|"+p.Region] {
			t.Fatal("duplicate product")
		}
		slugs[p.Slug], names[p.Name+"|"+p.Platform+"|"+p.Region] = true, true
		if len(p.Offers) < 2 || len(p.Offers) > 8 {
			t.Fatal("offer count outside bounds")
		}
		minOffers, maxOffers = min(minOffers, len(p.Offers)), max(maxOffers, len(p.Offers))
		sellers := map[int]bool{}
		for _, o := range p.Offers {
			if o.Seller < 0 || o.Seller >= 10 || sellers[o.Seller] {
				t.Fatal("invalid or duplicate seller")
			}
			sellers[o.Seller] = true
			if o.Keys < 5 || o.Keys > 30 {
				t.Fatal("key count outside bounds")
			}
			minKeys, maxKeys = min(minKeys, o.Keys), max(maxKeys, o.Keys)
			if o.PriceCents <= 0 || o.DiscountCents < 0 || o.DiscountCents >= o.PriceCents {
				t.Fatal("invalid price")
			}
			discounted = discounted || o.DiscountCents > 0
			fullPrice = fullPrice || o.DiscountCents == 0
		}
	}
	if minOffers != 2 || maxOffers != 8 || minKeys != 5 || maxKeys != 30 || !discounted || !fullPrice {
		t.Fatal("fixtures should cover inventory and price boundaries")
	}
}
func TestMoney(t *testing.T) {
	for cents, want := range map[int]string{0: "0.00", 1: "0.01", 99: "0.99", 100: "1.00", 12345: "123.45"} {
		if got := money(cents); got != want {
			t.Errorf("money(%d) = %s, want %s", cents, got, want)
		}
	}
}

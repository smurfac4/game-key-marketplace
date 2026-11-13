package products

import (
	"errors"
	"strings"
	"testing"
)

func TestParseQuery(t *testing.T) {
	tests := []struct{ name, raw, field string }{
		{"defaults", "", ""}, {"all options", "platform=Steam&region=Europe&genre=RPG&q=hello+world&min_price=1.2&max_price=12&sort=price_asc&page=2&page_size=100&currency=USD", ""},
		{"equal prices", "min_price=0&max_price=0", ""}, {"max price", "max_price=9999999999.99", ""},
		{"bad encoding", "q=%ZZ", "query"}, {"invalid utf8", "q=%FF", "q"}, {"control", "q=%00", "q"},
		{"unknown", "limit=20", "limit"}, {"duplicate", "genre=RPG&genre=Action", "genre"}, {"blank", "q=+", "q"},
		{"long search", "q=" + strings.Repeat("a", 201), "q"}, {"long filter", "platform=" + strings.Repeat("a", 81), "platform"},
		{"bad sort", "sort=price%3BDROP", "sort"}, {"negative page", "page=-1", "page"}, {"zero page", "page=0", "page"},
		{"page overflow", "page=999999999999999999999999", "page"}, {"page cap", "page=1000001", "page"}, {"fraction page", "page=1.5", "page"},
		{"zero size", "page_size=0", "page_size"}, {"large size", "page_size=101", "page_size"},
		{"negative price", "min_price=-1", "min_price"}, {"exponent", "max_price=1e2", "max_price"}, {"NaN", "min_price=NaN", "min_price"},
		{"precision", "min_price=1.001", "min_price"}, {"price overflow", "max_price=10000000000", "max_price"},
		{"reversed", "min_price=1.02&max_price=1.01", "min_price"}, {"lower currency", "currency=eur", "currency"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q, err := ParseQuery(tc.raw)
			if tc.field != "" {
				var e *ValidationError
				if !errors.As(err, &e) || e.Field != tc.field {
					t.Fatalf("got %v, want field %s", err, tc.field)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "defaults" && (q.Page != 1 || q.PageSize != 24 || q.Sort != "popularity" || q.Currency != "EUR") {
				t.Fatalf("defaults: %+v", q)
			}
			if tc.name == "all options" && (q.Search != "hello world" || *q.MinPrice != "1.20" || *q.MaxPrice != "12.00" || q.Page != 2 || q.Currency != "USD") {
				t.Fatalf("parsed: %+v", q)
			}
		})
	}
	for sort := range sortSQL {
		if _, err := ParseQuery("sort=" + sort); err != nil {
			t.Fatal(err)
		}
	}
}

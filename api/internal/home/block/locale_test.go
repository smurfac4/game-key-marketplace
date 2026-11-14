package block

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestLocalizedTitle(t *testing.T) {
	for _, tc := range []struct{ header, title, want string }{
		{"lt", "Best sellers", "Perkamiausi žaidimai"},
		{"lt-LT, en;q=0.8", "Discover your next adventure", "Atraskite naują nuotykį"},
		{" LT-lt ", "Explore this week's deals", "Atraskite šios savaitės pasiūlymus"},
		{"lt;q=0.9", "Favorites from our players", "Žaidėjų mėgstamiausi"},
		{"lt", "New releases", "Naujausi leidimai"},
		{"lt", "Explore the marketplace", "Atraskite žaidimų pasaulį"},
		{"lt", "Deals worth discovering", "Verti dėmesio pasiūlymai"},
		{"ru-RU", "Best sellers", "Лидеры продаж"},
		{"en", "Best sellers", "Best sellers"},
		{"de", "Best sellers", "Best sellers"},
		{"lt", "Custom catalog title", "Custom catalog title"},
	} {
		t.Run(tc.header+tc.title, func(t *testing.T) {
			if got := LocalizedTitle(tc.title, tc.header); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
func TestServeLithuanian(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/home/best-sellers", nil)
	r.Header.Set("Accept-Language", "lt-LT")
	rr := httptest.NewRecorder()
	Serve(rr, r, func(context.Context, string) (Response[Card], error) {
		return Response[Card]{Title: "Best sellers", Data: []Card{}}, nil
	})
	var result Response[Card]
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if rr.Code != 200 || result.Title != "Perkamiausi žaidimai" || rr.Header().Get("Vary") != "Accept-Language" {
		t.Fatal("localized block response or cache variation failed")
	}
}

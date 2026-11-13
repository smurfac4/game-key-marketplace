package httpapi

import (
	"example.com/game-key-marketplace/api/internal/money"
	"net/http"
	"strings"
)

func displayCurrency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := r.Header.Get("X-Display-Currency")
		if c != "" {
			if !money.Supported(c) {
				writeError(w, 400, "invalid_query", "unsupported currency", "currency")
				return
			}
			if r.Method == "GET" || strings.HasPrefix(r.URL.Path, "/api/cart") {
				q := r.URL.Query()
				q.Set("currency", c)
				r.URL.RawQuery = q.Encode()
				r = r.WithContext(money.WithCurrency(r.Context(), c))
				w.Header().Add("Vary", "X-Display-Currency")
			}
		}
		next.ServeHTTP(w, r)
	})
}

package httpapi

import (
	"context"
	"net/http"
	"time"
)

// Socket write timeouts do not cancel database work. Bound the entire request,
// including authentication and pool/lock waits; narrower feature deadlines win.
func requestBudget(budget time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), budget)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

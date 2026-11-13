package httpapi

import (
	"example.com/game-key-marketplace/api/internal/auth"
	"net"
	"net/http"
	"strconv"
)

func limitRequests(l *auth.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				key = r.RemoteAddr
			}
			if u, ok := CurrentUser(r.Context()); ok {
				key = "user:" + strconv.FormatInt(u.ID, 10)
			}
			if ok, retry := l.Allow(key); !ok {
				w.Header().Set("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
				writeError(w, 429, "rate_limited", "too many requests", "")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

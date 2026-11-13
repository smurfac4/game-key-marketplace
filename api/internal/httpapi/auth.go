package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/game-key-marketplace/api/internal/auth"
	"github.com/go-chi/chi/v5/middleware"
)

type Auth struct {
	Service *auth.Service
	Limiter *auth.Limiter
}

func NewAuth(service *auth.Service) *Auth { return &Auth{Service: service, Limiter: auth.NewLimiter()} }

type userContextKey struct{}

func CurrentUser(ctx context.Context) (auth.User, bool) {
	u, ok := ctx.Value(userContextKey{}).(auth.User)
	return u, ok
}

// Browser writes must pass both an explicit-origin policy and a non-simple header.
// Browsers cannot attach this header cross-origin without an allowed preflight.
func (a *Auth) BrowserSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Add("Vary", "Origin")
		if origin != "" {
			if !a.Service.Config.Origins[origin] {
				writeError(w, 403, "forbidden", "origin not allowed", "")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			if origin == "" {
				writeError(w, 403, "forbidden", "origin required", "")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Protection, Idempotency-Key, X-Display-Currency")
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" && r.Header.Get("X-CSRF-Protection") != "1" {
			writeError(w, 403, "csrf_failed", "X-CSRF-Protection: 1 is required", "")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		cookie, err := r.Cookie(auth.CookieName)
		if err != nil {
			writeError(w, 401, "unauthorized", "authentication required", "")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		user, err := a.Service.Authenticate(ctx, cookie.Value)
		if err != nil {
			a.failure(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey{}, user)))
	})
}
func (a *Auth) ProtectCommerce(next http.Handler) http.Handler {
	protected := a.Require(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, prefix := range []string{"/api/checkout", "/api/orders"} {
			if r.URL.Path == prefix || strings.HasPrefix(r.URL.Path, prefix+"/") {
				protected.ServeHTTP(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func (a *Auth) limited(w http.ResponseWriter, r *http.Request) bool {
	// Do not trust X-Forwarded-For from arbitrary clients. RealIP middleware is disabled.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	ok, retry := a.Limiter.Allow(ip)
	if !ok {
		w.Header().Set("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
		writeError(w, 429, "rate_limited", "too many authentication attempts", "")
	}
	return !ok
}
func readCredentials(w http.ResponseWriter, r *http.Request) (auth.Credentials, bool) {
	var c auth.Credentials
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, 415, "unsupported_media_type", "Content-Type must be application/json", "")
		return c, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		writeError(w, 400, "invalid_body", "invalid JSON credentials", "")
		return c, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, 400, "invalid_body", "expected one JSON object", "")
		return c, false
	}
	return c, true
}
func (a *Auth) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if a.limited(w, r) {
		return
	}
	c, ok := readCredentials(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	user, err := a.Service.Register(ctx, c)
	if err != nil {
		a.failure(w, r, err)
		return
	}
	writeJSON(w, 201, struct {
		Data auth.User `json:"data"`
	}{user})
}
func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if a.limited(w, r) {
		return
	}
	c, ok := readCredentials(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	user, token, err := a.Service.Login(ctx, c)
	if err != nil {
		a.failure(w, r, err)
		return
	}
	http.SetCookie(w, a.Service.Config.Cookie(token, time.Now()))
	writeJSON(w, 200, struct {
		Data auth.User `json:"data"`
	}{user})
}
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		if err = a.Service.Logout(ctx, cookie.Value); err != nil {
			a.failure(w, r, err)
			return
		}
	}
	http.SetCookie(w, a.Service.Config.ClearCookie())
	w.WriteHeader(204)
}
func (a *Auth) Me(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r.Context())
	writeJSON(w, 200, struct {
		Data auth.User `json:"data"`
	}{user})
}
func (a *Auth) failure(w http.ResponseWriter, r *http.Request, err error) {
	var validation *auth.ValidationError
	switch {
	case errors.As(err, &validation):
		writeError(w, 400, "invalid_input", validation.Message, validation.Field)
	case errors.Is(err, auth.ErrUnauthorized):
		writeError(w, 401, "unauthorized", "invalid credentials or session", "")
	case errors.Is(err, auth.ErrEmailTaken):
		writeError(w, 409, "email_taken", "email already registered", "")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, 504, "timeout", "request timed out", "")
	default:
		slog.Error("authentication operation failed", "request_id", middleware.GetReqID(r.Context()), "error", err)
		writeError(w, 500, "internal_error", "authentication operation failed", "")
	}
}

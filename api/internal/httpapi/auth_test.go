package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/game-key-marketplace/api/internal/auth"
	"example.com/game-key-marketplace/api/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthBrowserSecurity(t *testing.T) {
	a := &Auth{Service: &auth.Service{Config: auth.Config{Origins: map[string]bool{"https://web.example": true}}}}
	for _, tc := range []struct {
		method, origin, header string
		status                 int
	}{
		{"POST", "https://web.example", "1", 204}, {"POST", "https://evil.example", "1", 403},
		{"POST", "https://web.example", "", 403}, {"POST", "", "1", 204}, {"POST", "", "", 403},
		{"GET", "https://web.example", "", 204}, {"GET", "null", "", 403},
	} {
		t.Run(tc.method+tc.origin+tc.header, func(t *testing.T) {
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(tc.method, "/api/auth/login", nil)
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("X-CSRF-Protection", tc.header)
			a.BrowserSecurity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })).ServeHTTP(rr, r)
			if rr.Code != tc.status {
				t.Fatal(rr.Code, rr.Body.String())
			}
			if tc.origin == "https://web.example" && rr.Header().Get("Access-Control-Allow-Credentials") != "true" {
				t.Fatal("missing credential CORS")
			}
		})
	}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest("OPTIONS", "/api/auth/login", nil)
	r.Header.Set("Origin", "https://web.example")
	r.Header.Set("Access-Control-Request-Method", "POST")
	a.BrowserSecurity(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("preflight should not invoke handler") })).ServeHTTP(rr, r)
	if rr.Code != 204 {
		t.Fatal(rr.Code)
	}
}

func TestAuthIntegration(t *testing.T) {
	url := os.Getenv("TEST_AUTH_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_AUTH_DATABASE_URL to an empty *_test database")
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
	if err = pool.QueryRow(ctx, "SELECT current_database()").Scan(&db); err != nil || !strings.HasSuffix(db, "_test") {
		t.Fatal("must use test database", err)
	}
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'").Scan(&count); err != nil || count != 0 {
		t.Fatal("must use empty database", err)
	}
	if err = migrations.Run(url, "up"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		for i := 0; i < 5; i++ {
			if err := migrations.Run(url, "down"); err != nil {
				t.Error(err)
			}
		}
	}()
	service, err := auth.New(auth.Repository{DB: pool}, auth.Config{Secret: []byte(strings.Repeat("x", 32)), SameSite: http.SameSiteLaxMode, Origins: map[string]bool{"http://localhost:5173": true}})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewRouter(pool, &fakeStore{}, NewAuth(service))
	request := func(method, path, body, ip string, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = ip + ":12345"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-CSRF-Protection", "1")
		r.Header.Set("Origin", "http://localhost:5173")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, r)
		return rr
	}
	credentials := `{"email":" Person@Example.com ","password":"correct-password-123"}`
	for _, tc := range []struct {
		name, path, body string
		status           int
	}{
		{"register", "/api/auth/register", credentials, 201},
		{"duplicate", "/api/auth/register", credentials, 409},
		{"short password", "/api/auth/register", `{"email":"x@example.com","password":"short"}`, 400},
		{"invalid email", "/api/auth/register", `{"email":"bad","password":"correct-password-123"}`, 400},
		{"unknown field", "/api/auth/login", `{"email":"person@example.com","password":"correct-password-123","admin":true}`, 400},
		{"extra json", "/api/auth/login", credentials + " {}", 400},
		{"bad password", "/api/auth/login", `{"email":"person@example.com","password":"incorrect-password"}`, 401},
		{"missing user", "/api/auth/login", `{"email":"unknown@example.com","password":"incorrect-password"}`, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request("POST", tc.path, tc.body, "192.0.2.1", nil)
			if r.Code != tc.status {
				t.Fatal(r.Code, r.Body.String())
			}
			if strings.Contains(r.Body.String(), "password") && tc.status < 400 {
				t.Fatal("password leaked")
			}
		})
	}
	login := request("POST", "/api/auth/login", credentials, "192.0.2.2", nil)
	if login.Code != 200 {
		t.Fatal(login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing cookie")
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.MaxAge != 86400 {
		t.Fatal(cookie)
	}
	if strings.Contains(login.Body.String(), cookie.Value) {
		t.Fatal("JWT exposed in body")
	}
	me := request("GET", "/api/me", "", "192.0.2.2", cookie)
	if me.Code != 200 || me.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(me.Code, me.Body.String())
	}
	var result struct {
		Data auth.User `json:"data"`
	}
	if err = json.Unmarshal(me.Body.Bytes(), &result); err != nil || result.Data.Email != "person@example.com" {
		t.Fatal(result, err)
	}
	for _, path := range []string{"/api/me", "/api/checkout", "/api/checkout/confirm", "/api/orders", "/api/orders/123"} {
		rr := request("GET", path, "", "192.0.2.2", nil)
		if rr.Code != 401 {
			t.Fatal(path, rr.Code, rr.Body.String())
		}
		if path != "/api/me" {
			rr = request("GET", path, "", "192.0.2.2", cookie)
			expected := 404
			if path == "/api/orders" {
				expected = 200
			}
			if path == "/api/checkout" {
				expected = 405
			}
			if rr.Code != expected {
				t.Fatal("authenticated unimplemented endpoint", path, rr.Code)
			}
		}
	}
	logout := request("POST", "/api/auth/logout", "", "192.0.2.2", cookie)
	if logout.Code != 204 || logout.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal(logout.Code, logout.Body.String())
	}
	if rr := request("GET", "/api/me", "", "192.0.2.2", cookie); rr.Code != 401 {
		t.Fatal("revoked token accepted")
	}
	if rr := request("POST", "/api/auth/logout", "", "192.0.2.2", nil); rr.Code != 204 {
		t.Fatal("logout must be idempotent")
	}
	for i := 0; i < 11; i++ {
		r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{}`))
		r.RemoteAddr = "192.0.2.99:12345"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-CSRF-Protection", "1")
		r.Header.Set("X-Forwarded-For", time.Now().String())
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, r)
		if i < 10 && rr.Code != 400 {
			t.Fatal(rr.Code)
		}
		if i == 10 && (rr.Code != 429 || rr.Header().Get("Retry-After") == "") {
			t.Fatal("limiter bypassed", rr.Code)
		}
	}
	// Persisted expiry and deleted-user checks complement JWT expiration validation.
	login = request("POST", "/api/auth/login", credentials, "192.0.2.3", nil)
	cookie = login.Result().Cookies()[0]
	if _, err = pool.Exec(ctx, "UPDATE auth_sessions SET expires_at=now()-interval '1 second'"); err != nil {
		t.Fatal(err)
	}
	if rr := request("GET", "/api/me", "", "192.0.2.3", cookie); rr.Code != 401 {
		t.Fatal("expired stored session accepted")
	}
	login = request("POST", "/api/auth/login", credentials, "192.0.2.3", nil)
	cookie = login.Result().Cookies()[0]
	if _, err = pool.Exec(ctx, "DELETE FROM users"); err != nil {
		t.Fatal(err)
	}
	if rr := request("GET", "/api/me", "", "192.0.2.3", cookie); rr.Code != 401 {
		t.Fatal("deleted user accepted")
	}
}

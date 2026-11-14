package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/game-key-marketplace/api/internal/account"
	"example.com/game-key-marketplace/api/internal/auth"
	"example.com/game-key-marketplace/api/migrations"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type memoryMailer struct {
	code string
	fail bool
}

func (m *memoryMailer) Send(_ context.Context, _, code string) error {
	m.code = code
	if m.fail {
		return errors.New("delivery failure")
	}
	return nil
}
func TestSafeIdentityNext(t *testing.T) {
	for _, raw := range []string{"//evil.invalid", "https://evil.invalid", "/\\evil", "/login?next=/", "/register", "/\r\nLocation: evil"} {
		if safeIdentityNext(raw) != "/" {
			t.Fatal("unsafe return destination")
		}
	}
	if safeIdentityNext("/wishlist") != "/wishlist" {
		t.Fatal("valid destination lost")
	}
}
func TestAccessLogOmitsSecrets(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	defer slog.SetDefault(previous)
	router := NewRouter(fakeDB{}, &fakeStore{}, nil)
	r := httptest.NewRequest("GET", "/api/missing?code=private-auth-code&state=private-state", nil)
	router.ServeHTTP(httptest.NewRecorder(), r)
	if strings.Contains(output.String(), "private-") || strings.Contains(output.String(), "missing?") {
		t.Fatal("request parameters leaked")
	}
}
func TestAccountIdentityIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_ACCOUNT_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_ACCOUNT_DATABASE_URL to an empty *_test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var database string
	var count int
	if err = pool.QueryRow(ctx, `SELECT current_database()`).Scan(&database); err != nil || !strings.HasSuffix(database, "_test") {
		t.Fatal("disposable database required")
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("empty test database required")
	}
	if err = migrations.Run(databaseURL, "up"); err != nil {
		t.Fatal(err)
	}
	service, err := auth.New(auth.Repository{DB: pool}, auth.Config{Secret: []byte(strings.Repeat("x", 32)), Origins: map[string]bool{"http://localhost:5173": true}, SameSite: http.SameSiteLaxMode})
	if err != nil {
		t.Fatal(err)
	}
	mail := &memoryMailer{}
	codes := auth.Passwordless{DB: pool, Service: service, Mailer: mail}
	t.Run("codes create accounts expire lock replay and hide values", func(t *testing.T) {
		id, err := codes.Request(ctx, " Code@Example.invalid ")
		if err != nil {
			t.Fatal("code request failed", err)
		}
		var hash string
		pool.QueryRow(ctx, `SELECT code_hash FROM email_challenges WHERE id=$1`, id).Scan(&hash)
		if hash == mail.code || len(hash) != 64 {
			t.Fatal("code was not hashed")
		}
		user, token, err := codes.Verify(ctx, id, mail.code)
		if err != nil || user.ID == 0 || token == "" {
			t.Fatal("code verification failed")
		}
		if _, err = service.Authenticate(ctx, token); err != nil {
			t.Fatal("code session invalid")
		}
		if _, _, err = codes.Verify(ctx, id, mail.code); !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatal("code replay accepted")
		}
		id, _ = codes.Request(ctx, user.Email)
		wrong := "000000"
		if mail.code == wrong {
			wrong = "999999"
		}
		for i := 0; i < 5; i++ {
			if _, _, err = codes.Verify(ctx, id, wrong); !errors.Is(err, auth.ErrUnauthorized) {
				t.Fatal("incorrect code accepted")
			}
		}
		if _, _, err = codes.Verify(ctx, id, mail.code); !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatal("attempt limit bypass")
		}
		id, _ = codes.Request(ctx, user.Email)
		pool.Exec(ctx, `UPDATE email_challenges SET expires_at=now()-interval '1 second' WHERE id=$1`, id)
		if _, _, err = codes.Verify(ctx, id, mail.code); !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatal("expired code accepted")
		}
		mail.fail = true
		if _, err = codes.Request(ctx, "failed@example.invalid"); err == nil {
			t.Fatal("mail failure hidden")
		}
		mail.fail = false
		pool.QueryRow(ctx, `SELECT count(*) FROM email_challenges WHERE email='failed@example.invalid'`).Scan(&count)
		if count != 0 {
			t.Fatal("failed delivery challenge retained")
		}
		if _, _, err = service.Login(ctx, auth.Credentials{Email: user.Email, Password: "a-long-test-password"}); !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatal("passwordless account allowed password login")
		}
	})
	var first, second auth.User
	first, _, err = codes.ExternalUser(ctx, "https://issuer.invalid", "one", "first@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	second, _, err = codes.ExternalUser(ctx, "https://issuer.invalid", "two", "second@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("external subject remains authoritative", func(t *testing.T) {
		u, _, err := codes.ExternalUser(ctx, "https://issuer.invalid", "one", second.Email)
		if err != nil || u.ID != first.ID {
			t.Fatal("external identity remapped")
		}
	})
	if _, err = pool.Exec(ctx, `INSERT INTO sellers(name)VALUES('Test seller');INSERT INTO products(name,slug,platform,region,genre,image_url)SELECT 'Game '||n,'game-'||n,'Steam','Global','RPG','/media/games/original-rpg.png' FROM generate_series(1,201)n;INSERT INTO offers(product_id,seller_id,price,currency,stock)VALUES(1,1,10,'EUR',1),(1,1,5,'USD',1)`); err != nil {
		t.Fatal(err)
	}
	repo := account.Repository{DB: pool}
	profile := account.Profile{DisplayName: "Player One", CountryCode: "LT", Avatar: "orbit", PreferredPlatform: "Steam", MarketingOptIn: true}
	t.Run("profile scoped validation completion", func(t *testing.T) {
		if err := repo.Update(ctx, first.ID, profile); err != nil {
			t.Fatal(err)
		}
		own, err := repo.Get(ctx, first.ID, "USD")
		if err != nil || own.Profile != profile || own.Completion.Percent != 100 || own.Currency != "USD" {
			t.Fatal("profile round trip failed", err)
		}
		other, err := repo.Get(ctx, second.ID, "EUR")
		if err != nil || other.Profile.DisplayName != "" || other.Completion.Percent != 33 {
			t.Fatal("profile ownership failed")
		}
		bad := profile
		bad.PreferredPlatform = "not in catalog"
		if err = repo.Update(ctx, first.ID, bad); !errors.Is(err, account.ErrInvalid) {
			t.Fatal("unknown platform accepted")
		}
	})
	t.Run("illustrative rewards use only owners paid orders", func(t *testing.T) {
		_, err := pool.Exec(ctx, `INSERT INTO orders(user_id,status,currency,total_amount) VALUES($1,'paid','EUR',12.34),($1,'paid','USD',11),($1,'paid','GBP',8.50),($1,'pending','EUR',999)`, first.ID)
		if err != nil {
			t.Fatal(err)
		}
		own, err := repo.Get(ctx, first.ID, "GBP")
		if err != nil || own.OrderCount != 3 || own.RewardPoints != 32 || own.CashbackEarned != "0.27" || own.WalletBalance != "0.00" || !own.Demo {
			t.Fatal("incorrect illustrative totals", err)
		}
		other, err := repo.Get(ctx, second.ID, "GBP")
		if err != nil || other.OrderCount != 0 || other.RewardPoints != 0 || other.CashbackEarned != "0.00" {
			t.Fatal("another owners totals exposed", err)
		}
	})
	t.Run("wishlist ownership converted price idempotency and cap", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			if err = repo.Favorite(ctx, first.ID, 1, true); err != nil {
				t.Fatal(err)
			}
		}
		own, err := repo.Wishlist(ctx, first.ID, "GBP")
		if err != nil || len(own) != 1 || own[0].Price == nil || *own[0].Price != "3.86" {
			t.Fatal("wishlist conversion or repeat failed", err)
		}
		repo.Favorite(ctx, second.ID, 1, false)
		other, err := repo.Wishlist(ctx, second.ID, "EUR")
		if err != nil || len(other) != 0 {
			t.Fatal("wishlist ownership failed")
		}
		own, _ = repo.Wishlist(ctx, first.ID, "EUR")
		if len(own) != 1 {
			t.Fatal("other user removed favorite")
		}
		if err = repo.Favorite(ctx, first.ID, 999999, true); !errors.Is(err, account.ErrNotFound) {
			t.Fatal("missing product accepted")
		}
		pool.Exec(ctx, `INSERT INTO wishlists(user_id,product_id) SELECT $1,id FROM products WHERE id<=199 ON CONFLICT DO NOTHING`, first.ID)
		var wg sync.WaitGroup
		out := make(chan error, 2)
		for _, p := range []int64{200, 201} {
			wg.Add(1)
			go func(id int64) { defer wg.Done(); out <- repo.Favorite(ctx, first.ID, id, true) }(p)
		}
		wg.Wait()
		close(out)
		success, full := 0, 0
		for e := range out {
			if e == nil {
				success++
			} else if errors.Is(e, account.ErrFull) {
				full++
			} else {
				t.Fatal(e)
			}
		}
		if success != 1 || full != 1 {
			t.Fatal("concurrent wishlist cap failed")
		}
	})
	t.Run("HTTP account auth CSRF tamper methods codes", func(t *testing.T) {
		handler := NewRouter(pool, &fakeStore{}, NewAuth(service))
		_, session, err := service.StartSession(ctx, first)
		if err != nil {
			t.Fatal(err)
		}
		call := func(method, path, body, origin string, signed bool) *httptest.ResponseRecorder {
			r := httptest.NewRequest(method, path, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-CSRF-Protection", "1")
			r.Header.Set("Origin", origin)
			if signed {
				r.AddCookie(service.Config.Cookie(session, time.Now()))
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, r)
			return rr
		}
		for _, route := range []struct{ method, path string }{{"PATCH", "/api/account"}, {"PUT", "/api/wishlist/1"}, {"DELETE", "/api/wishlist/1"}, {"POST", "/api/auth/email/request"}, {"POST", "/api/auth/email/verify"}} {
			if call(route.method, route.path, `{}`, "https://foreign.invalid", true).Code != 403 {
				t.Fatal("foreign origin accepted")
			}
		}
		if call("GET", "/api/account", "", "", false).Code != 401 {
			t.Fatal("account auth bypass")
		}
		if call("PATCH", "/api/account", `{"user_id":2}`, "", true).Code != 400 {
			t.Fatal("profile owner tampering accepted")
		}
		if call("GET", "/api/auth/methods", "", "", false).Code != 200 {
			t.Fatal("methods unavailable")
		}
		identity := &Identity{Auth: NewAuth(service), Codes: codes}
		rr := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/auth/email/request", strings.NewReader(`{"email":"http@example.invalid"}`))
		r.Header.Set("Content-Type", "application/json")
		identity.Request(rr, r)
		if rr.Code != 202 || strings.Contains(rr.Body.String(), mail.code) || rr.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("code request response unsafe")
		}
	})
	t.Run("OIDC PKCE nonce verified email state and replay", func(t *testing.T) { testOIDCFlow(t, ctx, service, codes) })
}
func testOIDCFlow(t *testing.T, ctx context.Context, service *auth.Service, codes auth.Passwordless) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	nonce, challenge := "", ""
	verified := true
	var issuer *httptest.Server
	issuer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]any{"issuer": issuer.URL, "authorization_endpoint": issuer.URL + "/authorize", "token_endpoint": issuer.URL + "/token", "jwks_uri": issuer.URL + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "RSA", "kid": "test", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB"}}})
		case "/token":
			r.ParseForm()
			sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
				http.Error(w, "invalid PKCE", 400)
				return
			}
			claims := jwt.MapClaims{"iss": issuer.URL, "sub": "oidc-user", "aud": "test-client", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(), "nonce": nonce, "email": "oidc@example.invalid", "email_verified": verified}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "test"
			signed, _ := token.SignedString(key)
			json.NewEncoder(w).Encode(map[string]any{"access_token": "fixture-only", "token_type": "Bearer", "id_token": signed})
		default:
			w.WriteHeader(404)
		}
	}))
	defer issuer.Close()
	for name, value := range map[string]string{"OIDC_ISSUER": issuer.URL, "OIDC_CLIENT_ID": "test-client", "OIDC_CLIENT_SECRET": "test-only", "OIDC_REDIRECT_URL": "http://localhost:5173/api/auth/oidc/callback", "AUTH_WEB_URL": "http://localhost:5173"} {
		t.Setenv(name, value)
	}
	h := NewIdentity(NewAuth(service), codes)
	if h.Provider == nil {
		t.Fatal("local issuer not configured")
	}
	start := func() (string, *http.Cookie) {
		rr := httptest.NewRecorder()
		h.Start(rr, httptest.NewRequest("GET", "/api/auth/oidc/start?next=/wishlist", nil))
		if rr.Code != 302 {
			t.Fatal("OIDC start failed")
		}
		location, _ := url.Parse(rr.Header().Get("Location"))
		nonce = location.Query().Get("nonce")
		challenge = location.Query().Get("code_challenge")
		if nonce == "" || challenge == "" || location.Query().Get("code_challenge_method") != "S256" {
			t.Fatal("OIDC binding absent")
		}
		return location.Query().Get("state"), rr.Result().Cookies()[0]
	}
	callback := func(state string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/api/auth/oidc/callback?state="+url.QueryEscape(state)+"&code=fixture-code", nil)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		rr := httptest.NewRecorder()
		h.Callback(rr, r)
		return rr
	}
	state, cookie := start()
	if rr := callback(state, nil); !strings.Contains(rr.Header().Get("Location"), "identity_error") {
		t.Fatal("missing binding accepted")
	}
	rr := callback(state, cookie)
	if rr.Header().Get("Location") != "http://localhost:5173/wishlist" {
		t.Fatal("valid OIDC callback failed")
	}
	var session *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == auth.CookieName {
			session = c
		}
	}
	if session == nil {
		t.Fatal("OIDC session absent")
	}
	if _, err = service.Authenticate(ctx, session.Value); err != nil {
		t.Fatal("OIDC session invalid")
	}
	if !strings.Contains(callback(state, cookie).Header().Get("Location"), "identity_error") {
		t.Fatal("OIDC callback replay accepted")
	}
	state, cookie = start()
	nonce = "wrong"
	if !strings.Contains(callback(state, cookie).Header().Get("Location"), "identity_error") {
		t.Fatal("wrong nonce accepted")
	}
	state, cookie = start()
	verified = false
	if !strings.Contains(callback(state, cookie).Header().Get("Location"), "identity_error") {
		t.Fatal("unverified provider email accepted")
	}
}

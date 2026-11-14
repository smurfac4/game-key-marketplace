package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func TestConfig(t *testing.T) {
	for _, tc := range []struct {
		name, secret, secure, same, origins string
		valid                               bool
	}{
		{"valid", strings.Repeat("x", 32), "false", "lax", "http://localhost:5173", true},
		{"cross-site", strings.Repeat("x", 32), "true", "none", "https://web.example.com", true},
		{"short secret", "short", "true", "lax", "https://web.example.com", false},
		{"insecure none", strings.Repeat("x", 32), "false", "none", "https://web.example.com", false},
		{"bad bool", strings.Repeat("x", 32), "nope", "lax", "https://web.example.com", false},
		{"bad same", strings.Repeat("x", 32), "true", "weird", "https://web.example.com", false},
		{"wildcard", strings.Repeat("x", 32), "true", "lax", "*", false},
		{"path", strings.Repeat("x", 32), "true", "lax", "https://web.example.com/path", false},
		{"no origins", strings.Repeat("x", 32), "true", "lax", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{"JWT_SECRET": tc.secret, "COOKIE_SECURE": tc.secure, "COOKIE_SAMESITE": tc.same, "AUTH_ALLOWED_ORIGINS": tc.origins}
			c, err := LoadConfig(func(k string) string { return env[k] })
			if (err == nil) != tc.valid {
				t.Fatal(err)
			}
			if err == nil {
				cookie := c.Cookie("token", time.Now())
				if !cookie.HttpOnly || cookie.Path != "/" || cookie.Domain != "" {
					t.Fatal(cookie)
				}
				clear := c.ClearCookie()
				if clear.MaxAge != -1 || clear.SameSite != cookie.SameSite || clear.Secure != cookie.Secure {
					t.Fatal(clear)
				}
			}
		})
	}
}
func TestValidation(t *testing.T) {
	for _, tc := range []struct {
		email, password string
		valid           bool
	}{
		{" User@Example.com ", "long-password-123", true}, {"bad", "long-password-123", false},
		{"Name <a@example.com>", "long-password-123", false}, {"a@example.com", "short", false},
		{"a@example.com", strings.Repeat("x", 72), true}, {"a@example.com", strings.Repeat("x", 73), false},
		{"a@example.com", strings.Repeat("界", 25), false}, {"a@example.com", strings.Repeat("界", 12), true},
	} {
		_, err := Validate(Credentials{tc.email, tc.password})
		if (err == nil) != tc.valid {
			t.Errorf("email=%s bytes=%d err=%v", tc.email, len(tc.password), err)
		}
	}
	c, _ := Validate(Credentials{" User@Example.com ", "long-password-123"})
	if c.Email != "user@example.com" {
		t.Fatal(c.Email)
	}
}
func TestLimiter(t *testing.T) {
	now := time.Now()
	l := NewLimiter()
	l.now = func() time.Time { return now }
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _ := l.Allow("same-ip")
			if ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != 10 {
		t.Fatalf("allowed %d", allowed)
	}
	if ok, retry := l.Allow("same-ip"); ok || retry <= 0 {
		t.Fatal("expected rate limit")
	}
	now = now.Add(15 * time.Minute)
	if ok, _ := l.Allow("same-ip"); !ok {
		t.Fatal("window did not reset")
	}
	l.entries = map[string]bucket{}
	for i := 0; i < 10000; i++ {
		l.entries[string(rune(i))] = bucket{Until: now.Add(time.Hour)}
	}
	if ok, _ := l.Allow("new-ip"); ok {
		t.Fatal("capacity limit bypassed")
	}
}

type memoryStore struct {
	a        Account
	sessions map[string]int64
}

func (m *memoryStore) CreateUser(_ context.Context, e, h string) (User, error) {
	if m.a.ID != 0 {
		return User{}, ErrEmailTaken
	}
	m.a = Account{User: User{ID: 1, Email: e}, PasswordHash: h}
	return m.a.User, nil
}
func (m *memoryStore) FindByEmail(_ context.Context, e string) (Account, error) {
	if e != m.a.Email {
		return Account{}, ErrUnauthorized
	}
	return m.a, nil
}
func (m *memoryStore) CreateSession(_ context.Context, id string, u int64, _ time.Time) error {
	m.sessions[id] = u
	return nil
}
func (m *memoryStore) SessionUser(_ context.Context, id string, u int64) (User, error) {
	if m.sessions[id] != u {
		return User{}, ErrUnauthorized
	}
	return m.a.User, nil
}
func (m *memoryStore) DeleteSession(_ context.Context, id string) error {
	delete(m.sessions, id)
	return nil
}
func TestSessionLifecycleAndJWT(t *testing.T) {
	m := &memoryStore{sessions: map[string]int64{}}
	s, err := New(m, Config{Secret: []byte(strings.Repeat("x", 32)), SameSite: http.SameSiteLaxMode})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := Credentials{"me@example.com", "correct-password-123"}
	if _, err = s.Register(ctx, c); err != nil {
		t.Fatal(err)
	}
	if m.a.PasswordHash == c.Password || bcrypt.CompareHashAndPassword([]byte(m.a.PasswordHash), []byte(c.Password)) != nil {
		t.Fatal("password not hashed")
	}
	if cost, _ := bcrypt.Cost([]byte(m.a.PasswordHash)); cost != bcrypt.DefaultCost {
		t.Fatal("wrong bcrypt cost")
	}
	for _, bad := range []Credentials{{"other@example.com", c.Password}, {c.Email, "incorrect-password"}} {
		if _, _, err = s.Login(ctx, bad); !errors.Is(err, ErrUnauthorized) {
			t.Fatal(err)
		}
	}
	_, token, err := s.Login(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Authenticate(ctx, token); err != nil {
		t.Fatal(err)
	}
	if err = s.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Authenticate(ctx, token); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("logout replay succeeded")
	}
	now := time.Now()
	base := jwt.RegisteredClaims{Issuer: "marketplace-api", Audience: jwt.ClaimStrings{"marketplace-web"}, Subject: "1", ID: "id", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}
	for _, name := range []string{"expired", "missing expiry", "issuer", "audience", "algorithm", "signature", "subject", "future issued"} {
		t.Run(name, func(t *testing.T) {
			claims := base
			method := jwt.SigningMethodHS256
			key := s.Config.Secret
			switch name {
			case "expired":
				claims.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Hour))
			case "missing expiry":
				claims.ExpiresAt = nil
			case "issuer":
				claims.Issuer = "wrong"
			case "audience":
				claims.Audience = jwt.ClaimStrings{"wrong"}
			case "algorithm":
				method = jwt.SigningMethodHS384
			case "signature":
				key = []byte(strings.Repeat("z", 32))
			case "subject":
				claims.Subject = "-1"
			case "future issued":
				claims.IssuedAt = jwt.NewNumericDate(now.Add(time.Hour))
			}
			raw, err := jwt.NewWithClaims(method, claims).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.claims(raw); !errors.Is(err, ErrUnauthorized) {
				t.Fatal("accepted invalid JWT", err)
			}
		})
	}
}

func TestSMTPConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name           string
		env            map[string]string
		valid, enabled bool
	}{
		{"disabled", map[string]string{}, true, false},
		{"missing auth", map[string]string{"SMTP_ADDR": "mail.example.invalid:587", "SMTP_FROM": "sender@example.invalid"}, false, false},
		{"bad sender", map[string]string{"SMTP_ADDR": "mail.example.invalid:587", "SMTP_FROM": "Name <sender@example.invalid>", "SMTP_USER": "test", "SMTP_PASSWORD": "fixture-only"}, false, false},
		{"configured", map[string]string{"SMTP_ADDR": "mail.example.invalid:587", "SMTP_FROM": "sender@example.invalid", "SMTP_USER": "test", "SMTP_PASSWORD": "fixture-only"}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := NewSMTP(func(k string) string { return tc.env[k] })
			if (err == nil) != tc.valid || (m != nil) != tc.enabled {
				t.Fatal("unexpected SMTP configuration result")
			}
		})
	}
}

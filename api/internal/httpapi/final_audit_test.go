package httpapi

import (
	"errors"
	"example.com/game-key-marketplace/api/internal/auth"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFinalForeignOrigins(t *testing.T) {
	for _, sameSite := range []http.SameSite{http.SameSiteLaxMode, http.SameSiteNoneMode} {
		a := &Auth{Service: &auth.Service{Config: auth.Config{Origins: map[string]bool{"https://web.example": true}, SameSite: sameSite, Secure: true}}}
		for _, route := range []string{"POST /api/auth/register", "POST /api/auth/login", "POST /api/cart/items", "PATCH /api/cart/items/1", "DELETE /api/cart/items/1", "POST /api/checkout"} {
			parts := strings.Split(route, " ")
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(parts[0], parts[1], nil)
			r.Header.Set("Origin", "https://evil.example")
			r.Header.Set("X-CSRF-Protection", "1")
			a.BrowserSecurity(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("foreign origin reached handler") })).ServeHTTP(rr, r)
			if rr.Code != 403 {
				t.Fatal(route, rr.Code)
			}
		}
	}
}
func TestFinalErrorRedaction(t *testing.T) {
	rr := httptest.NewRecorder()
	commerceError(rr, errors.New("SQL password internal-sensitive-marker"))
	if rr.Code != 500 || strings.Contains(rr.Body.String(), "internal-sensitive-marker") {
		t.Fatal("error leaked")
	}
}
func TestFinalRouteLimit(t *testing.T) {
	h := limitRequests(auth.NewWindowLimiter(2, time.Minute))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for _, want := range []int{204, 204, 429} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
		if rr.Code != want {
			t.Fatal(rr.Code)
		}
		if want == 429 && rr.Header().Get("Retry-After") == "" {
			t.Fatal("missing retry")
		}
	}
}

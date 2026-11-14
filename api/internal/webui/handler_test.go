package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticFrontendAndAPIBoundary(t *testing.T) {
	directory := t.TempDir()
	for name, body := range map[string]string{"index.html": "<html>KEYSPACE</html>", "assets/app-hash.js": "console.log('app')", "media/cover.svg": "<svg></svg>", ".env": "private-test-fixture"} {
		target := filepath.Join(directory, name)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized"}}`))
	})
	handler, err := New(api, directory)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, method, target string
		status               int
		body, cache          string
	}{
		{"homepage", "GET", "/", 200, "KEYSPACE", "no-cache"},
		{"deep link", "GET", "/products/demo-game-001", 200, "KEYSPACE", "no-cache"},
		{"keys library deep link", "GET", "/account/keys", 200, "KEYSPACE", "no-cache"},
		{"asset", "GET", "/assets/app-hash.js", 200, "console.log", "public, max-age=31536000, immutable"},
		{"cover", "GET", "/media/cover.svg", 200, "<svg>", "public, max-age=3600"},
		{"head", "HEAD", "/store", 200, "", "no-cache"},
		{"missing asset", "GET", "/assets/missing.js", 404, "404", ""},
		{"extensionless missing asset", "GET", "/assets/missing", 404, "404", ""},
		{"missing cover", "GET", "/media/missing.svg", 404, "404", ""},
		{"directory listing", "GET", "/media/", 404, "404", ""},
		{"hidden file", "GET", "/.env", 404, "404", ""},
		{"hidden path traversal", "GET", "/media/../.env", 404, "404", ""},
		{"mutation", "POST", "/store", 405, "method not allowed", ""},
		{"private API", "GET", "/api/orders", 401, "unauthorized", "no-store"},
		{"API mutation", "POST", "/api/checkout", 401, "unauthorized", "no-store"},
		{"unknown API", "GET", "/api/missing", 401, "unauthorized", "no-store"},
		{"API root", "GET", "/api", 401, "unauthorized", "no-store"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tc.method, tc.target, nil))
			if response.Code != tc.status || !strings.Contains(response.Body.String(), tc.body) {
				t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
			}
			if response.Header().Get("Cache-Control") != tc.cache {
				t.Fatalf("cache=%q", response.Header().Get("Cache-Control"))
			}
			if response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("missing MIME protection")
			}
			if tc.method == "HEAD" && response.Body.Len() != 0 {
				t.Fatal("HEAD returned a body")
			}
			if strings.HasPrefix(tc.target, "/api") && !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
				t.Fatal("API fell through to HTML")
			}
		})
	}
}

func TestMissingFrontendBuild(t *testing.T) {
	if _, err := New(http.NotFoundHandler(), t.TempDir()); err == nil {
		t.Fatal("missing build must fail startup")
	}
}

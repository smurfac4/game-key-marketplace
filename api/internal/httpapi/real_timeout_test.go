package httpapi

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Real TCP HTTP request waiting for the sole occupied pgx connection: no mocked SQL/error.
func TestRealTwelveSecondPoolTimeout(t *testing.T) {
	url := os.Getenv("TEST_TIMEOUT_DATABASE_URL")
	if url == "" {
		t.Skip("separate disposable timeout database required")
	}
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal("pool configuration")
	}
	if !strings.HasSuffix(config.ConnConfig.Database, "_test") {
		t.Fatal("disposable database required")
	}
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal("pool creation")
	}
	defer pool.Close()
	held, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal("hold pool connection")
	}
	defer held.Release()
	handler := Commerce{}
	handler.Repo.DB = pool
	server := httptest.NewServer(requestBudget(12 * time.Second)(http.HandlerFunc(handler.Cart)))
	defer server.Close()
	client := &http.Client{Timeout: 16 * time.Second}
	start := time.Now()
	response, err := client.Get(server.URL + "/api/cart")
	if err != nil {
		t.Fatal("client hung or disconnected")
	}
	defer response.Body.Close()
	elapsed := time.Since(start)
	var body struct {
		Error struct{ Code, Message string }
	}
	if err = json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal("not JSON")
	}
	if response.StatusCode != 504 || body.Error.Code != "timeout" || body.Error.Message != "request timed out" || elapsed < 11500*time.Millisecond || elapsed > 15*time.Second {
		t.Fatalf("unexpected timeout response: status=%d elapsed=%s", response.StatusCode, elapsed)
	}
	t.Logf("real occupied-pool HTTP timeout: status=504 elapsed=%s", elapsed)
}

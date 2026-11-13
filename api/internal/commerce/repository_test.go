package commerce

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/game-key-marketplace/api/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMockPayment(t *testing.T) {
	for _, tc := range []struct {
		card string
		err  error
	}{{"4242 4242 4242 4242", nil}, {"4000000000000002", ErrPaymentDeclined}, {"123", ErrInvalidCard}} {
		if err := (MockPayment{}).Charge(context.Background(), Payment{CardNumber: tc.card}); !errors.Is(err, tc.err) {
			t.Fatal(err)
		}
	}
}

type gatePayment struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (p *gatePayment) Charge(ctx context.Context, _ Payment) error {
	p.once.Do(func() { close(p.entered) })
	select {
	case <-p.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func TestCommerceIntegration(t *testing.T) {
	url := os.Getenv("TEST_COMMERCE_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_COMMERCE_DATABASE_URL to an empty *_test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var name string
	var count int
	if err = pool.QueryRow(ctx, "SELECT current_database()").Scan(&name); err != nil || !strings.HasSuffix(name, "_test") {
		t.Fatal("test database required", err)
	}
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'").Scan(&count); err != nil || count != 0 {
		t.Fatal("empty database required", err)
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
	_, err = pool.Exec(ctx, `INSERT INTO users(email,password_hash) VALUES ('a@example.invalid','disabled'),('b@example.invalid','disabled');
 INSERT INTO sellers(name) VALUES ('Seller');
 INSERT INTO products(name,slug,platform,region,genre,image_url) VALUES ('Test','test','Steam','Global','RPG','https://example.invalid/1'),('Other','other','Steam','Global','RPG','https://example.invalid/2');
 INSERT INTO offers(product_id,seller_id,price,discount_price,currency,stock) VALUES (1,1,20,10,'EUR',1),(2,1,30,NULL,'EUR',3),(1,1,2,NULL,'USD',2);
 INSERT INTO game_keys(offer_id,key_value) VALUES (1,'FAKE-last-key'),(2,'FAKE-2'),(2,'FAKE-3'),(2,'FAKE-4'),(3,'FAKE-USD-1'),(3,'FAKE-USD-2');`)
	if err != nil {
		t.Fatal(err)
	}
	repo := Repository{DB: pool, Payments: MockPayment{}}
	for _, tc := range []struct {
		offer    int64
		quantity int
		want     error
	}{{2, 0, ErrInvalid}, {2, 11, ErrInvalid}, {999, 1, ErrNotFound}, {2, 4, ErrOutOfStock}, {2, 2, nil}, {3, 1, ErrCurrency}} {
		if err := repo.SetItem(ctx, 1, tc.offer, tc.quantity, true); !errors.Is(err, tc.want) {
			t.Fatalf("cart %v: %v", tc, err)
		}
	}
	if err := repo.SetItem(ctx, 1, 2, 1, false); err != nil {
		t.Fatal(err)
	}
	if other, err := repo.GetCart(ctx, 2); err != nil || len(other.Items) != 0 {
		t.Fatal("cart isolation", other, err)
	}
	// Provider failure must roll back every mutation, including allocation and stock.
	for _, card := range []string{"4000000000000002", "123"} {
		if _, err := repo.Checkout(ctx, 1, "failure-token-123456", card); err == nil {
			t.Fatal("expected payment failure")
		}
	}
	var unchanged bool
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM orders)=0 AND (SELECT count(*) FROM order_items)=0 AND (SELECT count(*) FROM game_keys WHERE status='sold')=0 AND (SELECT stock FROM offers WHERE id=2)=3 AND (SELECT count(*) FROM cart_items)=1`).Scan(&unchanged); err != nil || !unchanged {
		t.Fatal("rollback incomplete", err)
	}
	// A locked key is skipped, not allocated twice or waited on.
	if err := repo.RemoveItem(ctx, 1, 2); err != nil {
		t.Fatal(err)
	}
	for _, u := range []int64{1, 2} {
		if err := repo.SetItem(ctx, u, 1, 1, true); err != nil {
			t.Fatal(err)
		}
	}
	lock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lock.Exec(ctx, "SELECT id FROM game_keys WHERE offer_id=1 FOR UPDATE"); err != nil {
		t.Fatal(err)
	}
	quick, stop := context.WithTimeout(ctx, time.Second)
	_, err = repo.Checkout(quick, 1, "locked-token-123456", "4242424242424242")
	stop()
	_ = lock.Rollback(ctx)
	if !errors.Is(err, ErrOutOfStock) {
		t.Fatal("SKIP LOCKED behavior", err)
	}
	gate := &gatePayment{entered: make(chan struct{}), release: make(chan struct{})}
	concurrent := Repository{DB: pool, Payments: gate}
	type result struct {
		user, id int64
		err      error
	}
	results := make(chan result, 2)
	go func() {
		id, err := concurrent.Checkout(ctx, 1, "buyer-one-token-123", "4242424242424242")
		results <- result{1, id, err}
	}()
	select {
	case <-gate.entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	go func() {
		id, err := concurrent.Checkout(ctx, 2, "buyer-two-token-123", "4242424242424242")
		results <- result{2, id, err}
	}()
	select {
	case early := <-results:
		t.Fatalf("checkout escaped held payment transaction: %+v", early)
	case <-time.After(50 * time.Millisecond):
	}
	close(gate.release)
	a, b := <-results, <-results
	winner, loser := a, b
	if winner.err != nil {
		winner, loser = loser, winner
	}
	if winner.err != nil || !errors.Is(loser.err, ErrOutOfStock) {
		t.Fatalf("concurrent results: %+v %+v", a, b)
	}
	order, err := repo.GetOrder(ctx, winner.user, winner.id)
	if err != nil || order.Total != "10.00" || len(order.Items) != 1 || len(order.Items[0].Keys) != 1 || order.Items[0].Keys[0] != "FAKE-last-key" {
		t.Fatal("allocation/price", order, err)
	}
	if _, err = repo.GetOrder(ctx, loser.user, winner.id); !errors.Is(err, ErrNotFound) {
		t.Fatal("IDOR", err)
	}
	list, err := repo.Orders(ctx, loser.user, 1)
	if err != nil || list.Total != 0 {
		t.Fatal("order list IDOR", list, err)
	}
	var stock, sold int
	if err = pool.QueryRow(ctx, "SELECT stock,(SELECT count(*) FROM game_keys WHERE offer_id=1 AND status='sold') FROM offers WHERE id=1").Scan(&stock, &sold); err != nil || stock != 0 || sold != 1 {
		t.Fatal("duplicate key sale", stock, sold, err)
	}
	token := "buyer-one-token-123"
	if winner.user == 2 {
		token = "buyer-two-token-123"
	}
	if _, err := repo.Checkout(ctx, winner.user, token, "4000000000000002"); !errors.Is(err, ErrIdempotency) {
		t.Fatal("different request must conflict")
	}
	// Simulate losing the HTTP response after the order has committed. POST
	// transport retries must resolve the durable token, not allocate another key.
	var dropReply atomic.Bool
	dropReply.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := repo.Checkout(r.Context(), winner.user, token, "4242424242424242")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		if dropReply.Swap(false) {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
			return
		}
		fmt.Fprint(w, id)
	}))
	defer server.Close()
	if response, err := http.Post(server.URL, "application/json", strings.NewReader("{}")); err == nil {
		response.Body.Close()
		t.Fatal("expected lost response")
	}
	response, err := http.Post(server.URL, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal("retry failed")
	}
	raw, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || string(raw) != strconv.FormatInt(winner.id, 10) {
		t.Fatal("lost-response retry changed order")
	}
	replay, err := repo.Checkout(ctx, winner.user, token, "4242424242424242")
	if err != nil || replay != winner.id {
		t.Fatal("idempotency", replay, err)
	}
	if cart, err := repo.GetCart(ctx, winner.user); err != nil || len(cart.Items) != 0 {
		t.Fatal("cart not cleared", cart, err)
	}
	if cart, err := repo.GetCart(ctx, loser.user); err != nil || len(cart.Items) != 1 {
		t.Fatal("losing cart modified", cart, err)
	}
	if _, err = repo.Checkout(ctx, winner.user, fmt.Sprintf("new-empty-%020d", winner.user), "4242424242424242"); !errors.Is(err, ErrEmptyCart) {
		t.Fatal(err)
	}
}

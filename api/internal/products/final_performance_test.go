package products

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

type planDB struct {
	*pgxpool.Pool
	t *testing.T
}
type planTx struct {
	pgx.Tx
	t *testing.T
}

func plan(ctx context.Context, db interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, t *testing.T, sql string, args ...any) {
	rows, err := db.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		fmt.Println(line)
	}
	if rows.Err() != nil {
		t.Fatal(rows.Err())
	}
}
func (d planDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	plan(ctx, d.Pool, d.t, sql, args...)
	return d.Pool.QueryRow(ctx, sql, args...)
}
func (d planDB) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := d.Pool.Begin(ctx)
	return planTx{tx, d.t}, err
}
func (d planTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	plan(ctx, d.Tx, d.t, sql, args...)
	return d.Tx.QueryRow(ctx, sql, args...)
}
func TestFinalPerformance(t *testing.T) {
	url := os.Getenv("TEST_PERF_DATABASE_URL")
	if url == "" {
		t.Skip("disposable scaled database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var db string
	pool.QueryRow(ctx, "select current_database()").Scan(&db)
	if db != "eneba_final_perf_test" {
		t.Fatal("wrong database")
	}
	repo := Repository{DB: planDB{pool, t}}
	for _, raw := range []string{"sort=price_asc", "sort=popularity", "sort=price_asc&page=250", "q=cyberpnk"} {
		t.Run(raw, func(t *testing.T) {
			q, err := ParseQuery(raw)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = repo.List(ctx, q); err != nil {
				t.Fatal(err)
			}
		})
	}
	q, _ := ParseSuggest("q=cyberpnk")
	if _, err = repo.Suggest(ctx, q); err != nil {
		t.Fatal(err)
	}
}

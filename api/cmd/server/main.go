package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/game-key-marketplace/api/internal/auth"
	"example.com/game-key-marketplace/api/internal/httpapi"
	"example.com/game-key-marketplace/api/internal/products"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	config, err := auth.LoadConfig(os.Getenv)
	if err != nil {
		return err
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	// Do not log connection errors: driver errors can contain credentials.
	if err != nil {
		return errors.New("invalid database configuration")
	}
	defer pool.Close()
	startup, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = pool.Ping(startup)
	cancel()
	if err != nil {
		return errors.New("database unavailable at startup")
	}
	authentication, err := auth.New(auth.Repository{DB: pool}, config)
	if err != nil {
		return err
	}
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{
		Addr: addr, Handler: httpapi.NewRouter(pool, products.Repository{DB: pool}, httpapi.NewAuth(authentication)),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}
	failure := make(chan error, 1)
	go func() { failure <- server.ListenAndServe() }()
	slog.Info("API listening", "address", addr)
	select {
	case err := <-failure:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
	case <-ctx.Done():
		// The signal context is cancelled; shutdown needs its own deadline.
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown HTTP: %w", err)
		}
	}
	return nil
}

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/game-key-marketplace/api/internal/seed"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	ifEmpty := flag.Bool("if-empty", false, "initialize an empty demo catalog; preserve an existing catalog")
	homepage := flag.Bool("homepage", false, "add homepage fixtures to an existing demo catalog")
	showcase := flag.Bool("showcase", false, "refresh owned demo artwork and catalog descriptions without resetting purchases")
	flag.Parse()
	if (*homepage && *showcase) || (*ifEmpty && (*homepage || *showcase)) {
		return fmt.Errorf("choose one of homepage, showcase or if-empty")
	}
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return fmt.Errorf("invalid database configuration")
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		return fmt.Errorf("cannot connect to database")
	}
	var counts seed.Counts
	if *showcase {
		counts, err = seed.RunShowcase(ctx, pool)
	} else if *homepage {
		counts, err = seed.RunHomepage(ctx, pool)
	} else if *ifEmpty {
		counts, err = seed.RunIfEmpty(ctx, pool)
	} else {
		counts, err = seed.Run(ctx, pool)
	}
	if err != nil {
		return err
	}
	if *ifEmpty && counts.Products == 0 {
		fmt.Println("Catalog already initialized; seed skipped")
		return nil
	}
	fmt.Printf("Seeded %d products, %d sellers, %d offers, %d fake keys; %d banners, %d paid orders\n", counts.Products, counts.Sellers, counts.Offers, counts.Keys, counts.Banners, counts.Orders)
	return nil
}

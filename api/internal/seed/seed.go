package seed

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Offer struct {
	Seller                          int
	PriceCents, DiscountCents, Keys int
}
type Product struct {
	Name, Slug, Description, Platform, Region, Genre, ImageURL string
	ReleaseDate                                                time.Time
	Offers                                                     []Offer
}
type Counts struct{ Products, Sellers, Offers, Keys, Banners, Orders int }

func money(cents int) string { return fmt.Sprintf("%d.%02d", cents/100, cents%100) }

func Generate() []Product {
	rng := rand.New(rand.NewSource(42))
	adjectives := []string{"Astral", "Crimson", "Hidden", "Neon", "Frozen", "Solar", "Ancient", "Silent", "Golden", "Wild"}
	nouns := []string{"Frontier", "Citadel", "Odyssey", "Circuit", "Kingdom", "Voyage", "Echo", "Horizon", "Legends", "Expedition", "Rivals", "Valley", "Forge", "Realm", "Quest"}
	platforms := []string{"Steam", "Epic Games", "GOG", "Xbox", "PlayStation"}
	regions := []string{"Global", "Europe", "North America"}
	genres := []string{"Action", "Adventure", "RPG", "Strategy", "Simulation", "Racing"}
	products := make([]Product, 0, 150)
	for i := 0; i < 150; i++ {
		p := Product{
			Name: adjectives[i/15] + " " + nouns[i%15], Slug: fmt.Sprintf("demo-game-%03d", i+1),
			Description: "Fictional game created for marketplace development and testing. No redeemable content.",
			Platform:    platforms[rng.Intn(len(platforms))], Region: regions[rng.Intn(len(regions))], Genre: genres[rng.Intn(len(genres))],
			ImageURL:    fmt.Sprintf("https://placehold.co/600x800/png?text=Demo+Game+%03d", i+1),
			ReleaseDate: time.Date(2018+rng.Intn(8), time.Month(1+rng.Intn(12)), 1+rng.Intn(28), 0, 0, 0, 0, time.UTC),
		}
		sellers := rng.Perm(10)
		count := 2 + rng.Intn(7)
		for _, seller := range sellers[:count] {
			offer := Offer{Seller: seller, PriceCents: 499 + rng.Intn(9501), Keys: 5 + rng.Intn(26)}
			if rng.Intn(3) == 0 {
				offer.DiscountCents = offer.PriceCents * (60 + rng.Intn(31)) / 100
			}
			p.Offers = append(p.Offers, offer)
		}
		products = append(products, p)
	}
	applyShowcase(products)
	return products
}

func Run(ctx context.Context, pool *pgxpool.Pool) (Counts, error) {
	return run(ctx, pool, false)
}

func RunIfEmpty(ctx context.Context, pool *pgxpool.Pool) (Counts, error) {
	return run(ctx, pool, true)
}

func run(ctx context.Context, pool *pgxpool.Pool, skipExisting bool) (Counts, error) {
	var counts Counts
	tx, err := pool.Begin(ctx)
	if err != nil {
		return counts, err
	}
	defer tx.Rollback(context.Background())

	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(741530021)"); err != nil {
		return counts, err
	}

	if _, err = tx.Exec(ctx, "LOCK TABLE products, sellers IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return counts, err
	}
	var populated bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM products) OR EXISTS(SELECT 1 FROM sellers)").Scan(&populated); err != nil {
		return counts, err
	}
	if populated {
		if skipExisting {
			return counts, nil
		}
		return counts, errors.New("seed refused: products or sellers already exist; use an empty development database")
	}
	sellers := make([]int64, 10)
	for i := range sellers {
		err = tx.QueryRow(ctx, "INSERT INTO sellers(name,rating) VALUES ($1,$2) RETURNING id", sellerNames[i], money(350+i*15)).Scan(&sellers[i])
		if err != nil {
			return counts, err
		}
		counts.Sellers++
	}
	for _, p := range Generate() {
		var productID int64
		err = tx.QueryRow(ctx, `INSERT INTO products(name,slug,description,platform,region,genre,image_url,release_date) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, p.Name, p.Slug, p.Description, p.Platform, p.Region, p.Genre, p.ImageURL, p.ReleaseDate).Scan(&productID)
		if err != nil {
			return counts, err
		}
		counts.Products++
		for _, offer := range p.Offers {
			var offerID int64
			var discount any
			if offer.DiscountCents > 0 {
				discount = money(offer.DiscountCents)
			}
			err = tx.QueryRow(ctx, `INSERT INTO offers(product_id,seller_id,price,discount_price,currency,stock) VALUES ($1,$2,$3,$4,'EUR',$5) RETURNING id`, productID, sellers[offer.Seller], money(offer.PriceCents), discount, offer.Keys).Scan(&offerID)
			if err != nil {
				return counts, err
			}
			rows := make([][]any, offer.Keys)
			for k := range rows {
				rows[k] = []any{offerID, fmt.Sprintf("FAKE-NOT-REDEEMABLE-%s-%02d-%02d", p.Slug, offer.Seller+1, k+1)}
			}
			if _, err = tx.CopyFrom(ctx, pgx.Identifier{"game_keys"}, []string{"offer_id", "key_value"}, pgx.CopyFromRows(rows)); err != nil {
				return counts, err
			}
			counts.Offers++
			counts.Keys += offer.Keys
		}
	}
	home, err := addHomepage(ctx, tx)
	if err != nil {
		return Counts{}, err
	}
	counts.Banners, counts.Orders = home.Banners, home.Orders
	if err := tx.Commit(ctx); err != nil {
		return Counts{}, err
	}
	return counts, nil
}

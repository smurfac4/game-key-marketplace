package seed

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed showcase.json
var showcaseJSON []byte

type showcaseGame struct {
	AppID       int    `json:"app_id"`
	Name        string `json:"name"`
	Genre       string `json:"genre"`
	ReleaseDate string `json:"release_date"`
	Description string `json:"description"`
}

const demoNotice = "Demo inventory: fictional prices, sellers and non-redeemable keys. No publisher affiliation."

var sellerNames = []string{"Pixel Harbor", "Level Up Market", "Digital Orchard", "Next Quest", "Game District", "Night Owl Keys", "Play Avenue", "Checkpoint Store", "Arcade Supply", "Bright Byte"}

func applyShowcase(products []Product) {
	var games []showcaseGame
	if err := json.Unmarshal(showcaseJSON, &games); err != nil || len(games) != 50 {
		panic("invalid embedded showcase catalog")
	}
	for i := range products {
		g := games[i%len(games)]
		p := &products[i]
		p.Name, p.Genre = g.Name, g.Genre
		p.Description = g.Description + "\n\n" + demoNotice
		art := "adventure"
		if g.Genre == "RPG" || g.Genre == "Strategy" {
			art = "fantasy"
		}
		if g.Genre == "Action" || g.Genre == "Racing" || g.AppID == 1091500 {
			art = "scifi"
		}
		p.ImageURL = "/media/games/" + art + "-v3.png"
		p.Platform = "Steam"

		if i >= 50 && (g.AppID == 1091500 || g.AppID == 292030 || g.AppID == 413150 || g.AppID == 367520 || g.AppID == 250900 || g.AppID == 391540 || g.AppID == 427520) {
			p.Platform = "GOG"
		}
		if i >= 100 && (g.AppID == 271590 || g.AppID == 1174180 || g.AppID == 990080 || g.AppID == 870780 || g.AppID == 289070 || g.AppID == 255710) {
			p.Platform = "Epic Games"
		}
		p.Region = []string{"Global", "Europe", "North America"}[i/50]
		p.ReleaseDate, _ = time.Parse("2006-01-02", g.ReleaseDate)
	}
}

func RunShowcase(ctx context.Context, pool *pgxpool.Pool) (Counts, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Counts{}, err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(741530021)"); err != nil {
		return Counts{}, err
	}
	var owned int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM products WHERE slug ~ '^demo-game-[0-9]{3}$' AND (description=$1 OR description LIKE '%' || $2)`, "Fictional game created for marketplace development and testing. No redeemable content.", demoNotice).Scan(&owned); err != nil {
		return Counts{}, err
	}
	if owned != 150 {
		return Counts{}, errors.New("showcase refresh requires the owned 150-product demo catalog")
	}
	for _, p := range Generate() {
		if _, err = tx.Exec(ctx, `UPDATE products SET name=$1,description=$2,platform=$3,region=$4,genre=$5,image_url=$6,release_date=$7 WHERE slug=$8`, p.Name, p.Description, p.Platform, p.Region, p.Genre, p.ImageURL, p.ReleaseDate, p.Slug); err != nil {
			return Counts{}, err
		}
	}
	for i, name := range sellerNames {
		if _, err = tx.Exec(ctx, `UPDATE sellers SET name=$1 WHERE name=$2`, name, fmt.Sprintf("Demo Seller %02d", i+1)); err != nil {
			return Counts{}, err
		}
	}

	for i := range []int{1091500, 1245620, 1174180} {
		if _, err = tx.Exec(ctx, `UPDATE banners SET image_url=$1 WHERE position=$2 AND title=$3 AND (image_url LIKE 'https://placehold.co/%' OR image_url LIKE '/media/banners/%') AND EXISTS(SELECT 1 FROM homepage_blocks WHERE slug='banners' AND config->>'seed'='homepage-v1')`, "/media/banners/portal-v2.png", i, strings.Split("Discover your next adventure|Explore this week's deals|Favorites from our players", "|")[i]); err != nil {
			return Counts{}, err
		}
	}
	return Counts{Products: 150}, tx.Commit(ctx)
}

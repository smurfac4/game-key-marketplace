package account

import (
	"context"
	"encoding/json"
	"errors"
	"example.com/game-key-marketplace/api/internal/money"
	"github.com/jackc/pgx/v5/pgxpool"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrInvalid = errors.New("invalid profile")
var ErrFull = errors.New("wishlist limit reached")
var displayNamePattern = regexp.MustCompile(`^[\pL\pN][\pL\pN ._-]{1,31}$`)
var ErrNotFound = errors.New("product not found")

type Profile struct {
	DisplayName           string `json:"display_name"`
	CountryCode           string `json:"country_code"`
	Avatar                string `json:"avatar"`
	PreferredPlatform     string `json:"preferred_platform"`
	MarketingOptIn        bool   `json:"marketing_opt_in"`
	WishlistNotifications bool   `json:"wishlist_notifications"`
}
type Completion struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
	Percent   int `json:"percent"`
}
type Summary struct {
	Profile        Profile    `json:"profile"`
	Completion     Completion `json:"completion"`
	WishlistCount  int64      `json:"wishlist_count"`
	OrderCount     int64      `json:"order_count"`
	KeyCount       int64      `json:"key_count"`
	Currency       string     `json:"currency"`
	WalletBalance  string     `json:"wallet_balance"`
	CashbackEarned string     `json:"cashback_earned"`
	RewardPoints   int64      `json:"reward_points"`
	Demo           bool       `json:"demo"`
	Platforms      []string   `json:"platforms"`
	Countries      []string   `json:"countries"`
}

var Countries = []string{"AU", "CA", "DE", "ES", "FR", "GB", "IT", "LT", "NL", "PL", "US"}

func Progress(p Profile) Completion {
	n := 2
	for _, s := range []string{p.DisplayName, p.CountryCode, p.Avatar, p.PreferredPlatform} {
		if s != "" {
			n++
		}
	}
	return Completion{n, 6, n * 100 / 6}
}
func Validate(p Profile) (Profile, error) {
	p.DisplayName = strings.TrimSpace(p.DisplayName)
	if p.DisplayName != "" && (!utf8.ValidString(p.DisplayName) || !displayNamePattern.MatchString(p.DisplayName)) {
		return p, ErrInvalid
	}
	valid := p.CountryCode == ""
	for _, c := range Countries {
		valid = valid || c == p.CountryCode
	}
	if !valid {
		return p, ErrInvalid
	}
	valid = false
	for _, a := range []string{"", "nebula", "pixel", "orbit", "ember"} {
		valid = valid || a == p.Avatar
	}
	if !valid || len(p.PreferredPlatform) > 40 {
		return p, ErrInvalid
	}
	return p, nil
}

type Repository struct{ DB *pgxpool.Pool }

func (r Repository) Get(ctx context.Context, user int64, currency string) (Summary, error) {
	var result Summary
	var raw []byte
	err := r.DB.QueryRow(ctx, `SELECT json_build_object('profile',coalesce((SELECT to_jsonb(p)-'user_id'-'updated_at' FROM account_profiles p WHERE p.user_id=$1),'{}'::jsonb),'wishlist_count',(SELECT count(*) FROM wishlists WHERE user_id=$1),'order_count',(SELECT count(*) FROM orders WHERE user_id=$1 AND status='paid'),'key_count',(SELECT count(*) FROM game_keys k JOIN order_items i ON i.id=k.order_item_id JOIN orders o ON o.id=i.order_id WHERE o.user_id=$1 AND o.status='paid' AND k.status='sold'),'platforms',coalesce((SELECT json_agg(platform ORDER BY platform) FROM (SELECT DISTINCT platform FROM products) s),'[]'::json))`, user).Scan(&raw)
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		return result, err
	}
	result.Completion = Progress(result.Profile)
	result.Currency = currency
	result.WalletBalance = "0.00"
	result.Demo = true
	result.Countries = Countries
	rows, err := r.DB.Query(ctx, `SELECT currency,sum(total_amount)::text FROM orders WHERE user_id=$1 AND status='paid' GROUP BY currency`, user)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	var euro int64
	for rows.Next() {
		var c, s string
		if err = rows.Scan(&c, &s); err != nil {
			return result, err
		}
		n, e := money.Cents(s)
		if e != nil {
			return result, e
		}
		if money.Supported(c) {
			euro += money.Convert(n, c, "EUR")
		}
	}
	result.RewardPoints = euro / 100
	result.CashbackEarned = money.Text(money.Convert(euro/100, "EUR", currency))
	return result, rows.Err()
}
func (r Repository) Update(ctx context.Context, user int64, p Profile) error {
	p, err := Validate(p)
	if err != nil {
		return err
	}
	if p.PreferredPlatform != "" {
		var exists bool
		if err = r.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE platform=$1)`, p.PreferredPlatform).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrInvalid
		}
	}
	_, err = r.DB.Exec(ctx, `INSERT INTO account_profiles(user_id,display_name,country_code,avatar,preferred_platform,marketing_opt_in,wishlist_notifications) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(user_id) DO UPDATE SET display_name=excluded.display_name,country_code=excluded.country_code,avatar=excluded.avatar,preferred_platform=excluded.preferred_platform,marketing_opt_in=excluded.marketing_opt_in,wishlist_notifications=excluded.wishlist_notifications,updated_at=now()`, user, p.DisplayName, p.CountryCode, p.Avatar, p.PreferredPlatform, p.MarketingOptIn, p.WishlistNotifications)
	return err
}

type Favorite struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Slug     string  `json:"slug"`
	ImageURL string  `json:"image_url"`
	Platform string  `json:"platform"`
	Region   string  `json:"region"`
	Price    *string `json:"price"`
	Currency string  `json:"currency"`
}

func (r Repository) Wishlist(ctx context.Context, user int64, currency string) ([]Favorite, error) {
	result := []Favorite{}
	rows, err := r.DB.Query(ctx, money.OffersSQL(`SELECT p.id,p.name,p.slug,p.image_url,p.platform,p.region,o.price::text FROM wishlists w JOIN products p ON p.id=w.product_id LEFT JOIN LATERAL (SELECT min(coalesce(discount_price,price)) AS price FROM offers WHERE product_id=p.id AND stock>0 AND currency=$1) o ON true WHERE w.user_id=$2 ORDER BY w.created_at DESC,p.id LIMIT 200`), currency, user)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var f Favorite
		if err = rows.Scan(&f.ID, &f.Name, &f.Slug, &f.ImageURL, &f.Platform, &f.Region, &f.Price); err != nil {
			return result, err
		}
		f.Currency = currency
		result = append(result, f)
	}
	return result, rows.Err()
}
func (r Repository) Favorite(ctx context.Context, user, product int64, add bool) error {
	if product < 1 {
		return ErrInvalid
	}
	if !add {
		_, err := r.DB.Exec(ctx, `DELETE FROM wishlists WHERE user_id=$1 AND product_id=$2`, user, product)
		return err
	}
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		tx.Rollback(c)
	}()
	var owner int64
	if err = tx.QueryRow(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, user).Scan(&owner); err != nil {
		return err
	}
	var count int
	var present bool
	if err = tx.QueryRow(ctx, `SELECT count(*),coalesce(bool_or(product_id=$2),false) FROM wishlists WHERE user_id=$1`, user, product).Scan(&count, &present); err != nil {
		return err
	}
	if count >= 200 && !present {
		return ErrFull
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1)`, product).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	_, err = tx.Exec(ctx, `INSERT INTO wishlists(user_id,product_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, user, product)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

package commerce

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"example.com/game-key-marketplace/api/internal/money"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrIdempotency = errors.New("idempotency key reused with different request")
	ErrNotFound    = errors.New("resource not found")
	ErrOutOfStock  = errors.New("not enough available keys")
	ErrEmptyCart   = errors.New("cart is empty")
	ErrCurrency    = errors.New("all cart items must have the same currency")
	ErrInvalid     = errors.New("invalid quantity or identifier")
	ErrCartLimit   = errors.New("cart limit reached")
)
var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,80}$`)

func ValidToken(s string) bool { return tokenPattern.MatchString(s) }

type Repository struct {
	DB       *pgxpool.Pool
	Payments PaymentProvider
}
type CartItem struct {
	OfferID   int64  `json:"offer_id"`
	ProductID int64  `json:"product_id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	ImageURL  string `json:"image_url"`
	Seller    string `json:"seller"`
	Quantity  int    `json:"quantity"`
	Stock     int    `json:"stock"`
	UnitPrice string `json:"unit_price"`
	Total     string `json:"total"`
	Currency  string `json:"currency"`
}
type Cart struct {
	Items    []CartItem `json:"items"`
	Total    string     `json:"total"`
	Currency string     `json:"currency"`
}
type OrderItem struct {
	ID          int64    `json:"id"`
	ProductName string   `json:"product_name"`
	SellerName  string   `json:"seller_name"`
	Quantity    int      `json:"quantity"`
	UnitPrice   string   `json:"unit_price"`
	Currency    string   `json:"currency"`
	Keys        []string `json:"keys"`
}
type Order struct {
	ID        int64       `json:"id"`
	Status    string      `json:"status"`
	Currency  string      `json:"currency"`
	Total     string      `json:"total"`
	CreatedAt time.Time   `json:"created_at"`
	Items     []OrderItem `json:"items"`
}
type OrderList struct {
	Data       []Order `json:"data"`
	Page       int     `json:"page"`
	Total      int64   `json:"total"`
	TotalPages int64   `json:"total_pages"`
}

func decodeRow[T any](row pgx.Row) (T, error) {
	var result T
	var raw []byte
	err := row.Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrNotFound
	}
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(raw, &result)
	return result, err
}
func (r Repository) GetCart(ctx context.Context, user int64) (Cart, error) {
	return decodeRow[Cart](r.DB.QueryRow(ctx, `WITH lines AS (
 SELECT o.id AS offer_id,p.id AS product_id,p.slug,p.name,p.image_url,s.name AS seller,ci.quantity,o.stock,
 coalesce(o.discount_price,o.price)::text AS unit_price,(coalesce(o.discount_price,o.price)*ci.quantity)::text AS total,o.currency
 FROM carts c JOIN cart_items ci ON ci.cart_id=c.id JOIN offers o ON o.id=ci.offer_id JOIN products p ON p.id=o.product_id JOIN sellers s ON s.id=o.seller_id WHERE c.user_id=$1
 ) SELECT json_build_object('items',coalesce((SELECT json_agg(lines ORDER BY offer_id) FROM lines),'[]'::json),
 'total',(SELECT coalesce(sum(total::numeric),0.00)::text FROM lines),'currency',coalesce((SELECT min(currency) FROM lines),'EUR'))`, user))
}
func lockCart(ctx context.Context, tx pgx.Tx, user int64) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `INSERT INTO carts(user_id) VALUES ($1) ON CONFLICT(user_id) DO UPDATE SET user_id=excluded.user_id RETURNING id`, user).Scan(&id)
	return id, err
}

// All cart mutations and checkout lock the same cart row. Inventory is not reserved by a cart.
func (r Repository) SetItem(ctx context.Context, user, offer int64, quantity int, add bool) error {
	if user <= 0 || offer <= 0 || quantity < 1 || quantity > 10 {
		return ErrInvalid
	}
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	cart, err := lockCart(ctx, tx, user)
	if err != nil {
		return err
	}
	var stock int
	var currency string
	err = tx.QueryRow(ctx, "SELECT stock,currency FROM offers WHERE id=$1", offer).Scan(&stock, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	var old int
	err = tx.QueryRow(ctx, "SELECT quantity FROM cart_items WHERE cart_id=$1 AND offer_id=$2", cart, offer).Scan(&old)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if !add && errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if add {
		quantity += old
	}
	if quantity > 10 {
		return ErrCartLimit
	}
	if quantity > stock {
		return ErrOutOfStock
	}
	var otherCurrency bool
	var lines int
	if err = tx.QueryRow(ctx, `SELECT count(*),coalesce(bool_or(o.currency<>$2),false) FROM cart_items ci JOIN offers o ON o.id=ci.offer_id WHERE cart_id=$1`, cart, currency).Scan(&lines, &otherCurrency); err != nil {
		return err
	}
	if otherCurrency {
		return ErrCurrency
	}
	if lines >= 50 && old == 0 {
		return ErrCartLimit
	}
	_, err = tx.Exec(ctx, `INSERT INTO cart_items(cart_id,offer_id,quantity) VALUES ($1,$2,$3) ON CONFLICT(cart_id,offer_id) DO UPDATE SET quantity=excluded.quantity`, cart, offer, quantity)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r Repository) RemoveItem(ctx context.Context, user, offer int64) error {
	if offer <= 0 {
		return ErrInvalid
	}
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	cart, err := lockCart(ctx, tx, user)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM cart_items WHERE cart_id=$1 AND offer_id=$2", cart, offer); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type purchaseLine struct {
	OfferID                   int64
	Quantity, Stock           int
	Cents                     int64
	Currency, Product, Seller string
}

func amountText(cents int64) string { return fmt.Sprintf("%d.%02d", cents/100, cents%100) }
func (r Repository) Checkout(ctx context.Context, user int64, token, card string, requested ...string) (int64, error) {
	if !ValidToken(token) || len(card) > 32 {
		return 0, ErrInvalid
	}
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(context.Background())
	cart, err := lockCart(ctx, tx, user)
	if err != nil {
		return 0, err
	}
	target := ""
	if len(requested) > 0 {
		target = requested[0]
		if !money.Supported(target) {
			return 0, ErrCurrency
		}
	}
	fingerprintBytes := sha256.Sum256([]byte(card + "\x00" + target))
	fingerprint := hex.EncodeToString(fingerprintBytes[:])
	var previousFingerprint *string
	var existing int64
	err = tx.QueryRow(ctx, "SELECT id,checkout_fingerprint FROM orders WHERE user_id=$1 AND checkout_token=$2", user, token).Scan(&existing, &previousFingerprint)
	if err == nil {
		if previousFingerprint == nil || *previousFingerprint != fingerprint {
			return 0, ErrIdempotency
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	// Lock offers in a global order: concurrent multi-offer checkouts cannot deadlock by lock inversion.
	rows, err := tx.Query(ctx, `SELECT o.id,ci.quantity,o.stock,(coalesce(o.discount_price,o.price)*100)::bigint,o.currency,p.name,s.name
 FROM cart_items ci JOIN offers o ON o.id=ci.offer_id JOIN products p ON p.id=o.product_id JOIN sellers s ON s.id=o.seller_id
 WHERE ci.cart_id=$1 ORDER BY o.id FOR UPDATE OF o`, cart)
	if err != nil {
		return 0, err
	}
	lines := []purchaseLine{}
	for rows.Next() {
		var l purchaseLine
		if err = rows.Scan(&l.OfferID, &l.Quantity, &l.Stock, &l.Cents, &l.Currency, &l.Product, &l.Seller); err != nil {
			rows.Close()
			return 0, err
		}
		lines = append(lines, l)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	if len(lines) == 0 {
		return 0, ErrEmptyCart
	}
	if len(lines) > 50 {
		return 0, ErrCartLimit
	}
	baseCurrency := lines[0].Currency
	currency := baseCurrency
	if target != "" {
		currency = target
	}
	var total int64
	for i := range lines {
		l := &lines[i]
		if l.Currency != baseCurrency {
			return 0, ErrCurrency
		}
		if l.Quantity < 1 || l.Quantity > 10 {
			return 0, ErrInvalid
		}
		if l.Stock < l.Quantity {
			return 0, ErrOutOfStock
		}
		if target != "" {
			if !money.Supported(l.Currency) {
				return 0, ErrCurrency
			}
			l.Cents = money.Convert(l.Cents, l.Currency, target)
		}
		total += l.Cents * int64(l.Quantity)
	}
	if total > 999999999999 {
		return 0, ErrCartLimit
	}
	var orderID int64
	if err = tx.QueryRow(ctx, `INSERT INTO orders(user_id,currency,total_amount,checkout_token,checkout_fingerprint) VALUES ($1,$2,$3,$4,$5) RETURNING id`, user, currency, amountText(total), token, fingerprint).Scan(&orderID); err != nil {
		return 0, err
	}
	for _, l := range lines {
		keys, err := tx.Query(ctx, `SELECT id FROM game_keys WHERE offer_id=$1 AND status='available' ORDER BY id LIMIT $2 FOR UPDATE SKIP LOCKED`, l.OfferID, l.Quantity)
		if err != nil {
			return 0, err
		}
		ids := []int64{}
		for keys.Next() {
			var id int64
			if err = keys.Scan(&id); err != nil {
				keys.Close()
				return 0, err
			}
			ids = append(ids, id)
		}
		err = keys.Err()
		keys.Close()
		if err != nil {
			return 0, err
		}
		if len(ids) != l.Quantity {
			return 0, ErrOutOfStock
		}
		var itemID int64
		if err = tx.QueryRow(ctx, `INSERT INTO order_items(order_id,offer_id,product_name,seller_name,quantity,unit_price,currency) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, orderID, l.OfferID, l.Product, l.Seller, l.Quantity, money.Text(l.Cents), currency).Scan(&itemID); err != nil {
			return 0, err
		}
		if _, err = tx.Exec(ctx, `UPDATE game_keys SET status='sold',order_item_id=$1 WHERE id=ANY($2::bigint[])`, itemID, ids); err != nil {
			return 0, err
		}
		if _, err = tx.Exec(ctx, "UPDATE offers SET stock=stock-$2 WHERE id=$1", l.OfferID, l.Quantity); err != nil {
			return 0, err
		}
	}
	// Only a side-effect-free mock is permitted here. A real provider requires an outbox/saga,
	// provider idempotency and compensation; a DB rollback cannot refund a real charge.
	if r.Payments == nil {
		return 0, errors.New("payment provider not configured")
	}
	if err = r.Payments.Charge(ctx, Payment{orderID, total, currency, card}); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, "UPDATE orders SET status='paid' WHERE id=$1", orderID); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM cart_items WHERE cart_id=$1", cart); err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return orderID, nil
}

// Every key-bearing query is scoped by authenticated user ID, never a client-supplied owner.
const orderJSON = `json_build_object('id',o.id,'status',o.status,'currency',o.currency,'total',o.total_amount::text,'created_at',o.created_at,
 'items',coalesce((SELECT json_agg(json_build_object('id',oi.id,'product_name',oi.product_name,'seller_name',oi.seller_name,'quantity',oi.quantity,'unit_price',oi.unit_price::text,'currency',oi.currency,
 'keys',coalesce((SELECT json_agg(k.key_value ORDER BY k.id) FROM game_keys k WHERE k.order_item_id=oi.id AND k.status='sold'),'[]'::json)) ORDER BY oi.id) FROM order_items oi WHERE oi.order_id=o.id),'[]'::json))`

func (r Repository) GetOrder(ctx context.Context, user, id int64) (Order, error) {
	return decodeRow[Order](r.DB.QueryRow(ctx, "SELECT "+orderJSON+" FROM orders o WHERE o.user_id=$1 AND o.id=$2", user, id))
}
func (r Repository) Orders(ctx context.Context, user int64, page int) (OrderList, error) {
	return decodeRow[OrderList](r.DB.QueryRow(ctx, `WITH mine AS (SELECT * FROM orders WHERE user_id=$1),page_rows AS (SELECT * FROM mine ORDER BY created_at DESC,id DESC LIMIT 20 OFFSET $2)
 SELECT json_build_object('data',coalesce((SELECT json_agg(`+orderJSON+` ORDER BY o.created_at DESC,o.id DESC) FROM page_rows o),'[]'::json),
 'page',$3::integer,'total',(SELECT count(*) FROM mine),'total_pages',(SELECT (count(*)+19)/20 FROM mine))`, user, (page-1)*20, page))
}

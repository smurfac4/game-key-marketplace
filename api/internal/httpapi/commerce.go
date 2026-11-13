package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"example.com/game-key-marketplace/api/internal/money"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"example.com/game-key-marketplace/api/internal/commerce"
	"github.com/go-chi/chi/v5"
)

type Commerce struct{ Repo commerce.Repository }

func bodyJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	typ, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || typ != "application/json" {
		writeError(w, 415, "unsupported_media_type", "Content-Type must be application/json", "")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(target); err != nil {
		writeError(w, 400, "invalid_body", "invalid JSON body", "")
		return false
	}
	if err = decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, 400, "invalid_body", "expected one JSON object", "")
		return false
	}
	return true
}
func commerceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, commerce.ErrIdempotency):
		writeError(w, 409, "idempotency_conflict", "Idempotency key was already used with a different request.", "")
	case errors.Is(err, commerce.ErrNotFound):
		writeError(w, 404, "not_found", "resource not found", "")
	case errors.Is(err, commerce.ErrOutOfStock):
		writeError(w, 409, "out_of_stock", "Some keys are no longer available. Update your cart and try again.", "")
	case errors.Is(err, commerce.ErrEmptyCart):
		writeError(w, 409, "empty_cart", "Your cart is empty.", "")
	case errors.Is(err, commerce.ErrCurrency):
		writeError(w, 409, "currency_mismatch", "Choose offers in one currency per order.", "")
	case errors.Is(err, commerce.ErrCartLimit):
		writeError(w, 400, "cart_limit", "Maximum 10 keys per offer and 50 offers per cart.", "")
	case errors.Is(err, commerce.ErrPaymentDeclined):
		writeError(w, 402, "payment_declined", "Mock payment declined. Your cart and keys have not changed.", "")
	case errors.Is(err, commerce.ErrInvalidCard):
		writeError(w, 400, "invalid_card", "Use a supported test card number.", "card_number")
	case errors.Is(err, commerce.ErrInvalid):
		writeError(w, 400, "invalid_input", "Invalid identifier, quantity, or idempotency key.", "")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, 504, "timeout", "request timed out", "")
	default:
		writeError(w, 500, "internal_error", "Unable to complete request.", "")
	}
}
func routeID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, commerce.ErrInvalid
	}
	return id, nil
}
func (c Commerce) Cart(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r.Context())
	cart, err := c.Repo.GetCart(r.Context(), u.ID)
	if err != nil {
		commerceError(w, err)
		return
	}
	target := r.URL.Query().Get("currency")
	if target != "" {
		if !money.Supported(target) {
			commerceError(w, commerce.ErrCurrency)
			return
		}
		var total int64
		for i := range cart.Items {
			l := &cart.Items[i]
			if !money.Supported(l.Currency) {
				commerceError(w, commerce.ErrCurrency)
				return
			}
			l.UnitPrice = money.Value(l.UnitPrice, l.Currency, target)
			n, _ := money.Cents(l.UnitPrice)
			l.Total = money.Text(n * int64(l.Quantity))
			total += n * int64(l.Quantity)
			l.Currency = target
		}
		cart.Currency = target
		cart.Total = money.Text(total)
	}
	writeJSON(w, 200, map[string]any{"data": cart})
}
func (c Commerce) Add(w http.ResponseWriter, r *http.Request) {
	var b struct {
		OfferID  int64 `json:"offer_id"`
		Quantity int   `json:"quantity"`
	}
	if !bodyJSON(w, r, &b) {
		return
	}
	u, _ := CurrentUser(r.Context())
	if err := c.Repo.SetItem(r.Context(), u.ID, b.OfferID, b.Quantity, true); err != nil {
		commerceError(w, err)
		return
	}
	c.Cart(w, r)
}
func (c Commerce) Update(w http.ResponseWriter, r *http.Request) {
	id, err := routeID(r)
	if err != nil {
		commerceError(w, err)
		return
	}
	var b struct {
		Quantity int `json:"quantity"`
	}
	if !bodyJSON(w, r, &b) {
		return
	}
	u, _ := CurrentUser(r.Context())
	if err = c.Repo.SetItem(r.Context(), u.ID, id, b.Quantity, false); err != nil {
		commerceError(w, err)
		return
	}
	c.Cart(w, r)
}
func (c Commerce) Remove(w http.ResponseWriter, r *http.Request) {
	id, err := routeID(r)
	if err != nil {
		commerceError(w, err)
		return
	}
	u, _ := CurrentUser(r.Context())
	if err = c.Repo.RemoveItem(r.Context(), u.ID, id); err != nil {
		commerceError(w, err)
		return
	}
	c.Cart(w, r)
}
func (c Commerce) Checkout(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Card     string `json:"card_number"`
		Currency string `json:"currency"`
	}
	if !bodyJSON(w, r, &b) {
		return
	}
	u, _ := CurrentUser(r.Context())
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	id, err := c.Repo.Checkout(ctx, u.ID, r.Header.Get("Idempotency-Key"), b.Card, checkoutCurrency(b.Currency))
	if err != nil {
		commerceError(w, err)
		return
	}
	order, err := c.Repo.GetOrder(ctx, u.ID, id)
	if err != nil {
		commerceError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"data": order})
}
func (c Commerce) Order(w http.ResponseWriter, r *http.Request) {
	id, err := routeID(r)
	if err != nil {
		commerceError(w, err)
		return
	}
	u, _ := CurrentUser(r.Context())
	order, err := c.Repo.GetOrder(r.Context(), u.ID, id)
	if err != nil {
		commerceError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"data": order})
}
func (c Commerce) Orders(w http.ResponseWriter, r *http.Request) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		commerceError(w, commerce.ErrInvalid)
		return
	}
	page := 1
	for k, v := range values {
		if k != "page" || len(v) != 1 {
			commerceError(w, commerce.ErrInvalid)
			return
		}
	}
	if v, ok := values["page"]; ok {
		page, err = strconv.Atoi(v[0])
	}
	if err != nil || page < 1 || page > 1000000 {
		commerceError(w, commerce.ErrInvalid)
		return
	}
	u, _ := CurrentUser(r.Context())
	orders, err := c.Repo.Orders(r.Context(), u.ID, page)
	if err != nil {
		commerceError(w, err)
		return
	}
	writeJSON(w, 200, orders)
}

func checkoutCurrency(c string) string {
	if c == "" {
		return "EUR"
	}
	return c
}

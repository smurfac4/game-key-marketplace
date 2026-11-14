package httpapi

import (
	"errors"
	"example.com/game-key-marketplace/api/internal/account"
	"example.com/game-key-marketplace/api/internal/money"
	"net/http"
)

type Account struct{ Repo account.Repository }

func accountCurrency(r *http.Request) string {
	c := money.Currency(r.Context())
	if c == "" {
		c = "EUR"
	}
	return c
}
func accountError(w http.ResponseWriter, err error) {
	if errors.Is(err, account.ErrFull) {
		writeError(w, 409, "wishlist_full", "Your wishlist can hold up to 200 products.", "")
		return
	}
	if errors.Is(err, account.ErrInvalid) {
		writeError(w, 400, "invalid_profile", "Check your display name, country, avatar and platform.", "")
		return
	}
	if errors.Is(err, account.ErrNotFound) {
		writeError(w, 404, "not_found", "product not found", "")
		return
	}
	commerceError(w, err)
}
func (a Account) Get(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r.Context())
	data, err := a.Repo.Get(r.Context(), u.ID, accountCurrency(r))
	if err != nil {
		accountError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"data": data})
}
func (a Account) Update(w http.ResponseWriter, r *http.Request) {
	var p account.Profile
	if !bodyJSON(w, r, &p) {
		return
	}
	u, _ := CurrentUser(r.Context())
	if err := a.Repo.Update(r.Context(), u.ID, p); err != nil {
		accountError(w, err)
		return
	}
	a.Get(w, r)
}
func (a Account) Wishlist(w http.ResponseWriter, r *http.Request) {
	u, _ := CurrentUser(r.Context())
	data, err := a.Repo.Wishlist(r.Context(), u.ID, accountCurrency(r))
	if err != nil {
		accountError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"data": data})
}
func (a Account) Favorite(w http.ResponseWriter, r *http.Request) {
	id, err := routeID(r)
	if err != nil {
		commerceError(w, err)
		return
	}
	u, _ := CurrentUser(r.Context())
	if err = a.Repo.Favorite(r.Context(), u.ID, id, r.Method == "PUT"); err != nil {
		accountError(w, err)
		return
	}
	w.WriteHeader(204)
}

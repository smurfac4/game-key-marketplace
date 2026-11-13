package httpapi

import (
	"context"
	"errors"
	"example.com/game-key-marketplace/api/internal/money"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"example.com/game-key-marketplace/api/internal/products"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func productDetail(store ProductLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		result, err := store.Detail(ctx, chi.URLParam(r, "slug"))
		if errors.Is(err, products.ErrNotFound) {
			writeError(w, 404, "not_found", "product not found", "")
			return
		}
		if err != nil {
			slog.Error("product detail failed", "request_id", middleware.GetReqID(r.Context()), "error", err)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
				writeError(w, 504, "timeout", "request timed out", "")
			} else {
				writeError(w, 500, "internal_error", "unable to load product", "")
			}
			return
		}
		target := r.URL.Query().Get("currency")
		if target != "" {
			if !money.Supported(target) {
				writeError(w, 400, "invalid_query", "unsupported currency", "currency")
				return
			}
			for i := range result.Data.Offers {
				o := &result.Data.Offers[i]
				if !money.Supported(o.Currency) {
					continue
				}
				o.Price = money.Value(o.Price, o.Currency, target)
				o.EffectivePrice = money.Value(o.EffectivePrice, o.Currency, target)
				if o.DiscountPrice != nil {
					v := money.Value(*o.DiscountPrice, o.Currency, target)
					o.DiscountPrice = &v
				}
				o.Currency = target
			}
			sort.SliceStable(result.Data.Offers, func(i, j int) bool {
				a, _ := money.Cents(result.Data.Offers[i].EffectivePrice)
				b, _ := money.Cents(result.Data.Offers[j].EffectivePrice)
				return a < b
			})
		}
		writeJSON(w, 200, result)
	}
}

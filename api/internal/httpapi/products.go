package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"example.com/game-key-marketplace/api/internal/products"
	"github.com/go-chi/chi/v5/middleware"
)

type ProductLister interface {
	Detail(context.Context, string) (products.DetailResponse, error)
	List(context.Context, products.Query) (products.Response, error)
	Suggest(context.Context, products.Query) (products.Suggestions, error)
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, message, field string) {
	writeJSON(w, status, struct {
		Error errorBody `json:"error"`
	}{errorBody{code, message, field}})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func listProducts(store ProductLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q, err := products.ParseQuery(r.URL.RawQuery)
		if err != nil {
			var validation *products.ValidationError
			if errors.As(err, &validation) {
				writeError(w, 400, "invalid_query", validation.Message, validation.Field)
				return
			}
			writeError(w, 400, "invalid_query", "invalid query parameters", "")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		result, err := store.List(ctx, q)
		if err != nil {

			slog.Error("list products failed", "request_id", middleware.GetReqID(r.Context()), "error", err)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
				writeError(w, 504, "timeout", "request timed out", "")
				return
			}
			writeError(w, 500, "internal_error", "unable to list products", "")
			return
		}
		writeJSON(w, 200, result)
	}
}
func recoverJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				if value == http.ErrAbortHandler {
					panic(value)
				}
				slog.Error("HTTP handler panic", "request_id", middleware.GetReqID(r.Context()))
				writeError(w, 500, "internal_error", "internal server error", "")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func suggestProducts(store ProductLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q, err := products.ParseSuggest(r.URL.RawQuery)
		if err != nil {
			var v *products.ValidationError
			if errors.As(err, &v) {
				writeError(w, 400, "invalid_query", v.Message, v.Field)
			} else {
				writeError(w, 400, "invalid_query", "invalid query parameters", "")
			}
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		result, err := store.Suggest(ctx, q)
		if err != nil {
			slog.Error("suggest products failed", "request_id", middleware.GetReqID(r.Context()), "error", err)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
				writeError(w, 504, "timeout", "request timed out", "")
			} else {
				writeError(w, 500, "internal_error", "unable to suggest products", "")
			}
			return
		}
		writeJSON(w, 200, result)
	}
}

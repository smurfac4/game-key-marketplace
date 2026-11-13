// Package block contains only shared homepage transport plumbing, not block queries.
package block

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type DB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Response[T any] struct {
	Title string `json:"title"`
	Data  []T    `json:"data"`
}
type Card struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	ImageURL      string  `json:"image_url"`
	Platform      string  `json:"platform"`
	Region        string  `json:"region"`
	Price         string  `json:"price"`
	OriginalPrice *string `json:"original_price"`
	Currency      string  `json:"currency"`
	ReleaseDate   *string `json:"release_date"`
	UnitsSold     int64   `json:"units_sold,omitempty"`
}

func Read[T any](ctx context.Context, db DB, query string, args ...any) (Response[T], error) {
	result := Response[T]{Data: []T{}}
	var raw []byte
	err := db.QueryRow(ctx, query, args...).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(raw, &result)
	return result, err
}

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

func Currency(raw string) (string, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "", err
	}
	for k, v := range values {
		if k != "currency" || len(v) != 1 || !currencyPattern.MatchString(v[0]) {
			return "", errors.New("only one uppercase three-letter currency parameter is supported")
		}
	}
	if value := values.Get("currency"); value != "" {
		return value, nil
	}
	return "EUR", nil
}
func Serve[T any](w http.ResponseWriter, r *http.Request, fetch func(context.Context, string) (Response[T], error)) {
	currency, err := Currency(r.URL.RawQuery)
	if err != nil {
		writeError(w, 400, "invalid_query", "expected an optional three-letter uppercase currency")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	result, err := fetch(ctx, currency)
	if err != nil {
		slog.Error("homepage block failed", "path", r.URL.Path, "error", err)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			writeError(w, 504, "timeout", "request timed out")
		} else {
			writeError(w, 500, "internal_error", "unable to load homepage block")
		}
		return
	}
	w.Header().Add("Vary", "Accept-Language")
	if strings.HasPrefix(r.Header.Get("Accept-Language"), "ru") {
		titles := map[string]string{"Best sellers": "Лидеры продаж", "Deals worth discovering": "Выгодные предложения", "New releases": "Новые релизы", "Explore the marketplace": "Откройте мир игр"}
		if v, ok := titles[result.Title]; ok {
			result.Title = v
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}

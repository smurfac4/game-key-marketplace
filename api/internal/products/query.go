package products

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Query struct {
	Platform, Region, Genre, Search, Sort, Currency string
	MinPrice, MaxPrice                              *string
	Page, PageSize                                  int
}

type ValidationError struct{ Field, Message string }

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

var decimal = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,2})?$`)
var integer = regexp.MustCompile(`^[1-9][0-9]*$`)
var currency = regexp.MustCompile(`^[A-Z]{3}$`)
var sortSQL = map[string]string{
	"relevance":  relevanceOrder,
	"price_asc":  "price ASC, id ASC",
	"price_desc": "price DESC, id ASC",
	"popularity": "popularity DESC, id ASC",
	"newest":     "release_date DESC NULLS LAST, id ASC",
	"name":       `lower(name) COLLATE "C" ASC, id ASC`,
}

func ParseQuery(raw string) (Query, error) {
	q := Query{Page: 1, PageSize: 24, Sort: "popularity", Currency: "EUR"}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return q, &ValidationError{"query", "malformed query encoding"}
	}
	allowed := map[string]bool{"platform": true, "region": true, "genre": true, "q": true, "sort": true, "currency": true, "min_price": true, "max_price": true, "page": true, "page_size": true}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !allowed[key] {
			return q, &ValidationError{key, "unknown query parameter"}
		}
		if len(values[key]) != 1 {
			return q, &ValidationError{key, "must be provided once"}
		}
		value := values.Get(key)
		if !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
			return q, &ValidationError{key, "must contain valid text without control characters"}
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return q, &ValidationError{key, "must not be empty"}
		}
		values.Set(key, value)
	}
	for _, field := range []struct {
		key    string
		target *string
		limit  int
	}{{"platform", &q.Platform, 80}, {"region", &q.Region, 80}, {"genre", &q.Genre, 80}, {"q", &q.Search, 200}} {
		if v, ok := values[field.key]; ok {
			if utf8.RuneCountInString(v[0]) > field.limit {
				return q, &ValidationError{field.key, fmt.Sprintf("must be at most %d characters", field.limit)}
			}
			*field.target = v[0]
		}
	}
	if q.Search != "" {
		q.Sort = "relevance"
	}
	if v := values.Get("sort"); v != "" {
		if _, ok := sortSQL[v]; !ok {
			return q, &ValidationError{"sort", "must be relevance, price_asc, price_desc, popularity, newest, or name"}
		}
		q.Sort = v
	}
	if v := values.Get("currency"); v != "" {
		if !currency.MatchString(v) {
			return q, &ValidationError{"currency", "must be three uppercase letters"}
		}
		q.Currency = v
	}
	for _, field := range []struct {
		key    string
		target *int
		max    int
	}{{"page", &q.Page, 1000000}, {"page_size", &q.PageSize, 100}} {
		if v := values.Get(field.key); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || !integer.MatchString(v) || n > field.max {
				return q, &ValidationError{field.key, fmt.Sprintf("must be an integer from 1 to %d", field.max)}
			}
			*field.target = n
		}
	}
	var min, max int64
	for _, field := range []struct {
		key    string
		target **string
		cents  *int64
	}{{"min_price", &q.MinPrice, &min}, {"max_price", &q.MaxPrice, &max}} {
		if v := values.Get(field.key); v != "" {
			if !decimal.MatchString(v) {
				return q, &ValidationError{field.key, "must be a nonnegative decimal with at most 10 whole digits and 2 decimal places"}
			}
			parts := strings.SplitN(v, ".", 2)
			whole, _ := strconv.ParseInt(parts[0], 10, 64)
			fraction := "00"
			if len(parts) == 2 {
				fraction = (parts[1] + "0")[:2]
			}
			cents, _ := strconv.ParseInt(fraction, 10, 64)
			*field.cents = whole*100 + cents
			normalized := parts[0] + "." + fraction
			*field.target = &normalized
		}
	}
	if q.MinPrice != nil && q.MaxPrice != nil && min > max {
		return q, &ValidationError{"min_price", "must be less than or equal to max_price"}
	}
	return q, nil
}

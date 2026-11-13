package money

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

var Rates = map[string]int64{"EUR": 100, "USD": 110, "GBP": 85}

func Supported(c string) bool { return Rates[c] > 0 }
func Cents(s string) (int64, error) {
	p := strings.Split(s, ".")
	if len(p) != 2 || len(p[1]) != 2 {
		return 0, fmt.Errorf("invalid money")
	}
	n, e := strconv.ParseInt(p[0]+p[1], 10, 64)
	return n, e
}
func Text(n int64) string { return fmt.Sprintf("%d.%02d", n/100, n%100) }
func Convert(n int64, from, to string) int64 {
	if from == to {
		return n
	}
	a, b := Rates[from], Rates[to]
	if a == 0 || b == 0 {
		return n
	}
	return (n*b + a/2) / a
}
func Value(s, from, to string) string {
	n, e := Cents(s)
	if e != nil {
		return s
	}
	return Text(Convert(n, from, to))
}

func OffersSQL(query string) string {
	rate := `(CASE $1 WHEN 'EUR' THEN 100 WHEN 'USD' THEN 110 WHEN 'GBP' THEN 85 END)`
	source := `(CASE currency WHEN 'EUR' THEN 100 WHEN 'USD' THEN 110 WHEN 'GBP' THEN 85 END)`
	factor := `CASE WHEN $1 IN ('EUR','USD','GBP') AND currency IN ('EUR','USD','GBP') THEN ` + rate + `::numeric/` + source + ` ELSE 1 END`
	table := `FROM (SELECT id,product_id,seller_id,round(price*(` + factor + `),2) AS price,round(discount_price*(` + factor + `),2) AS discount_price,CASE WHEN $1 IN ('EUR','USD','GBP') AND currency IN ('EUR','USD','GBP') THEN $1 ELSE currency END AS currency,stock FROM offers)`
	query = strings.ReplaceAll(query, "FROM offers o", "__OFFER_ALIAS_O__")
	query = strings.ReplaceAll(query, "FROM offers", table+" converted_offers")
	return strings.ReplaceAll(query, "__OFFER_ALIAS_O__", table+" o")
}

type contextKey struct{}

func WithCurrency(ctx context.Context, c string) context.Context {
	return context.WithValue(ctx, contextKey{}, c)
}
func Currency(ctx context.Context) string { c, _ := ctx.Value(contextKey{}).(string); return c }
func Query(ctx context.Context, q string) string {
	if Currency(ctx) == "" {
		return q
	}
	return OffersSQL(q)
}

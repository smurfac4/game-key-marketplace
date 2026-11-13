package products

import (
	"context"
	"encoding/json"
	"example.com/game-key-marketplace/api/internal/money"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const searchPredicate = `($5::text='' OR p.name ILIKE $11 ESCAPE '\' OR
 (char_length($5)>=4 AND (p.name % $5 OR p.name %> $5)))`
const searchTier = `CASE WHEN lower(p.name)=lower($5) THEN 0
 WHEN p.name ILIKE $12 ESCAPE '\' THEN 1
 WHEN p.name ILIKE $11 ESCAPE '\' THEN 2 ELSE 3 END`
const relevanceOrder = "search_tier ASC, search_score DESC, word_score DESC, id ASC"

func searchPatterns(search string) (string, string) {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search)
	return "%" + escaped + "%", escaped + "%"
}

func (r Repository) searchRow(ctx context.Context, sql string, args ...any) ([]byte, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL pg_trgm.similarity_threshold = 0.3; SET LOCAL pg_trgm.word_similarity_threshold = 0.5`); err != nil {
		return nil, err
	}
	var raw []byte
	if err = tx.QueryRow(ctx, sql, args...).Scan(&raw); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return raw, nil
}

type Suggestion struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ImageURL string `json:"image_url"`
}
type Suggestions struct {
	Data []Suggestion `json:"data"`
}

func ParseSuggest(raw string) (Query, error) {
	q, err := ParseQuery(raw)
	if err != nil {
		return q, err
	}
	values, _ := url.ParseQuery(raw)
	for key := range values {
		if key != "q" && key != "currency" {
			return q, &ValidationError{key, "unknown query parameter"}
		}
	}
	if q.Search == "" {
		return q, &ValidationError{"q", "is required"}
	}
	return q, nil
}
func (r Repository) Suggest(ctx context.Context, q Query) (Suggestions, error) {
	result := Suggestions{Data: []Suggestion{}}

	if utf8.RuneCountInString(q.Search) < 2 {
		return result, nil
	}
	contains, prefix := searchPatterns(q.Search)
	predicate, tier := searchExpressions("$2", "$3", "$4")
	sql := `WITH candidates AS (
 SELECT p.id,p.name,p.slug,p.image_url,` + tier + ` AS search_tier,
 similarity(p.name,$2) AS search_score,word_similarity($2,p.name) AS word_score
 FROM products p WHERE ` + predicate + `
 AND EXISTS(SELECT 1 FROM offers o WHERE o.product_id=p.id AND o.stock>0 AND o.currency=$1)
 ), top AS (SELECT *,row_number() OVER (ORDER BY ` + relevanceOrder + `) AS position
 FROM candidates ORDER BY ` + relevanceOrder + ` LIMIT 8)
 SELECT coalesce(json_agg(json_build_object('id',id,'name',name,'slug',slug,'image_url',image_url) ORDER BY position),'[]'::json) FROM top`
	raw, err := r.searchRow(ctx, money.Query(ctx, sql), q.Currency, q.Search, contains, prefix)
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal(raw, &result.Data); err != nil {
		return result, fmt.Errorf("decode suggestions: %w", err)
	}
	return result, nil
}

type searchDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
}

func searchExpressions(query, contains, prefix string) (string, string) {
	replace := strings.NewReplacer("$5", query, "$11", contains, "$12", prefix)
	return replace.Replace(searchPredicate), replace.Replace(searchTier)
}

package products

import (
	"context"
	"encoding/json"
	"example.com/game-key-marketplace/api/internal/money"
	"fmt"
	"strings"
)

type Product struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	Platform    string  `json:"platform"`
	Region      string  `json:"region"`
	Genre       string  `json:"genre"`
	ImageURL    string  `json:"image_url"`
	ReleaseDate *string `json:"release_date"`
	Price       string  `json:"price"`
	Currency    string  `json:"currency"`
	FuzzyMatch  bool    `json:"fuzzy_match"`
	Popularity  int64   `json:"popularity"`
}
type Facet struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}
type Facets struct {
	Platform []Facet `json:"platform"`
	Region   []Facet `json:"region"`
	Genre    []Facet `json:"genre"`
}
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}
type Response struct {
	Data       []Product  `json:"data"`
	Pagination Pagination `json:"pagination"`
	Facets     Facets     `json:"facets"`
}
type Repository struct{ DB searchDB }

func (r Repository) List(ctx context.Context, q Query) (Response, error) {
	order, ok := sortSQL[q.Sort]
	if !ok {
		return Response{}, fmt.Errorf("unsupported sort")
	}
	contains, prefix := searchPatterns(q.Search)
	statement := strings.NewReplacer("__ORDER__", order, "__PREDICATE__", searchPredicate, "__TIER__", searchTier).Replace(listSQL)
	statement = money.Query(ctx, statement)
	args := []any{q.Currency, q.Platform, q.Region, q.Genre, q.Search, q.MinPrice, q.MaxPrice, q.PageSize, (q.Page - 1) * q.PageSize, q.Page, contains, prefix}
	var raw []byte
	var err error
	if q.Search != "" {
		raw, err = r.searchRow(ctx, statement, args...)
	} else {
		err = r.DB.QueryRow(ctx, statement, args...).Scan(&raw)
	}

	if err != nil {
		return Response{}, err
	}
	var result Response
	if err = json.Unmarshal(raw, &result); err != nil {
		return Response{}, fmt.Errorf("decode product results: %w", err)
	}
	return result, nil
}

const listSQL = `
WITH priced AS (
 SELECT p.*, o.price, __TIER__ AS search_tier,
 similarity(p.name,$5) AS search_score,word_similarity($5,p.name) AS word_score
 FROM products p JOIN LATERAL (
 SELECT min(coalesce(discount_price,price)) AS price FROM offers
 WHERE product_id=p.id AND stock>0 AND currency=$1 HAVING count(*)>0
 ) o ON true
 WHERE __PREDICATE__
) , eligible AS MATERIALIZED (
 SELECT * FROM priced WHERE ($6::numeric IS NULL OR price >= $6::numeric)
 AND ($7::numeric IS NULL OR price <= $7::numeric)
), matched AS MATERIALIZED (
 SELECT p.*, coalesce((SELECT sum(oi.quantity) FROM order_items oi
 JOIN orders ord ON ord.id=oi.order_id JOIN offers sold ON sold.id=oi.offer_id
 WHERE sold.product_id=p.id AND ord.status='paid'),0) AS popularity
 FROM eligible p WHERE ($2='' OR p.platform=$2) AND ($3='' OR p.region=$3) AND ($4='' OR p.genre=$4)
), page AS (
 SELECT id,name,slug,description,platform,region,genre,image_url,release_date,
 price::text AS price,$1::text AS currency,popularity,(search_tier=3) AS fuzzy_match,
 row_number() OVER (ORDER BY __ORDER__) AS position
 FROM matched
 ORDER BY position LIMIT $8 OFFSET $9
)
SELECT json_build_object(
 'data',coalesce((SELECT json_agg(to_jsonb(page)-'position' ORDER BY position) FROM page),'[]'::json),
 'pagination',json_build_object('page',$10::int,'page_size',$8::int,'total',(SELECT count(*) FROM matched),
 'total_pages',(SELECT (count(*)+$8::int-1)/$8::int FROM matched)),
 'facets',json_build_object(
 'platform',coalesce((SELECT json_agg(f ORDER BY value) FROM (SELECT platform AS value,count(*) AS count FROM eligible WHERE ($3='' OR region=$3) AND ($4='' OR genre=$4) GROUP BY platform) f),'[]'::json),
 'region',coalesce((SELECT json_agg(f ORDER BY value) FROM (SELECT region AS value,count(*) AS count FROM eligible WHERE ($2='' OR platform=$2) AND ($4='' OR genre=$4) GROUP BY region) f),'[]'::json),
 'genre',coalesce((SELECT json_agg(f ORDER BY value) FROM (SELECT genre AS value,count(*) AS count FROM eligible WHERE ($2='' OR platform=$2) AND ($3='' OR region=$3) GROUP BY genre) f),'[]'::json)
 ))`

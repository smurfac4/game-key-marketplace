package products

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("product not found")

type Seller struct {
	ID     int64   `json:"id"`
	Name   string  `json:"name"`
	Rating float64 `json:"rating"`
}
type Offer struct {
	ID             int64   `json:"id"`
	Seller         Seller  `json:"seller"`
	Price          string  `json:"price"`
	DiscountPrice  *string `json:"discount_price"`
	EffectivePrice string  `json:"effective_price"`
	Currency       string  `json:"currency"`
	Stock          int     `json:"stock"`
}
type DetailProduct struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	Platform    string  `json:"platform"`
	Region      string  `json:"region"`
	Genre       string  `json:"genre"`
	ImageURL    string  `json:"image_url"`
	ReleaseDate *string `json:"release_date"`
	Offers      []Offer `json:"offers"`
}
type SimilarProduct struct {
	Suggestion
	Platform string `json:"platform"`
	Region   string `json:"region"`
	Genre    string `json:"genre"`
}
type DetailResponse struct {
	Data            DetailProduct    `json:"data"`
	SimilarProducts []SimilarProduct `json:"similar_products"`
}

func (r Repository) Detail(ctx context.Context, slug string) (DetailResponse, error) {
	var raw []byte
	err := r.DB.QueryRow(ctx, detailSQL, slug).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return DetailResponse{}, ErrNotFound
	}
	if err != nil {
		return DetailResponse{}, err
	}
	var result DetailResponse
	if err = json.Unmarshal(raw, &result); err != nil {
		return result, fmt.Errorf("decode product detail: %w", err)
	}
	return result, nil
}

// One statement keeps the product, offers and recommendations in one snapshot.
// Explicit fields prevent inventory key values from ever entering the response.
const detailSQL = `
WITH target AS (
 SELECT id,name,slug,description,platform,region,genre,image_url,release_date
 FROM products WHERE slug=$1
)
SELECT json_build_object(
 'data',to_jsonb(p)||jsonb_build_object('offers',coalesce((
 SELECT jsonb_agg(jsonb_build_object(
 'id',o.id,'seller',jsonb_build_object('id',s.id,'name',s.name,'rating',s.rating),
 'price',o.price::text,'discount_price',o.discount_price::text,
 'effective_price',coalesce(o.discount_price,o.price)::text,'currency',o.currency,'stock',o.stock
 ) ORDER BY coalesce(o.discount_price,o.price),o.currency,o.id)
 FROM offers o JOIN sellers s ON s.id=o.seller_id WHERE o.product_id=p.id
 ),'[]'::jsonb)),
 'similar_products',coalesce((
 SELECT json_agg(to_jsonb(recommended)-'position' ORDER BY position) FROM (
 SELECT other.id,other.name,other.slug,other.image_url,other.platform,other.region,other.genre,
 row_number() OVER (ORDER BY (other.platform=p.platform) DESC,(other.region=p.region) DESC,other.id) AS position
 FROM products other
 WHERE other.id<>p.id AND other.genre=p.genre
 AND EXISTS(SELECT 1 FROM offers o WHERE o.product_id=other.id AND o.stock>0)
 ORDER BY position LIMIT 8
 ) recommended
 ),'[]'::json)
) FROM target p`

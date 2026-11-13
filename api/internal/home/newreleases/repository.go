package newreleases

import (
	"context"
	"example.com/game-key-marketplace/api/internal/home/block"
	"example.com/game-key-marketplace/api/internal/money"
)

type Repository struct{ DB block.DB }

func (r Repository) List(ctx context.Context, currency string) (block.Response[block.Card], error) {
	return block.Read[block.Card](ctx, r.DB, money.Query(ctx, query), currency)
}

const query = `WITH cards AS (
 SELECT p.id,p.name,p.slug,p.image_url,p.platform,p.region,o.price::text AS price,o.original_price::text AS original_price,o.currency,p.release_date,row_number() OVER (ORDER BY p.release_date DESC,p.id) AS position
 FROM products p JOIN LATERAL (
 SELECT coalesce(discount_price,price) AS price,
 CASE WHEN discount_price IS NOT NULL THEN price END AS original_price,currency FROM offers
 WHERE product_id=p.id AND stock>0 AND currency=$1
 ORDER BY coalesce(discount_price,price),id LIMIT 1) o ON true
 WHERE p.release_date<=CURRENT_DATE
 ORDER BY position LIMIT 12
)
SELECT json_build_object('title',h.title,'data',coalesce((
 SELECT json_agg(to_jsonb(cards)-'position' ORDER BY position) FROM cards),'[]'::json))
FROM homepage_blocks h WHERE h.slug='new-releases' AND h.active`

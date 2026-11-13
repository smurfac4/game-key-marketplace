package bestsellers

import (
	"context"
	"example.com/game-key-marketplace/api/internal/home/block"
	"example.com/game-key-marketplace/api/internal/money"
)

type Repository struct{ DB block.DB }

func (r Repository) List(ctx context.Context, currency string) (block.Response[block.Card], error) {
	return block.Read[block.Card](ctx, r.DB, money.Query(ctx, query), currency)
}

const query = `WITH sales AS (
 SELECT sold.product_id,sum(oi.quantity) AS units_sold FROM order_items oi
 JOIN orders ord ON ord.id=oi.order_id JOIN offers sold ON sold.id=oi.offer_id
 WHERE ord.status='paid' GROUP BY sold.product_id
), cards AS (
 SELECT p.id,p.name,p.slug,p.image_url,p.platform,p.region,o.price::text AS price,o.original_price::text AS original_price,o.currency,p.release_date,sales.units_sold,
 row_number() OVER (ORDER BY sales.units_sold DESC,p.id) AS position
 FROM sales JOIN products p ON p.id=sales.product_id
 JOIN LATERAL (SELECT coalesce(discount_price,price) AS price,
 CASE WHEN discount_price IS NOT NULL THEN price END AS original_price,currency
 FROM offers WHERE product_id=p.id AND stock>0 AND currency=$1
 ORDER BY coalesce(discount_price,price),id LIMIT 1) o ON true
 ORDER BY position LIMIT 12
)
SELECT json_build_object('title',h.title,'data',coalesce((
 SELECT json_agg(to_jsonb(cards)-'position' ORDER BY position) FROM cards),'[]'::json))
FROM homepage_blocks h WHERE h.slug='best-sellers' AND h.active`

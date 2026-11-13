package banners

import (
	"context"
	"example.com/game-key-marketplace/api/internal/home/block"
)

type Banner struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	ImageURL string `json:"image_url"`
	LinkURL  string `json:"link_url"`
}
type Repository struct{ DB block.DB }

func (r Repository) List(ctx context.Context, _ string) (block.Response[Banner], error) {
	return block.Read[Banner](ctx, r.DB, `SELECT json_build_object('title',h.title,'data',coalesce((
 SELECT json_agg(b ORDER BY position,id) FROM (
 SELECT id,title,image_url,link_url,position FROM banners
 WHERE active AND (starts_at IS NULL OR starts_at<=now()) AND (ends_at IS NULL OR ends_at>now())
 ORDER BY position,id LIMIT 8) b),'[]'::json))
 FROM homepage_blocks h WHERE h.slug='banners' AND h.active`)
}

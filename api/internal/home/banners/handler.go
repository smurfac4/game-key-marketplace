package banners

import (
	"context"
	"example.com/game-key-marketplace/api/internal/home/block"
	"net/http"
)

func Handler(repo Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		block.Serve(w, r, func(ctx context.Context, c string) (block.Response[Banner], error) {
			result, err := repo.List(ctx, c)
			for i := range result.Data {
				result.Data[i].Title = block.LocalizedTitle(result.Data[i].Title, r.Header.Get("Accept-Language"))
			}
			return result, err
		})
	}
}

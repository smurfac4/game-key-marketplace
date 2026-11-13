package banners

import (
	"context"
	"example.com/game-key-marketplace/api/internal/home/block"
	"net/http"
	"strings"
)

func Handler(repo Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		block.Serve(w, r, func(ctx context.Context, c string) (block.Response[Banner], error) {
			result, err := repo.List(ctx, c)
			if strings.HasPrefix(r.Header.Get("Accept-Language"), "ru") {
				titles := map[string]string{"Discover your next adventure": "Твоя следующая игровая вселенная", "Explore this week's deals": "Большие игры. Маленькие цены.", "Favorites from our players": "Игры, которые остаются с тобой"}
				for i := range result.Data {
					if v, ok := titles[result.Data[i].Title]; ok {
						result.Data[i].Title = v
					}
				}
			}
			return result, err
		})
	}
}

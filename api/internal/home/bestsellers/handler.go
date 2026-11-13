package bestsellers

import (
	"example.com/game-key-marketplace/api/internal/home/block"
	"net/http"
)

func Handler(repo Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { block.Serve(w, r, repo.List) }
}

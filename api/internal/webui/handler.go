package webui

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

func New(api http.Handler, directory string) (http.Handler, error) {
	files := os.DirFS(directory)
	index, err := fs.Stat(files, "index.html")
	if err != nil || !index.Mode().IsRegular() {
		return nil, errors.New("WEB_DIST must contain a built index.html")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		for _, segment := range strings.Split(name, "/") {
			if strings.HasPrefix(segment, ".") {
				http.NotFound(w, r)
				return
			}
		}
		if name == "" || name == "." {
			name = "index.html"
		}
		info, err := fs.Stat(files, name)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) || path.Ext(name) != "" || name == "assets" || strings.HasPrefix(name, "assets/") || name == "media" || strings.HasPrefix(name, "media/") {
				http.NotFound(w, r)
				return
			}
			name = "index.html"
		} else if !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		if name == "index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		} else if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		http.ServeFileFS(w, r, files, name)
	}), nil
}

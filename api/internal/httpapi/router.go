package httpapi

import (
	"context"
	"example.com/game-key-marketplace/api/internal/account"
	"example.com/game-key-marketplace/api/internal/auth"
	"example.com/game-key-marketplace/api/internal/commerce"
	"example.com/game-key-marketplace/api/internal/home/banners"
	"example.com/game-key-marketplace/api/internal/home/bestsellers"
	"example.com/game-key-marketplace/api/internal/home/block"
	"example.com/game-key-marketplace/api/internal/home/deals"
	"example.com/game-key-marketplace/api/internal/home/newreleases"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Pinger interface {
	Ping(context.Context) error
}

func NewRouter(db Pinger, catalog ProductLister, authentication *Auth) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, accessLog, recoverJSON, requestBudget(12*time.Second), displayCurrency)
	if authentication != nil {
		r.Use(authentication.BrowserSecurity, authentication.ProtectCommerce)
		r.Post("/api/auth/register", authentication.Register)
		r.Post("/api/auth/login", authentication.Login)
		r.Post("/api/auth/logout", authentication.Logout)
		r.With(authentication.Require).Get("/api/me", authentication.Me)
		if pool, ok := db.(*pgxpool.Pool); ok {
			mailer, mailErr := auth.NewSMTP(os.Getenv)
			if mailErr != nil {
				mailer = nil
				slog.Warn("Email sign-in disabled: invalid SMTP configuration")
			}
			identity := NewIdentity(authentication, auth.Passwordless{DB: pool, Service: authentication.Service, Mailer: mailer})
			if identity.InitErr {
				slog.Warn("External sign-in disabled: check OIDC configuration and issuer availability")
			}
			r.Get("/api/auth/methods", identity.Methods)
			r.Post("/api/auth/email/request", identity.Request)
			r.Post("/api/auth/email/verify", identity.Verify)
			r.With(limitRequests(auth.NewWindowLimiter(10, time.Minute))).Get("/api/auth/oidc/start", identity.Start)
			r.Get("/api/auth/oidc/callback", identity.Callback)
			accountHandler := Account{Repo: account.Repository{DB: pool}}
			r.With(authentication.Require).Get("/api/account", accountHandler.Get)
			r.With(authentication.Require).Patch("/api/account", accountHandler.Update)
			r.With(authentication.Require).Get("/api/wishlist", accountHandler.Wishlist)
			r.With(authentication.Require).Put("/api/wishlist/{id}", accountHandler.Favorite)
			r.With(authentication.Require).Delete("/api/wishlist/{id}", accountHandler.Favorite)
			cart := Commerce{Repo: commerce.Repository{DB: pool, Payments: commerce.MockPayment{}}}
			r.With(authentication.Require).Get("/api/cart", cart.Cart)
			r.With(authentication.Require).Post("/api/cart/items", cart.Add)
			r.With(authentication.Require).Patch("/api/cart/items/{id}", cart.Update)
			r.With(authentication.Require).Delete("/api/cart/items/{id}", cart.Remove)
			r.With(limitRequests(auth.NewWindowLimiter(10, time.Minute))).Post("/api/checkout", cart.Checkout)
			r.Get("/api/orders", cart.Orders)
			r.Get("/api/orders/{id}", cart.Order)
		}
	}
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, 404, "not_found", "resource not found", "")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, 405, "method_not_allowed", "method not allowed", "")
	})
	if homeDB, ok := db.(block.DB); ok {
		r.Get("/api/home/banners", banners.Handler(banners.Repository{DB: homeDB}))
		r.Get("/api/home/best-sellers", bestsellers.Handler(bestsellers.Repository{DB: homeDB}))
		r.Get("/api/home/deals", deals.Handler(deals.Repository{DB: homeDB}))
		r.Get("/api/home/new-releases", newreleases.Handler(newreleases.Repository{DB: homeDB}))
	}
	r.Get("/api/products", listProducts(catalog))
	r.Get("/api/products/{slug}", productDetail(catalog))
	r.With(limitRequests(auth.NewWindowLimiter(60, time.Minute))).Get("/api/search/suggest", suggestProducts(catalog))
	r.Get("/api/health/live", func(w http.ResponseWriter, r *http.Request) {
		writeStatus(w, http.StatusOK, `{"status":"ok"}`)
	})
	r.Get("/api/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "unavailable", "database unavailable", "")
			return
		}
		writeStatus(w, http.StatusOK, `{"status":"ready"}`)
	})
	return r
}

func writeStatus(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body + "\n"))
}

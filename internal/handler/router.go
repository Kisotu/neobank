package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/username/banking-app/internal/auth"
	"github.com/username/banking-app/internal/config"
)

func NewRouter(
	cfg *config.Config,
	jwtManager *auth.JWTManager,
	authHandler *AuthHandler,
	accountHandler *AccountHandler,
	transferHandler *TransferHandler,
	transactionHandler *TransactionHandler,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RequestLogger(nil))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(CORS())
	r.Use(SecurityHeaders())
	r.Use(RateLimit(cfg.RateLimiter.GeneralPerMinute, cfg.RateLimiter.LoginPerMinute, cfg.RateLimiter.TransferPerMinute))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Route("/auth", func(ar chi.Router) {
			ar.Post("/register", authHandler.Register)
			ar.Post("/login", authHandler.Login)
			ar.Post("/refresh", authHandler.Refresh)
			ar.Post("/logout", authHandler.Logout)
		})

		api.Group(func(protected chi.Router) {
			protected.Use(auth.RequireAuth(jwtManager))

			protected.Get("/auth/profile", authHandler.GetProfile)
			protected.Put("/auth/profile", authHandler.UpdateProfile)

			protected.Route("/accounts", func(ar chi.Router) {
				ar.Post("/", accountHandler.Create)
				ar.Get("/", accountHandler.List)
				ar.Get("/{id}", accountHandler.GetByID)
				ar.Get("/{id}/balance", accountHandler.GetBalance)
				ar.Get("/{id}/transfers", transferHandler.ListByAccount)
				ar.Get("/{id}/transactions", transactionHandler.ListByAccount)
			})

			protected.Route("/transfers", func(tr chi.Router) {
				tr.Post("/", transferHandler.Create)
				tr.Get("/{id}", transferHandler.GetByID)
			})

			protected.Route("/transactions", func(tr chi.Router) {
				tr.Get("/{id}", transactionHandler.GetByID)
			})
		})
	})

	return r
}

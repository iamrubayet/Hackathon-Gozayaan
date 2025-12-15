package api

import (
	"net/http"

	"rickshaw-app/internal/config"
	"rickshaw-app/internal/handlers"
	"rickshaw-app/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, rdb *redis.Client, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	authHandler := handlers.NewAuthHandler(db, rdb, cfg)
	driverHandler := handlers.NewDriverHandler(db, rdb, cfg)
	rideHandler := handlers.NewRideHandler(db, rdb, cfg)
	adminHandler := handlers.NewAdminHandler(db)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(cfg.JWTSecret))

			r.Post("/auth/logout", authHandler.Logout)
			r.Get("/auth/me", authHandler.Me)

			r.Post("/driver/profile", driverHandler.CreateProfile)
			r.Get("/driver/profile", driverHandler.GetProfile)
			r.Patch("/driver/location", driverHandler.UpdateLocation)
			r.Patch("/driver/availability", driverHandler.UpdateAvailability)

			r.Post("/rides", rideHandler.CreateRide)
			r.Post("/fares", rideHandler.CreateFare)
			r.Get("/rides", rideHandler.GetRides)
			r.Get("/rides/{id}", rideHandler.GetRide)
			r.Post("/rides/{id}/accept", rideHandler.AcceptRide)
			r.Post("/rides/{id}/start", rideHandler.StartRide)
			r.Post("/rides/{id}/complete", rideHandler.CompleteRide)
			r.Post("/rides/{id}/cancel", rideHandler.CancelRide)
			r.Post("/rides/{id}/rate", rideHandler.RateRide)

			r.Get("/drivers/nearby", driverHandler.GetNearbyDrivers)
		})
	})

	r.Route("/admin", func(ar chi.Router) {
		handlers.RegisterAdminRoutes(ar, adminHandler)
	})

	return r
}

package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	infradb "shipment_ms/internal/infrastructure/db"
	"shipment_ms/internal/infrastructure/logger"
	"shipment_ms/internal/infrastructure/temporal"
	httphandler "shipment_ms/internal/interface/http"
	"shipment_ms/internal/repository/postgres"
	"shipment_ms/internal/usecase"
)

const shutdownTimeout = 15 * time.Second

func main() {
	// Load .env if present. Silently ignored in production where env vars
	// are injected by the runtime (Docker, Kubernetes, etc.).
	_ = godotenv.Load()

	log := logger.New()

	dbURL := mustEnv("DATABASE_URL", log)

	if err := infradb.RunMigrations(dbURL, "file://db/migrations"); err != nil {
		log.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Error("pgxpool init failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	temporalClient, err := temporal.NewClient(mustEnv("TEMPORAL_HOST", log), log)
	if err != nil {
		log.Error("temporal client init failed", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()

	repo := postgres.NewShipmentRepository(pool)
	uc := usecase.NewShipmentUseCase(repo, temporalClient, log)
	handler := httphandler.NewShipmentHandler(uc, log)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	srv := &http.Server{
		Addr:         ":8083",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background.
	go func() {
		log.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Block until SIGINT or SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutdown signal received", "signal", sig.String())

	// Give in-flight requests up to shutdownTimeout to complete.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	log.Info("server stopped")
}

func mustEnv(key string, log *slog.Logger) string {
	v := os.Getenv(key)
	if v == "" {
		log.Error("missing required env var", "key", key)
		os.Exit(1)
	}
	return v
}

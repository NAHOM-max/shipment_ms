package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	infradb "shipment_ms/internal/infrastructure/db"
	"shipment_ms/internal/infrastructure/kafka"
	"shipment_ms/internal/infrastructure/logger"
	"shipment_ms/internal/infrastructure/outbox"
	"shipment_ms/internal/infrastructure/temporal"
	httphandler "shipment_ms/internal/interface/http"
	"shipment_ms/internal/repository/postgres"
	"shipment_ms/internal/usecase"
)

const shutdownTimeout = 15 * time.Second

func main() {
	_ = godotenv.Load()

	log := logger.New()

	dbURL := mustEnv("DATABASE_URL", log)

	if err := infradb.RunMigrations(dbURL, "file://db/migrations"); err != nil {
		log.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	kafkaBrokers := strings.Split(mustEnv("KAFKA_BROKERS", log), ",")
	kafkaProducer := kafka.NewKafkaProducer(kafkaBrokers, log)
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			log.Error("kafka producer close failed", "error", err)
		}
	}()

	// ── repositories ─────────────────────────────────────────────────────────
	shipmentRepo := postgres.NewShipmentRepository(pool)
	outboxRepo := postgres.NewOutboxRepository(pool)

	// ── use case ─────────────────────────────────────────────────────────────
	uc := usecase.NewShipmentUseCase(shipmentRepo, temporalClient, log)

	// ── outbox worker ─────────────────────────────────────────────────────────
	pollInterval := 2 * time.Second
	worker := outbox.NewWorker(outboxRepo, kafkaProducer, pollInterval, log)
	go worker.Run(ctx)

	// ── HTTP server ───────────────────────────────────────────────────────────
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

	go func() {
		log.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Block until signal.
	<-ctx.Done()
	log.Info("shutdown signal received")

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

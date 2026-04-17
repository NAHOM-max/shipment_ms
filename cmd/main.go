package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	infradb "shipment_ms/internal/infrastructure/db"
	"shipment_ms/internal/infrastructure/temporal"
	httphandler "shipment_ms/internal/interface/http"
	"shipment_ms/internal/repository/postgres"
	"shipment_ms/internal/usecase"
)

func main() {
	ctx := context.Background()

	dbURL := mustEnv("DATABASE_URL")

	if err := infradb.RunMigrations(dbURL, "file://db/migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("pgxpool: %v", err)
	}
	defer pool.Close()

	temporalClient, err := temporal.NewClient(mustEnv("TEMPORAL_HOST"))
	if err != nil {
		log.Fatalf("temporal: %v", err)
	}
	defer temporalClient.Close()

	repo := postgres.NewShipmentRepository(pool)
	uc := usecase.NewShipmentUseCase(repo, temporalClient)
	handler := httphandler.NewShipmentHandler(uc)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return v
}

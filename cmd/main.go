package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"

	httphandler "shipment_ms/internal/interface/http"
	"shipment_ms/internal/infrastructure/temporal"
	"shipment_ms/internal/repository/postgres"
	"shipment_ms/internal/usecase"
)

func main() {
	db, err := sql.Open("postgres", mustEnv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()

	temporalClient, err := temporal.NewClient(mustEnv("TEMPORAL_HOST"))
	if err != nil {
		log.Fatalf("temporal: %v", err)
	}
	defer temporalClient.Close()

	repo := postgres.NewShipmentRepository(db)
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

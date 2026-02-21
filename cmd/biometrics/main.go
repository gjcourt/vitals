// Package main is the entry point for the biometrics application.
package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	adapthttp "biometrics/internal/adapter/http"
	"biometrics/internal/adapter/memory"
	"biometrics/internal/adapter/postgres"
	"biometrics/internal/app"
	"biometrics/internal/domain"
)

func main() {
	addr := env("ADDR", ":8080")
	webDir := env("WEB_DIR", "web")

	var (
		weightRepo       domain.WeightRepository
		waterRepo        domain.WaterRepository
		chartsWeightRepo domain.WeightRepository
		chartsWaterRepo  domain.WaterRepository
		userRepo         domain.UserRepository
		sessionRepo      domain.SessionRepository
	)

	useMemory := os.Getenv("POSTGRES_URL") == ""

	// DB configuration
	if useMemory {
		log.Println("Using in-memory database")
		mem := memory.New()
		weightRepo = mem
		waterRepo = mem
		chartsWeightRepo = mem
		chartsWaterRepo = mem
		userRepo = mem
		sessionRepo = mem.NewSessionRepo()
	} else {
		log.Println("Using PostgreSQL database")
		connStr := os.Getenv("POSTGRES_URL")

		// Map custom env vars to lib/pq standard vars if provided
		if v := os.Getenv("POSTGRES_USER"); v != "" {
			_ = os.Setenv("PGUSER", v)
		}
		if v := os.Getenv("POSTGRES_PASSWORD"); v != "" {
			_ = os.Setenv("PGPASSWORD", v)
		}

		db, err := postgres.Open(connStr)
		if err != nil {
			log.Fatalf("db open: %v", err)
		}
		defer func() { _ = db.Close() }()

		weightRepo = db
		waterRepo = db
		chartsWeightRepo = db
		chartsWaterRepo = db
		userRepo = db
		sessionRepo = postgres.NewSessionRepo(db)
	}

	weightSvc := app.NewWeightService(weightRepo)
	waterSvc := app.NewWaterService(waterRepo)
	chartsSvc := app.NewChartsService(chartsWeightRepo, chartsWaterRepo)
	authSvc := app.NewAuthService(userRepo, sessionRepo)

	srv := adapthttp.New(weightSvc, waterSvc, chartsSvc, authSvc, webDir)
	h := srv.Handler()

	log.Printf("listening on %s", addr)
	//nolint:gosec // ignoring timeout constraint for simple server
	if err := http.ListenAndServe(addr, h); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

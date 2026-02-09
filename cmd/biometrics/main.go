package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	adapthttp "biometrics/internal/adapter/http"
	"biometrics/internal/adapter/postgres"
	"biometrics/internal/app"
)

func main() {
	addr := env("ADDR", ":8080")
	webDir := env("WEB_DIR", "web")

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := postgres.Open(connStr)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer func() { _ = db.Close() }()

	weightSvc := app.NewWeightService(db)
	waterSvc := app.NewWaterService(db)
	chartsSvc := app.NewChartsService(db, db)

	h := adapthttp.New(weightSvc, waterSvc, chartsSvc, webDir).Handler()
	log.Printf("listening on %s", addr)
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

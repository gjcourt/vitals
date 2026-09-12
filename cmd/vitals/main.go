// Package main is the entry point for the vitals application.
package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	adapthttp "vitals/internal/adapter/http"
	"vitals/internal/adapter/memory"
	"vitals/internal/adapter/sqlite"
	"vitals/internal/app"
	"vitals/internal/domain"
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

	sqlitePath := os.Getenv("SQLITE_PATH")

	// DB configuration. SQLITE_PATH selects durable storage; unset means
	// in-memory, which is the dev default and loses everything on restart.
	if sqlitePath == "" {
		log.Println("Using in-memory database (set SQLITE_PATH for durable storage)")
		mem := memory.New()
		weightRepo = mem
		waterRepo = mem
		chartsWeightRepo = mem
		chartsWaterRepo = mem
		userRepo = mem
		sessionRepo = mem.NewSessionRepo()
	} else {
		log.Printf("Using SQLite database at %s", sqlitePath)

		db, err := sqlite.Open(sqlitePath)
		if err != nil {
			log.Fatalf("db open: %v", err)
		}
		defer func() { _ = db.Close() }()

		weightRepo = db
		waterRepo = db
		chartsWeightRepo = db
		chartsWaterRepo = db
		userRepo = db
		sessionRepo = sqlite.NewSessionRepo(db)
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

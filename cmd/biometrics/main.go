package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"biometrics/internal/db"
	"biometrics/internal/server"
)

func main() {
	addr := env("ADDR", ":8080")
	webDir := env("WEB_DIR", "web")

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is required")
	}

	d, err := db.Open(connStr)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer func() { _ = d.Close() }()

	h := server.New(d, webDir).Handler()
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

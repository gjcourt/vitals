// Package main is the entry point for the vitals application.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	adapthttp "vitals/internal/adapters/http"
	"vitals/internal/adapters/memory"
	"vitals/internal/adapters/postgres"
	"vitals/internal/app"
	outbound "vitals/internal/ports/outbound"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	addr := env("ADDR", ":8080")
	webDir := env("WEB_DIR", "web")

	var (
		weightRepo       outbound.WeightRepository
		waterRepo        outbound.WaterRepository
		chartsWeightRepo outbound.WeightRepository
		chartsWaterRepo  outbound.WaterRepository
		userRepo         outbound.UserRepository
		sessionRepo      outbound.SessionRepository
	)

	useMemory := os.Getenv("POSTGRES_URL") == ""

	// DB configuration
	if useMemory {
		logger.Info("using in-memory database")
		mem := memory.New()
		weightRepo = mem
		waterRepo = mem
		chartsWeightRepo = mem
		chartsWaterRepo = mem
		userRepo = mem
		sessionRepo = mem.NewSessionRepo()
	} else {
		logger.Info("using PostgreSQL database")
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
			logger.Error("db open failed", slog.Any("err", err))
			os.Exit(1)
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

	opts := adapthttp.Options{
		Logger:       logger,
		OIDC:         buildOIDCConfig(logger),
		CookieSecure: cookieSecureFromEnv(),
	}

	srv := adapthttp.New(weightSvc, waterSvc, chartsSvc, authSvc, webDir, opts)
	h := srv.Handler()

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("listening", slog.String("addr", addr))
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server exited with error", slog.Any("err", err))
		os.Exit(1)
	}
}

// buildOIDCConfig discovers the OIDC provider when SSO_ISSUER_URL is set
// and constructs the oauth2.Config used by the HTTP adapter.
func buildOIDCConfig(logger *slog.Logger) adapthttp.OIDCConfig {
	issuer := os.Getenv("SSO_ISSUER_URL")
	if issuer == "" {
		return adapthttp.OIDCConfig{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		logger.Error("failed to initialize OIDC provider", slog.Any("err", err))
		return adapthttp.OIDCConfig{}
	}

	cfg := adapthttp.OIDCConfig{
		Provider: provider,
		OAuth2Config: oauth2.Config{
			ClientID:     os.Getenv("SSO_CLIENT_ID"),
			ClientSecret: os.Getenv("SSO_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("SSO_REDIRECT_URL"),
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		Enabled: true,
	}
	logger.Info("SSO (OIDC) enabled")
	return cfg
}

// cookieSecureFromEnv returns true when the deployment runs behind a
// TLS-terminating proxy and we want auth cookies marked Secure.
func cookieSecureFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SECURE")))
	switch v {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

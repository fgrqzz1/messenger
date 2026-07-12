package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	httphandler "messenger/internal/handler/http"
	wshandler "messenger/internal/handler/ws"
	"messenger/internal/repository/postgres"
	"messenger/internal/service"
	"messenger/pkg/jwt"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := loadConfig()
	if err != nil {
		logger.Error("invalid config", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()

	db, err := postgres.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect postgres", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	userRepo := postgres.NewUserRepository(db)
	chatRepo := postgres.NewChatRepository(db)
	messageRepo := postgres.NewMessageRepository(db)
	memberRepo := postgres.NewMemberRepository(db)

	jwtManager := jwt.NewManager(jwt.Config{
		AccessSecret:  cfg.JWTAccessSecret,
		RefreshSecret: cfg.JWTRefreshSecret,
		AccessTTL:     cfg.AccessTokenTTL,
		RefreshTTL:    cfg.RefreshTokenTTL,
	})

	svc := service.New(userRepo, chatRepo, messageRepo, memberRepo, jwtManager)

	hub := wshandler.NewHub()
	wsHandler := wshandler.NewHandler(svc, jwtManager, hub, wshandler.Config{
		AllowedOrigins: parseAllowedOrigins(os.Getenv("WS_ALLOWED_ORIGINS")),
	}, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(pingCtx); err != nil {
			logger.Error("health check failed", "err", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("unavailable"))
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("GET /ws", wsHandler)
	mux.Handle("/", httphandler.NewMux(svc, jwtManager))

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.Error("server failed", "err", err)
		os.Exit(1)
	case sig := <-stop:
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown", "err", err)
	}

	hub.Shutdown(shutdownCtx)
	logger.Info("server stopped")
}

type config struct {
	DatabaseURL      string
	JWTAccessSecret  string
	JWTRefreshSecret string
	HTTPAddr         string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
}

func loadConfig() (config, error) {
	cfg := config{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		JWTAccessSecret:  os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
		HTTPAddr:         envOrDefault("HTTP_ADDR", ":8080"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}
	if cfg.JWTAccessSecret == "" {
		return cfg, errors.New("JWT_ACCESS_SECRET is required")
	}
	if cfg.JWTRefreshSecret == "" {
		return cfg, errors.New("JWT_REFRESH_SECRET is required")
	}

	accessTTL, err := time.ParseDuration(envOrDefault("ACCESS_TOKEN_TTL", "15m"))
	if err != nil {
		return cfg, fmt.Errorf("ACCESS_TOKEN_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(envOrDefault("REFRESH_TOKEN_TTL", "168h"))
	if err != nil {
		return cfg, fmt.Errorf("REFRESH_TOKEN_TTL: %w", err)
	}

	cfg.AccessTokenTTL = accessTTL
	cfg.RefreshTokenTTL = refreshTTL

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseAllowedOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

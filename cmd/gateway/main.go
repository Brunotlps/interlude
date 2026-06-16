package main

import (
	"context"
	"errors"
	"fmt"
	"interlude/internal/config"
	"interlude/internal/middleware"
	"interlude/internal/router"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "interlude/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	rt, err := router.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create router: %v", err)
	}

	validKeys := make(map[string]struct{}, len(cfg.Auth.APIKeys))
	for _, k := range cfg.Auth.APIKeys {
		validKeys[k] = struct{}{}
	}

	handler := middleware.Logging(
		middleware.RateLimit(cfg.RateLimit)(
			middleware.Auth(validKeys)(rt),
		),
	)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: handler,
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	metricsSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.MetricsPort),
		Handler: mux,
	}

	errCh := make(chan error, 2)

	go func() {
		slog.Info("metrics server started", "addr", metricsSrv.Addr)
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("metrics server: %w", err)
		}
	}()

	go func() {
		slog.Info("gateway started", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("gateway server: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		// normal shutdown via SIGTERM or interrupt
	case err := <-errCh:
		slog.Error("server error, initiating shutdown", "err", err)
	}
	stop() // release signal resources

	slog.Info("shutdown signal received, draining connections")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("gateway shutdown error", "err", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
			slog.Error("metrics server shutdown error", "err", err)
		}
	}()

	wg.Wait()

	slog.Info("shutdown complete")
}

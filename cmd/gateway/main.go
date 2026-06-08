package main

import (
	"context"
	"fmt"
	"interlude/internal/config"
	"interlude/internal/middleware"
	"interlude/internal/router"
	"log"
	"log/slog"
	"net/http"
	"os"

	_ "interlude/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	// Configures the logger
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Load configuration from YAML file
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rt, err := router.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create router: %v", err)
	}

	// Set the server address
	addr := fmt.Sprintf(":%d", cfg.Server.Port)

	// wiring
	validKeys := make(map[string]struct{}, len(cfg.Auth.APIKeys))
	for _, k := range cfg.Auth.APIKeys {
		validKeys[k] = struct{}{}
	}

	routePrefixes := make([]string, len(cfg.Routes))
	for i, r := range cfg.Routes {
		routePrefixes[i] = r.Prefix
	}

	handler := middleware.Logging(
		routePrefixes,
		middleware.RateLimit(cfg.RateLimit)(
			middleware.Auth(validKeys)(rt),
		),
	)

	slog.Info("gateway started", "addr", addr)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		metricsAddr := fmt.Sprintf(":%d", cfg.Server.MetricsPort)
		slog.Info("metrics server started", "addr", metricsAddr)
		if err := http.ListenAndServe(metricsAddr, mux); err != nil {
			log.Fatalf("metrics server failed: %v", err)
		}
	}()

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}

}

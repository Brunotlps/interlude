package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Each Route maps a URL Prefix to a Backends list destination
// Ex:
//
//	prefix: "/api/users"
//	backends:
//		- "https://example1.com"
//		- "https://example2.com"

type HealthCheckConfig struct {
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Path     string        `yaml:"path"`
}

type Route struct {
	Prefix   string   `yaml:"prefix"` // These yaml tags tell the parser which field in the file the struct field refers to
	Backends []string `yaml:"backends"`
}

// HTTP server configs
type ServerConfig struct {
	Port        int `yaml:"port"`
	MetricsPort int `yaml:"metrics_port"`
}

type AuthConfig struct {
	APIKeys []string `yaml:"api_keys"`
}

type RateLimitConfig struct {
	RequestsPerSecond float64 `yaml:"requests_per_second"`
	Burst             float64 `yaml:"burst"`
}

type CircuitBreakerConfig struct {
	MaxFailures     int32         `yaml:"max_failures"`
	RecoveryTimeout time.Duration `yaml:"recovery_timeout"`
}

type RetryConfig struct {
	MaxAttempts int `yaml:"max_attempts"`
}

// Root settings structure (exactly mirrors the structure of config.yaml)
type Config struct {
	Server         ServerConfig         `yaml:"server"`
	Auth           AuthConfig           `yaml:"auth"`
	RateLimit      RateLimitConfig      `yaml:"rate_limit"`
	HealthCheck    HealthCheckConfig    `yaml:"health_check"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	Retry          RetryConfig          `yaml:"retry"`
	Routes         []Route              `yaml:"routes"`
}

func Load(path string) (*Config, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Initializes default Config
	cfg := &Config{
		Server: ServerConfig{Port: 8080, MetricsPort: 9091},
		HealthCheck: HealthCheckConfig{
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Path:     "/health",
		},
	}

	// Convert yaml bytes to go struct
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}

	if len(cfg.Routes) == 0 {
		return nil, fmt.Errorf("invalid config: no routes defined")
	}

	return cfg, nil

}

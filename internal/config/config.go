package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Each Route maps a URL Prefix to a Backend destination
// Ex:
//
//	prefix: "/api/users"
//	backend: "https://example.com"
type Route struct {
	Prefix  string `yaml:"prefix"` // These yaml tags tell the parser which field in the file the struct field refers to
	Backend string `yaml:"backend"`
}

// HTTP server configs
type ServerConfig struct {
	Port int `yaml:"port"`
}

// Root settings structure (exactly mirrors the structure of config.yaml)
type Config struct {
	Server ServerConfig `yaml:"server"`
	Routes []Route      `yaml:"routes"`
}

func Load(path string) (*Config, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Initializes default Config
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
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

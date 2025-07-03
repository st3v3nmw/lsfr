package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

const configPath = "lsfr.yaml"

type Config struct {
	Challenge string `yaml:"challenge"`
	Stage     int    `yaml:"stage"`
}

func Load() (*Config, error) {
	// Parse config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("not in a challenge directory\nRun this command from a directory created with 'lsfr new <challenge>'")
	}

	bytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validation
	if cfg.Challenge == "" {
		return nil, fmt.Errorf("challenge name cannot be empty")
	}

	if cfg.Stage <= 0 {
		cfg.Stage = 1
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(configPath, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

const configPath = "lsfr.yaml"

type Stages struct {
	Current   string   `yaml:"current"`
	Completed []string `yaml:"completed"`
}

type Config struct {
	Challenge string `yaml:"challenge"`
	Stages    Stages `yaml:"stages"`
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

	if cfg.Stages.Completed == nil {
		cfg.Stages.Completed = []string{}
	}

	// Validation
	if cfg.Challenge == "" {
		return nil, fmt.Errorf("challenge name cannot be empty")
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	return SaveTo(cfg, configPath)
}

func SaveTo(cfg *Config, path string) error {
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

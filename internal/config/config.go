package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	defaultBaseURL = "http://localhost:8000"
	defaultTimeout = 10
)

type Config struct {
	BaseURL        string   `yaml:"base_url"`
	TimeoutSeconds int      `yaml:"timeout_seconds"`
	Scopes         []string `yaml:"scopes"`
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bookmark-cli", "config.yaml"), nil
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		BaseURL:        defaultBaseURL,
		TimeoutSeconds: defaultTimeout,
	}

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = defaultTimeout
	}
	return cfg, nil
}

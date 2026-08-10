package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	defaultBaseURL = "http://localhost:8000"
	defaultTimeout = 10
	DefaultProfile = "default"
)

type Profile struct {
	BaseURL        string   `toml:"base_url"`
	TimeoutSeconds int      `toml:"timeout_seconds"`
	Scopes         []string `toml:"scopes"`
}

type Config struct {
	Profiles map[string]Profile `toml:"profile"`
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bookmark-cli", "config.toml"), nil
}

func Load(path string) (*Config, error) {
	cfg := &Config{}

	if path == "" {
		cfg.Profiles = map[string]Profile{DefaultProfile: defaultProfile()}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg.Profiles = map[string]Profile{DefaultProfile: defaultProfile()}
			return cfg, nil
		}
		return nil, err
	}

	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, err
	}
	if len(cfg.Profiles) == 0 {
		cfg.Profiles = map[string]Profile{DefaultProfile: defaultProfile()}
	}
	for name, profile := range cfg.Profiles {
		cfg.Profiles[name] = normalizeProfile(profile)
	}
	return cfg, nil
}

func (c *Config) Profile(name string) (Profile, error) {
	profileName := strings.TrimSpace(name)
	if profileName == "" {
		profileName = DefaultProfile
	}

	profile, ok := c.Profiles[profileName]
	if !ok {
		return Profile{}, fmt.Errorf("profile %q not found", profileName)
	}
	return normalizeProfile(profile), nil
}

func defaultProfile() Profile {
	return Profile{
		BaseURL:        defaultBaseURL,
		TimeoutSeconds: defaultTimeout,
	}
}

func normalizeProfile(profile Profile) Profile {
	if profile.BaseURL == "" {
		profile.BaseURL = defaultBaseURL
	}
	if profile.TimeoutSeconds <= 0 {
		profile.TimeoutSeconds = defaultTimeout
	}
	return profile
}

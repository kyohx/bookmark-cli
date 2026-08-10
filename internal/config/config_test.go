package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadReturnsDefaultProfileWhenFileDoesNotExist(t *testing.T) {
	t.Parallel()

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	profile, err := cfg.Profile(DefaultProfile)
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}
	if profile.BaseURL != defaultBaseURL {
		t.Fatalf("BaseURL = %q, want %q", profile.BaseURL, defaultBaseURL)
	}
	if profile.TimeoutSeconds != defaultTimeout {
		t.Fatalf("TimeoutSeconds = %d, want %d", profile.TimeoutSeconds, defaultTimeout)
	}
	if len(profile.Scopes) != 0 {
		t.Fatalf("Scopes = %v, want empty", profile.Scopes)
	}
}

func TestLoadParsesTOMLProfiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	data := `
[profile.default]
base_url = "https://api.example.com"
timeout_seconds = 30
scopes = ["read", "write"]

[profile.work]
base_url = "https://work.example.com"
scopes = ["admin"]
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	def, err := cfg.Profile(DefaultProfile)
	if err != nil {
		t.Fatalf("Profile(default) error = %v", err)
	}
	if def.BaseURL != "https://api.example.com" {
		t.Fatalf("default BaseURL = %q", def.BaseURL)
	}
	if def.TimeoutSeconds != 30 {
		t.Fatalf("default TimeoutSeconds = %d, want 30", def.TimeoutSeconds)
	}
	if !reflect.DeepEqual(def.Scopes, []string{"read", "write"}) {
		t.Fatalf("default Scopes = %v", def.Scopes)
	}

	work, err := cfg.Profile("work")
	if err != nil {
		t.Fatalf("Profile(work) error = %v", err)
	}
	if work.BaseURL != "https://work.example.com" {
		t.Fatalf("work BaseURL = %q", work.BaseURL)
	}
	if work.TimeoutSeconds != defaultTimeout {
		t.Fatalf("work TimeoutSeconds = %d, want %d", work.TimeoutSeconds, defaultTimeout)
	}
	if !reflect.DeepEqual(work.Scopes, []string{"admin"}) {
		t.Fatalf("work Scopes = %v", work.Scopes)
	}
}

func TestProfileReturnsErrorForUnknownProfile(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profiles: map[string]Profile{
			DefaultProfile: defaultProfile(),
		},
	}

	if _, err := cfg.Profile("missing"); err == nil {
		t.Fatal("Profile(missing) error = nil, want error")
	}
}

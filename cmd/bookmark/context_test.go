package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveProfileAppliesSelectedProfileAndOverrides(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
[profile.default]
base_url = "https://default.example.com"
timeout_seconds = 10
scopes = ["read"]

[profile.work]
base_url = "https://work.example.com"
timeout_seconds = 20
scopes = ["admin"]
`)

	profile, err := resolveProfile(&rootOptions{
		ConfigPath: path,
		Profile:    "work",
		BaseURL:    "https://override.example.com",
		TimeoutSec: 45,
		TimeoutSet: true,
	})
	if err != nil {
		t.Fatalf("resolveProfile() error = %v", err)
	}

	if profile.Name != "work" {
		t.Fatalf("Name = %q, want %q", profile.Name, "work")
	}
	if profile.BaseURL != "https://override.example.com" {
		t.Fatalf("BaseURL = %q, want %q", profile.BaseURL, "https://override.example.com")
	}
	if profile.TimeoutSeconds != 45 {
		t.Fatalf("TimeoutSeconds = %d, want %d", profile.TimeoutSeconds, 45)
	}
	if len(profile.Scopes) != 1 || profile.Scopes[0] != "admin" {
		t.Fatalf("Scopes = %v, want [admin]", profile.Scopes)
	}
}

func TestShowProfileUsesParsedPersistentFlags(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
[profile.default]
base_url = "https://default.example.com"

[profile.work]
base_url = "https://work.example.com"
timeout_seconds = 25
scopes = ["read", "write"]
`)

	cmd := newRootCmd()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--config", path,
		"--profile", "work",
		"--show-profile",
		"version",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != version {
		t.Fatalf("stdout = %q, want %q", got, version)
	}
	errText := stderr.String()
	for _, want := range []string{
		`"name": "work"`,
		`"base_url": "https://work.example.com"`,
		`"timeout_seconds": 25`,
		`"scopes": [`,
	} {
		if !strings.Contains(errText, want) {
			t.Fatalf("stderr = %q, want substring %q", errText, want)
		}
	}
}

func writeConfigFile(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(body)), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

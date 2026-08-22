package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != version {
		t.Fatalf("version output = %q, want %q", got, version)
	}
}

func TestAPIVersionCommandIsRegistered(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()

	found, _, err := cmd.Find([]string{"api-version"})
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found == nil {
		t.Fatal("api-version command was not found")
	}
	if got := found.Name(); got != "api-version" {
		t.Fatalf("command name = %q, want %q", got, "api-version")
	}
}

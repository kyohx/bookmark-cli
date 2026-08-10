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

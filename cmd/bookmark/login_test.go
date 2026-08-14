package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadUsernamePromptsWithoutNewline(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	username, err := readUsername(strings.NewReader("alice\n"), &stdout)
	if err != nil {
		t.Fatalf("readUsername() error = %v", err)
	}

	if username != "alice" {
		t.Fatalf("username = %q, want %q", username, "alice")
	}
	if got := stdout.String(); got != "Username: " {
		t.Fatalf("prompt = %q, want %q", got, "Username: ")
	}
}

func TestReadUsernameAcceptsEOFWithoutTrailingNewline(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	username, err := readUsername(strings.NewReader("alice"), &stdout)
	if err != nil {
		t.Fatalf("readUsername() error = %v", err)
	}

	if username != "alice" {
		t.Fatalf("username = %q, want %q", username, "alice")
	}
}

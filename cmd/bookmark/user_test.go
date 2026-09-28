package main

import (
	"io"
	"strings"
	"testing"
)

func TestUserCommandsRegistered(t *testing.T) {
	for _, name := range []string{"add", "get", "list", "update"} {
		cmd, _, err := newRootCmd().Find([]string{"user", name})
		if err != nil || cmd.Name() != name {
			t.Fatalf("user %s not registered: %v", name, err)
		}
	}
}

func TestUserCommandsRejectIncompleteOptionsBeforeAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"add without authority", []string{"user", "add", "test_user"}, "--authority is required"},
		{"add invalid authority", []string{"user", "add", "test_user", "--authority", "3"}, "authority must be"},
		{"update without fields", []string{"user", "update", "test_user"}, "specify at least one field"},
		{"update conflicting password input", []string{"user", "update", "test_user", "--password", "--password-stdin"}, "cannot be used together"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

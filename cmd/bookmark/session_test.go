package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bookmark-cli/internal/session"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

func TestSessionCommands(t *testing.T) {
	keyring.MockInit()
	if err := session.NewKeyringStore("work").Save(context.Background(), &oauth2.Token{AccessToken: "admin-token", TokenType: "Bearer"}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, method, path, response, want, wantError string
		args                                          []string
		status                                        int
	}{
		{name: "list", method: "GET", path: "/users/test_user/sessions", args: []string{"session", "list", "test_user"}, status: 200, response: `{"sessions":[{"id":"session1","created_at":"2026-10-01 12:34:56","last_used_at":"2026-10-02 12:34:56","expires_at":"2026-10-08 12:34:56","revoked":false,"user_agent":null}]}`, want: `"user_agent": null`},
		{name: "empty list", method: "GET", path: "/users/test_user/sessions", args: []string{"session", "list", "test_user"}, status: 200, response: `{"sessions":[]}`, want: "[]\n"},
		{name: "revoke", method: "DELETE", path: "/users/test_user/sessions/session1", args: []string{"session", "revoke", "test_user", "session1"}, status: 204, want: "Revoked.\n"},
		{name: "list forbidden", method: "GET", path: "/users/test_user/sessions", args: []string{"session", "list", "test_user"}, status: 403, response: `{"detail":"admin only"}`, wantError: `status=403, body={"detail":"admin only"}`},
		{name: "revoke not found", method: "DELETE", path: "/users/test_user/sessions/missing", args: []string{"session", "revoke", "test_user", "missing"}, status: 404, response: `{"detail":"session not found"}`, wantError: `status=404, body={"detail":"session not found"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != tc.method || r.URL.Path != tc.path || r.URL.RawQuery != "" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer admin-token" {
					t.Error("selected profile's token not used")
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.response)
			}))
			defer server.Close()
			path := writeConfigFile(t, "[profile.work]\nbase_url = \""+server.URL+"\"\n")
			cmd := newRootCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(append([]string{"--config", path, "--profile", "work"}, tc.args...))
			err := cmd.Execute()
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				if out.Len() != 0 {
					t.Fatalf("unexpected success output: %s", out.String())
				}
			} else if err != nil || !strings.Contains(out.String(), tc.want) {
				t.Fatalf("output = %q, error = %v, want %q", out.String(), err, tc.want)
			}
			if requests != 1 {
				t.Errorf("requests = %d, want 1", requests)
			}
		})
	}
}

func TestSessionCommandsRequireArguments(t *testing.T) {
	for _, args := range [][]string{
		{"session", "list"}, {"session", "list", "user", "extra"},
		{"session", "revoke"}, {"session", "revoke", "user"}, {"session", "revoke", "user", "id", "extra"},
	} {
		cmd := newRootCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "arg(s)") {
			t.Errorf("%v: expected argument error, got %v", args, err)
		}
	}
}

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestListUserSessions(t *testing.T) {
	raw, err := os.ReadFile("testdata/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Components struct {
			Schemas map[string]struct {
				Required []string `json:"required"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	for _, response := range []string{
		`{"sessions":[{"id":"session1","created_at":"2026-10-01 12:34:56","last_used_at":"2026-10-02 12:34:56","expires_at":"2026-10-08 12:34:56","revoked":false,"user_agent":null},{"id":"session2","created_at":"2026-10-02 12:34:56","last_used_at":"2026-10-03 12:34:56","expires_at":"2026-10-09 12:34:56","revoked":true,"user_agent":"bookmark-cli"}]}`,
		`{"sessions":[]}`,
	} {
		t.Run(response, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/users/test_user/sessions" || r.URL.RawQuery != "" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer token" {
					t.Error("missing authorization")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, response)
			}))
			defer server.Close()
			client := &Client{baseURL: server.URL, httpClient: server.Client(), tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), authService: &fakeAuthRefresher{}}
			sessions, err := client.ListUserSessions(context.Background(), "test_user")
			if err != nil {
				t.Fatal(err)
			}
			actual, err := json.Marshal(sessions)
			if err != nil {
				t.Fatal(err)
			}
			var expected struct {
				Sessions []map[string]any `json:"sessions"`
			}
			if err := json.Unmarshal([]byte(response), &expected); err != nil {
				t.Fatal(err)
			}
			var got []map[string]any
			if err := json.Unmarshal(actual, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, expected.Sessions) {
				t.Fatalf("sessions = %s, want %s", actual, response)
			}
			for _, session := range got {
				for _, field := range spec.Components.Schemas["SessionDetail"].Required {
					if _, ok := session[field]; !ok {
						t.Errorf("missing required OpenAPI field %q", field)
					}
				}
			}
		})
	}
}

func TestRevokeUserSession(t *testing.T) {
	for _, id := range []string{"session1", "session /?#+%あ", strings.Repeat("あ", 128)} {
		t.Run(id, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || r.URL.Path != "/users/test_user/sessions/"+id || r.URL.RawQuery != "" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if id == "session /?#+%あ" && r.URL.EscapedPath() != "/users/test_user/sessions/session%20%2F%3F%23+%25%E3%81%82" {
					t.Errorf("session ID not escaped: %s", r.URL.EscapedPath())
				}
				if r.Header.Get("Authorization") != "Bearer token" {
					t.Error("missing authorization")
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 {
					t.Errorf("unexpected request body: %q, %v", body, err)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			client := &Client{baseURL: server.URL, httpClient: server.Client(), tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), authService: &fakeAuthRefresher{}}
			if err := client.RevokeUserSession(context.Background(), "test_user", id); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSessionValidationBeforeRequest(t *testing.T) {
	client := &Client{}
	for _, name := range []string{"", "bad/name", "bad-name", "日本語", strings.Repeat("a", 33)} {
		if _, err := client.ListUserSessions(context.Background(), name); err == nil {
			t.Errorf("ListUserSessions(%q) accepted invalid name", name)
		}
		if err := client.RevokeUserSession(context.Background(), name, "session1"); err == nil {
			t.Errorf("RevokeUserSession(%q) accepted invalid name", name)
		}
	}
	for _, id := range []string{"", strings.Repeat("a", 129), strings.Repeat("あ", 129)} {
		if err := client.RevokeUserSession(context.Background(), "test_user", id); err == nil {
			t.Errorf("accepted invalid session ID %q", id)
		}
	}
}

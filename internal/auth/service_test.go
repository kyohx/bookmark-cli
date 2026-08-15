package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type memStore struct {
	mu    sync.Mutex
	token *oauth2.Token
}

func (m *memStore) Save(_ context.Context, token *oauth2.Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *token
	m.token = &copied
	return nil
}

func (m *memStore) Load(_ context.Context) (*oauth2.Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.token == nil {
		return nil, io.EOF
	}
	copied := *m.token
	return &copied, nil
}

func (m *memStore) Delete(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = nil
	return nil
}

func TestLoginUsesTokenEndpointAndScope(t *testing.T) {
	var gotScope string
	var hitToken bool
	var hitLogin bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			hitLogin = true
			w.WriteHeader(http.StatusNotFound)
		case "/token":
			hitToken = true
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm() error = %v", err)
			}
			gotScope = r.Form.Get("scope")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"` + makeJWTWithExp(time.Now().Add(5*time.Minute)) + `","refresh_token":"r1","token_type":"bearer"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	store := &memStore{}
	svc := NewService(server.URL, 2*time.Second, store)
	err := svc.Login(context.Background(), "testuser", []byte("pass"), []string{"read", "write", "read", "  "})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if gotScope != "read write" {
		t.Fatalf("scope = %q, want %q", gotScope, "read write")
	}
	if !hitToken {
		t.Fatalf("expected /token endpoint to be used")
	}
	if hitLogin {
		t.Fatalf("did not expect /login endpoint to be used")
	}

	tok, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if tok.AccessToken == "" || tok.RefreshToken != "r1" {
		t.Fatalf("unexpected token saved: %#v", tok)
	}
	if tok.Expiry.IsZero() {
		t.Fatalf("expected parsed expiry from JWT")
	}
}

func TestRefreshIfPossible(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/refresh" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"a2","refresh_token":"r2","token_type":"bearer"}`))
	}))
	defer server.Close()

	store := &memStore{
		token: &oauth2.Token{
			AccessToken:  "a1",
			RefreshToken: "r1",
			TokenType:    "bearer",
		},
	}
	svc := NewService(server.URL, 2*time.Second, store)
	if err := svc.RefreshIfPossible(context.Background()); err != nil {
		t.Fatalf("RefreshIfPossible() error = %v", err)
	}

	tok, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if tok.AccessToken != "a2" || tok.RefreshToken != "r2" {
		t.Fatalf("unexpected token after refresh: %#v", tok)
	}
}

func TestLoginIncludesJSONErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"invalid credentials"}`))
	}))
	defer server.Close()

	svc := NewService(server.URL, 2*time.Second, &memStore{})
	err := svc.Login(context.Background(), "testuser", []byte("wrong"), nil)
	if err == nil {
		t.Fatal("Login() error should not be nil")
	}
	if !strings.Contains(err.Error(), `status=401, body={"detail":"invalid credentials"}`) {
		t.Fatalf("error should contain compact JSON body, got %v", err)
	}
}

func TestRefreshIncludesJSONErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"refresh token expired"}`))
	}))
	defer server.Close()

	store := &memStore{
		token: &oauth2.Token{
			AccessToken:  "a1",
			RefreshToken: "r1",
			TokenType:    "bearer",
		},
	}
	svc := NewService(server.URL, 2*time.Second, store)
	err := svc.RefreshIfPossible(context.Background())
	if err == nil {
		t.Fatal("RefreshIfPossible() error should not be nil")
	}
	if !strings.Contains(err.Error(), `status=401, body={"detail":"refresh token expired"}`) {
		t.Fatalf("error should contain compact JSON body, got %v", err)
	}
}

func makeJWTWithExp(exp time.Time) string {
	header := map[string]any{"alg": "none", "typ": "JWT"}
	payload := map[string]any{"exp": exp.Unix()}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)
	return base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(pb) + "." +
		strings.TrimSpace("")
}

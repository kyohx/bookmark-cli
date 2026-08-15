package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type fakeTokenSource struct {
	tokens []*oauth2.Token
	index  int32
}

func (f *fakeTokenSource) Token() (*oauth2.Token, error) {
	i := int(atomic.AddInt32(&f.index, 1) - 1)
	if i >= len(f.tokens) {
		i = len(f.tokens) - 1
	}
	return f.tokens[i], nil
}

type fakeAuthRefresher struct {
	calls int32
	err   error
}

func (f *fakeAuthRefresher) RefreshIfPossible(ctx context.Context) error {
	atomic.AddInt32(&f.calls, 1)
	return f.err
}

func TestClientRetriesOnceOn401(t *testing.T) {
	var reqCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&reqCount, 1)
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"testuser","authority":2}`))
	}))
	defer server.Close()

	refresher := &fakeAuthRefresher{}
	client := &Client{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 2 * time.Second},
		tokenSource: &fakeTokenSource{
			tokens: []*oauth2.Token{
				{AccessToken: "a1", TokenType: "Bearer"},
				{AccessToken: "a2", TokenType: "Bearer"},
			},
		},
		authService: refresher,
	}

	user, err := client.Me(context.Background())
	if err != nil {
		t.Fatalf("Me() error = %v", err)
	}
	if user.Name != "testuser" || user.Authority != 2 {
		t.Fatalf("unexpected user: %#v", user)
	}
	if got := atomic.LoadInt32(&refresher.calls); got != 1 {
		t.Fatalf("RefreshIfPossible calls = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&reqCount); got != 2 {
		t.Fatalf("request count = %d, want 2", got)
	}
}

func TestClientStopsAfterSingleRetry(t *testing.T) {
	var reqCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&reqCount, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	refresher := &fakeAuthRefresher{}
	client := &Client{
		baseURL:     server.URL,
		httpClient:  &http.Client{Timeout: 2 * time.Second},
		tokenSource: &fakeTokenSource{tokens: []*oauth2.Token{{AccessToken: "a1", TokenType: "Bearer"}}},
		authService: refresher,
	}

	_, err := client.Me(context.Background())
	if err == nil {
		t.Fatalf("Me() error should not be nil")
	}
	if !strings.Contains(err.Error(), "status=401") {
		t.Fatalf("error should contain status=401, got %v", err)
	}
	if got := atomic.LoadInt32(&refresher.calls); got != 1 {
		t.Fatalf("RefreshIfPossible calls = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&reqCount); got != 2 {
		t.Fatalf("request count = %d, want 2", got)
	}
}

func TestClientIncludesJSONErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"invalid tag","fields":["tag"]}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:     server.URL,
		httpClient:  &http.Client{Timeout: 2 * time.Second},
		tokenSource: &fakeTokenSource{tokens: []*oauth2.Token{{AccessToken: "a1", TokenType: "Bearer"}}},
		authService: &fakeAuthRefresher{},
	}

	_, err := client.ListBookmarks(context.Background(), []string{"bad"}, 1, 10)
	if err == nil {
		t.Fatal("ListBookmarks() error should not be nil")
	}
	if !strings.Contains(err.Error(), `status=400, body={"detail":"invalid tag","fields":["tag"]}`) {
		t.Fatalf("error should contain compact JSON body, got %v", err)
	}
}

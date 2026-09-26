package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// Use upstream response examples so old, flat responses cannot mask regressions.
func TestBookmarkOpenAPIResponses(t *testing.T) {
	raw, err := os.ReadFile("testdata/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Components struct {
			Schemas map[string]struct {
				Examples []json.RawMessage `json:"examples"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 64)
	for _, tc := range []struct{ name, method, path, schema, field string }{
		{"add", "POST", "/bookmarks", "ResponseForAddBookmark", "added_bookmark"},
		{"get", "GET", "/bookmarks/" + id, "ResponseForGetBookmark", "bookmark"},
		{"update", "PATCH", "/bookmarks/" + id, "ResponseForUpdateBookmark", "updated_bookmark"},
		{"list", "GET", "/bookmarks", "ResponseForGetBookmarkList", "bookmarks"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			example := spec.Components.Schemas[tc.schema].Examples[0]
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer token" {
					t.Error("missing authorization")
				}
				if tc.name == "add" || tc.name == "update" {
					var body map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if string(body["memo"]) != `""` {
						t.Errorf("empty memo not preserved: %s", body["memo"])
					}
					if tc.name == "update" {
						if _, ok := body["tags"]; ok {
							t.Error("unspecified tags must be omitted")
						}
					}
				}
				if tc.name == "list" {
					if !reflect.DeepEqual(r.URL.Query()["tag"], []string{"work", "test"}) || r.URL.Query().Get("page") != "2" || r.URL.Query().Get("size") != "100" {
						t.Errorf("unexpected query: %s", r.URL.RawQuery)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(example)
			}))
			defer server.Close()
			client := &Client{baseURL: server.URL, httpClient: &http.Client{Timeout: time.Second}, tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), authService: &fakeAuthRefresher{}}
			var got any
			var err error
			switch tc.name {
			case "add":
				got, err = client.AddBookmark(context.Background(), AddBookmarkRequest{URL: "https://example.com", Memo: "", Tags: []string{"work"}})
			case "get":
				got, err = client.GetBookmark(context.Background(), id)
			case "update":
				memo := ""
				got, err = client.UpdateBookmark(context.Background(), id, &memo, nil, false)
			case "list":
				got, err = client.ListBookmarks(context.Background(), []string{"work", "test"}, 2, 100)
			}
			if err != nil {
				t.Fatal(err)
			}
			var expected map[string]json.RawMessage
			if err := json.Unmarshal(example, &expected); err != nil {
				t.Fatal(err)
			}
			actual, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			var a, b any
			_ = json.Unmarshal(actual, &a)
			_ = json.Unmarshal(expected[tc.field], &b)
			if !reflect.DeepEqual(a, b) {
				t.Errorf("response = %s, want %s", actual, expected[tc.field])
			}
		})
	}
}

func TestBookmarkValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*Client) error
	}{
		{"page", func(c *Client) error { _, err := c.ListBookmarks(context.Background(), nil, 0, 10); return err }},
		{"size", func(c *Client) error { _, err := c.ListBookmarks(context.Background(), nil, 1, 101); return err }},
		{"tag", func(c *Client) error {
			_, err := c.ListBookmarks(context.Background(), []string{""}, 1, 10)
			return err
		}},
		{"id", func(c *Client) error { return c.DeleteBookmark(context.Background(), "invalid") }},
		{"memo", func(c *Client) error {
			memo := strings.Repeat("あ", 401)
			_, err := c.UpdateBookmark(context.Background(), strings.Repeat("a", 64), &memo, nil, false)
			return err
		}},
		{"tags", func(c *Client) error {
			_, err := c.UpdateBookmark(context.Background(), strings.Repeat("a", 64), nil, []string{}, true)
			return err
		}},
		{"url", func(c *Client) error {
			_, err := c.AddBookmark(context.Background(), AddBookmarkRequest{URL: "relative", Tags: []string{"work"}})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(&Client{}); err == nil {
				t.Fatal("expected validation error before HTTP request")
			}
		})
	}
	if err := validateAddBookmark(AddBookmarkRequest{URL: "https://example.com", Memo: strings.Repeat("あ", 400), Tags: []string{strings.Repeat("あ", 100)}}); err != nil {
		t.Fatal(err)
	}
}

func TestBookmarkUpdateTagsAndDelete(t *testing.T) {
	id := strings.Repeat("b", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bookmarks/"+id {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		switch r.Method {
		case http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if _, ok := body["memo"]; ok {
				t.Error("unspecified memo must be omitted")
			}
			if !reflect.DeepEqual(body["tags"], []any{"private", "test"}) {
				t.Errorf("unexpected tags: %v", body)
			}
			_, _ = w.Write([]byte(`{"updated_bookmark":{"hashed_id":"` + id + `","url":"https://example.com","memo":"","tags":["private","test"],"created_at":"2025-01-01 12:34:56","updated_at":"2025-01-02 09:45:01"}}`))
		case http.MethodDelete:
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	client := &Client{baseURL: server.URL, httpClient: server.Client(), tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), authService: &fakeAuthRefresher{}}
	bookmark, err := client.UpdateBookmark(context.Background(), id, nil, []string{"private", "test"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if bookmark.HashedID != id {
		t.Errorf("unexpected bookmark: %v", bookmark)
	}
	if err := client.DeleteBookmark(context.Background(), id); err != nil {
		t.Fatal(err)
	}
}

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"golang.org/x/oauth2"
)

func TestUserOpenAPIRequestsAndResponses(t *testing.T) {
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
	for _, tc := range []struct{ name, method, path, schema, field string }{
		{"add", http.MethodPost, "/users", "ResponseForAddUser", "added_user"},
		{"get", http.MethodGet, "/users/test_user", "ResponseForGetUser", "user"},
		{"list", http.MethodGet, "/users", "ResponseForGetUserList", "users"},
		{"update", http.MethodPatch, "/users/test_user", "ResponseForUpdateUser", "updated_user"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			example := spec.Components.Schemas[tc.schema].Examples[0]
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer token" {
					t.Error("missing authorization")
				}
				if tc.name == "list" && (r.URL.Query().Get("page") != "2" || r.URL.Query().Get("size") != "100") {
					t.Errorf("query = %s", r.URL.RawQuery)
				}
				if tc.name == "add" || tc.name == "update" {
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if tc.name == "add" && !reflect.DeepEqual(body, map[string]any{"name": "test_user", "password": "password", "authority": float64(2)}) {
						t.Errorf("add body = %v", body)
					}
					if tc.name == "update" && !reflect.DeepEqual(body, map[string]any{"disabled": false}) {
						t.Errorf("update body = %v", body)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(example)
			}))
			defer server.Close()
			client := &Client{baseURL: server.URL, httpClient: server.Client(), tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), authService: &fakeAuthRefresher{}}
			var got any
			var err error
			switch tc.name {
			case "add":
				got, err = client.AddUser(context.Background(), AddUserRequest{Name: "test_user", Password: "password", Authority: 2})
			case "get":
				got, err = client.GetUser(context.Background(), "test_user")
			case "list":
				got, err = client.ListUsers(context.Background(), 2, 100)
			case "update":
				v := false
				got, err = client.UpdateUser(context.Background(), "test_user", UpdateUserRequest{Disabled: &v})
			}
			if err != nil {
				t.Fatal(err)
			}
			var expected map[string]json.RawMessage
			if err := json.Unmarshal(example, &expected); err != nil {
				t.Fatal(err)
			}
			actual, _ := json.Marshal(got)
			var a, b any
			_ = json.Unmarshal(actual, &a)
			_ = json.Unmarshal(expected[tc.field], &b)
			if !reflect.DeepEqual(a, b) {
				t.Errorf("response = %s, want %s", actual, expected[tc.field])
			}
		})
	}
}

func TestUserValidation(t *testing.T) {
	client := &Client{}
	for _, name := range []string{"", "bad/name", "a-", "abcdefghijklmnopqrstuvwxyz1234567"} {
		if _, err := client.GetUser(context.Background(), name); err == nil {
			t.Errorf("GetUser(%q) accepted invalid name", name)
		}
	}
	if _, err := client.AddUser(context.Background(), AddUserRequest{Name: "valid", Password: "short", Authority: 2}); err == nil {
		t.Error("accepted short password")
	}
	if _, err := client.AddUser(context.Background(), AddUserRequest{Name: "valid", Password: "password", Authority: 3}); err == nil {
		t.Error("accepted invalid authority")
	}
	if _, err := client.ListUsers(context.Background(), 0, 10); err == nil {
		t.Error("accepted invalid page")
	}
	if _, err := client.UpdateUser(context.Background(), "valid", UpdateUserRequest{}); err == nil {
		t.Error("accepted empty update")
	}
	bad := "bad/name"
	if _, err := client.UpdateUser(context.Background(), "valid", UpdateUserRequest{Name: &bad}); err == nil {
		t.Error("accepted invalid new name")
	}
}

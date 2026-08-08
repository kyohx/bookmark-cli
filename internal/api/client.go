package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"bookmark-cli/internal/auth"
	"golang.org/x/oauth2"
)

type Client struct {
	baseURL     string
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
	authService authRefresher
}

type authRefresher interface {
	RefreshIfPossible(ctx context.Context) error
}

type CurrentUser struct {
	Name      string `json:"name"`
	Authority int    `json:"authority"`
}

type Bookmark struct {
	HashedID  string   `json:"hashed_id"`
	URL       string   `json:"url"`
	Memo      string   `json:"memo"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
	Tags      []string `json:"tags"`
}

type AddBookmarkRequest struct {
	URL  string   `json:"url"`
	Memo string   `json:"memo"`
	Tags []string `json:"tags"`
}

type listBookmarksResponse struct {
	Bookmarks []Bookmark `json:"bookmarks"`
}

type meResponse = CurrentUser

type addBookmarkResponse struct {
	HashedID string `json:"hashed_id"`
}

func NewClient(baseURL string, timeout time.Duration, authSvc *auth.Service) (*Client, error) {
	ts, err := authSvc.TokenSource(context.Background())
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		httpClient:  &http.Client{Timeout: timeout},
		tokenSource: ts,
		authService: authSvc,
	}, nil
}

func (c *Client) Me(ctx context.Context) (*CurrentUser, error) {
	var out meResponse
	if err := c.doJSON(ctx, http.MethodGet, "/me", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListBookmarks(ctx context.Context, tags []string, page, size int) ([]Bookmark, error) {
	q := make(url.Values)
	for _, tag := range tags {
		if strings.TrimSpace(tag) == "" {
			continue
		}
		q.Add("tag", tag)
	}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if size > 0 {
		q.Set("size", strconv.Itoa(size))
	}

	var out listBookmarksResponse
	if err := c.doJSON(ctx, http.MethodGet, "/bookmarks", q, nil, &out); err != nil {
		return nil, err
	}
	return out.Bookmarks, nil
}

func (c *Client) AddBookmark(ctx context.Context, req AddBookmarkRequest) (string, error) {
	var out addBookmarkResponse
	if err := c.doJSON(ctx, http.MethodPost, "/bookmarks", nil, req, &out); err != nil {
		return "", err
	}
	return out.HashedID, nil
}

func (c *Client) DeleteBookmark(ctx context.Context, hashedID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/bookmarks/"+hashedID, nil, nil, nil)
}

func (c *Client) UpdateBookmark(ctx context.Context, hashedID string, memo *string, tags []string, hasTags bool) error {
	var req map[string]any
	req = make(map[string]any)
	if memo != nil {
		req["memo"] = *memo
	}
	if hasTags {
		req["tags"] = tags
	}
	return c.doJSON(ctx, http.MethodPatch, "/bookmarks/"+hashedID, nil, req, nil)
}

func (c *Client) doJSON(
	ctx context.Context,
	method, path string,
	query url.Values,
	reqBody any,
	out any,
) error {
	if c.httpClient.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.httpClient.Timeout)
		defer cancel()
	}

	var rawBody []byte
	var err error
	if reqBody != nil {
		rawBody, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
	}

	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint = endpoint + "?" + query.Encode()
	}

	exec := func() (*http.Response, error) {
		var body io.Reader
		if rawBody != nil {
			body = bytes.NewReader(rawBody)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		if reqBody != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")

		token, err := c.tokenSource.Token()
		if err != nil {
			return nil, err
		}
		setAuthHeader(req, token)
		return c.httpClient.Do(req)
	}

	resp, err := exec()
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.authService.RefreshIfPossible(ctx); err != nil {
			return fmt.Errorf("refresh failed: %w", err)
		}

		resp.Body.Close()
		resp, err = exec()
		if err != nil {
			return fmt.Errorf("retry request failed: %w", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("api %s %s failed: status=%d", method, path, resp.StatusCode)
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

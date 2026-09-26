package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"bookmark-cli/internal/auth"
	"bookmark-cli/internal/cli"

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

type VersionInfo struct {
	Version string `json:"version"`
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
	Bookmark Bookmark `json:"added_bookmark"`
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

func (c *Client) Version(ctx context.Context) (*VersionInfo, error) {
	var out VersionInfo
	if err := c.doJSON(ctx, http.MethodGet, "/version", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListBookmarks(ctx context.Context, tags []string, page, size int) ([]Bookmark, error) {
	if err := validatePagination(page, size); err != nil {
		return nil, err
	}
	for _, tag := range tags {
		if err := validateTag(tag); err != nil {
			return nil, err
		}
	}
	q := make(url.Values)
	for _, tag := range tags {
		q.Add("tag", tag)
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("size", strconv.Itoa(size))

	var out listBookmarksResponse
	if err := c.doJSON(ctx, http.MethodGet, "/bookmarks", q, nil, &out); err != nil {
		return nil, err
	}
	return out.Bookmarks, nil
}

func (c *Client) AddBookmark(ctx context.Context, req AddBookmarkRequest) (*Bookmark, error) {
	if err := validateAddBookmark(req); err != nil {
		return nil, err
	}
	var out addBookmarkResponse
	if err := c.doJSON(ctx, http.MethodPost, "/bookmarks", nil, req, &out); err != nil {
		return nil, err
	}
	return &out.Bookmark, nil
}

func (c *Client) DeleteBookmark(ctx context.Context, hashedID string) error {
	if err := validateHashedID(hashedID); err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodDelete, "/bookmarks/"+hashedID, nil, nil, nil)
}

func (c *Client) UpdateBookmark(ctx context.Context, hashedID string, memo *string, tags []string, hasTags bool) (*Bookmark, error) {
	if err := validateHashedID(hashedID); err != nil {
		return nil, err
	}
	if memo != nil && utf8.RuneCountInString(*memo) > 400 {
		return nil, fmt.Errorf("memo must be at most 400 characters")
	}
	if hasTags {
		if err := validateTags(tags); err != nil {
			return nil, err
		}
	}
	var req map[string]any
	req = make(map[string]any)
	if memo != nil {
		req["memo"] = *memo
	}
	if hasTags {
		req["tags"] = tags
	}
	var out struct {
		Bookmark Bookmark `json:"updated_bookmark"`
	}
	if err := c.doJSON(ctx, http.MethodPatch, "/bookmarks/"+hashedID, nil, req, &out); err != nil {
		return nil, err
	}
	return &out.Bookmark, nil
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
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if err != nil {
			return fmt.Errorf("read error response: %w", err)
		}
		return cli.WrapHTTPError(fmt.Sprintf("api %s %s failed", method, path), resp.StatusCode, bodyBytes)
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) GetBookmark(ctx context.Context, hashedID string) (*Bookmark, error) {
	if err := validateHashedID(hashedID); err != nil {
		return nil, err
	}
	var out struct {
		Bookmark Bookmark `json:"bookmark"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/bookmarks/"+hashedID, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out.Bookmark, nil
}

func validatePagination(page, size int) error {
	if page < 1 {
		return fmt.Errorf("page must be at least 1")
	}
	if size < 1 || size > 100 {
		return fmt.Errorf("size must be between 1 and 100")
	}
	return nil
}

func validateHashedID(id string) error {
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(id) {
		return fmt.Errorf("hashed_id must be 64 lowercase hexadecimal characters")
	}
	return nil
}

func validateTag(tag string) error {
	n := utf8.RuneCountInString(tag)
	if n < 1 || n > 100 {
		return fmt.Errorf("each tag must contain 1 to 100 characters")
	}
	return nil
}

func validateTags(tags []string) error {
	if len(tags) < 1 || len(tags) > 10 {
		return fmt.Errorf("tags must contain 1 to 10 items")
	}
	for _, tag := range tags {
		if err := validateTag(tag); err != nil {
			return err
		}
	}
	return nil
}

func validateAddBookmark(req AddBookmarkRequest) error {
	u, err := url.Parse(req.URL)
	if err != nil || u.Scheme == "" || utf8.RuneCountInString(req.URL) < 1 || utf8.RuneCountInString(req.URL) > 400 {
		return fmt.Errorf("url must be an absolute URI of at most 400 characters")
	}
	if utf8.RuneCountInString(req.Memo) > 400 {
		return fmt.Errorf("memo must be at most 400 characters")
	}
	return validateTags(req.Tags)
}

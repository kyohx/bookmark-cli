package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"bookmark-cli/internal/session"
	"golang.org/x/oauth2"
)

type Service struct {
	baseURL string
	client  *http.Client
	store   session.TokenStore
	mu      sync.Mutex
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func NewService(baseURL string, timeout time.Duration, store session.TokenStore) *Service {
	return &Service{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
		store:   store,
	}
}

func (s *Service) Login(ctx context.Context, username string, password []byte, scopes []string) error {
	if s.client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.client.Timeout)
		defer cancel()
	}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", username)
	form.Set("password", string(password))
	form.Set("scope", normalizeScopes(scopes))

	out, status, _, err := s.loginWithPath(ctx, "/token", form)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("login failed: status=%d", status)
	}
	return s.saveToken(ctx, out, "")
}

func (s *Service) Logout(ctx context.Context) error {
	return s.store.Delete(ctx)
}

func (s *Service) TokenSource(_ context.Context) (oauth2.TokenSource, error) {
	return &tokenSource{svc: s}, nil
}

func (s *Service) CurrentToken(ctx context.Context) (*oauth2.Token, error) {
	return s.store.Load(ctx)
}

func (s *Service) RefreshIfPossible(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tok, err := s.store.Load(ctx)
	if err != nil {
		return err
	}
	if tok.RefreshToken == "" {
		return errors.New("refresh token is empty")
	}
	return s.refresh(ctx, tok.RefreshToken)
}

func (s *Service) refresh(ctx context.Context, refreshToken string) error {
	if s.client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.client.Timeout)
		defer cancel()
	}

	body := refreshRequest{RefreshToken: refreshToken}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal refresh request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/refresh", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh failed: status=%d", resp.StatusCode)
	}

	var out loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode refresh response: %w", err)
	}
	return s.saveToken(ctx, out, refreshToken)
}

func (s *Service) saveToken(ctx context.Context, res loginResponse, fallbackRefresh string) error {
	refresh := res.RefreshToken
	if refresh == "" {
		refresh = fallbackRefresh
	}
	token := &oauth2.Token{
		AccessToken:  res.AccessToken,
		RefreshToken: refresh,
		TokenType:    res.TokenType,
		Expiry:       parseJWTExpiry(res.AccessToken),
	}
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	if err := s.store.Save(ctx, token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}

func (s *Service) loginWithPath(ctx context.Context, path string, form url.Values) (loginResponse, int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return loginResponse{}, 0, "", fmt.Errorf("build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return loginResponse{}, 0, "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return loginResponse{}, 0, "", fmt.Errorf("read login response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return loginResponse{}, resp.StatusCode, string(bodyBytes), nil
	}

	var out loginResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return loginResponse{}, 0, "", fmt.Errorf("decode login response: %w", err)
	}
	return out, resp.StatusCode, "", nil
}

func parseJWTExpiry(accessToken string) time.Time {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return time.Time{}
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}
	}

	expVal, ok := claims["exp"]
	if !ok {
		return time.Time{}
	}

	switch v := expVal.(type) {
	case float64:
		return time.Unix(int64(v), 0).UTC()
	case int64:
		return time.Unix(v, 0).UTC()
	default:
		return time.Time{}
	}
}

func normalizeScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}

	seen := make(map[string]struct{}, len(scopes))
	normalized := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		normalized = append(normalized, scope)
	}
	return strings.Join(normalized, " ")
}

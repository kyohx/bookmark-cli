package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"unicode/utf8"
)

type SessionDetail struct {
	ID         string  `json:"id"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt string  `json:"last_used_at"`
	ExpiresAt  string  `json:"expires_at"`
	Revoked    bool    `json:"revoked"`
	UserAgent  *string `json:"user_agent"`
}

func (c *Client) ListUserSessions(ctx context.Context, name string) ([]SessionDetail, error) {
	if err := validateUserName(name); err != nil {
		return nil, err
	}
	var out struct {
		Sessions []SessionDetail `json:"sessions"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/users/"+name+"/sessions", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Sessions, nil
}

func (c *Client) RevokeUserSession(ctx context.Context, name, sessionID string) error {
	if err := validateUserName(name); err != nil {
		return err
	}
	if n := utf8.RuneCountInString(sessionID); n < 1 || n > 128 {
		return fmt.Errorf("session_id must contain 1 to 128 characters")
	}
	return c.doJSON(ctx, http.MethodDelete, "/users/"+name+"/sessions/"+url.PathEscape(sessionID), nil, nil, nil)
}

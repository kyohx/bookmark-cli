package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

var ErrNotFound = errors.New("session not found")

const serviceName = "bookmark-cli"

var (
	keyringSet    = keyring.Set
	keyringGet    = keyring.Get
	keyringDelete = keyring.Delete
)

type TokenStore interface {
	Save(ctx context.Context, token *oauth2.Token) error
	Load(ctx context.Context) (*oauth2.Token, error)
	Delete(ctx context.Context) error
}

type KeyringStore struct {
	user string
}

type storedToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Expiry       string `json:"expiry,omitempty"`
}

func NewKeyringStore(profileName string) *KeyringStore {
	return &KeyringStore{
		user: "session:" + profileName,
	}
}

func (s *KeyringStore) Save(_ context.Context, token *oauth2.Token) error {
	st := storedToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
	}
	if !token.Expiry.IsZero() {
		st.Expiry = token.Expiry.UTC().Format(time.RFC3339)
	}

	raw, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := keyringSet(serviceName, s.user, string(raw)); err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}
	return nil
}

func (s *KeyringStore) Load(_ context.Context) (*oauth2.Token, error) {
	raw, err := keyringGet(serviceName, s.user)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("keyring get: %w", err)
	}

	var st storedToken
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}

	tok := &oauth2.Token{
		AccessToken:  st.AccessToken,
		RefreshToken: st.RefreshToken,
		TokenType:    st.TokenType,
	}
	if st.Expiry != "" {
		t, err := time.Parse(time.RFC3339, st.Expiry)
		if err != nil {
			return nil, fmt.Errorf("parse expiry: %w", err)
		}
		tok.Expiry = t
	}
	return tok, nil
}

func (s *KeyringStore) Delete(_ context.Context) error {
	err := keyringDelete(serviceName, s.user)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("keyring delete: %w", err)
	}
	return nil
}

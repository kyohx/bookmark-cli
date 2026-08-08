package auth

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

type tokenSource struct {
	svc *Service
}

func (t *tokenSource) Token() (*oauth2.Token, error) {
	ctx := context.Background()

	tok, err := t.svc.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if tok.Valid() {
		return tok, nil
	}

	if tok.RefreshToken == "" {
		return nil, fmt.Errorf("token expired and refresh token is empty")
	}
	if err := t.svc.refresh(ctx, tok.RefreshToken); err != nil {
		return nil, err
	}
	return t.svc.store.Load(ctx)
}

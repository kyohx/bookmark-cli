package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

func TestKeyringStoreSaveAndLoad(t *testing.T) {
	origSet := keyringSet
	origGet := keyringGet
	t.Cleanup(func() {
		keyringSet = origSet
		keyringGet = origGet
	})

	var savedService string
	var savedUser string
	var savedRaw string
	keyringSet = func(service, user, password string) error {
		savedService = service
		savedUser = user
		savedRaw = password
		return nil
	}
	keyringGet = func(service, user string) (string, error) {
		if service != savedService || user != savedUser {
			return "", errors.New("unexpected key")
		}
		return savedRaw, nil
	}

	store := NewKeyringStore("http://localhost:8000")
	wantExpiry := time.Now().UTC().Truncate(time.Second)
	err := store.Save(context.Background(), &oauth2.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Expiry:       wantExpiry,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.AccessToken != "access" {
		t.Fatalf("AccessToken = %q, want %q", got.AccessToken, "access")
	}
	if got.RefreshToken != "refresh" {
		t.Fatalf("RefreshToken = %q, want %q", got.RefreshToken, "refresh")
	}
	if got.TokenType != "Bearer" {
		t.Fatalf("TokenType = %q, want %q", got.TokenType, "Bearer")
	}
	if !got.Expiry.Equal(wantExpiry) {
		t.Fatalf("Expiry = %v, want %v", got.Expiry, wantExpiry)
	}
}

func TestKeyringStoreLoadNotFound(t *testing.T) {
	origGet := keyringGet
	t.Cleanup(func() {
		keyringGet = origGet
	})
	keyringGet = func(service, user string) (string, error) {
		return "", keyring.ErrNotFound
	}

	store := NewKeyringStore("http://localhost:8000")
	_, err := store.Load(context.Background())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load() error = %v, want ErrNotFound", err)
	}
}

func TestKeyringStoreDelete(t *testing.T) {
	origDelete := keyringDelete
	t.Cleanup(func() {
		keyringDelete = origDelete
	})

	keyringDelete = func(service, user string) error {
		return keyring.ErrNotFound
	}
	store := NewKeyringStore("http://localhost:8000")
	if err := store.Delete(context.Background()); err != nil {
		t.Fatalf("Delete() with ErrNotFound should succeed, got %v", err)
	}

	keyringDelete = func(service, user string) error {
		return errors.New("boom")
	}
	if err := store.Delete(context.Background()); err == nil {
		t.Fatalf("Delete() should fail on generic error")
	}
}

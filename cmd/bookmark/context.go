package main

import (
	"fmt"
	"strings"
	"time"

	"bookmark-cli/internal/api"
	"bookmark-cli/internal/auth"
	"bookmark-cli/internal/config"
	"bookmark-cli/internal/session"
)

type commandDeps struct {
	BaseURL string
	Timeout time.Duration
	Scopes  []string
	Auth    *auth.Service
	API     *api.Client
}

func loadDeps(opts rootOptions, withAPI bool) (*commandDeps, error) {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	profileName := strings.TrimSpace(opts.Profile)
	if profileName == "" {
		profileName = config.DefaultProfile
	}

	profile, err := cfg.Profile(profileName)
	if err != nil {
		return nil, fmt.Errorf("load profile: %w", err)
	}

	baseURL := profile.BaseURL
	if strings.TrimSpace(opts.BaseURL) != "" {
		baseURL = strings.TrimSpace(opts.BaseURL)
	}
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is empty")
	}

	timeoutSec := profile.TimeoutSeconds
	if opts.TimeoutSec > 0 {
		timeoutSec = opts.TimeoutSec
	}
	timeout := time.Duration(timeoutSec) * time.Second

	store := session.NewKeyringStore(profileName)
	authSvc := auth.NewService(baseURL, timeout, store)

	deps := &commandDeps{
		BaseURL: baseURL,
		Timeout: timeout,
		Scopes:  profile.Scopes,
		Auth:    authSvc,
	}
	if withAPI {
		apiClient, err := api.NewClient(baseURL, timeout, authSvc)
		if err != nil {
			return nil, fmt.Errorf("build api client: %w", err)
		}
		deps.API = apiClient
	}
	return deps, nil
}

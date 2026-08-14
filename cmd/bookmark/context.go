package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"bookmark-cli/internal/api"
	"bookmark-cli/internal/auth"
	"bookmark-cli/internal/cli"
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

type resolvedProfile struct {
	Name           string   `json:"name"`
	ConfigPath     string   `json:"config_path,omitempty"`
	BaseURL        string   `json:"base_url"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	Scopes         []string `json:"scopes,omitempty"`
}

func resolveProfile(opts *rootOptions) (resolvedProfile, error) {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return resolvedProfile{}, fmt.Errorf("load config: %w", err)
	}

	profileName := strings.TrimSpace(opts.Profile)
	if profileName == "" {
		profileName = config.DefaultProfile
	}

	profile, err := cfg.Profile(profileName)
	if err != nil {
		return resolvedProfile{}, fmt.Errorf("load profile: %w", err)
	}

	baseURL := profile.BaseURL
	if strings.TrimSpace(opts.BaseURL) != "" {
		baseURL = strings.TrimSpace(opts.BaseURL)
	}
	if baseURL == "" {
		return resolvedProfile{}, fmt.Errorf("base URL is empty")
	}

	timeoutSec := profile.TimeoutSeconds
	if opts.TimeoutSet && opts.TimeoutSec > 0 {
		timeoutSec = opts.TimeoutSec
	}

	return resolvedProfile{
		Name:           profileName,
		ConfigPath:     opts.ConfigPath,
		BaseURL:        baseURL,
		TimeoutSeconds: timeoutSec,
		Scopes:         slices.Clone(profile.Scopes),
	}, nil
}

func maybePrintProfile(cmd interface{ ErrOrStderr() io.Writer }, opts *rootOptions) error {
	if !opts.ShowProfile {
		return nil
	}

	profile, err := resolveProfile(opts)
	if err != nil {
		return err
	}
	return cli.PrintJSONTo(cmd.ErrOrStderr(), profile)
}

func loadDeps(opts *rootOptions, withAPI bool) (*commandDeps, error) {
	profile, err := resolveProfile(opts)
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(profile.TimeoutSeconds) * time.Second

	store := session.NewKeyringStore(profile.Name)
	authSvc := auth.NewService(profile.BaseURL, timeout, store)

	deps := &commandDeps{
		BaseURL: profile.BaseURL,
		Timeout: timeout,
		Scopes:  profile.Scopes,
		Auth:    authSvc,
	}
	if withAPI {
		apiClient, err := api.NewClient(profile.BaseURL, timeout, authSvc)
		if err != nil {
			return nil, fmt.Errorf("build api client: %w", err)
		}
		deps.API = apiClient
	}
	return deps, nil
}

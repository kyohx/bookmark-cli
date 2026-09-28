package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"unicode/utf8"
)

type User struct {
	Name      string `json:"name"`
	Authority int    `json:"authority"`
	Disabled  bool   `json:"disabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type AddUserRequest struct {
	Name      string `json:"name"`
	Password  string `json:"password"`
	Authority int    `json:"authority"`
}

type UpdateUserRequest struct {
	Name      *string `json:"name,omitempty"`
	Password  *string `json:"password,omitempty"`
	Authority *int    `json:"authority,omitempty"`
	Disabled  *bool   `json:"disabled,omitempty"`
}

var userNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{1,32}$`)

func validateUserName(name string) error {
	if !userNamePattern.MatchString(name) {
		return fmt.Errorf("name must contain 1 to 32 ASCII letters, digits, or underscores")
	}
	return nil
}

func validateAuthority(authority int) error {
	switch authority {
	case 0, 1, 2, 9:
		return nil
	default:
		return fmt.Errorf("authority must be 0, 1, 2, or 9")
	}
}

func ValidateAddUserOptions(name string, authority int) error {
	if err := validateUserName(name); err != nil {
		return err
	}
	return validateAuthority(authority)
}

func (c *Client) AddUser(ctx context.Context, req AddUserRequest) (*User, error) {
	if err := ValidateAddUserOptions(req.Name, req.Authority); err != nil {
		return nil, err
	}
	if n := utf8.RuneCountInString(req.Password); n < 8 || n > 64 {
		return nil, fmt.Errorf("password must contain 8 to 64 characters")
	}
	var out struct {
		User User `json:"added_user"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/users", nil, req, &out); err != nil {
		return nil, err
	}
	return &out.User, nil
}

func (c *Client) GetUser(ctx context.Context, name string) (*User, error) {
	if err := validateUserName(name); err != nil {
		return nil, err
	}
	var out struct {
		User User `json:"user"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/users/"+name, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out.User, nil
}

func (c *Client) ListUsers(ctx context.Context, page, size int) ([]User, error) {
	if err := validatePagination(page, size); err != nil {
		return nil, err
	}
	q := make(url.Values)
	q.Set("page", strconv.Itoa(page))
	q.Set("size", strconv.Itoa(size))
	var out struct {
		Users []User `json:"users"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/users", q, nil, &out); err != nil {
		return nil, err
	}
	return out.Users, nil
}

func (c *Client) UpdateUser(ctx context.Context, name string, req UpdateUserRequest) (*User, error) {
	if err := validateUserName(name); err != nil {
		return nil, err
	}
	if req.Name == nil && req.Password == nil && req.Authority == nil && req.Disabled == nil {
		return nil, fmt.Errorf("specify at least one field to update")
	}
	if req.Name != nil {
		if err := validateUserName(*req.Name); err != nil {
			return nil, err
		}
	}
	if req.Password != nil {
		if n := utf8.RuneCountInString(*req.Password); n < 8 || n > 64 {
			return nil, fmt.Errorf("password must contain 8 to 64 characters")
		}
	}
	if req.Authority != nil {
		if err := validateAuthority(*req.Authority); err != nil {
			return nil, err
		}
	}
	var out struct {
		User User `json:"updated_user"`
	}
	if err := c.doJSON(ctx, http.MethodPatch, "/users/"+name, nil, req, &out); err != nil {
		return nil, err
	}
	return &out.User, nil
}

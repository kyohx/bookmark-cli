package api

import (
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

func setAuthHeader(req *http.Request, token *oauth2.Token) {
	tokenType := token.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}
	if !strings.EqualFold(tokenType, "bearer") {
		req.Header.Set("Authorization", tokenType+" "+token.AccessToken)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
}

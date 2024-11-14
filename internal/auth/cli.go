package auth

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

var ErrAuthTokenExpired = fmt.Errorf("authorization token expired")

func EnsureValidAuthData(ctx context.Context, auth *AuthorizationResponse, orgURL string, clientID string) (*AuthorizationResponse, error) {
	if auth == nil {
		return nil, nil
	}

	config, err := NewOAuth2Config(ctx, orgURL, clientID)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken:  auth.AccessToken,
		RefreshToken: auth.RefreshToken,
		Expiry:       time.Unix(auth.ExpiresAt, 0),
	}

	// If the token is valid, no need to refresh
	if token.Valid() {
		return auth, nil
	}

	newToken, err := Refresh(ctx, config, auth.RefreshToken)
	if err != nil {
		return nil, err
	}

	auth.AccessToken = newToken.AccessToken
	auth.RefreshToken = newToken.RefreshToken
	auth.ExpiresAt = newToken.Expiry.Unix()

	return auth, nil
}

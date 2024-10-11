package auth

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

var ErrAuthTokenExpired = fmt.Errorf("authorization token expired")

func EnsureValidAuthData(ctx context.Context, auth *AuthorizationResponse, orgURL string, clientID string) (*AuthorizationResponse, error) {
	if auth == nil {
		return nil, nil
	}

	config := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  orgURL + "/oauth2/default/v1/device/authorize",
			TokenURL: orgURL + "/oauth2/default/v1/token",
		},
		Scopes: []string{"openid", "profile", "offline_access"},
	}

	newToken, err := Refresh(ctx, config, auth.RefreshToken)
	if err != nil {
		return nil, err
	}

	auth.AccessToken = newToken.AccessToken

	return auth, nil
}

package auth

import (
	"context"
	"fmt"
)

var ErrAuthTokenExpired = fmt.Errorf("authorization token expired")

func EnsureValidAuthData(ctx context.Context, auth *AuthorizationResponse, orgURL string, clientID string) (*AuthorizationResponse, error) {
	if auth == nil {
		return nil, nil
	}

	if auth.IsExpired() {
		updatedAuth, err := Refresh(ctx, orgURL, clientID, auth.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh authentication token: %w", err)
		}

		return updatedAuth, err
	}

	return auth, nil
}

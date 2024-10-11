package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

func waitForAuthorization(ctx context.Context, config *oauth2.Config, deviceCode string) (AuthorizationResponse, error) {
	for {
		// Prepare the token request
		values := url.Values{
			"client_id":   {config.ClientID},
			"device_code": {deviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.Endpoint.TokenURL, strings.NewReader(values.Encode()))
		if err != nil {
			return AuthorizationResponse{}, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Print(err)
			return AuthorizationResponse{}, nil
		}
		defer resp.Body.Close()

		var token AuthorizationResponse
		if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
			log.Print(err)
			return AuthorizationResponse{}, nil
		}
		fmt.Printf("%+v\n", token)

		switch token.Error {
		case "":
			// Successfully received a token
			return token, nil
		case "authorization_pending":
			// Do nothing, just wait
		case "access_denied":
			return token, fmt.Errorf("access denied")
		default:
			return token, fmt.Errorf("authorization failed: %v", token.Error)
		}

		select {
		case <-ctx.Done():
			return token, ctx.Err()
		case <-time.After(time.Duration(2) * time.Second):
			// next loop iteration
		}
	}
}

func Refresh(ctx context.Context, config *oauth2.Config, refreshToken string) (*oauth2.Token, error) {
	tokenSource := config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return newToken, nil
}

func Login(ctx context.Context, orgURL string, clientID string) (AuthorizationResponse, error) {
	config := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  orgURL + "/oauth2/default/v1/device/authorize",
			TokenURL: orgURL + "/oauth2/default/v1/token",
		},
		Scopes: []string{"openid", "profile", "offline_access"},
	}

	values := url.Values{
		"client_id": {clientID},
		"scope":     {"openid profile offline_access"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.Endpoint.AuthURL, strings.NewReader(values.Encode()))
	if err != nil {
		return AuthorizationResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return AuthorizationResponse{}, fmt.Errorf("failed to get device code: %w", err)
	}
	defer resp.Body.Close()

	var loginResponse LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		return AuthorizationResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Println("Please visit:", loginResponse.VerificationURIComplete)
	fmt.Println("Code:", loginResponse.UserCode)

	if err := openbrowser(loginResponse.VerificationURIComplete); err != nil {
		log.Print(err)
	}

	authResponse, err := waitForAuthorization(ctx, config, loginResponse.DeviceCode)
	if err != nil {
		return AuthorizationResponse{}, fmt.Errorf("failed to get authorization: %w", err)
	}

	authResponse.ExpiresAt = authResponse.GetExpiresAt()
	return authResponse, nil
}

func openbrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")

	}
	if err != nil {
		log.Println("Error opening browser:", err)
	}

	return err
}

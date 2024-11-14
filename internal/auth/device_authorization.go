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
		switch token.Error {
		case "":
			// Successfully received a token
			return token, nil
		case "authorization_pending":
			// Do nothing, just wait
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

func NewOAuth2Config(ctx context.Context, orgURL string, clientID string) (*oauth2.Config, error) {
	return &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  orgURL + "/oauth2/v2.0/devicecode",
			TokenURL: orgURL + "/oauth2/v2.0/token",
		},
		Scopes: []string{"openid", "profile", "offline_access", "api://provisioner/provisioner"},
	}, nil
}

func Refresh(ctx context.Context, config *oauth2.Config, refreshToken string) (*oauth2.Token, error) {
	data := url.Values{
		"client_id":     {config.ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {strings.Join(config.Scopes, " ")},
	}

	req, err := http.NewRequest("POST", config.Endpoint.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	var token oauth2.Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &token, nil
}

func Login(ctx context.Context, orgURL string, clientID string) (AuthorizationResponse, error) {
	config, err := NewOAuth2Config(ctx, orgURL, clientID)
	if err != nil {
		return AuthorizationResponse{}, fmt.Errorf("failed to create config: %w", err)
	}

	values := url.Values{
		"client_id": {config.ClientID},
		"scope":     {strings.Join(config.Scopes, " ")},
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
	if dErr := json.NewDecoder(resp.Body).Decode(&loginResponse); dErr != nil {
		return AuthorizationResponse{}, fmt.Errorf("failed to decode response: %w", dErr)
	}

	if oErr := openbrowser(loginResponse.VerificationURI); oErr != nil {
		fmt.Println("Unable to open browser: %w", oErr)
		fmt.Println(loginResponse.Message)
	} else {
		fmt.Println("Please enter the following code in the browser: ", loginResponse.UserCode)
	}

	authResponse, err := waitForAuthorization(ctx, config, loginResponse.DeviceCode)
	if err != nil {
		return AuthorizationResponse{}, fmt.Errorf("failed to get authorization: %w", err)
	}

	authResponse.ExpiresAt = authResponse.GetExpiresAt()
	fmt.Println("Authentication successful")

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

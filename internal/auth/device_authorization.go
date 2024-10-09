package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func waitForAuthorization(ctx context.Context, orgURL string, clientID string, code string) (AuthorizationResponse, error) {
	values := url.Values{
		"client_id":   {clientID},
		"device_code": {code},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}
	for {
		var token AuthorizationResponse

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, orgURL+"/oauth2/v1/token", strings.NewReader(values.Encode()))
		if err != nil {
			return token,
				fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Print(err)
			return token, nil
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Print(err)
			return token, nil

		}

		err = json.Unmarshal(body, &token)
		if err != nil {
			log.Print(err)
			return token, nil
		}

		fmt.Println(string(body))

		switch token.Error {
		case "":
			// if error is empty, we got a token
			return token, nil
		case "authorization_pending":
			// do nothing, just wait
		case "access_denied":
			return token, fmt.Errorf("Access denied")
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

func Refresh(ctx context.Context, orgURL, clientID, refreshToken string) (*AuthorizationResponse, error) {
	var token AuthorizationResponse

	values := url.Values{
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
		"scope":         {"openid profile offline_access"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, orgURL+"/oauth2/v1/token", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err)
		return nil, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return nil, nil
	}

	err = json.Unmarshal(body, &token)
	if err != nil {
		log.Print(err)
		return &token, nil
	}

	if token.Error != "" {
		return &token, fmt.Errorf("authorization failed: %v", token.Error)
	}

	return &token, nil
}

func Login(ctx context.Context, orgURL string, clientID string) (AuthorizationResponse, error) {
	var login AuthorizationResponse
	fullUrl := orgURL + "/oauth2/v1/device/authorize"

	values := url.Values{
		"client_id": {clientID},
		"scope":     {"openid profile offline_access"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullUrl, strings.NewReader(values.Encode()))
	if err != nil {
		log.Print(err)
		return login, nil
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err)
		return login, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return login, nil

	}

	var loginResponse LoginResponse

	err = json.Unmarshal(body, &loginResponse)
	if err != nil {
		log.Print(err)
		return login, nil
	}

	fmt.Println(string(body))
	err = openbrowser(loginResponse.VerificationURIComplete)
	if err != nil {
		log.Print(err)
	}

	fmt.Println("-------------------------------------------------------------")
	fmt.Println("If your browser isn't already open, copy and past the following URL directly:")
	fmt.Println(loginResponse.VerificationURIComplete)
	fmt.Println("Code:", loginResponse.UserCode)
	fmt.Println("-------------------------------------------------------------")

	login, err = waitForAuthorization(ctx, orgURL, clientID, loginResponse.DeviceCode)
	if err != nil {
		log.Print(err)
		return login, nil
	}

	return login, nil
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

package auth

import "time"

const ContextKeyAuthData = "authData"

type ServerConfig struct {
	Issuer    string   `json:"issuer"`
	ClientIDs []string `json:"client_ids"`
	Audience  string   `json:"audience"`
}

func (s ServerConfig) IsValid() bool {
	return s.Issuer != "" && len(s.ClientIDs) > 0 && s.Audience != ""
}

type LoginResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
	VerificationURIComplete string `json:"verification_uri_complete"`
}

type AuthorizationResponse struct {
	AccessToken      string `json:"access_token"`
	IDToken          string `json:"id_token"`
	Scope            string `json:"scope"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresAt        int64  `json:"expires_at"`
}

func (a *AuthorizationResponse) GetExpiresAt() int64 {
	return time.Now().Unix() + int64(a.ExpiresIn)
}

func (a AuthorizationResponse) IsExpired() bool {
	return time.Now().Unix() > a.ExpiresAt
}

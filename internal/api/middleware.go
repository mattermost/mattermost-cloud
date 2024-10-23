package api

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"strings"

	keyfunc "github.com/MicahParks/keyfunc/v3"
	jwt "github.com/golang-jwt/jwt/v5"
)

type ContextKeyUserID struct{}

type CustomClaims struct {
	Oid   string `json:"oid"`
	AppID string `json:"appid"`
	jwt.RegisteredClaims
}

func AuthMiddleware(next http.Handler, apiContext *Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// If auth isn't configured, don't worry about authentication
		if apiContext.AuthConfig == nil {
			next.ServeHTTP(w, r)
			return
		}

		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		k, err := keyfunc.NewDefaultCtx(r.Context(), []string{apiContext.AuthConfig.JWKSURL})
		if err != nil {
			log.Printf("Error creating keyfunc: %v", err)
			http.Error(w, "Failed to create keyfunc", http.StatusInternalServerError)
			return
		}

		// Parse and validate the JWT
		token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, k.Keyfunc, jwt.WithAudience(apiContext.AuthConfig.Audience), jwt.WithIssuer(apiContext.AuthConfig.Issuer))

		if err != nil || !token.Valid {
			log.Printf("Error validating token: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		var uid string
		var cid string
		if claims, ok := token.Claims.(*CustomClaims); ok {
			uid = claims.Oid
			cid = claims.AppID

			if uid == "" {
				// If the user ID is missing, this is a client credentials token, so use the cid as the uid
				uid = cid
			}
		}

		endpoint := r.URL.Path
		if !isAccessAllowed(cid, endpoint, apiContext.AuthConfig.RestrictedClientIDs, apiContext.AuthConfig.RestrictedClientAllowedEndpointsList) {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		// Add user ID to request context for use in handlers
		ctx := context.WithValue(r.Context(), ContextKeyUserID{}, uid)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func isAccessAllowed(clientID string, endpoint string, restrictedClientIDs []string, restrictedEndpoints []string) bool {
	if isRestrictedClient(clientID, restrictedClientIDs) {
		if !isRestrictedAllowedEndpoint(endpoint, restrictedEndpoints) {
			return false
		}
	}
	return true
}

func isRestrictedClient(clientID string, restrictedClientIDs []string) bool {
	for _, id := range restrictedClientIDs {
		if id == clientID {
			return true
		}
	}
	return false
}

func isRestrictedAllowedEndpoint(endpoint string, restrictedEndpoints []string) bool {

	for _, e := range restrictedEndpoints {
		// Check if the endpoint is a regex pattern (for exact matches, or others)
		if strings.HasPrefix(e, "^") && strings.HasSuffix(e, "$") {
			if matched, _ := regexp.MatchString(e, endpoint); matched {
				return true
			}
			// Otherwise, it's treated as a wildcard prefix match
		} else if strings.HasPrefix(endpoint, e) {
			return true // Access denied for prefix match
		}
	}
	return false
}

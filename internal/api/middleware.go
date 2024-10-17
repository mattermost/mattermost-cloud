package api

import (
	"context"
	"log"
	"net/http"
	"strings"

	jwtverifier "github.com/okta/okta-jwt-verifier-golang"
)

type ContextKeyUserID struct{}

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

		toValidate := map[string]string{}
		toValidate["aud"] = apiContext.AuthConfig.Audience

		jwtVerifierSetup := jwtverifier.JwtVerifier{
			Issuer:           apiContext.AuthConfig.Issuer,
			ClaimsToValidate: toValidate,
		}

		verifier := jwtVerifierSetup.New()

		token, err := verifier.VerifyAccessToken(tokenString)
		if err != nil {
			log.Printf("Error verifying token: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Extract user ID (or other claims)
		var userID string
		if uid, ok := token.Claims["uid"]; ok {
			userID = uid.(string)
		} else if sub, ok := token.Claims["sub"]; ok {
			userID = sub.(string)
		} else {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		log.Printf("%+v", token.Claims)

		// Add user ID to request context for use in handlers
		ctx := context.WithValue(r.Context(), ContextKeyUserID{}, userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

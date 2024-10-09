package api

import (
	"context"
	"log"
	"net/http"
	"strings"

	jwtverifier "github.com/okta/okta-jwt-verifier-golang"
)

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

		apiContext.Logger.Println("AuthConfig")
		apiContext.Logger.Println(apiContext.AuthConfig)

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		toValidate := map[string]string{}
		toValidate["cid"] = apiContext.AuthConfig.ClientIDs[0]
		toValidate["aud"] = apiContext.AuthConfig.Audience
		// TODO: validate audience as well

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
		userID := token.Claims["uid"].(string) // Type assertion to string

		log.Printf("%+v", token.Claims)

		// Add user ID to request context for use in handlers
		ctx := context.WithValue(r.Context(), "userID", userID)
		r = r.WithContext(ctx)

		// Log the request and user ID
		log.Printf("User %s accessed %s %s", userID, r.Method, r.URL)

		next.ServeHTTP(w, r)
	})
}

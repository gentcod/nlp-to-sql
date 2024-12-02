package chat

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gentcod/nlp-to-sql/token"
)

const (
	authorizationHeaderKey  = "Authorization"
	authorizationTypeBearer = "Bearer"
)

type contextKey string

const authorizationPayloadKey contextKey = "authorization_payload"

type AuthHandlerFunc func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)

// AuthMiddleware creates a gin middleware for authorization
func authMiddleware(tokenGenerator token.Generator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract the Authorization header
			authorizationHeader := r.Header.Get(authorizationHeaderKey)

			if len(authorizationHeader) == 0 {
				http.Error(w, "Authorization header is not provided", http.StatusUnauthorized)
				return
			}

			fields := strings.Fields(authorizationHeader)
			if len(fields) < 2 {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			authorizationType := strings.ToLower(fields[0])
			if authorizationType != authorizationTypeBearer {
				http.Error(w, fmt.Sprintf("Unsupported authorization type %s", authorizationType), http.StatusUnauthorized)
				return
			}

			accessToken := fields[1]
			payload, err := tokenGenerator.VerifyToken(accessToken)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), authorizationPayloadKey, payload)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gentcod/nlp-to-sql/token"
	"github.com/gin-gonic/gin"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

// AuthMiddleware creates a gin middleware for authorization.
// Supports both standard "Authorization: Bearer <token>" headers and "?token=<token>" query parameters (for browser WebSockets).
func authMiddleware(tokenGenerator token.Generator) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)

		var accessToken string
		if len(authorizationHeader) == 0 {
			// Fallback to query parameter for browser WebSocket upgrades
			if queryToken := strings.TrimSpace(ctx.Query("token")); queryToken != "" {
				accessToken = queryToken
			} else {
				err := errors.New("authorization header is not provided")
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, apiErrorResponse(err))
				return
			}
		} else {
			fields := strings.Fields(authorizationHeader)
			if len(fields) < 2 {
				err := errors.New("invalid authorization header format")
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, apiErrorResponse(err))
				return
			}

			authorizationType := strings.ToLower(fields[0])
			if authorizationType != authorizationTypeBearer {
				err := fmt.Errorf("unsuppoerted authorization type %s", authorizationType)
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, apiErrorResponse(err))
				return
			}

			accessToken = fields[1]
		}

		payload, err := tokenGenerator.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, apiErrorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

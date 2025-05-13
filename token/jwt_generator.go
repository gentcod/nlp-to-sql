package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

const minSecretKeySize = 32

// JWTGenerator is a JSON Web Token maker
type JWTGenerator struct {
	secretKey string
}

// NewJWTGenerator creates a new JWT Token Generator object
func NewJWTGenerator(secretKey string) (Generator, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %v characters", minSecretKeySize)
	}
	return &JWTGenerator{secretKey}, nil
}

// CreateToken creates a new token for a specific username and duration
func (maker *JWTGenerator) CreateToken(username string, userID uuid.UUID, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(username, userID, duration)
	if err != nil {
		return "", payload, err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString([]byte(maker.secretKey))
	return token, payload, err
}

// VerifyToken checks if the token is valid or not
func (maker *JWTGenerator) VerifyToken(token string) (*Payload, error) {
	//Returns the secretKey from a token and an error
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(maker.secretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		verr, ok := err.(*jwt.ValidationError)
		if ok && errors.Is(verr.Inner, ErrExpiredToken) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		return nil, ErrInvalidToken
	}

	return payload, nil
}

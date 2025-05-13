package token

import (
	"time"

	"github.com/google/uuid"
)

// Generator is an interface for managing tokens
type Generator interface {
	//CreateToken creates a new token for a specific username and duration
	CreateToken(username string, userID uuid.UUID, duration time.Duration) (string, *Payload, error)

	//VerifyToken checks if the token is valid or not
	VerifyToken(token string) (*Payload, error)
}

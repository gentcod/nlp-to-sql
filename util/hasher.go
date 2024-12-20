package util

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns the bcrypt hash of the password and an error.
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash passowrd: %w", err)
	}
	return string(hashedPassword), err
}

// CheckPassword checks if password and hashedpassword are the same.
func CheckPassword(password string, hashedPassowrd string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassowrd), []byte(password))
}

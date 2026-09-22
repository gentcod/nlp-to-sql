package mcp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"
)

// CredentialManager manages database credentials securely
type CredentialManager struct {
	encryptionKey []byte
	credentials   map[string]*EncryptedCredential
	mu            sync.RWMutex
	cacheTTL      time.Duration
}

// EncryptedCredential stores encrypted connection credentials
type EncryptedCredential struct {
	Encrypted string
	Nonce     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// DatabaseCredential represents database connection credentials
type DatabaseCredential struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Driver   string
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager(encryptionKey []byte) (*CredentialManager, error) {
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes")
	}

	return &CredentialManager{
		encryptionKey: encryptionKey,
		credentials:   make(map[string]*EncryptedCredential),
		cacheTTL:      5 * time.Minute,
	}, nil
}

// StoreCredential stores and encrypts credentials
func (cm *CredentialManager) StoreCredential(id string, cred *DatabaseCredential) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Serialize credentials
	credJSON := fmt.Sprintf(`{"host":"%s","port":%d,"user":"%s","password":"%s","database":"%s","driver":"%s"}`,
		cred.Host, cred.Port, cred.User, cred.Password, cred.Database, cred.Driver)

	// Encrypt
	encrypted, nonce, err := cm.encrypt([]byte(credJSON))
	if err != nil {
		return err
	}

	cm.credentials[id] = &EncryptedCredential{
		Encrypted: base64.StdEncoding.EncodeToString(encrypted),
		Nonce:     base64.StdEncoding.EncodeToString(nonce),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(cm.cacheTTL),
	}

	return nil
}

// encrypt encrypts data securely using AES-GCM
func (cm *CredentialManager) encrypt(data []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, data, nil)
	return ciphertext, nonce, nil
}

package mcp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// CredentialManager manages database credentials securely with explicit TTL state transitions
type CredentialManager struct {
	encryptionKey []byte
	credentials   map[string]*EncryptedCredential
	mu            sync.RWMutex
	cacheTTL      time.Duration
}

// EncryptedCredential stores encrypted connection credentials
type EncryptedCredential struct {
	Encrypted string    `json:"encrypted"`
	Nonce     string    `json:"nonce"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// DatabaseCredential represents database connection credentials
type DatabaseCredential struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	Driver   string `json:"driver"`
	URL      string `json:"url,omitempty"`
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

// SetCacheTTL updates the TTL duration for newly stored credentials
func (cm *CredentialManager) SetCacheTTL(ttl time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.cacheTTL = ttl
}

// StoreCredential stores and encrypts credentials (transitions state -> FRESH)
func (cm *CredentialManager) StoreCredential(id string, cred *DatabaseCredential) error {
	if id == "" {
		return fmt.Errorf("credential id cannot be empty")
	}
	if cred == nil {
		return fmt.Errorf("credential payload cannot be nil")
	}

	credJSON, err := json.Marshal(cred)
	if err != nil {
		return fmt.Errorf("failed to serialize credential: %w", err)
	}

	encrypted, nonce, err := cm.encrypt(credJSON)
	if err != nil {
		return err
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	cm.credentials[id] = &EncryptedCredential{
		Encrypted: base64.StdEncoding.EncodeToString(encrypted),
		Nonce:     base64.StdEncoding.EncodeToString(nonce),
		CreatedAt: now,
		ExpiresAt: now.Add(cm.cacheTTL),
	}

	return nil
}

// GetCredentialState returns the explicit CacheState of a stored credential
func (cm *CredentialManager) GetCredentialState(id string) CacheState {
	cm.mu.RLock()
	entry, ok := cm.credentials[id]
	cm.mu.RUnlock()

	if !ok {
		return CacheStateMissing
	}
	if time.Now().After(entry.ExpiresAt) {
		return CacheStateExpired
	}
	return CacheStateFresh
}

// GetCredential retrieves and decrypts a credential if its state is FRESH; evicts if EXPIRED
func (cm *CredentialManager) GetCredential(id string) (*DatabaseCredential, CacheState, error) {
	cm.mu.RLock()
	entry, ok := cm.credentials[id]
	cm.mu.RUnlock()

	if !ok {
		return nil, CacheStateMissing, fmt.Errorf("credential %q not found", id)
	}

	if time.Now().After(entry.ExpiresAt) {
		cm.mu.Lock()
		delete(cm.credentials, id)
		cm.mu.Unlock()
		return nil, CacheStateExpired, fmt.Errorf("credential %q has expired", id)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(entry.Encrypted)
	if err != nil {
		return nil, CacheStateFresh, fmt.Errorf("corrupted ciphertext: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(entry.Nonce)
	if err != nil {
		return nil, CacheStateFresh, fmt.Errorf("corrupted nonce: %w", err)
	}

	plaintext, err := cm.decrypt(ciphertext, nonce)
	if err != nil {
		return nil, CacheStateFresh, fmt.Errorf("decryption failed: %w", err)
	}

	var cred DatabaseCredential
	if err := json.Unmarshal(plaintext, &cred); err != nil {
		return nil, CacheStateFresh, fmt.Errorf("failed to unmarshal credential: %w", err)
	}

	return &cred, CacheStateFresh, nil
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

// decrypt decrypts AES-GCM data
func (cm *CredentialManager) decrypt(ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return aesgcm.Open(nil, nonce, ciphertext, nil)
}

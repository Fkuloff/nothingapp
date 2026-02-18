package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	errInvalidKey    = errors.New("encryption key must be exactly 32 bytes (AES-256)")
	errDecryptFailed = errors.New("decryption failed: invalid ciphertext or key")
)

// MessageEncryptor handles AES-256-GCM encryption for message text at rest.
// It is safe for concurrent use after initialization.
type MessageEncryptor struct {
	gcm cipher.AEAD
}

// NewMessageEncryptor creates an encryptor from a base64-encoded 32-byte key.
func NewMessageEncryptor(keyBase64 string) (*MessageEncryptor, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}
	if len(keyBytes) != 32 {
		return nil, errInvalidKey
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	return &MessageEncryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext and returns base64-encoded ciphertext and IV.
// Each call generates a unique random 12-byte nonce.
func (e *MessageEncryptor) Encrypt(plaintext string) (ciphertext, iv string, err error) {
	if plaintext == "" {
		return "", "", nil
	}

	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", fmt.Errorf("generate nonce: %w", err)
	}

	sealed := e.gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed),
		base64.StdEncoding.EncodeToString(nonce),
		nil
}

// Decrypt decrypts base64-encoded ciphertext with base64-encoded IV.
// Returns the original plaintext.
func (e *MessageEncryptor) Decrypt(ciphertextB64, ivB64 string) (string, error) {
	if ciphertextB64 == "" || ivB64 == "" {
		return ciphertextB64, nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(ivB64)
	if err != nil {
		return "", fmt.Errorf("decode IV: %w", err)
	}

	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errDecryptFailed
	}

	return string(plaintext), nil
}

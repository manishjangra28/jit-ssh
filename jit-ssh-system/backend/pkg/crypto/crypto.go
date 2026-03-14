package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

// GetMasterKey retrieves the 32-byte master key from the environment.
// It is used to encrypt and decrypt sensitive cloud integration credentials.
func GetMasterKey() ([]byte, error) {
	keyStr := os.Getenv("JIT_MASTER_KEY")
	if keyStr == "" {
		return nil, errors.New("JIT_MASTER_KEY environment variable is not set")
	}

	// First, attempt to decode it as a base64 string
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil || len(key) != 32 {
		// Fallback: assume the raw string itself is exactly 32 characters
		key = []byte(keyStr)
	}

	if len(key) != 32 {
		return nil, errors.New("JIT_MASTER_KEY must resolve to exactly 32 bytes for AES-256")
	}

	return key, nil
}

// Encrypt takes a plaintext byte slice and a 32-byte key,
// and returns a base64-encoded ciphertext string using AES-256-GCM.
func Encrypt(plaintext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a nonce. Nonce size is standard GCM size (12 bytes)
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Seal encrypts and authenticates plaintext, appending the result to nonce
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	// Return base64 encoded string for easy storage in text fields
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt takes a base64-encoded ciphertext string and a 32-byte key,
// and returns the original plaintext byte slice.
func Decrypt(encodedCiphertext string, key []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// Extract nonce and actual ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Open decrypts and authenticates ciphertext
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// EncryptString is a convenience wrapper around Encrypt for string inputs.
func EncryptString(plaintext string, key []byte) (string, error) {
	return Encrypt([]byte(plaintext), key)
}

// DecryptString is a convenience wrapper around Decrypt returning a string.
func DecryptString(encodedCiphertext string, key []byte) (string, error) {
	plaintext, err := Decrypt(encodedCiphertext, key)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

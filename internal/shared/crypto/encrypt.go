package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"golang.org/x/crypto/pbkdf2"
)

const (
	keyLen     = 32     // AES-256
	iterations = 100000 // PBKDF2 iterations
	saltLen    = 16
	nonceLen   = 12 // AES-GCM nonce
)

// DeriveKey derives a 32-byte AES key from a master key string using PBKDF2.
func DeriveKey(masterKey string, salt []byte) []byte {
	return pbkdf2.Key([]byte(masterKey), salt, iterations, keyLen, sha256.New)
}

// Encrypt encrypts plaintext using AES-256-GCM. Returns base64-encoded string
// containing: salt (16 bytes) + nonce (12 bytes) + ciphertext.
func Encrypt(plaintext []byte, masterKey string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key := DeriveKey(masterKey, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Concatenate: salt + nonce + ciphertext
	combined := make([]byte, 0, saltLen+nonceLen+len(ciphertext))
	combined = append(combined, salt...)
	combined = append(combined, nonce...)
	combined = append(combined, ciphertext...)

	return base64.StdEncoding.EncodeToString(combined), nil
}

// Decrypt decrypts a base64-encoded string produced by Encrypt.
func Decrypt(encoded string, masterKey string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	if len(data) < saltLen+nonceLen+1 {
		return nil, errors.New("ciphertext too short")
	}

	salt := data[:saltLen]
	nonce := data[saltLen : saltLen+nonceLen]
	ciphertext := data[saltLen+nonceLen:]

	key := DeriveKey(masterKey, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}

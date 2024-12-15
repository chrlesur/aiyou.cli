// internal/auth/storage/crypto.go
package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// Encrypter définit l'interface pour le chiffrement/déchiffrement
type Encrypter interface {
	// Encrypt chiffre les données
	Encrypt(data []byte) ([]byte, error)

	// Decrypt déchiffre les données
	Decrypt(data []byte) ([]byte, error)
}

// AESEncrypter implémente Encrypter avec AES-GCM
type AESEncrypter struct {
	key []byte
}

// NewAESEncrypter crée un nouveau chiffreur AES
func NewAESEncrypter(key []byte) (*AESEncrypter, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}
	return &AESEncrypter{key: key}, nil
}

// Encrypt implémente Encrypter.Encrypt
func (a *AESEncrypter) Encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Chiffrer et combiner nonce + données chiffrées
	ciphertext := aesgcm.Seal(nil, nonce, data, nil)
	return append(nonce, ciphertext...), nil
}

// Decrypt implémente Encrypter.Decrypt
func (a *AESEncrypter) Decrypt(data []byte) ([]byte, error) {
	if len(data) < NonceSize {
		return nil, ErrInvalidFormat
	}

	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := data[:NonceSize]
	ciphertext := data[NonceSize:]

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrCorruptedData
	}

	return plaintext, nil
}

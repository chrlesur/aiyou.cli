// internal/auth/storage/errors.go
package storage

import (
	"errors"
	"time"
)

var (
	ErrInvalidKey       = errors.New("invalid encryption key")
	ErrCorruptedData    = errors.New("corrupted or tampered data")
	ErrInvalidFormat    = errors.New("invalid file format")
	ErrPermissionDenied = errors.New("permission denied accessing storage")
)

const (
	// Tailles pour la cryptographie
	KeySize   = 32 // 256 bits
	NonceSize = 12 // 96 bits pour GCM

	// Valeurs par défaut
	DefaultTokenTTL = 24 * time.Hour
)

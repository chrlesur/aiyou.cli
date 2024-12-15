// internal/auth/storage/types.go
package storage

import (
	"context"
	"time"
)

// TokenData représente les informations de token à stocker
type TokenData struct {
	Token     string    `json:"token"`
	Expiry    time.Time `json:"expiry"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenStore définit l'interface pour le stockage des tokens
type TokenStore interface {
	// Save stocke un token de manière sécurisée
	Save(ctx context.Context, data TokenData) error

	// Load charge le token stocké
	Load(ctx context.Context) (*TokenData, error)

	// Clear supprime le token stocké
	Clear(ctx context.Context) error
}

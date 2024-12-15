// internal/auth/storage/factory.go
package storage

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
)

// StorageFactory gère la création des instances de TokenStore
type StorageFactory struct {
	configDir string
}

// NewStorageFactory crée une nouvelle factory
func NewStorageFactory(configDir string) *StorageFactory {
	return &StorageFactory{
		configDir: configDir,
	}
}

// CreateStore crée une nouvelle instance de TokenStore
func (f *StorageFactory) CreateStore() (TokenStore, error) {
	// Générer une clé unique pour ce store
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Créer le chemin du fichier de stockage
	storePath := filepath.Join(f.configDir, "auth", "tokens.dat")

	// Créer le store
	store, err := NewFileStore(storePath, key)
	if err != nil {
		return nil, fmt.Errorf("failed to create token store: %w", err)
	}

	return store, nil
}

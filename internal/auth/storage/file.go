// internal/auth/storage/file.go
package storage

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	// Permissions pour les fichiers de stockage
	// FilePerms = 0600 // Ne fonctionne pas correctement sur Windows
	FilePerms = 0666 // Plus compatible avec Windows
	DirPerms  = 0777 // Plus compatible avec Windows
)

// FileStore implémente TokenStore avec un stockage fichier chiffré
type FileStore struct {
	filePath  string
	encrypter Encrypter
	mu        sync.RWMutex
}

// NewFileStore crée une nouvelle instance de FileStore
func NewFileStore(filePath string, key []byte) (*FileStore, error) {
	encrypter, err := NewAESEncrypter(key)
	if err != nil {
		return nil, err
	}

	// Normaliser le chemin
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, DirPerms); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &FileStore{
		filePath:  absPath,
		encrypter: encrypter,
	}, nil
}

// Save implémente TokenStore.Save
func (f *FileStore) Save(ctx context.Context, data TokenData) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Sérialiser
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Chiffrer
	encrypted, err := f.encrypter.Encrypt(jsonData)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	// Écrire
	if err := os.WriteFile(f.filePath, encrypted, FilePerms); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load implémente TokenStore.Load
func (f *FileStore) Load(ctx context.Context) (*TokenData, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Lire
	encrypted, err := os.ReadFile(f.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Déchiffrer
	decrypted, err := f.encrypter.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	// Désérialiser
	var data TokenData
	if err := json.Unmarshal(decrypted, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %w", err)
	}

	return &data, nil
}

// Clear implémente TokenStore.Clear
func (f *FileStore) Clear(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	err := os.Remove(f.filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token file: %w", err)
	}
	return nil
}

// GetKeyPath retourne le chemin du fichier de clé
func GetKeyPath(configDir string) string {
	return filepath.Join(configDir, "auth", "key.dat")
}

// LoadOrCreateKey charge la clé existante ou en crée une nouvelle
func LoadOrCreateKey(configDir string) ([]byte, error) {
	keyPath := GetKeyPath(configDir)

	// Essayer de charger la clé existante
	key, err := os.ReadFile(keyPath)
	if err == nil && len(key) == KeySize {
		return key, nil
	}

	// Créer une nouvelle clé
	key = make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Assurer que le répertoire existe
	keyDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(keyDir, DirPerms); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	// Sauvegarder la clé
	if err := os.WriteFile(keyPath, key, FilePerms); err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}

	return key, nil
}

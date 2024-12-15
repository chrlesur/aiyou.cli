// internal/auth/storage/file_test.go
package storage

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	mathrand "math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	mathrand.Seed(time.Now().UnixNano())
}

func TestFileStore(t *testing.T) {
	// Créer un répertoire temporaire pour les tests
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "tokens.dat")

	// Générer une clé de test
	key := make([]byte, KeySize)
	_, err := cryptorand.Read(key)
	require.NoError(t, err)

	// Créer le store
	store, err := NewFileStore(filePath, key)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("save and load token data", func(t *testing.T) {
		data := TokenData{
			Token:     "test-token",
			Expiry:    time.Now().Add(time.Hour),
			CreatedAt: time.Now(),
		}

		// Sauvegarder
		err := store.Save(ctx, data)
		require.NoError(t, err)

		// Vérifier que le fichier existe
		info, err := os.Stat(filePath)
		require.NoError(t, err)

		// Sur Windows, les permissions peuvent être différentes
		if runtime.GOOS == "windows" {
			// Vérifier simplement que le fichier est accessible en lecture/écriture
			assert.True(t, info.Mode().Perm()&0600 != 0)
		} else {
			// Sur Unix, vérifier les permissions exactes
			assert.Equal(t, os.FileMode(FilePerms), info.Mode().Perm())
		}

		// Charger et vérifier
		loaded, err := store.Load(ctx)
		require.NoError(t, err)
		require.NotNil(t, loaded)

		assert.Equal(t, data.Token, loaded.Token)
		assert.WithinDuration(t, data.Expiry, loaded.Expiry, time.Second)
		assert.WithinDuration(t, data.CreatedAt, loaded.CreatedAt, time.Second)
	})

	t.Run("load non-existent file", func(t *testing.T) {
		// Supprimer le fichier s'il existe
		_ = os.Remove(filePath)

		loaded, err := store.Load(ctx)
		assert.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("clear token", func(t *testing.T) {
		// Sauvegarder des données
		err := store.Save(ctx, TokenData{Token: "test"})
		require.NoError(t, err)

		// Vérifier que le fichier existe
		_, err = os.Stat(filePath)
		require.NoError(t, err)

		// Clear
		err = store.Clear(ctx)
		require.NoError(t, err)

		// Vérifier que le fichier n'existe plus
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("invalid directory permissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		invalidDir := filepath.Join(os.TempDir(), "nonexistent", "subdir")
		_, err := NewFileStore(filepath.Join(invalidDir, "token.dat"), key)
		assert.Error(t, err)
	})

	t.Run("concurrent access", func(t *testing.T) {
		const goroutines = 10
		var wg sync.WaitGroup
		errors := make(chan error, goroutines)

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				data := TokenData{
					Token:     fmt.Sprintf("token-%d", id),
					CreatedAt: time.Now(),
					Expiry:    time.Now().Add(time.Hour),
				}

				// Ajouter un délai aléatoire pour mieux tester la concurrence
				time.Sleep(time.Duration(mathrand.Intn(100)) * time.Millisecond)

				if err := store.Save(ctx, data); err != nil {
					errors <- err
					return
				}

				// Ajouter un délai avant la lecture
				time.Sleep(time.Duration(mathrand.Intn(100)) * time.Millisecond)

				_, err := store.Load(ctx)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		// Attendre la fin de toutes les goroutines
		wg.Wait()
		close(errors)

		// Vérifier les erreurs
		for err := range errors {
			t.Errorf("Concurrent operation failed: %v", err)
		}
	})

	t.Run("data corruption detection", func(t *testing.T) {
		original := TokenData{
			Token:     "test-token",
			CreatedAt: time.Now(),
			Expiry:    time.Now().Add(time.Hour),
		}

		// Sauvegarder les données
		err := store.Save(ctx, original)
		require.NoError(t, err)

		// Corrompre le fichier
		f, err := os.OpenFile(filePath, os.O_WRONLY, 0600)
		require.NoError(t, err)
		_, err = f.WriteAt([]byte{0x00}, 10) // Écrire un octet nul au milieu du fichier
		f.Close()
		require.NoError(t, err)

		// Tenter de charger les données corrompues
		_, err = store.Load(ctx)
		assert.Error(t, err)
	})
}

// Benchmark pour les performances
func BenchmarkFileStore(b *testing.B) {
	tempDir := b.TempDir()
	filePath := filepath.Join(tempDir, "bench_tokens.dat")

	key := make([]byte, KeySize)
	_, err := cryptorand.Read(key)
	require.NoError(b, err)

	store, err := NewFileStore(filePath, key)
	require.NoError(b, err)

	ctx := context.Background()
	data := TokenData{
		Token:     "benchmark-token",
		CreatedAt: time.Now(),
		Expiry:    time.Now().Add(time.Hour),
	}

	b.Run("save", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := store.Save(ctx, data)
			require.NoError(b, err)
		}
	})

	b.Run("load", func(b *testing.B) {
		// Préparer les données
		err := store.Save(ctx, data)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := store.Load(ctx)
			require.NoError(b, err)
		}
	})
}

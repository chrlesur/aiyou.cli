// internal/auth/storage/crypto_test.go
package storage

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAESEncrypter(t *testing.T) {
	// Générer une clé de test
	key := make([]byte, KeySize)
	_, err := rand.Read(key)
	require.NoError(t, err)

	t.Run("encrypt and decrypt", func(t *testing.T) {
		encrypter, err := NewAESEncrypter(key)
		require.NoError(t, err)

		original := []byte("test data")

		// Chiffrer
		encrypted, err := encrypter.Encrypt(original)
		require.NoError(t, err)
		assert.NotEqual(t, original, encrypted)

		// Déchiffrer
		decrypted, err := encrypter.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, original, decrypted)
	})

	t.Run("invalid key size", func(t *testing.T) {
		_, err := NewAESEncrypter([]byte("too short"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("corrupted data", func(t *testing.T) {
		encrypter, err := NewAESEncrypter(key)
		require.NoError(t, err)

		encrypted, err := encrypter.Encrypt([]byte("test"))
		require.NoError(t, err)

		// Corrompre les données
		encrypted[len(encrypted)-1]++

		_, err = encrypter.Decrypt(encrypted)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrCorruptedData)
	})
}

// Package memory_test contient les tests pour l'implémentation en mémoire du cache.
package memory

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/chrlesur/aiyou.cli/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMemoryCache vérifie la création correcte d'une instance de cache.
func TestNewMemoryCache(t *testing.T) {
	t.Run("création avec configuration valide", func(t *testing.T) {
		config := cache.DefaultConfig()
		c, err := NewMemoryCache(config)
		require.NoError(t, err)
		require.NotNil(t, c)
		defer c.Close()

		stats := c.GetStats(context.Background())
		assert.Equal(t, 0, stats.ItemCount)
		assert.Equal(t, int64(0), stats.Size)
	})

	t.Run("vérification des valeurs par défaut", func(t *testing.T) {
		config := cache.DefaultConfig()
		c, err := NewMemoryCache(config)
		require.NoError(t, err)
		defer c.Close()

		mc := c.(*memoryCache)
		assert.Equal(t, config.MaxSize, mc.config.MaxSize)
		assert.Equal(t, config.CleanupInterval, mc.config.CleanupInterval)
	})
}

// TestMemoryCache_Set teste les opérations de stockage dans le cache.
func TestMemoryCache_Set(t *testing.T) {
	ctx := context.Background()
	config := cache.DefaultConfig()
	c, err := NewMemoryCache(config)
	require.NoError(t, err)
	defer c.Close()

	tests := []struct {
		name      string
		key       string
		value     interface{}
		cacheType cache.CacheType
		wantErr   bool
	}{
		{
			name:      "stockage valeur simple",
			key:       "test1",
			value:     "value1",
			cacheType: cache.QueryCache,
			wantErr:   false,
		},
		{
			name:      "clé vide",
			key:       "",
			value:     "value2",
			cacheType: cache.QueryCache,
			wantErr:   true,
		},
		{
			name:      "valeur nil",
			key:       "test3",
			value:     nil,
			cacheType: cache.QueryCache,
			wantErr:   true,
		},
		{
			name:      "grande valeur",
			key:       "test4",
			value:     make([]byte, config.MaxSize+1),
			cacheType: cache.QueryCache,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Set(ctx, tt.key, tt.value, tt.cacheType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				entry, exists := c.Get(ctx, tt.key)
				assert.True(t, exists)
				assert.Equal(t, tt.value, entry.Value)
			}
		})
	}
}

// TestMemoryCache_Get teste les opérations de récupération du cache.
func TestMemoryCache_Get(t *testing.T) {
	ctx := context.Background()
	c, err := NewMemoryCache(cache.DefaultConfig())
	require.NoError(t, err)
	defer c.Close()

	t.Run("vérification statistiques", func(t *testing.T) {
		// Réinitialisation des stats
		err := c.Clear(ctx)
		require.NoError(t, err)

		err = c.Set(ctx, "stats", "value", cache.QueryCache)
		require.NoError(t, err)

		// Premier accès (hit)
		_, exists := c.Get(ctx, "stats")
		require.True(t, exists)

		// Accès manqué (miss)
		_, exists = c.Get(ctx, "missing")
		require.False(t, exists)

		stats := c.GetStats(ctx)
		assert.Equal(t, int64(1), stats.HitCount, "Hit count devrait être 1")
		assert.Equal(t, int64(1), stats.MissCount, "Miss count devrait être 1")
	})
}

// TestMemoryCache_Delete teste les opérations de suppression du cache.
func TestMemoryCache_Delete(t *testing.T) {
	ctx := context.Background()
	c, err := NewMemoryCache(cache.DefaultConfig())
	require.NoError(t, err)
	defer c.Close()

	t.Run("suppression valeur existante", func(t *testing.T) {
		err := c.Set(ctx, "key", "value", cache.QueryCache)
		require.NoError(t, err)

		err = c.Delete(ctx, "key")
		assert.NoError(t, err)

		_, exists := c.Get(ctx, "key")
		assert.False(t, exists)
	})

	t.Run("suppression valeur inexistante", func(t *testing.T) {
		err = c.Delete(ctx, "nonexistent")
		assert.NoError(t, err)
	})

	t.Run("vérification taille après suppression", func(t *testing.T) {
		err := c.Set(ctx, "size_test", "value", cache.QueryCache)
		require.NoError(t, err)

		statsBefore := c.GetStats(ctx)
		err = c.Delete(ctx, "size_test")
		require.NoError(t, err)
		statsAfter := c.GetStats(ctx)

		assert.Greater(t, statsBefore.Size, statsAfter.Size)
	})
}

// TestMemoryCache_Clear teste le nettoyage complet du cache.
func TestMemoryCache_Clear(t *testing.T) {
	ctx := context.Background()
	c, err := NewMemoryCache(cache.DefaultConfig())
	require.NoError(t, err)
	defer c.Close()

	// Ajout de plusieurs entrées
	for i := 0; i < 5; i++ {
		err := c.Set(ctx, fmt.Sprintf("key%d", i), i, cache.QueryCache)
		require.NoError(t, err)
	}

	// Vérification avant Clear
	statsBefore := c.GetStats(ctx)
	assert.Greater(t, statsBefore.ItemCount, 0)

	// Clear
	err = c.Clear(ctx)
	require.NoError(t, err)

	// Vérification après Clear
	statsAfter := c.GetStats(ctx)
	assert.Equal(t, 0, statsAfter.ItemCount)
	assert.Equal(t, int64(0), statsAfter.Size)
}

// TestMemoryCache_Concurrent teste les accès concurrents au cache.
func TestMemoryCache_Concurrent(t *testing.T) {
	ctx := context.Background()
	c, err := NewMemoryCache(cache.DefaultConfig())
	require.NoError(t, err)
	defer c.Close()

	const goroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key%d-%d", id, j)

				// Set
				err := c.Set(ctx, key, j, cache.QueryCache)
				assert.NoError(t, err)

				// Get
				_, _ = c.Get(ctx, key)

				// Delete
				err = c.Delete(ctx, key)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()
}

// TestMemoryCache_Cleanup teste le nettoyage automatique des entrées expirées.
func TestMemoryCache_Cleanup(t *testing.T) {
	ctx := context.Background()
	config := cache.DefaultConfig()
	config.CleanupInterval = 50 * time.Millisecond
	config.QueryTTL = 100 * time.Millisecond

	c, err := NewMemoryCache(config)
	require.NoError(t, err)
	defer c.Close()

	// Ajout d'entrées
	err = c.Set(ctx, "test", "value", cache.QueryCache)
	require.NoError(t, err)

	// Vérification initiale
	statsBefore := c.GetStats(ctx)
	assert.Equal(t, 1, statsBefore.ItemCount)

	// Attente de l'expiration et du nettoyage
	time.Sleep(200 * time.Millisecond)

	// Vérification après nettoyage
	statsAfter := c.GetStats(ctx)
	assert.Equal(t, 0, statsAfter.ItemCount)
}

// TestMemoryCache_Context teste la gestion du contexte.
func TestMemoryCache_Context(t *testing.T) {
	c, err := NewMemoryCache(cache.DefaultConfig())
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Annulation immédiate du contexte

	t.Run("Set avec contexte annulé", func(t *testing.T) {
		err := c.Set(ctx, "key", "value", cache.QueryCache)
		assert.Error(t, err)
	})

	t.Run("Get avec contexte annulé", func(t *testing.T) {
		_, exists := c.Get(ctx, "key")
		assert.False(t, exists)
	})

	t.Run("Delete avec contexte annulé", func(t *testing.T) {
		err := c.Delete(ctx, "key")
		assert.Error(t, err)
	})

	t.Run("Clear avec contexte annulé", func(t *testing.T) {
		err := c.Clear(ctx)
		assert.Error(t, err)
	})
}

// BenchmarkMemoryCache_Set effectue des tests de performance pour l'opération Set.
func BenchmarkMemoryCache_Set(b *testing.B) {
	ctx := context.Background()
	c, _ := NewMemoryCache(cache.DefaultConfig())
	defer c.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		_ = c.Set(ctx, key, "value", cache.QueryCache)
	}
}

// BenchmarkMemoryCache_Get effectue des tests de performance pour l'opération Get.
func BenchmarkMemoryCache_Get(b *testing.B) {
	ctx := context.Background()
	c, _ := NewMemoryCache(cache.DefaultConfig())
	defer c.Close()

	// Préparation des données
	_ = c.Set(ctx, "bench_key", "value", cache.QueryCache)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Get(ctx, "bench_key")
	}
}

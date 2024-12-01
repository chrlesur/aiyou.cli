// Package memory fournit une implémentation en mémoire du système de cache.
// Cette implémentation est thread-safe et gère automatiquement l'expiration des entrées.
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chrlesur/aiyou.cli/internal/cache"
)

// memoryCache implémente l'interface cache.Cache avec un stockage en mémoire.
// Elle utilise une map pour stocker les données et assure la thread-safety via mutex.
type memoryCache struct {
	// mu protège l'accès concurrent aux données du cache
	mu sync.RWMutex

	// items stocke les entrées du cache avec leur clé associée
	items map[string]cache.CacheEntry

	// config contient les paramètres de configuration du cache
	config cache.CacheConfig

	// stats maintient les statistiques d'utilisation du cache
	stats cache.CacheStats

	// stopChan est utilisé pour signaler l'arrêt de la routine de nettoyage
	stopChan chan struct{}
}

// NewMemoryCache crée et initialise une nouvelle instance de cache en mémoire.
// Il démarre également la routine de nettoyage automatique des entrées expirées.
//
// Paramètres :
// - config: Configuration du cache définissant les TTL et autres paramètres
//
// Retourne :
// - Une instance de Cache
// - Une erreur si l'initialisation échoue
func NewMemoryCache(config cache.CacheConfig) (cache.Cache, error) {
	mc := &memoryCache{
		items:    make(map[string]cache.CacheEntry),
		config:   config,
		stopChan: make(chan struct{}),
	}

	// Démarrage de la routine de nettoyage en arrière-plan
	go mc.startCleanup()

	return mc, nil
}

// Set ajoute ou met à jour une entrée dans le cache.
//
// Paramètres :
// - ctx: Contexte pour la gestion de l'annulation
// - key: Clé unique identifiant l'entrée
// - value: Valeur à mettre en cache
// - cacheType: Type de l'entrée déterminant sa durée de vie
//
// Retourne une erreur si :
// - Le contexte est annulé
// - La clé est invalide
// - La taille maximale du cache serait dépassée
func (m *memoryCache) Set(ctx context.Context, key string, value interface{}, cacheType cache.CacheType) error {
	// Vérification du contexte
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error in Set: %w", err)
	}

	// Validation de la clé
	if key == "" {
		return cache.ErrCacheKeyInvalid
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Calcul de la taille de la nouvelle entrée
	size, err := calculateSize(value)
	if err != nil {
		return fmt.Errorf("failed to calculate entry size: %w", err)
	}

	// Vérification de la limite de taille
	if m.stats.Size+size > m.config.MaxSize {
		return cache.ErrCacheFull
	}

	// Détermination du TTL selon le type
	ttl := m.getTTLForType(cacheType)

	// Création de l'entrée
	entry := cache.CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(ttl),
		Type:       cacheType,
		Size:       size,
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
	}

	// Mise à jour du cache et des statistiques
	m.items[key] = entry
	m.updateStats(size, true)

	return nil
}

// Get récupère une entrée du cache.
//
// Paramètres :
// - ctx: Contexte pour la gestion de l'annulation
// - key: Clé de l'entrée à récupérer
//
// Retourne :
// - L'entrée trouvée
// - Un booléen indiquant si l'entrée existe et n'est pas expirée
func (m *memoryCache) Get(ctx context.Context, key string) (cache.CacheEntry, bool) {
	if err := ctx.Err(); err != nil {
		return cache.CacheEntry{}, false
	}

	m.mu.Lock() // Changé de RLock à Lock car nous modifions les stats
	defer m.mu.Unlock()

	entry, exists := m.items[key]
	if !exists {
		m.stats.MissCount++
		return cache.CacheEntry{}, false
	}

	// Vérification de l'expiration
	if time.Now().After(entry.Expiration) {
		m.stats.MissCount++
		delete(m.items, key) // Nettoyage de l'entrée expirée
		return cache.CacheEntry{}, false
	}

	// Mise à jour du dernier accès et des statistiques
	entry.LastAccess = time.Now()
	m.items[key] = entry
	m.stats.HitCount++

	return entry, true
}

// Delete supprime une entrée du cache.
//
// Paramètres :
// - ctx: Contexte pour la gestion de l'annulation
// - key: Clé de l'entrée à supprimer
//
// Retourne une erreur si le contexte est annulé
func (m *memoryCache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error in Delete: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, exists := m.items[key]; exists {
		m.updateStats(entry.Size, false)
		delete(m.items, key)
	}

	return nil
}

// Clear supprime toutes les entrées du cache.
//
// Paramètres :
// - ctx: Contexte pour la gestion de l'annulation
//
// Retourne une erreur si le contexte est annulé
func (m *memoryCache) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error in Clear: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.items = make(map[string]cache.CacheEntry)
	m.stats = cache.CacheStats{}

	return nil
}

// GetStats retourne les statistiques actuelles du cache.
//
// Paramètres :
// - ctx: Contexte pour la gestion de l'annulation
//
// Retourne les statistiques d'utilisation du cache
func (m *memoryCache) GetStats(ctx context.Context) cache.CacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// Close arrête proprement le cache en arrêtant la routine de nettoyage.
//
// Retourne toujours nil car l'opération ne peut pas échouer
func (m *memoryCache) Close() error {
	close(m.stopChan)
	return nil
}

// startCleanup lance la routine de nettoyage périodique des entrées expirées.
// Cette méthode s'exécute dans une goroutine séparée.
func (m *memoryCache) startCleanup() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("Recovered from panic in cleanup: %v\n", r)
					}
				}()
				m.cleanup()
			}()
		case <-m.stopChan:
			return
		}
	}
}

// cleanup effectue le nettoyage des entrées expirées du cache.
// Cette méthode est appelée périodiquement par startCleanup.
func (m *memoryCache) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, entry := range m.items {
		if now.After(entry.Expiration) {
			m.updateStats(entry.Size, false)
			delete(m.items, key)
		}
	}
}

// updateStats met à jour les statistiques du cache lors de l'ajout ou
// de la suppression d'entrées.
//
// Paramètres :
// - size: Taille de l'entrée en bytes
// - isAdd: true pour un ajout, false pour une suppression
func (m *memoryCache) updateStats(size int64, isAdd bool) {
	if isAdd {
		m.stats.Size += size
		m.stats.ItemCount++
	} else {
		m.stats.Size -= size
		m.stats.ItemCount--
	}
}

// getTTLForType retourne la durée de vie appropriée selon le type de cache.
//
// Paramètres :
// - cacheType: Type de l'entrée
//
// Retourne la durée de vie configurée pour ce type
func (m *memoryCache) getTTLForType(cacheType cache.CacheType) time.Duration {
	switch cacheType {
	case cache.AssistantCache:
		return m.config.AssistantTTL
	case cache.QueryCache:
		return m.config.QueryTTL
	case cache.ModelCache:
		return m.config.ModelTTL
	case cache.UserCache:
		return m.config.UserTTL
	case cache.ConfigCache:
		return m.config.ConfigTTL
	default:
		return m.config.DefaultTTL
	}
}

// calculateSize estime la taille en bytes d'une valeur.
//
// Paramètres :
// - value: Valeur dont on veut estimer la taille
//
// Retourne :
// - La taille estimée en bytes
// - Une erreur si la taille ne peut pas être calculée
func calculateSize(value interface{}) (int64, error) {
	switch v := value.(type) {
	case string:
		return int64(len(v)), nil
	case []byte:
		return int64(len(v)), nil
	case nil:
		return 0, cache.ErrCacheValueInvalid
	default:
		// Estimation par défaut pour les autres types
		// Note: Cette estimation pourrait être améliorée avec une réflexion plus précise
		return 100, nil
	}
}

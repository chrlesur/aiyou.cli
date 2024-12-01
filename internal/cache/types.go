// Package cache fournit une interface et une implémentation pour la gestion du cache
// dans l'application AI.YOU CLI. Il permet de stocker temporairement différents types
// de données avec des durées de vie configurables et une gestion automatique de l'expiration.
package cache

import (
	"context"
	"errors"
	"time"
)

// Définition des erreurs standard du système de cache.
// Ces erreurs sont utilisées pour identifier clairement les problèmes courants.
var (
	// ErrCacheKeyNotFound est retourné lorsqu'une clé n'existe pas dans le cache
	ErrCacheKeyNotFound = errors.New("cache key not found")

	// ErrCacheKeyInvalid est retourné lorsqu'une clé de cache est invalide (vide par exemple)
	ErrCacheKeyInvalid = errors.New("invalid cache key")

	// ErrCacheValueInvalid est retourné lorsqu'une valeur ne peut pas être mise en cache
	ErrCacheValueInvalid = errors.New("invalid cache value")

	// ErrCacheFull est retourné lorsque le cache a atteint sa taille maximale
	ErrCacheFull = errors.New("cache is full")
)

// CacheType représente les différents types de données pouvant être mis en cache.
// Chaque type peut avoir des paramètres de durée de vie différents.
type CacheType string

// Définition des différents types de cache supportés par le système.
const (
	// AssistantCache stocke les informations relatives aux assistants AI
	AssistantCache CacheType = "assistant"

	// QueryCache stocke les résultats des requêtes fréquentes
	QueryCache CacheType = "query"

	// ModelCache stocke les métadonnées des modèles AI
	ModelCache CacheType = "model"

	// UserCache stocke les informations temporaires sur l'utilisateur
	UserCache CacheType = "user"

	// ConfigCache stocke les configurations qui changent peu fréquemment
	ConfigCache CacheType = "config"
)

// CacheEntry représente une entrée individuelle dans le cache.
// Elle contient la valeur stockée ainsi que des métadonnées associées.
type CacheEntry struct {
	// Value contient la donnée mise en cache
	Value interface{}

	// Expiration définit le moment où l'entrée devient invalide
	Expiration time.Time

	// Type identifie le type de donnée stockée
	Type CacheType

	// Size représente la taille approximative en bytes de l'entrée
	Size int64

	// CreatedAt est l'horodatage de création de l'entrée
	CreatedAt time.Time

	// LastAccess est l'horodatage du dernier accès à l'entrée
	LastAccess time.Time
}

// CacheStats maintient des statistiques sur l'utilisation du cache.
// Ces informations sont utiles pour le monitoring et l'optimisation.
type CacheStats struct {
	// ItemCount représente le nombre total d'éléments dans le cache
	ItemCount int

	// Size représente la taille totale occupée par le cache en bytes
	Size int64

	// HitCount compte le nombre d'accès réussis au cache
	HitCount int64

	// MissCount compte le nombre d'accès manqués au cache
	MissCount int64
}

// CacheConfig définit les paramètres de configuration du cache.
// Elle permet de personnaliser le comportement du cache selon les besoins.
type CacheConfig struct {
	// Durées de vie pour chaque type de cache
	AssistantTTL time.Duration
	QueryTTL     time.Duration
	ModelTTL     time.Duration
	UserTTL      time.Duration
	ConfigTTL    time.Duration
	DefaultTTL   time.Duration

	// MaxSize définit la taille maximale du cache en bytes
	MaxSize int64

	// CleanupInterval définit la fréquence de nettoyage des entrées expirées
	CleanupInterval time.Duration
}

// DefaultConfig retourne une configuration par défaut pour le cache.
// Ces valeurs peuvent être ajustées selon les besoins spécifiques.
func DefaultConfig() CacheConfig {
	return CacheConfig{
		AssistantTTL:    time.Hour,        // 1 heure pour les assistants
		QueryTTL:        15 * time.Minute, // 15 minutes pour les requêtes
		ModelTTL:        24 * time.Hour,   // 24 heures pour les modèles
		UserTTL:         time.Hour,        // 1 heure pour les données utilisateur
		ConfigTTL:       12 * time.Hour,   // 12 heures pour les configurations
		DefaultTTL:      time.Hour,        // 1 heure par défaut
		MaxSize:         10 * 1024 * 1024, // 10MB maximum
		CleanupInterval: 10 * time.Minute, // Nettoyage toutes les 10 minutes
	}
}

// Cache définit l'interface pour toutes les opérations de cache supportées.
// Cette interface peut être implémentée par différents backends de stockage.
type Cache interface {
	// Set ajoute ou met à jour une entrée dans le cache.
	// Retourne une erreur si l'opération échoue.
	Set(ctx context.Context, key string, value interface{}, cacheType CacheType) error

	// Get récupère une entrée du cache.
	// Retourne l'entrée et un booléen indiquant si elle a été trouvée.
	Get(ctx context.Context, key string) (CacheEntry, bool)

	// Delete supprime une entrée du cache.
	// Retourne une erreur si l'opération échoue.
	Delete(ctx context.Context, key string) error

	// Clear vide entièrement le cache.
	// Retourne une erreur si l'opération échoue.
	Clear(ctx context.Context) error

	// GetStats retourne les statistiques actuelles du cache.
	GetStats(ctx context.Context) CacheStats

	// Close ferme proprement le cache et libère les ressources.
	// Doit être appelé lors de l'arrêt de l'application.
	Close() error
}

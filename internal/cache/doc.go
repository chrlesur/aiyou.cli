// internal/cache/doc.go

// Package cache fournit une solution de mise en cache pour l'application AI.YOU CLI.
//
// # Architecture
//
// Le package est organisé autour d'une interface Cache centrale qui définit
// les opérations standard de cache. L'implémentation actuelle (memoryCache)
// fournit un stockage en mémoire avec :
// - Gestion automatique de l'expiration des entrées
// - Support de différents types de données avec TTL configurables
// - Métriques et statistiques d'utilisation
// - Thread-safety pour les accès concurrents
//
// Utilisation
//
//	// Création d'un nouveau cache avec la configuration par défaut
//	config := cache.DefaultConfig()
//	cache, err := memory.NewMemoryCache(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer cache.Close()
//
//	// Stockage d'une valeur
//	err = cache.Set(ctx, "key", "value", cache.QueryCache)
//
//	// Récupération d'une valeur
//	entry, exists := cache.Get(ctx, "key")
//
// # Configuration
//
// Le cache peut être configuré via la structure CacheConfig qui permet de définir :
// - Les durées de vie (TTL) pour chaque type de données
// - La taille maximale du cache
// - L'intervalle de nettoyage des entrées expirées
//
// # Monitoring
//
// Le cache fournit des statistiques d'utilisation via la méthode GetStats :
// - Nombre d'entrées
// - Taille totale utilisée
// - Taux de succès/échecs
//
// # Thread-Safety
//
// Toutes les opérations du cache sont thread-safe et peuvent être utilisées
// en toute sécurité depuis plusieurs goroutines.
package cache

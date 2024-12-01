// internal/cache/example_test.go
package cache_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chrlesur/aiyou.cli/internal/cache"
	"github.com/chrlesur/aiyou.cli/internal/cache/memory"
)

func Example() {
	// Création d'un nouveau cache avec la configuration par défaut
	config := cache.DefaultConfig()
	c, err := memory.NewMemoryCache(config)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()

	// Stockage d'une valeur
	err = c.Set(ctx, "greeting", "Hello, World!", cache.QueryCache)
	if err != nil {
		log.Fatal(err)
	}

	// Récupération de la valeur
	entry, exists := c.Get(ctx, "greeting")
	if exists {
		fmt.Println(entry.Value)
	}
	// Output: Hello, World!
}

func Example_withExpiration() {
	config := cache.DefaultConfig()
	config.QueryTTL = 1 * time.Second
	c, _ := memory.NewMemoryCache(config)
	defer c.Close()

	ctx := context.Background()

	c.Set(ctx, "temp", "temporary value", cache.QueryCache)

	entry, exists := c.Get(ctx, "temp")
	fmt.Printf("Immediately: %v\n", exists)

	time.Sleep(2 * time.Second)

	entry, exists = c.Get(ctx, "temp")
	fmt.Printf("After expiration: %v\n", exists)

	// Output:
	// Immediately: true
	// After expiration: false
}

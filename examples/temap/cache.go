package main

import (
	"fmt"
	"time"

	"github.com/majiddarvishan/snipgo/temap"
)

type Cache struct {
	data   *temap.TimedMap
	hits   int64
	misses int64
}

func NewCache() *Cache {
	return &Cache{
		data: temap.NewWithCapacity(10000, func(key, value any) {
			fmt.Printf("Cache entry expired: %s\n", key)
		}),
	}
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.data.SetTemporary(key, value, ttl)
}

func (c *Cache) Get(key string) (interface{}, bool) {
	val, ok := c.data.Get(key)
	if ok {
		c.hits++
	} else {
		c.misses++
	}
	return val, ok
}

func (c *Cache) Stats() (hits, misses int64, hitRate float64) {
	total := c.hits + c.misses
	if total == 0 {
		return 0, 0, 0
	}
	return c.hits, c.misses, float64(c.hits) / float64(total) * 100
}

func (c *Cache) Size() int {
	return c.data.Size()
}

func main() {
	cache := NewCache()

	// Store data with different TTLs
	cache.Set("fast", "data1", 2*time.Second)
	cache.Set("medium", "data2", 5*time.Second)
	cache.Set("slow", "data3", 10*time.Second)

	fmt.Println("Cache size:", cache.Size())

	// Simulate access patterns
	for i := 0; i < 10; i++ {
		keys := []string{"fast", "medium", "slow", "nonexistent"}
		for _, key := range keys {
			if val, ok := cache.Get(key); ok {
				fmt.Printf("Hit: %s = %v\n", key, val)
			}
		}
		time.Sleep(1 * time.Second)
	}

	hits, misses, hitRate := cache.Stats()
	fmt.Printf("\nCache Stats:\n")
	fmt.Printf("Hits: %d\n", hits)
	fmt.Printf("Misses: %d\n", misses)
	fmt.Printf("Hit Rate: %.2f%%\n", hitRate)
}

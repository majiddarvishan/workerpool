package temap_test

import (
	"fmt"
	"time"

	"github.com/majiddarvishan/snipgo/temap"
)

// Example demonstrates basic usage of temap
func Example() {
	m := temap.New(func(key, value any) {
		fmt.Printf("Expired: %s\n", key)
	})

	// Set temporary values
	m.SetTemporary("session", "user123", 2*time.Second)

	// Set permanent values
	m.SetPermanent("config", "permanent_data")

	// Get values
	if val, ok := m.Get("session"); ok {
		fmt.Println("Found:", val)
	}

	fmt.Println("Size:", m.Size())

	// Output:
	// Found: user123
	// Size: 2
}

// Example_batchOperations demonstrates batch operations
func Example_batchOperations() {
	m := temap.NewWithCapacity(100, nil)

	// Set multiple items
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		m.SetTemporary(key, i*10, 5*time.Second)
	}

	// Get multiple items at once
	keys := []string{"key0", "key1", "key2"}
	values := m.GetMultiple(keys)
	fmt.Printf("Found %d values\n", len(values))

	// Remove multiple items at once
	removed := m.RemoveMultiple([]string{"key0", "key1", "key2"})
	fmt.Printf("Removed %d items\n", removed)

	// Output:
	// Found 3 values
	// Removed 3 items
}

// Example_setExpiry demonstrates changing TTL
func Example_setExpiry() {
	m := temap.New(nil)

	// Set initial TTL
	m.SetTemporary("session", "active", 5*time.Second)

	// Extend TTL later
	newExpiry := time.Now().Add(10 * time.Second)
	if m.SetExpiry("session", newExpiry) {
		fmt.Println("Expiry updated")
	}

	// Output:
	// Expiry updated
}

// Example_forEach demonstrates iteration
func Example_forEach() {
	m := temap.New(nil)

	m.SetPermanent("user1", "Alice")
	m.SetPermanent("user2", "Bob")
	m.SetPermanent("user3", "Charlie")

	// Get all keys first to ensure deterministic output
	keys := m.Keys()
	fmt.Printf("Total keys: %d\n", len(keys))

	// Output:
	// Total keys: 3
}

// Example_highPerformance demonstrates high-performance configuration
func Example_highPerformance() {
	// For maximum performance with known workload
	m := temap.NewWithShards(32, 1000, nil)

	// Pre-populate
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("item_%d", i)
		m.SetTemporary(key, i, 1*time.Minute)
	}

	fmt.Println("Items:", m.Size())
	// Output:
	// Items: 10000
}

// Example_sessionManagement demonstrates a real-world use case
func Example_sessionManagement() {
	// Create session store with 30 minute TTL
	sessions := temap.New(func(key, value any) {
		fmt.Printf("Session %s expired\n", key)
	})

	// Create session
	sessions.SetTemporary("sess_123", map[string]string{
		"user_id": "user_456",
		"role":    "admin",
	}, 30*time.Minute)

	// Get session
	if val, ok := sessions.Get("sess_123"); ok {
		sessionData := val.(map[string]string)
		fmt.Printf("User: %s, Role: %s\n", sessionData["user_id"], sessionData["role"])
	}

	// Output:
	// User: user_456, Role: admin
}

// Example_cache demonstrates cache implementation
func Example_cache() {
	cache := temap.NewWithCapacity(1000, func(key, value any) {
		fmt.Printf("Cache miss: %s\n", key)
	})

	// Cache some data
	cache.SetTemporary("user:123", "John Doe", 5*time.Minute)
	cache.SetTemporary("user:456", "Jane Smith", 5*time.Minute)

	// Retrieve from cache
	if val, ok := cache.Get("user:123"); ok {
		fmt.Println("Cached:", val)
	}

	fmt.Println("Cache size:", cache.Size())

	// Output:
	// Cached: John Doe
	// Cache size: 2
}

// Example_rateLimiter demonstrates rate limiting
func Example_rateLimiter() {
	limiter := temap.New(nil)
	maxRequests := 5
	window := 1 * time.Minute

	checkLimit := func(clientID string) bool {
		count := 0
		if val, ok := limiter.Get(clientID); ok {
			count = val.(int)
		}

		if count >= maxRequests {
			return false // Rate limited
		}

		limiter.SetTemporary(clientID, count+1, window)
		return true // Allowed
	}

	// Simulate requests
	for i := 0; i < 7; i++ {
		if checkLimit("client1") {
			fmt.Printf("Request %d: Allowed\n", i+1)
		} else {
			fmt.Printf("Request %d: Rate limited\n", i+1)
		}
	}

	// Output:
	// Request 1: Allowed
	// Request 2: Allowed
	// Request 3: Allowed
	// Request 4: Allowed
	// Request 5: Allowed
	// Request 6: Rate limited
	// Request 7: Rate limited
}

// Example_permanentAndTemporary demonstrates mixing permanent and temporary keys
func Example_permanentAndTemporary() {
	m := temap.New(nil)

	// Permanent configuration
	m.SetPermanent("app_name", "MyApp")
	m.SetPermanent("version", "1.0.0")

	// Temporary data
	m.SetTemporary("active_users", 42, 1*time.Minute)
	m.SetTemporary("cache_hit_rate", 0.95, 30*time.Second)

	fmt.Println("Total items:", m.Size())

	// Output:
	// Total items: 4
}

// Example_updateExisting demonstrates updating existing keys
func Example_updateExisting() {
	m := temap.New(nil)

	// Set initial value
	m.SetTemporary("counter", 1, 10*time.Second)

	// Update to new value
	m.SetTemporary("counter", 2, 10*time.Second)

	// Convert to permanent
	m.SetPermanent("counter", 3)

	if val, ok := m.Get("counter"); ok {
		fmt.Println("Counter:", val)
	}

	// Output:
	// Counter: 3
}

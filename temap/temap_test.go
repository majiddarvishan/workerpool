package temap

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// Test: Basic functionality
func TestBasicOperations(t *testing.T) {
	m := New(nil)

	// Test SetPermanent and Get
	m.SetPermanent("key1", "value1")
	if val, ok := m.Get("key1"); !ok || val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}

	// Test SetTemporary
	m.SetTemporary("key2", "value2", 100*time.Millisecond)
	if val, ok := m.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}

	// Test Size
	if m.Size() != 2 {
		t.Errorf("Expected size 2, got %d", m.Size())
	}

	// Test Remove
	if !m.Remove("key1") {
		t.Error("Remove should return true")
	}
	if _, ok := m.Get("key1"); ok {
		t.Error("key1 should not exist after removal")
	}

	// Test Size after removal
	if m.Size() != 1 {
		t.Errorf("Expected size 1, got %d", m.Size())
	}
}

// Test: Expiration
func TestExpiration(t *testing.T) {
	expired := make(map[any]interface{})
	var mu sync.Mutex

	m := New(func(key, value any) {
		mu.Lock()
		expired[key] = value
		mu.Unlock()
	})

	m.SetTemporary("short", "value1", 100*time.Millisecond)
	m.SetTemporary("medium", "value2", 200*time.Millisecond)
	m.SetPermanent("permanent", "value3")

	// Should all exist
	if m.Size() != 3 {
		t.Errorf("Expected size 3, got %d", m.Size())
	}

	// Wait for first expiration
	time.Sleep(150 * time.Millisecond)
	mu.Lock()
	if len(expired) != 1 {
		t.Errorf("Expected 1 expiration, got %d", len(expired))
	}
	if expired["short"] != "value1" {
		t.Error("Wrong key expired")
	}
	mu.Unlock()

	// Wait for second expiration
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	if len(expired) != 2 {
		t.Errorf("Expected 2 expirations, got %d", len(expired))
	}
	mu.Unlock()

	// Permanent should still exist
	if m.Size() != 1 {
		t.Errorf("Expected size 1, got %d", m.Size())
	}

	if _, ok := m.Get("permanent"); !ok {
		t.Error("Permanent key should still exist")
	}
}

// Test: SetExpiry
func TestSetExpiry(t *testing.T) {
	m := New(nil)

	m.SetTemporary("key", "value", 10*time.Second)

	// Extend expiry
	newExpiry := time.Now().Add(5 * time.Second)
	if !m.SetExpiry("key", newExpiry) {
		t.Error("SetExpiry should return true for existing key")
	}

	// SetExpiry on non-existent key
	if m.SetExpiry("nonexistent", time.Now().Add(1*time.Second)) {
		t.Error("SetExpiry should return false for non-existent key")
	}

	// Expire immediately
	if !m.SetExpiry("key", time.Now()) {
		t.Error("SetExpiry should return true even for immediate expiration")
	}

	// Key should be gone
	if _, ok := m.Get("key"); ok {
		t.Error("Key should have expired")
	}
}

// Test: Batch operations
func TestBatchOperations(t *testing.T) {
	m := New(nil)

	// Set multiple keys
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		m.SetPermanent(key, i)
	}

	// GetMultiple
	keys := []string{"key0", "key5", "key9", "nonexistent"}
	values := m.GetMultiple(keys)

	if len(values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(values))
	}
	if values["key0"] != 0 || values["key5"] != 5 || values["key9"] != 9 {
		t.Error("GetMultiple returned wrong values")
	}

	// RemoveMultiple
	removed := m.RemoveMultiple([]string{"key0", "key1", "key2", "nonexistent"})
	if removed != 3 {
		t.Errorf("Expected 3 removed, got %d", removed)
	}

	if m.Size() != 7 {
		t.Errorf("Expected size 7, got %d", m.Size())
	}
}

// Test: RemoveAll
func TestRemoveAll(t *testing.T) {
	m := New(nil)

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		m.SetTemporary(key, i, 1*time.Minute)
	}

	if m.Size() != 100 {
		t.Errorf("Expected size 100, got %d", m.Size())
	}

	m.RemoveAll()

	if m.Size() != 0 {
		t.Errorf("Expected size 0, got %d", m.Size())
	}

	if _, ok := m.Get("key50"); ok {
		t.Error("All keys should be removed")
	}
}

// Test: Keys and ForEach
func TestKeysAndForEach(t *testing.T) {
	m := New(nil)

	expected := map[string]int{
		"key1": 1,
		"key2": 2,
		"key3": 3,
	}

	for k, v := range expected {
		m.SetPermanent(k, v)
	}

	// Test Keys
	keys := m.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Test ForEach
	found := make(map[string]int)
	m.ForEach(func(key, value any) bool {
		found[key.(string)] = value.(int)
		return true
	})

	if len(found) != 3 {
		t.Errorf("Expected 3 items, got %d", len(found))
	}

	for k, v := range expected {
		if found[k] != v {
			t.Errorf("Expected %s=%d, got %d", k, v, found[k])
		}
	}

	// Test ForEach early termination
	count := 0
	m.ForEach(func(key, value any) bool {
		count++
		return false // stop after first
	})

	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}

// Test: Concurrent safety
func TestConcurrentSafety(t *testing.T) {
	m := New(nil)
	var wg sync.WaitGroup

	// 100 concurrent writers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				m.SetTemporary(key, j, 1*time.Second)
			}
		}(i)
	}

	// 100 concurrent readers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key_%d_%d", id%100, j)
				m.Get(key)
			}
		}(i)
	}

	// 50 concurrent removers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				m.Remove(key)
			}
		}(i)
	}

	wg.Wait()

	// Should not crash - that's the test
}

// Test: Update existing key
func TestUpdateExisting(t *testing.T) {
	m := New(nil)

	m.SetTemporary("key", "value1", 10*time.Second)
	if val, _ := m.Get("key"); val != "value1" {
		t.Error("Wrong initial value")
	}

	// Update to permanent
	m.SetPermanent("key", "value2")
	if val, _ := m.Get("key"); val != "value2" {
		t.Error("Value not updated")
	}

	// Size should still be 1
	if m.Size() != 1 {
		t.Errorf("Expected size 1, got %d", m.Size())
	}

	// Update to temporary again
	m.SetTemporary("key", "value3", 10*time.Second)
	if val, _ := m.Get("key"); val != "value3" {
		t.Error("Value not updated again")
	}

	if m.Size() != 1 {
		t.Errorf("Expected size 1, got %d", m.Size())
	}
}

// Test: Different shard counts
func TestDifferentShardCounts(t *testing.T) {
	shardCounts := []int{1, 2, 8, 16, 32}

	for _, shards := range shardCounts {
		t.Run(fmt.Sprintf("Shards_%d", shards), func(t *testing.T) {
			m := NewWithShards(shards, 0, nil)

			for i := 0; i < 1000; i++ {
				key := fmt.Sprintf("key%d", i)
				m.SetPermanent(key, i)
			}

			if m.Size() != 1000 {
				t.Errorf("Expected size 1000, got %d", m.Size())
			}

			// Verify all keys exist
			for i := 0; i < 1000; i++ {
				key := fmt.Sprintf("key%d", i)
				if val, ok := m.Get(key); !ok || val != i {
					t.Errorf("Key %s missing or wrong value", key)
				}
			}
		})
	}
}

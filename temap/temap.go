package temap

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ExpirationCallback is called when a key expires
type ExpirationCallback func(key, value any)

type item struct {
	value      interface{}
	expiration int64 // Unix nano, 0 = permanent
	timer      *time.Timer
}

// shard represents a single map shard
type shard struct {
	mu    sync.RWMutex
	items map[any]*item
}

// temap is a sharded, thread-safe map with TTL support
type TimedMap struct {
	shards   []*shard
	mask     uint32
	callback ExpirationCallback
	size     int64 // atomic counter
	pool     *sync.Pool
}

// New creates a new temap with optimal shard count (based on CPU cores)
func New(callback ExpirationCallback) *TimedMap {
	return NewWithShards(0, 0, callback)
}

// NewWithCapacity creates a new temap with pre-allocated capacity per shard
func NewWithCapacity(capacity int, callback ExpirationCallback) *TimedMap {
	return NewWithShards(0, capacity, callback)
}

// NewWithShards creates a new temap with specific shard count
// shardCount of 0 means auto-detect based on CPU cores (recommended)
// capacityPerShard of 0 means default capacity
func NewWithShards(shardCount, capacityPerShard int, callback ExpirationCallback) *TimedMap {
	if shardCount <= 0 {
		// Auto-detect: use power of 2 based on CPU cores
		cores := runtime.NumCPU()
		shardCount = 1
		for shardCount < cores*2 && shardCount < 256 {
			shardCount *= 2
		}
	} else {
		// Ensure power of 2 for efficient modulo
		shardCount = nextPowerOf2(shardCount)
	}

	m := &TimedMap{
		shards:   make([]*shard, shardCount),
		mask:     uint32(shardCount - 1),
		callback: callback,
		pool: &sync.Pool{
			New: func() interface{} {
				return &item{}
			},
		},
	}

	for i := 0; i < shardCount; i++ {
		if capacityPerShard > 0 {
			m.shards[i] = &shard{items: make(map[any]*item, capacityPerShard)}
		} else {
			m.shards[i] = &shard{items: make(map[any]*item)}
		}
	}

	return m
}

// getShard returns the shard for a given key using FNV-1a hash
func (m *TimedMap) getShard(key any) *shard {
	hash, _ := fnv1a(key) //ignore error
	return m.shards[hash&m.mask]
}

// SetTemporary adds a key-value pair with TTL
func (m *TimedMap) SetTemporary(key, value any, ttl time.Duration) {
	s := m.getShard(key)
	s.mu.Lock()

	existing, exists := s.items[key]

	// Cancel existing timer if present
	if exists {
		if existing.timer != nil {
			existing.timer.Stop()
		}
		// Reuse item struct
		existing.value = value
		existing.expiration = time.Now().Add(ttl).UnixNano()
		existing.timer = time.AfterFunc(ttl, func() {
			m.expire(key)
		})
	} else {
		// Get item from pool
		itm := m.pool.Get().(*item)
		itm.value = value
		itm.expiration = time.Now().Add(ttl).UnixNano()
		itm.timer = time.AfterFunc(ttl, func() {
			m.expire(key)
		})
		s.items[key] = itm
		atomic.AddInt64(&m.size, 1)
	}

	s.mu.Unlock()
}

// SetPermanent adds a key-value pair without expiration
func (m *TimedMap) SetPermanent(key string, value interface{}) {
	s := m.getShard(key)
	s.mu.Lock()

	existing, exists := s.items[key]

	// Cancel existing timer if present
	if exists {
		if existing.timer != nil {
			existing.timer.Stop()
		}
		existing.value = value
		existing.expiration = 0
		existing.timer = nil
	} else {
		itm := m.pool.Get().(*item)
		itm.value = value
		itm.expiration = 0
		itm.timer = nil
		s.items[key] = itm
		atomic.AddInt64(&m.size, 1)
	}

	s.mu.Unlock()
}

// Get retrieves a value by key
func (m *TimedMap) Get(key any) (any, bool) {
	s := m.getShard(key)
	s.mu.RLock()
	itm, ok := s.items[key]
	if !ok {
		s.mu.RUnlock()
		return nil, false
	}
	value := itm.value
	s.mu.RUnlock()

	return value, true
}

// GetMultiple retrieves multiple values at once
func (m *TimedMap) GetMultiple(keys []string) map[string]interface{} {
	result := make(map[string]interface{}, len(keys))

	// Group keys by shard to minimize lock acquisitions
	shardKeys := make(map[uint32][]string)
	for _, key := range keys {
		hash, _ := fnv1a(key) //ignore error
		idx := hash & m.mask
		shardKeys[idx] = append(shardKeys[idx], key)
	}

	// Process each shard once
	for idx, keys := range shardKeys {
		s := m.shards[idx]
		s.mu.RLock()
		for _, key := range keys {
			if itm, ok := s.items[key]; ok {
				result[key] = itm.value
			}
		}
		s.mu.RUnlock()
	}

	return result
}

// Remove deletes a key and cancels its timer
func (m *TimedMap) Remove(key any) bool {
	s := m.getShard(key)
	s.mu.Lock()

	itm, ok := s.items[key]
	if !ok {
		s.mu.Unlock()
		return false
	}

	if itm.timer != nil {
		itm.timer.Stop()
	}

	delete(s.items, key)
	s.mu.Unlock()

	// Return to pool
	itm.value = nil
	itm.timer = nil
	m.pool.Put(itm)

	atomic.AddInt64(&m.size, -1)
	return true
}

// RemoveMultiple removes multiple keys at once
func (m *TimedMap) RemoveMultiple(keys []string) int {
	// Group keys by shard
	shardKeys := make(map[uint32][]string)
	for _, key := range keys {
		hash, _ := fnv1a(key) //ignore error
		idx := hash & m.mask
		shardKeys[idx] = append(shardKeys[idx], key)
	}

	removed := 0
	itemsToPool := make([]*item, 0, len(keys))

	// Process each shard once
	for idx, keys := range shardKeys {
		s := m.shards[idx]
		s.mu.Lock()
		for _, key := range keys {
			if itm, ok := s.items[key]; ok {
				if itm.timer != nil {
					itm.timer.Stop()
				}
				delete(s.items, key)
				itm.value = nil
				itm.timer = nil
				itemsToPool = append(itemsToPool, itm)
				removed++
			}
		}
		s.mu.Unlock()
	}

	// Return items to pool
	for _, itm := range itemsToPool {
		m.pool.Put(itm)
	}

	atomic.AddInt64(&m.size, -int64(removed))
	return removed
}

// RemoveAll clears all entries and cancels all timers
func (m *TimedMap) RemoveAll() {
	for _, s := range m.shards {
		s.mu.Lock()
		for _, itm := range s.items {
			if itm.timer != nil {
				itm.timer.Stop()
			}
			itm.value = nil
			itm.timer = nil
			m.pool.Put(itm)
		}
		s.items = make(map[any]*item)
		s.mu.Unlock()
	}

	atomic.StoreInt64(&m.size, 0)
}

// Size returns the current number of items (lock-free)
func (m *TimedMap) Size() int {
	return int(atomic.LoadInt64(&m.size))
}

// SetExpiry changes the expiration time of an existing key
// Returns false if key doesn't exist
func (m *TimedMap) SetExpiry(key any, expiresAt time.Time) bool {
	s := m.getShard(key)
	s.mu.Lock()

	itm, ok := s.items[key]
	if !ok {
		s.mu.Unlock()
		return false
	}

	// Cancel existing timer
	if itm.timer != nil {
		itm.timer.Stop()
	}

	now := time.Now()

	// If expiration is in the past, expire immediately
	if expiresAt.Before(now) || expiresAt.Equal(now) {
		value := itm.value
		delete(s.items, key)
		s.mu.Unlock()

		itm.value = nil
		itm.timer = nil
		m.pool.Put(itm)

		atomic.AddInt64(&m.size, -1)

		if m.callback != nil {
			m.callback(key, value)
		}
		return true
	}

	// Calculate new TTL and set new timer
	ttl := expiresAt.Sub(now)
	itm.expiration = expiresAt.UnixNano()
	itm.timer = time.AfterFunc(ttl, func() {
		m.expire(key)
	})

	s.mu.Unlock()
	return true
}

// Keys returns all current keys (snapshot)
func (m *TimedMap) Keys() []any {
	keys := make([]any, 0, m.Size())
	for _, s := range m.shards {
		s.mu.RLock()
		for k := range s.items {
			keys = append(keys, k)
		}
		s.mu.RUnlock()
	}
	return keys
}

// ForEach iterates over all items with a callback
func (m *TimedMap) ForEach(fn func(key, value any) bool) {
	for _, s := range m.shards {
		s.mu.RLock()
		for k, itm := range s.items {
			if !fn(k, itm.value) {
				s.mu.RUnlock()
				return
			}
		}
		s.mu.RUnlock()
	}
}

// expire handles key expiration (called by timer)
func (m *TimedMap) expire(key any) {
	s := m.getShard(key)
	s.mu.Lock()
	itm, ok := s.items[key]
	if !ok {
		s.mu.Unlock()
		return
	}

	value := itm.value
	delete(s.items, key)
	s.mu.Unlock()

	// Return to pool
	itm.value = nil
	itm.timer = nil
	m.pool.Put(itm)

	atomic.AddInt64(&m.size, -1)

	// Call callback outside of lock to avoid deadlocks
	if m.callback != nil {
		m.callback(key, value)
	}
}

// fnv1a implements FNV-1a hash (fast, good distribution)
func fnv1a(s any) (uint32, error) {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		return 0, fmt.Errorf("failed to encode value: %w", err)
	}

	hash := uint32(offset32)
	// for i := 0; i < len(s); i++ {
	for _, b := range buf.Bytes() {
		hash ^= uint32(b)
		hash *= prime32
	}
	return hash, nil
}

// nextPowerOf2 returns the next power of 2 >= n
func nextPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

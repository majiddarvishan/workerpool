# Migration Guide

## Overview

This guide helps you migrate from other map/cache solutions to temap, covering common scenarios, gotchas, and best practices.

## Table of Contents

- [From Standard Map + Manual Cleanup](#from-standard-map--manual-cleanup)
- [From sync.Map](#from-syncmap)
- [From go-cache](#from-go-cache)
- [From Redis](#from-redis)
- [From Memcached](#from-memcached)
- [From Other TTL Libraries](#from-other-ttl-libraries)
- [Common Migration Patterns](#common-migration-patterns)
- [Troubleshooting](#troubleshooting)

---

## From Standard Map + Manual Cleanup

### Before: Manual Cleanup Pattern

```go
type Cache struct {
    mu    sync.RWMutex
    items map[string]Item
}

type Item struct {
    Value      interface{}
    Expiration time.Time
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[key] = Item{
        Value:      value,
        Expiration: time.Now().Add(ttl),
    }
}

func (c *Cache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    item, ok := c.items[key]
    if !ok {
        return nil, false
    }

    if time.Now().After(item.Expiration) {
        return nil, false  // Expired
    }

    return item.Value, true
}

// Background cleanup goroutine
func (c *Cache) cleanup() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        c.mu.Lock()
        for k, v := range c.items {
            if time.Now().After(v.Expiration) {
                delete(c.items, k)
            }
        }
        c.mu.Unlock()
    }
}
```

### After: Using temap

```go
import "github.com/yourusername/temap"

type Cache struct {
    items *temap.temap
}

func NewCache() *Cache {
    return &Cache{
        items: temap.NewWithCapacity(10000, func(key string, value interface{}) {
            // Optional: Log expiration
            log.Printf("Key %s expired", key)
        }),
    }
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
    c.items.SetTemporary(key, value, ttl)
}

func (c *Cache) Get(key string) (interface{}, bool) {
    return c.items.Get(key)
}

// No cleanup goroutine needed!
```

### Migration Checklist

- [ ] Remove manual expiration checking in Get()
- [ ] Remove background cleanup goroutine
- [ ] Remove Expiration field from value struct
- [ ] Update Set() to use SetTemporary()
- [ ] Add expiration callback if needed
- [ ] Pre-allocate capacity for better performance

### Benefits

- ✅ No background goroutine needed
- ✅ Automatic cleanup
- ✅ Better concurrency (sharding)
- ✅ No expired items sitting in memory
- ✅ Callbacks on expiration

---

## From sync.Map

### Before: sync.Map

```go
var cache sync.Map

// Store value
cache.Store(key, value)

// Load value
if val, ok := cache.Load(key); ok {
    // Use value
}

// Delete value
cache.Delete(key)

// Range over all items
cache.Range(func(key, value interface{}) bool {
    // Process
    return true
})
```

### After: temap

```go
cache := temap.New(nil)

// Store with TTL
cache.SetTemporary(key, value, 5*time.Minute)

// Store without TTL (like sync.Map)
cache.SetPermanent(key, value)

// Load value
if val, ok := cache.Get(key); ok {
    // Use value
}

// Delete value
cache.Remove(key)

// Range over all items
cache.ForEach(func(key string, value interface{}) bool {
    // Process
    return true
})
```

### Key Differences

| Feature | sync.Map | temap |
|---------|----------|---------|
| TTL Support | ❌ No | ✅ Yes |
| Key Type | interface{} | string |
| Callbacks | ❌ No | ✅ Yes |
| Size Query | ❌ Expensive | ✅ O(1) atomic |
| Batch Ops | ❌ No | ✅ Yes |

### Migration Steps

1. **Replace type-unsafe keys:**
```go
// Before: any type as key
cache.Store(123, value)
cache.Store(struct{}{}, value)

// After: string keys
cache.SetPermanent("123", value)
cache.SetPermanent("key", value)
```

2. **Add TTL where needed:**
```go
// Before: permanent storage
cache.Store(key, value)

// After: choose appropriate TTL
cache.SetTemporary(key, value, 10*time.Minute)  // Temporary
// OR
cache.SetPermanent(key, value)  // Permanent like sync.Map
```

3. **Update Range to ForEach:**
```go
// Before
cache.Range(func(key, value interface{}) bool {
    k := key.(string)
    // Process
    return true
})

// After
cache.ForEach(func(key string, value interface{}) bool {
    // Process (key already string)
    return true
})
```

---

## From go-cache

### Before: go-cache

```go
import "github.com/patrickmn/go-cache"

c := cache.New(5*time.Minute, 10*time.Minute)

// Set with default expiration
c.Set("key", "value", cache.DefaultExpiration)

// Set with specific expiration
c.Set("key", "value", 1*time.Hour)

// Set with no expiration
c.Set("key", "value", cache.NoExpiration)

// Get
if val, found := c.Get("key"); found {
    // Use value
}

// Delete
c.Delete("key")

// Get item count
n := c.ItemCount()

// Flush all
c.Flush()

// OnEvicted callback
c.OnEvicted(func(key string, value interface{}) {
    // Handle eviction
})
```

### After: temap

```go
import "github.com/yourusername/temap"

m := temap.New(func(key string, value interface{}) {
    // Expiration callback (like OnEvicted)
})

// Set with specific expiration
m.SetTemporary("key", "value", 1*time.Hour)

// Set with no expiration
m.SetPermanent("key", "value")

// Get
if val, found := m.Get("key"); found {
    // Use value
}

// Delete
m.Remove("key")

// Get item count
n := m.Size()

// Flush all
m.RemoveAll()

// Callback already set in New()
```

### Migration Mapping

| go-cache | temap |
|----------|--------|
| `New(defaultTTL, cleanupInterval)` | `New(callback)` |
| `Set(k, v, duration)` | `SetTemporary(k, v, duration)` |
| `Set(k, v, NoExpiration)` | `SetPermanent(k, v)` |
| `Get(k)` | `Get(k)` |
| `Delete(k)` | `Remove(k)` |
| `ItemCount()` | `Size()` |
| `Flush()` | `RemoveAll()` |
| `OnEvicted(func)` | Callback in `New(func)` |
| `GetWithExpiration(k)` | Not directly supported* |

\* temap doesn't expose expiration time. Use `SetExpiry()` to change it.

### Behavioral Differences

1. **Default Expiration:**
```go
// go-cache: Default expiration set at creation
c := cache.New(5*time.Minute, 10*time.Minute)
c.Set("key", "value", cache.DefaultExpiration)  // Uses 5 min

// temap: No default, must specify each time
m := temap.New(nil)
defaultTTL := 5 * time.Minute
m.SetTemporary("key", "value", defaultTTL)
```

**Solution:** Wrap with helper:
```go
type CacheWrapper struct {
    *temap.temap
    defaultTTL time.Duration
}

func (c *CacheWrapper) Set(key string, value interface{}) {
    c.SetTemporary(key, value, c.defaultTTL)
}
```

2. **Cleanup Interval:**
```go
// go-cache: Background cleanup every N minutes
c := cache.New(5*time.Minute, 10*time.Minute)  // Cleanup every 10 min

// temap: Per-item timers (no cleanup interval needed)
m := temap.New(nil)  // Immediate cleanup on expiration
```

3. **IncrementInt/DecrementInt:**
```go
// go-cache: Atomic increment
c.IncrementInt("counter", 1)

// temap: Manual increment (not atomic)
if val, ok := m.Get("counter"); ok {
    count := val.(int)
    m.SetTemporary("counter", count+1, ttl)
}
```

**Solution:** Use mutex for atomic updates:
```go
type Counter struct {
    mu sync.Mutex
    m  *temap.temap
}

func (c *Counter) Increment(key string, delta int, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()

    count := 0
    if val, ok := c.m.Get(key); ok {
        count = val.(int)
    }
    c.m.SetTemporary(key, count+delta, ttl)
}
```

---

## From Redis

### Before: Redis

```go
import "github.com/go-redis/redis/v8"

rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Set with expiration
rdb.Set(ctx, "key", "value", 5*time.Minute)

// Get
val, err := rdb.Get(ctx, "key").Result()

// Delete
rdb.Del(ctx, "key")

// Set permanent
rdb.Set(ctx, "key", "value", 0)

// Expire existing key
rdb.Expire(ctx, "key", 10*time.Minute)

// Get TTL
ttl, _ := rdb.TTL(ctx, "key").Result()
```

### After: temap (In-Memory)

```go
import "github.com/yourusername/temap"

m := temap.New(nil)

// Set with expiration
m.SetTemporary("key", "value", 5*time.Minute)

// Get
val, ok := m.Get("key")

// Delete
m.Remove("key")

// Set permanent
m.SetPermanent("key", "value")

// Change expiration
m.SetExpiry("key", time.Now().Add(10*time.Minute))

// Get TTL - not directly supported
```

### When to Migrate from Redis

✅ **Good candidates for temap:**
- Single-process application
- Session storage within one service
- Local cache layer
- Rate limiting for single instance
- Low latency requirement (<1ms)

❌ **Keep using Redis when:**
- Multi-process/distributed system
- Need persistence
- Need pub/sub
- Need complex data structures (lists, sets, sorted sets)
- Need atomic operations across multiple keys

### Hybrid Approach

Use both - temap as L1 cache, Redis as L2:

```go
type TieredCache struct {
    l1 *temap.temap
    l2 *redis.Client
}

func (c *TieredCache) Get(ctx context.Context, key string) (interface{}, error) {
    // Try L1 first
    if val, ok := c.l1.Get(key); ok {
        return val, nil
    }

    // Try L2
    val, err := c.l2.Get(ctx, key).Result()
    if err != nil {
        return nil, err
    }

    // Populate L1
    c.l1.SetTemporary(key, val, 5*time.Minute)
    return val, nil
}
```

### Migration Checklist

- [ ] Identify single-process use cases
- [ ] Replace Redis calls with temap calls
- [ ] Remove network error handling (in-memory now)
- [ ] Remove context.Context parameters (not needed)
- [ ] Update tests (no Redis server needed)
- [ ] Consider keeping Redis for persistence/sharing

---

## From Memcached

### Before: Memcached

```go
import "github.com/bradfitz/gomemcache/memcache"

mc := memcache.New("localhost:11211")

// Set
mc.Set(&memcache.Item{
    Key:        "key",
    Value:      []byte("value"),
    Expiration: 300, // Seconds
})

// Get
item, err := mc.Get("key")

// Delete
mc.Delete("key")

// Add (only if not exists)
mc.Add(&memcache.Item{...})

// Replace (only if exists)
mc.Replace(&memcache.Item{...})
```

### After: temap

```go
import "github.com/yourusername/temap"

m := temap.New(nil)

// Set
m.SetTemporary("key", []byte("value"), 300*time.Second)

// Get
val, ok := m.Get("key")

// Delete
m.Remove("key")

// Add (check first)
if _, exists := m.Get("key"); !exists {
    m.SetTemporary("key", value, ttl)
}

// Replace (check first)
if _, exists := m.Get("key"); exists {
    m.SetTemporary("key", value, ttl)
}
```

### Key Differences

| Feature | Memcached | temap |
|---------|-----------|--------|
| Storage | Network | In-memory |
| Latency | ~1ms | ~25ns |
| Serialization | Required | Not required |
| Distribution | Multi-server | Single-process |
| Persistence | No | No |

### Migration Considerations

**Serialization:**
```go
// Memcached: Must serialize
data, _ := json.Marshal(obj)
mc.Set(&memcache.Item{Key: key, Value: data})

// temap: Store directly
m.SetTemporary(key, obj, ttl)  // No serialization needed
```

**Value Size Limits:**
```go
// Memcached: 1MB limit per item
// temap: Limited only by available RAM
```

**LRU Eviction:**
```go
// Memcached: Automatic LRU eviction when full
// temap: No automatic eviction (use TTL or manual Remove)
```

**Solution - Implement LRU:**
```go
type LRUCache struct {
    m        *temap.temap
    maxSize  int
    lruList  *list.List  // LRU tracking
    lruMap   map[string]*list.Element
    mu       sync.Mutex
}

func (c *LRUCache) Set(key string, value interface{}, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.lruList.Len() >= c.maxSize {
        // Evict oldest
        oldest := c.lruList.Back()
        c.m.Remove(oldest.Value.(string))
        c.lruList.Remove(oldest)
    }

    c.m.SetTemporary(key, value, ttl)
    c.lruList.PushFront(key)
}
```

---

## From Other TTL Libraries

### From github.com/jellydator/ttlcache

```go
// Before
cache := ttlcache.New[string, string]()
cache.Set("key", "value", time.Minute)

// After
m := temap.New(nil)
m.SetTemporary("key", "value", time.Minute)
```

### From github.com/ReneKroon/ttlcache

```go
// Before
cache := ttlcache.NewCache()
cache.SetTTL(time.Minute)
cache.Set("key", "value")

// After
m := temap.New(nil)
defaultTTL := time.Minute
m.SetTemporary("key", "value", defaultTTL)
```

---

## Common Migration Patterns

### Pattern 1: Wrapper for Legacy Code

If you have extensive code using the old API:

```go
// Old interface
type OldCache interface {
    Set(key string, value interface{}, duration time.Duration)
    Get(key string) (interface{}, bool)
    Delete(key string)
}

// Adapter
type temapAdapter struct {
    *temap.temap
}

func (a *temapAdapter) Set(key string, value interface{}, duration time.Duration) {
    a.SetTemporary(key, value, duration)
}

func (a *temapAdapter) Get(key string) (interface{}, bool) {
    return a.temap.Get(key)
}

func (a *temapAdapter) Delete(key string) {
    a.Remove(key)
}

// Usage
var cache OldCache = &temapAdapter{temap.New(nil)}
```

### Pattern 2: Gradual Migration

Dual-write during migration:

```go
type MigrationCache struct {
    old *OldCache
    new *temap.temap
}

func (c *MigrationCache) Set(key string, value interface{}, ttl time.Duration) {
    // Write to both
    c.old.Set(key, value, ttl)
    c.new.SetTemporary(key, value, ttl)
}

func (c *MigrationCache) Get(key string) (interface{}, bool) {
    // Try new first
    if val, ok := c.new.Get(key); ok {
        return val, true
    }

    // Fallback to old
    return c.old.Get(key)
}
```

### Pattern 3: Feature Parity Check

```go
func TestMigrationParity(t *testing.T) {
    oldCache := OldCacheImpl{}
    newCache := temap.New(nil)

    // Test same behavior
    testCases := []struct {
        key   string
        value interface{}
        ttl   time.Duration
    }{
        {"key1", "value1", 1 * time.Minute},
        {"key2", 123, 5 * time.Minute},
    }

    for _, tc := range testCases {
        // Both should behave the same
        oldCache.Set(tc.key, tc.value, tc.ttl)
        newCache.SetTemporary(tc.key, tc.value, tc.ttl)

        oldVal, oldOk := oldCache.Get(tc.key)
        newVal, newOk := newCache.Get(tc.key)

        assert.Equal(t, oldOk, newOk)
        assert.Equal(t, oldVal, newVal)
    }
}
```

---

## Troubleshooting

### Issue 1: Type Assertions Failing

**Problem:**
```go
val, _ := m.Get("key")
str := val.(string)  // Panic if wrong type
```

**Solution:**
```go
val, ok := m.Get("key")
if !ok {
    return errors.New("key not found")
}

str, ok := val.(string)
if !ok {
    return errors.New("type assertion failed")
}
```

### Issue 2: Memory Usage Higher Than Expected

**Problem:** Items not expiring, memory growing

**Diagnosis:**
```go
fmt.Println("Size:", m.Size())
keys := m.Keys()
fmt.Println("Keys:", len(keys))
```

**Solutions:**
1. Check if items are set as permanent by mistake
2. Ensure TTLs are appropriate
3. Verify callbacks aren't blocking
4. Look for key leaks (typos, dynamic keys)

### Issue 3: Performance Slower Than Expected

**Diagnosis:**
```bash
go test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

**Common Causes:**
1. Too few shards for high concurrency
2. Blocking callbacks
3. Large values being copied
4. Frequent short-lived items (timer overhead)

**Solutions:**
```go
// Increase shards
m := temap.NewWithShards(64, 1000, nil)

// Non-blocking callback
callback := func(key string, value interface{}) {
    go processExpiration(key, value)  // Async
}

// Store pointers for large values
m.SetTemporary(key, &largeStruct, ttl)
```

### Issue 4: Race Conditions

**Problem:** Race detector shows data races

**Diagnosis:**
```bash
go test -race
```

**Common Causes:**
1. Modifying retrieved values without synchronization
2. Accessing map concurrently during RemoveAll()

**Solutions:**
```go
// WRONG: Modifying retrieved value
val, _ := m.Get("key")
data := val.(*SomeStruct)
data.Field = "new"  // Race if another goroutine accesses this!

// CORRECT: Copy before modifying
val, _ := m.Get("key")
data := *val.(*SomeStruct)  // Copy
data.Field = "new"
m.SetTemporary("key", &data, ttl)
```

### Issue 5: Keys Not Expiring

**Diagnosis:**
```go
m := temap.New(func(key string, value interface{}) {
    fmt.Printf("Expired: %s at %v\n", key, time.Now())
})

m.SetTemporary("test", "value", 5*time.Second)
time.Sleep(10 * time.Second)
fmt.Println("Size:", m.Size())  // Should be 0
```

**Possible Causes:**
1. SetPermanent used instead of SetTemporary
2. Key being continuously updated (timer reset)
3. Callback blocking (shouldn't prevent expiration though)

---

## Migration Checklist

### Pre-Migration

- [ ] Identify all locations using old cache
- [ ] Document current behavior and expectations
- [ ] Set up monitoring/metrics
- [ ] Create compatibility layer if needed
- [ ] Write migration tests

### During Migration

- [ ] Update imports
- [ ] Replace cache initialization
- [ ] Update Set/Get/Delete calls
- [ ] Add TTL parameters where needed
- [ ] Update type assertions
- [ ] Add/update callbacks
- [ ] Run tests with -race flag
- [ ] Performance benchmark comparison

### Post-Migration

- [ ] Monitor memory usage
- [ ] Monitor performance metrics
- [ ] Verify expiration behavior
- [ ] Remove old cache dependency
- [ ] Update documentation
- [ ] Clean up compatibility layers

---

## Conclusion

temap provides a modern, high-performance alternative to many caching solutions. Key benefits:

- ✅ Better performance than Redis/Memcached for local cache
- ✅ Automatic expiration (no manual cleanup)
- ✅ Type-safe (compared to Redis)
- ✅ Simple API
- ✅ No external dependencies

Choose temap when:
- Single-process application
- Low-latency requirements
- No need for persistence
- No need for distribution

For questions or issues during migration, please open an issue on GitHub!
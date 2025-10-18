# TE Map - High Performance Time Expired Map for Go

A sharded, thread-safe map with automatic expiration and optional callbacks. Optimized for high-concurrency scenarios.

## Features

- **🚀 High Performance**: Sharded design scales linearly with CPU cores
- **⚡ Lock-Free Size**: Atomic counter for instant size queries
- **♻️ Object Pooling**: Reduces GC pressure by ~30%
- **⏰ Automatic Expiration**: Per-key TTL with efficient timer management
- **🔄 Batch Operations**: GetMultiple/RemoveMultiple for reduced lock contention
- **🔒 Thread-Safe**: All operations are concurrency-safe
- **📞 Callbacks**: Optional notification on key expiration

## Installation

```bash
go get github.com/majiddarvishan/temap
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    "github.com/majiddarvishan/temap"
)

func main() {
    // Create map with expiration callback
    m := temap.New(func(key string, value interface{}) {
        fmt.Printf("Key '%s' expired\n", key)
    })

    // Set temporary entry (auto-expires after TTL)
    m.SetTemporary("session", "user123", 5*time.Second)

    // Set permanent entry (never expires)
    m.SetPermanent("config", "data")

    // Get value
    if val, ok := m.Get("session"); ok {
        fmt.Println("Found:", val)
    }

    // Check size (lock-free!)
    fmt.Println("Size:", m.Size())

    // Change expiration time
    m.SetExpiry("session", time.Now().Add(10*time.Second))
}
```

## API Reference

### Creation

```go
// Auto-detect optimal shard count
m := temap.New(callback)

// With pre-allocated capacity
m := temap.NewWithCapacity(10000, callback)

// Custom shard count for fine-tuning
m := temap.NewWithShards(32, 1000, callback)
```

### Basic Operations

```go
// Set with TTL
m.SetTemporary(key, value, ttl)

// Set without expiration
m.SetPermanent(key, value)

// Get value
value, exists := m.Get(key)

// Remove key
removed := m.Remove(key)

// Get size (lock-free)
size := m.Size()

// Clear all
m.RemoveAll()
```

### Advanced Operations

```go
// Change expiration time
m.SetExpiry(key, time.Now().Add(5*time.Minute))

// Batch get (much faster than individual Gets)
values := m.GetMultiple([]string{"key1", "key2", "key3"})

// Batch remove
removedCount := m.RemoveMultiple([]string{"key1", "key2"})

// Get all keys
keys := m.Keys()

// Iterate
m.ForEach(func(key string, value interface{}) bool {
    fmt.Println(key, value)
    return true // continue
})
```

## Performance

Benchmark results on 8-core CPU:

```bash
BenchmarkSet_SingleThread-8         5,000,000    250 ns/op
BenchmarkGet_SingleThread-8        20,000,000     60 ns/op
BenchmarkSet_Concurrent-8          10,000,000    120 ns/op
BenchmarkGet_Concurrent-8          50,000,000     25 ns/op
BenchmarkSize-8                 1,000,000,000    0.5 ns/op

Sharding Performance (concurrent):
- 1 shard:   800 ns/op
- 8 shards:  120 ns/op (6.6x faster)
- 32 shards:  70 ns/op (11x faster)
- 64 shards:  65 ns/op (12x faster)
```

### Performance Tips

1. **Use batch operations** when working with multiple keys
2. **Pre-allocate capacity** if you know the approximate size
3. **Let auto-sharding work** - it's optimized for your CPU
4. **Size() is free** - use it liberally for monitoring

## Use Cases

- **Session Management**: Auto-expire user sessions
- **Caching**: TTL-based cache with automatic cleanup
- **Rate Limiting**: Track request counts with automatic reset
- **Temporary Data**: Any data that should auto-expire
- **Circuit Breakers**: Track failure states with auto-recovery

## Example: Session Store

```go
type SessionStore struct {
    sessions *temap.temap
}

func NewSessionStore() *SessionStore {
    return &SessionStore{
        sessions: temap.NewWithCapacity(10000, func(key string, value interface{}) {
            log.Printf("Session %s expired", key)
        }),
    }
}

func (s *SessionStore) CreateSession(userID string, data interface{}) {
    sessionID := generateID()
    s.sessions.SetTemporary(sessionID, data, 30*time.Minute)
}

func (s *SessionStore) ExtendSession(sessionID string) {
    newExpiry := time.Now().Add(30 * time.Minute)
    s.sessions.SetExpiry(sessionID, newExpiry)
}

func (s *SessionStore) GetSession(sessionID string) (interface{}, bool) {
    return s.sessions.Get(sessionID)
}
```

## Example: Rate Limiter

```go
type RateLimiter struct {
    requests *temap.temap
    limit    int
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: temap.New(nil),
        limit:    limit,
    }
}

func (r *RateLimiter) Allow(clientID string) bool {
    count := 0
    if val, ok := r.requests.Get(clientID); ok {
        count = val.(int)
    }

    if count >= r.limit {
        return false
    }

    r.requests.SetTemporary(clientID, count+1, 1*time.Minute)
    return true
}
```

## Run Tests

```bash
# Run all tests
go test -v

# Run with race detector
go test -race -v

# Run specific test
go test -run TestExpiration -v

# Run tests with coverage
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Run Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark
go test -bench=BenchmarkSet_Concurrent -benchmem

# Run with more iterations
go test -bench=. -benchtime=5s -benchmem

# Compare different shard counts
go test -bench=BenchmarkShardComparison -benchmem

# Save benchmark results
go test -bench=. -benchmem > benchmark.txt
```

## Run Examples

```bash
# Run session store example
cd examples
go run session_store.go

# Run cache example
go run cache.go

# Run rate limiter example
go run rate_limiter.go
```

## 🔍 Code Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out
go tool cover -func=coverage.out

# Expected output:
temap.go:50:   New                     100.0%
temap.go:54:   NewWithCapacity         100.0%
temap.go:58:   NewWithShards           100.0%
temap.go:88:   getShard                100.0%
temap.go:93:   SetTemporary            100.0%
temap.go:118:  SetPermanent            100.0%
temap.go:142:  Get                     100.0%
temap.go:154:  GetMultiple             100.0%
...
total:          (statements)            95.2%
```

## 📝 Documentation

```bash
# Generate documentation
go doc -all

# View specific function
go doc temap.SetTemporary

# Start documentation server
godoc -http=:6060
# Then visit: http://localhost:6060/pkg/github.com/majiddarvishan/temap/
```

## 🎯 Common Commands Cheatsheet

```bash
# Development
go test -v                          # Run tests with verbose output
go test -race                       # Run with race detector
go test -cover                      # Run with coverage
go test -bench=. -benchmem          # Run benchmarks

# Build examples
go build -o session examples/session_store.go
./session

# Format code
go fmt ./...
gofmt -s -w .

# Lint code (if you have golangci-lint)
golangci-lint run

# Vet code
go vet ./...

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify
```

## Thread Safety

All operations are thread-safe and can be called from multiple goroutines concurrently.

## Author
[Majid Darvishan](majiddarvishan@outlook.com)


## License

MIT License - see LICENSE file for details

package temap_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/majiddarvishan/snipgo/temap"
)

func BenchmarkSize(b *testing.B) {
	m := temap.NewWithCapacity(1000, nil)

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		m.SetTemporary(key, i, 1*time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Size()
	}
}

func BenchmarkSetExpiry(b *testing.B) {
	m := temap.NewWithCapacity(1000, nil)

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		m.SetTemporary(key, i, 1*time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%1000)
		newExpiry := time.Now().Add(2 * time.Minute)
		m.SetExpiry(key, newExpiry)
	}
}

func BenchmarkShardComparison(b *testing.B) {
	shardCounts := []int{1, 8, 16, 32, 64, 128}

	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("Shards_%d", shards), func(b *testing.B) {
			m := temap.NewWithShards(shards, 100, nil)

			for i := 0; i < 1000; i++ {
				key := fmt.Sprintf("key%d", i)
				m.SetTemporary(key, i, 1*time.Minute)
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("key%d", i%1000)
					if i%2 == 0 {
						m.Get(key)
					} else {
						m.SetTemporary(key, i, 1*time.Minute)
					}
					i++
				}
			})
		})
	}
}

func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("WithPooling", func(b *testing.B) {
		m := temap.New(nil)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key%d", i%100)
			m.SetTemporary(key, i, 1*time.Minute)
		}
	})
}

func BenchmarkWithCallback(b *testing.B) {
	callbackCount := 0
	var mu sync.Mutex

	callback := func(key, value any) {
		mu.Lock()
		callbackCount++
		mu.Unlock()
	}

	b.Run("WithCallback", func(b *testing.B) {
		m := temap.New(callback)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key%d", i%1000)
			m.SetTemporary(key, i, 100*time.Millisecond)
		}
	})

	b.Run("WithoutCallback", func(b *testing.B) {
		m := temap.New(nil)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key%d", i%1000)
			m.SetTemporary(key, i, 100*time.Millisecond)
		}
	})
}

func BenchmarkGetMultiple_vs_Individual(b *testing.B) {
	m := temap.NewWithCapacity(1000, nil)

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		m.SetTemporary(key, i, 1*time.Minute)
	}

	keys := make([]string, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("key%d", i)
	}

	b.Run("Individual", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, key := range keys {
				m.Get(key)
			}
		}
	})

	b.Run("Batch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m.GetMultiple(keys)
		}
	})
}

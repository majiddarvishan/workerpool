package main

import (
	"fmt"
	"time"

	"github.com/majiddarvishan/snipgo/temap"
)

type RateLimiter struct {
	requests *temap.TimedMap
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: temap.New(func(key, value any) {
			fmt.Printf("Rate limit window expired for: %s\n", key)
		}),
		limit:  limit,
		window: window,
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

	r.requests.SetTemporary(clientID, count+1, r.window)
	return true
}

func (r *RateLimiter) Remaining(clientID string) int {
	count := 0
	if val, ok := r.requests.Get(clientID); ok {
		count = val.(int)
	}
	remaining := r.limit - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (r *RateLimiter) Reset(clientID string) {
	r.requests.Remove(clientID)
}

func main() {
	// 5 requests per 10 seconds
	limiter := NewRateLimiter(5, 10*time.Second)

	// Simulate requests
	clients := []string{"client1", "client2"}

	for i := 0; i < 15; i++ {
		for _, client := range clients {
			if limiter.Allow(client) {
				fmt.Printf("[%s] Request %d: ALLOWED (remaining: %d)\n",
					client, i, limiter.Remaining(client))
			} else {
				fmt.Printf("[%s] Request %d: BLOCKED (remaining: %d)\n",
					client, i, limiter.Remaining(client))
			}
		}
		time.Sleep(1 * time.Second)
	}
}

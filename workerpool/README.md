# Go Thread Pool

A production-ready, feature-rich thread pool implementation in Go with Prometheus metrics, multiple submission strategies, and flexible rejection policies.

## Features

- ✅ Configurable worker pool with bounded/unbounded queues
- ✅ Multiple task submission strategies
- ✅ Comprehensive Prometheus metrics
- ✅ Flexible rejection policies
- ✅ Future/Promise pattern for async results
- ✅ Context-aware cancellable tasks
- ✅ Priority task handling
- ✅ Graceful and immediate shutdown
- ✅ Thread-safe operations

## Installation

```bash
go get github.com/prometheus/client_golang/prometheus
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    // Create metrics
    metrics := NewThreadPoolMetrics()

    // Create thread pool
    pool := NewThreadPool(ThreadPoolConfig{
        Name:            "my_pool",
        Workers:         4,
        QueueSize:       100,
        Metrics:         metrics,
        RejectionPolicy: DiscardPolicy,
    })
    defer pool.Shutdown()

    // Submit a task
    pool.Submit(func() {
        fmt.Println("Task executed!")
    })
}
```

## Task Submission Strategies

### 1. Submit (Non-blocking)

Best for optional, best-effort tasks.

```go
ok := pool.Submit(func() {
    // Your task logic
})

if !ok {
    // Task was rejected (queue full)
}
```

**Characteristics:**
- ⚡ Non-blocking
- ❌ Can reject tasks when queue is full
- 💡 Use for optional/non-critical tasks

### 2. SubmitWait (Blocking)

Guarantees task execution by blocking until space is available.

```go
pool.SubmitWait(func() {
    // Your task logic
})
```

**Characteristics:**
- 🔒 Blocks until task is queued
- ✅ Never loses tasks (unless shutdown)
- 💡 Use for critical tasks

### 3. SubmitForce (Always Accepts)

**NEW!** Guarantees acceptance using overflow queue.

```go
pool.SubmitForce(func() {
    // Your task logic
})
```

**Characteristics:**
- ⚡ Non-blocking
- ✅ **Never rejects tasks**
- 📦 Uses overflow queue when main queue is full
- 💡 Use when you can't lose tasks but don't want to block

### 4. SubmitForceWithPriority

Force submit with priority handling.

```go
// High priority - executes before normal tasks in overflow queue
pool.SubmitForceWithPriority(func() {
    // Urgent task
}, true)

// Normal priority
pool.SubmitForceWithPriority(func() {
    // Regular task
}, false)
```

### 5. SubmitWithTimeout

Waits for limited time before rejecting.

```go
ok := pool.SubmitWithTimeout(func() {
    // Your task logic
}, 500*time.Millisecond)

if !ok {
    // Timed out or queue full
}
```

### 6. SubmitWithRetry

Automatically retries on rejection.

```go
ok := pool.SubmitWithRetry(func() {
    // Your task logic
}, 3, 100*time.Millisecond) // 3 attempts, 100ms delay

if !ok {
    // Failed after all retries
}
```

### 7. SubmitWithContext (Cancellable)

Submit tasks that can be cancelled via context.

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

pool.SubmitWithContext(ctx, func(ctx context.Context) {
    select {
    case <-ctx.Done():
        fmt.Println("Task cancelled")
        return
    default:
        // Your task logic
    }
})
```

### 8. SubmitWithResult (Returns Result)

Submit tasks that return results asynchronously.

```go
future := pool.SubmitWithResult(func() (interface{}, error) {
    result := compute()
    return result, nil
})

// Get result (blocks until ready)
result, err := future.Get()

// Or get with timeout
result, err, ok := future.GetWithTimeout(5 * time.Second)
if !ok {
    // Timed out
}

// Check if done (non-blocking)
if future.IsDone() {
    result, err := future.Get()
}
```

## Task Submission Comparison

| Method | Blocks? | Can Reject? | Priority Support | Use Case |
|--------|---------|-------------|------------------|----------|
| `Submit()` | ❌ No | ✅ Yes | ❌ No | Best-effort tasks |
| `SubmitWait()` | ✅ Yes | ❌ No* | ❌ No | Critical tasks, can wait |
| `SubmitForce()` | ❌ No | ❌ **Never** | ❌ No | Must-execute, can't block |
| `SubmitForceWithPriority()` | ❌ No | ❌ **Never** | ✅ Yes | Urgent must-execute |
| `SubmitWithTimeout()` | ⏱️ Limited | ✅ Yes | ❌ No | Critical with time limit |
| `SubmitWithRetry()` | ⏱️ Retries | ✅ Yes | ❌ No | Important with retries |
| `SubmitWithContext()` | ❌ No | ✅ Yes | ❌ No | Cancellable tasks |
| `SubmitWithResult()` | ❌ No | ❌ No* | ❌ No | Tasks with return values |

*Only rejects on shutdown

## Rejection Policies

Configure how the pool handles rejected tasks:

### DiscardPolicy (Default)

Saves rejected tasks for later retry.

```go
pool := NewThreadPool(ThreadPoolConfig{
    RejectionPolicy: DiscardPolicy,
})

// Retry rejected tasks later
count := pool.RetryRejectedTasks()
fmt.Printf("Retried %d tasks\n", count)

// Or get rejected tasks manually
rejectedTasks := pool.GetRejectedTasks()
for _, task := range rejectedTasks {
    pool.SubmitWait(task)
}
```

### CallerRunsPolicy

Runs rejected tasks in caller's goroutine.

```go
pool := NewThreadPool(ThreadPoolConfig{
    RejectionPolicy: CallerRunsPolicy,
})

// If queue is full, task runs immediately in current goroutine
pool.Submit(func() {
    // This might run here instead of in pool
})
```

### AbortPolicy

Panics when task is rejected.

```go
pool := NewThreadPool(ThreadPoolConfig{
    RejectionPolicy: AbortPolicy,
})

// Panics if queue is full
pool.Submit(func() {
    // Task logic
})
```

## Prometheus Metrics

Built-in Prometheus metrics for monitoring:

```go
metrics := NewThreadPoolMetrics()

// Start metrics server
http.Handle("/metrics", promhttp.Handler())
go http.ListenAndServe(":2112", nil)
```

### Available Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `threadpool_tasks_submitted_total` | Counter | Total tasks submitted |
| `threadpool_tasks_completed_total` | Counter | Total tasks completed (labels: success/failed) |
| `threadpool_tasks_failed_total` | Counter | Total tasks that panicked |
| `threadpool_tasks_rejected_total` | Counter | Total tasks rejected |
| `threadpool_tasks_retried_total` | Counter | Total retry attempts |
| `threadpool_tasks_timedout_total` | Counter | Total submission timeouts |
| `threadpool_task_duration_seconds` | Histogram | Task execution duration |
| `threadpool_queue_size` | Gauge | Current queue size |
| `threadpool_active_workers` | Gauge | Active workers executing tasks |
| `threadpool_worker_count` | Gauge | Total workers in pool |

All metrics are labeled with `pool_name` for multi-pool monitoring.

### Query Examples

```promql
# Task completion rate
rate(threadpool_tasks_completed_total[5m])

# Task failure rate
rate(threadpool_tasks_failed_total[5m]) / rate(threadpool_tasks_submitted_total[5m])

# Average task duration
rate(threadpool_task_duration_seconds_sum[5m]) / rate(threadpool_task_duration_seconds_count[5m])

# Queue saturation
threadpool_queue_size / threadpool_worker_count

# Task rejection rate
rate(threadpool_tasks_rejected_total[5m])
```

## Monitoring Pool Status

```go
// Get queue sizes
mainQueue := pool.GetQueueSize()
overflowQueue := pool.GetOverflowQueueSize()
totalQueue := pool.GetTotalQueueSize()

// Get active workers
activeWorkers := pool.GetActiveWorkers()

fmt.Printf("Main queue: %d, Overflow: %d, Total: %d, Active: %d\n",
    mainQueue, overflowQueue, totalQueue, activeWorkers)
```

## Shutdown

### Graceful Shutdown

Waits for all queued tasks to complete.

```go
pool.Shutdown()
```

### Immediate Shutdown

Stops immediately, may not execute pending tasks.

```go
pool.ShutdownNow()
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    // Initialize metrics
    metrics := NewThreadPoolMetrics()

    // Start metrics server
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        http.ListenAndServe(":2112", nil)
    }()

    // Create thread pool
    pool := NewThreadPool(ThreadPoolConfig{
        Name:            "workers",
        Workers:         4,
        QueueSize:       10,
        Metrics:         metrics,
        RejectionPolicy: DiscardPolicy,
    })
    defer pool.Shutdown()

    // Example 1: Simple task
    pool.Submit(func() {
        fmt.Println("Hello from thread pool!")
    })

    // Example 2: Task with parameters
    userID := 123
    pool.Submit(func() {
        processUser(userID)
    })

    // Example 3: Force submit (never rejects)
    for i := 0; i < 100; i++ {
        taskID := i
        pool.SubmitForce(func() {
            fmt.Printf("Task %d\n", taskID)
            time.Sleep(100 * time.Millisecond)
        })
    }

    // Example 4: Task with result
    future := pool.SubmitWithResult(func() (interface{}, error) {
        return compute(), nil
    })
    result, err := future.Get()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Result: %v\n", result)
    }

    // Example 5: Cancellable task
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    pool.SubmitWithContext(ctx, func(ctx context.Context) {
        select {
        case <-ctx.Done():
            fmt.Println("Task cancelled")
        case <-time.After(1 * time.Second):
            fmt.Println("Task completed")
        }
    })

    // Example 6: Monitor status
    fmt.Printf("Queue size: %d\n", pool.GetQueueSize())
    fmt.Printf("Active workers: %d\n", pool.GetActiveWorkers())

    // Wait for completion
    time.Sleep(2 * time.Second)
}

func processUser(userID int) {
    fmt.Printf("Processing user %d\n", userID)
}

func compute() int {
    return 42
}
```

## Best Practices

1. **Choose the right submission method:**
   - Use `Submit()` for optional tasks
   - Use `SubmitWait()` for critical tasks that can block
   - Use `SubmitForce()` for critical tasks that can't block
   - Use `SubmitWithResult()` when you need return values

2. **Set appropriate queue size:**
   - Small queue (10-100): Low memory, may reject tasks
   - Large queue (1000+): High memory, rarely rejects
   - Use `SubmitForce()` for unbounded acceptance

3. **Monitor metrics:**
   - Watch rejection rates
   - Monitor queue saturation
   - Track task duration for performance

4. **Handle rejections:**
   - Check return values from `Submit()`
   - Use appropriate rejection policy
   - Implement retry logic for critical tasks

5. **Graceful shutdown:**
   - Always call `Shutdown()` or `ShutdownNow()`
   - Use `defer pool.Shutdown()` pattern

## Thread Safety

All thread pool operations are thread-safe and can be called from multiple goroutines concurrently.

## License

MIT License

## Contributing

Contributions welcome! Please open an issue or submit a pull request.

## Acknowledgments

Built with:
- [Prometheus Go client](https://github.com/prometheus/client_golang)
- Go standard library (sync, context, time)


## 🔑 How to use

- make build → builds the example binary
- make run → runs the example directly
- make test → runs all tests under ./...
- make tidy → cleans up go.mod and go.sum
- make clean → removes the built binary
- make run-metrics → runs the example and exposes Prometheus metrics at http://localhost:2112/metrics



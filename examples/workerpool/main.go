package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	workerpool "github.com/majiddarvishan/snipgo/workerpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize metrics
	metrics := workerpool.NewThreadPoolMetrics()

	// Start Prometheus metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Println("Prometheus metrics available at http://localhost:2112/metrics")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			fmt.Printf("Error starting metrics server: %v\n", err)
		}
	}()

	// Example 1: Pool with discard policy (saves rejected tasks)
	pool1 := workerpool.NewThreadPool(workerpool.ThreadPoolConfig{
		Name:            "discard_pool",
		Workers:         2,
		QueueSize:       3,
		Metrics:         metrics,
		RejectionPolicy: workerpool.DiscardPolicy,
	})

	fmt.Println("=== Example 1: Discard Policy (saves rejected tasks) ===")
	for i := 0; i < 10; i++ {
		taskID := i
		ok := pool1.Submit(func() {
			fmt.Printf("[discard_pool] Task %d executing\n", taskID)
			time.Sleep(300 * time.Millisecond)
		})
		if !ok {
			fmt.Printf("Task %d rejected\n", taskID)
		}
	}

	// Retry rejected tasks
	time.Sleep(1 * time.Second)
	fmt.Println("\n=== Retrying rejected tasks ===")
	count := pool1.RetryRejectedTasks()
	fmt.Printf("Successfully retried %d tasks\n", count)

	// Example 2: Pool with caller runs policy
	pool2 := workerpool.NewThreadPool(workerpool.ThreadPoolConfig{
		Name:            "caller_runs_pool",
		Workers:         2,
		QueueSize:       2,
		Metrics:         metrics,
		RejectionPolicy: workerpool.CallerRunsPolicy,
	})

	fmt.Println("\n=== Example 2: Caller Runs Policy ===")
	for i := 0; i < 5; i++ {
		taskID := i
		pool2.Submit(func() {
			fmt.Printf("[caller_runs_pool] Task %d executing\n", taskID)
			time.Sleep(200 * time.Millisecond)
		})
	}

	// Example 3: Submit with timeout
	pool3 := workerpool.NewThreadPool(workerpool.ThreadPoolConfig{
		Name:            "timeout_pool",
		Workers:         2,
		QueueSize:       5,
		Metrics:         metrics,
		RejectionPolicy: workerpool.DiscardPolicy,
	})

	fmt.Println("\n=== Example 3: Submit with Timeout ===")
	for i := 0; i < 3; i++ {
		taskID := i
		ok := pool3.SubmitWithTimeout(func() {
			fmt.Printf("[timeout_pool] Task %d executing\n", taskID)
			time.Sleep(100 * time.Millisecond)
		}, 500*time.Millisecond)

		if !ok {
			fmt.Printf("Task %d timed out\n", taskID)
		}
	}

	// Example 4: Submit with retry
	fmt.Println("\n=== Example 4: Submit with Retry ===")
	for i := 0; i < 3; i++ {
		taskID := i
		ok := pool3.SubmitWithRetry(func() {
			fmt.Printf("[timeout_pool] Retry task %d executing\n", taskID)
			time.Sleep(100 * time.Millisecond)
		}, 3, 100*time.Millisecond)

		if !ok {
			fmt.Printf("Task %d failed after retries\n", taskID)
		}
	}

	// Example 5: Submit with context (cancellable)
	fmt.Println("\n=== Example 5: Submit with Context ===")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for i := 0; i < 3; i++ {
		taskID := i
		pool3.SubmitWithContext(ctx, func(ctx context.Context) {
			select {
			case <-ctx.Done():
				fmt.Printf("[timeout_pool] Task %d cancelled\n", taskID)
				return
			case <-time.After(500 * time.Millisecond):
				fmt.Printf("[timeout_pool] Context task %d completed\n", taskID)
			}
		})
	}

	// Example 6: Submit with result
	fmt.Println("\n=== Example 6: Submit with Result ===")
	futures := make([]*workerpool.Future, 3)
	for i := 0; i < 3; i++ {
		taskID := i
		futures[i] = pool3.SubmitWithResult(func() (interface{}, error) {
			time.Sleep(200 * time.Millisecond)
			return taskID * taskID, nil
		})
	}

	for i, future := range futures {
		result, err := future.Get()
		if err != nil {
			fmt.Printf("Task %d error: %v\n", i, err)
		} else {
			fmt.Printf("Task %d result: %v\n", i, result)
		}
	}

	// Example 7: Monitor pool status
	fmt.Println("\n=== Example 7: Pool Status ===")
	fmt.Printf("Queue size: %d\n", pool3.GetQueueSize())
	fmt.Printf("Active workers: %d\n", pool3.GetActiveWorkers())

	// Wait for tasks to complete
	time.Sleep(2 * time.Second)

	// Shutdown all pools
	fmt.Println("\n=== Shutting down thread pools ===")
	pool1.Shutdown()
	pool2.Shutdown()
	pool3.Shutdown()
	fmt.Println("Thread pools shut down successfully")
	fmt.Println("\nCheck metrics at http://localhost:2112/metrics")

	// Keep server running
	time.Sleep(5 * time.Second)
}
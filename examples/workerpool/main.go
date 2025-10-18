package main

import (
    "fmt"
    "net/http"
    "time"

    "github.com/prometheus/client_golang/prometheus/promhttp"
    workerpool "github.com/majiddarvishan/snipgo/workerpool"
)

func main() {
    metrics := workerpool.NewThreadPoolMetrics()

	// Start Prometheus metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Println("Prometheus metrics available at http://localhost:2112/metrics")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			fmt.Printf("Error starting metrics server: %v\n", err)
		}
	}()

    pool1 := workerpool.NewThreadPool("main_pool", 3, 10, metrics)
    pool2 := workerpool.NewThreadPool("secondary_pool", 2, 5, metrics)

	fmt.Println("=== Example 1: Simple tasks ===")
	for i := 0; i < 5; i++ {
		taskID := i
		pool1.SubmitWait(func() {
			fmt.Printf("[main_pool] Task %d executing\n", taskID)
			time.Sleep(200 * time.Millisecond)
		})
	}

	fmt.Println("\n=== Example 2: Tasks with results ===")
	futures := make([]*workerpool.Future, 3)
	for i := 0; i < 3; i++ {
		taskID := i
		futures[i] = pool1.SubmitWithResult(func() (interface{}, error) {
			time.Sleep(300 * time.Millisecond)
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

	fmt.Println("\n=== Example 3: Multiple pools ===")
	for i := 0; i < 5; i++ {
		taskID := i
		pool2.SubmitWait(func() {
			fmt.Printf("[secondary_pool] Task %d executing\n", taskID)
			time.Sleep(150 * time.Millisecond)
		})
	}

	fmt.Println("\n=== Example 4: Task rejection (queue full) ===")
	for i := 0; i < 15; i++ {
		taskID := i
		ok := pool2.Submit(func() {
			fmt.Printf("[secondary_pool] Task %d executing\n", taskID)
			time.Sleep(500 * time.Millisecond)
		})
		if !ok {
			fmt.Printf("Task %d rejected (queue full)\n", taskID)
		}
	}

	// Wait for metrics to be collected
	time.Sleep(2 * time.Second)

	fmt.Println("\n=== Shutting down thread pools ===")
	pool1.Shutdown()
	pool2.Shutdown()
	fmt.Println("Thread pools shut down successfully")
	fmt.Println("\nCheck metrics at http://localhost:2112/metrics")

	// Keep server running to check metrics
	time.Sleep(5 * time.Second)
}
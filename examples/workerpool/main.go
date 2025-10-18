package main

import (
    // "context"
    "fmt"
    // "net/http"
    "time"

    // "github.com/prometheus/client_golang/prometheus/promhttp"
    workerpool "github.com/majiddarvishan/snipgo/workerpool"
)

// func main() {
//     // Start Prometheus metrics endpoint
//     go func() {
//         http.Handle("/metrics", promhttp.Handler())
//         http.ListenAndServe(":2112", nil)
//     }()

//     ctx := context.Background()
//     metrics := workerpool.NewMetrics("myapp", "workerpool")

//     pool := workerpool.New[int](ctx, workerpool.Options[int]{
//         NumWorkers:      4,
//         QueueSize:       20,
//         DrainOnShutdown: true,
//         Metrics:         metrics, // attach metrics
//     })

//     for i := 0; i < 10; i++ {
//         n := i
//         _ = pool.Submit(func(ctx context.Context) (int, error) {
//             time.Sleep(200 * time.Millisecond)
//             return n * n, nil
//         })
//     }

//     go func() {
//         for res := range pool.Results() {
//             if res.Err == nil {
//                 fmt.Println("Result:", res.Value)
//             }
//         }
//     }()

//     time.Sleep(3 * time.Second)
//     pool.Shutdown()
// }

func main() {
	pool := workerpool.NewThreadPool(3, 10)

	fmt.Println("=== Example 1: Simple tasks without return ===")
	for i := 0; i < 3; i++ {
		taskID := i
		pool.SubmitWait(func() {
			fmt.Printf("Simple task %d executing\n", taskID)
			time.Sleep(200 * time.Millisecond)
		})
	}

	fmt.Println("\n=== Example 2: Tasks with parameters ===")
	for i := 0; i < 3; i++ {
		taskID := i
		pool.SubmitWait(func() {
			processTask(taskID, fmt.Sprintf("data-%d", taskID))
		})
	}

	fmt.Println("\n=== Example 3: Tasks with results ===")
	// Submit tasks that return results
	futures := make([]*workerpool.Future, 5)
	for i := 0; i < 5; i++ {
		taskID := i
		futures[i] = pool.SubmitWithResult(func() (interface{}, error) {
			return calculateSquare(taskID)
		})
	}

	// Retrieve results
	for i, future := range futures {
		result, err := future.Get()
		if err != nil {
			fmt.Printf("Task %d error: %v\n", i, err)
		} else {
			fmt.Printf("Task %d result: %v\n", i, result)
		}
	}

	fmt.Println("\n=== Example 4: Tasks with complex results ===")
	// Task returning struct
	type UserInfo struct {
		ID   int
		Name string
		Age  int
	}

	future := pool.SubmitWithResult(func() (interface{}, error) {
		time.Sleep(300 * time.Millisecond)
		return UserInfo{ID: 1, Name: "Alice", Age: 30}, nil
	})

	result, err := future.Get()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		user := result.(UserInfo)
		fmt.Printf("User: ID=%d, Name=%s, Age=%d\n", user.ID, user.Name, user.Age)
	}

	fmt.Println("\n=== Example 5: Tasks with errors ===")
	// Task that returns an error
	errorFuture := pool.SubmitWithResult(func() (interface{}, error) {
		return nil, fmt.Errorf("something went wrong")
	})

	result, err = errorFuture.Get()
	if err != nil {
		fmt.Printf("Got expected error: %v\n", err)
	}

	fmt.Println("\n=== Example 6: Tasks with timeout ===")
	slowFuture := pool.SubmitWithResult(func() (interface{}, error) {
		time.Sleep(2 * time.Second)
		return "slow result", nil
	})

	result, err, ok := slowFuture.GetWithTimeout(500 * time.Millisecond)
	if !ok {
		fmt.Println("Task timed out")
	} else {
		fmt.Printf("Result: %v, Error: %v\n", result, err)
	}

	fmt.Println("\n=== Example 7: Check if done ===")
	quickFuture := pool.SubmitWithResult(func() (interface{}, error) {
		return "quick result", nil
	})

	time.Sleep(100 * time.Millisecond)
	if quickFuture.IsDone() {
		result, _ := quickFuture.Get()
		fmt.Printf("Task completed with result: %v\n", result)
	}

	fmt.Println("\n=== Shutting down thread pool ===")
	pool.Shutdown()
	fmt.Println("Thread pool shut down successfully")
}

// Example task functions

func processTask(id int, data string) {
	fmt.Printf("Processing task %d with data: %s\n", id, data)
	time.Sleep(200 * time.Millisecond)
}

func calculateSquare(n int) (interface{}, error) {
	fmt.Printf("Calculating square of %d\n", n)
	time.Sleep(300 * time.Millisecond)
	return n * n, nil
}
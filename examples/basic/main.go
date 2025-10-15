package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/prometheus/client_golang/prometheus/promhttp"
    workerpool "github.com/majiddarvishan/workerpool/pkg/workerpool"
)

func main() {
    // Start Prometheus metrics endpoint
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        http.ListenAndServe(":2112", nil)
    }()

    ctx := context.Background()
    metrics := workerpool.NewMetrics("myapp", "workerpool")

    pool := workerpool.New[int](ctx, workerpool.Options[int]{
        NumWorkers:      4,
        QueueSize:       20,
        DrainOnShutdown: true,
        Metrics:         metrics, // attach metrics
    })

    for i := 0; i < 10; i++ {
        n := i
        _ = pool.Submit(func(ctx context.Context) (int, error) {
            time.Sleep(200 * time.Millisecond)
            return n * n, nil
        })
    }

    go func() {
        for res := range pool.Results() {
            if res.Err == nil {
                fmt.Println("Result:", res.Value)
            }
        }
    }()

    time.Sleep(3 * time.Second)
    pool.Shutdown()
}

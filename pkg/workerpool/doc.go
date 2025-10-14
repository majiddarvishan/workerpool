// Package workerpool provides a generic, tunable worker pool with
// support for context cancellation, dynamic resizing, lifecycle hooks,
// panic recovery, and Prometheus metrics integration.
//
// Typical usage:
//
//   ctx := context.Background()
//   metrics := workerpool.NewMetrics("myapp", "workerpool")
//   pool := workerpool.New[int](ctx, workerpool.Options[int]{
//       NumWorkers:      4,
//       QueueSize:       20,
//       DrainOnShutdown: true,
//       Metrics:         metrics,
//   })
//
//   for i := 0; i < 10; i++ {
//       n := i
//       _ = pool.Submit(func(ctx context.Context) (int, error) {
//           return n * n, nil
//       })
//   }
//
//   go func() {
//       for res := range pool.Results() {
//           if res.Err == nil {
//               fmt.Println("Result:", res.Value)
//           }
//       }
//   }()
//
//   pool.Shutdown()
//
// This package is designed for production use: it is safe for concurrent
// submission, supports observability via Prometheus, and allows graceful
// or immediate shutdown depending on configuration.
package workerpool

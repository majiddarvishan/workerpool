package workerpool

import (
    "context"
    "errors"
    "fmt"
    "sync"
    "sync/atomic"
    "time"
)

// Result wraps the outcome of a task.
type Result[T any] struct {
    Value T
    Err   error
}

// Task is a unit of work.
type Task[T any] func(ctx context.Context) (T, error)

// Pool is a tunable worker pool.
type Pool[T any] struct {
    tasks    chan Task[T]
    results  chan Result[T]
    ctx      context.Context
    cancel   context.CancelFunc
    wg       sync.WaitGroup
    opts     Options[T]

    // metrics
    activeWorkers int32
    completed     int64
    failed        int64

    mu         sync.Mutex
    numWorkers int
    shutdown   bool
}

// New creates a new pool with the given options.
func New[T any](parent context.Context, opts Options[T]) *Pool[T] {
    if opts.NumWorkers <= 0 {
        opts.NumWorkers = 1
    }
    if opts.QueueSize <= 0 {
        opts.QueueSize = 1
    }
    ctx, cancel := context.WithCancel(parent)
    p := &Pool[T]{
        tasks:      make(chan Task[T], opts.QueueSize),
        results:    make(chan Result[T], opts.QueueSize),
        ctx:        ctx,
        cancel:     cancel,
        opts:       opts,
        numWorkers: opts.NumWorkers,
    }
    p.start(opts.NumWorkers)
    return p
}

func (p *Pool[T]) start(n int) {
    for i := 0; i < n; i++ {
        p.wg.Add(1)
        go p.worker()
    }
    atomic.AddInt32(&p.activeWorkers, int32(n))
    if p.opts.Metrics != nil {
        p.opts.Metrics.ActiveWorkers.Set(float64(p.numWorkers))
    }
}

func (p *Pool[T]) worker() {
    defer p.wg.Done()
    for {
        select {
        case <-p.ctx.Done():
            return
        case task, ok := <-p.tasks:
            if !ok {
                return
            }
            if p.opts.Hooks.OnStart != nil {
                p.opts.Hooks.OnStart()
            }
            var res Result[T]
            start := time.Now()
            func() {
                defer func() {
                    if r := recover(); r != nil {
                        err := fmt.Errorf("panic: %v", r)
                        res.Err = err
                        if p.opts.Hooks.OnError != nil {
                            p.opts.Hooks.OnError(err)
                        }
                    }
                }()
                val, err := task(p.ctx)
                res = Result[T]{Value: val, Err: err}
            }()
            latency := time.Since(start).Seconds()
            if p.opts.Metrics != nil {
                p.opts.Metrics.TaskLatency.Observe(latency)
            }
            if res.Err != nil {
                atomic.AddInt64(&p.failed, 1)
                if p.opts.Metrics != nil {
                    p.opts.Metrics.TasksFailed.Inc()
                }
                if p.opts.Hooks.OnError != nil {
                    p.opts.Hooks.OnError(res.Err)
                }
            } else {
                atomic.AddInt64(&p.completed, 1)
                if p.opts.Metrics != nil {
                    p.opts.Metrics.TasksCompleted.Inc()
                }
            }
            if p.opts.Hooks.OnFinish != nil {
                p.opts.Hooks.OnFinish(res)
            }
            p.results <- res
        }
    }
}

// Submit enqueues a task.
func (p *Pool[T]) Submit(task Task[T]) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.shutdown {
        return errors.New("pool shutting down")
    }
    if p.opts.Hooks.OnSubmit != nil {
        p.opts.Hooks.OnSubmit()
    }
    if p.opts.Metrics != nil {
        p.opts.Metrics.TasksSubmitted.Inc()
    }
    select {
    case p.tasks <- task:
        return nil
    case <-p.ctx.Done():
        return errors.New("pool closed")
    }
}

// Resize changes the number of workers dynamically.
func (p *Pool[T]) Resize(n int) {
    p.mu.Lock()
    defer p.mu.Unlock()
    diff := n - p.numWorkers
    if diff > 0 {
        p.start(diff)
    } else if diff < 0 {
        // send poison pills
        for i := 0; i < -diff; i++ {
            p.tasks <- func(ctx context.Context) (T, error) {
                return *new(T), errors.New("worker exit")
            }
        }
    }
    p.numWorkers = n
    if p.opts.Metrics != nil {
        p.opts.Metrics.ActiveWorkers.Set(float64(p.numWorkers))
    }
}

// Results returns the results channel.
func (p *Pool[T]) Results() <-chan Result[T] {
    return p.results
}

// Metrics snapshot.
func (p *Pool[T]) Metrics() (active int32, completed, failed int64, queued int) {
    return atomic.LoadInt32(&p.activeWorkers),
        atomic.LoadInt64(&p.completed),
        atomic.LoadInt64(&p.failed),
        len(p.tasks)
}

// Shutdown stops the pool.
func (p *Pool[T]) Shutdown() {
    p.mu.Lock()
    if p.shutdown {
        p.mu.Unlock()
        return
    }
    p.shutdown = true
    p.mu.Unlock()

    if !p.opts.DrainOnShutdown {
        p.cancel()
    }
    close(p.tasks)
    p.wg.Wait()
    close(p.results)
}

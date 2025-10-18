package workerpool

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Task represents a unit of work
type Task func()

// TaskWithContext represents a unit of work that can be cancelled
type TaskWithContext func(ctx context.Context)

// Future represents a future result of an asynchronous computation
type Future struct {
	result interface{}
	err    error
	done   chan struct{}
	once   sync.Once
}

// Get blocks until the result is available and returns it
func (f *Future) Get() (interface{}, error) {
	<-f.done
	return f.result, f.err
}

// GetWithTimeout waits for the result with a timeout
func (f *Future) GetWithTimeout(timeout time.Duration) (interface{}, error, bool) {
	select {
	case <-f.done:
		return f.result, f.err, true
	case <-time.After(timeout):
		return nil, nil, false
	}
}

// IsDone checks if the task has completed
func (f *Future) IsDone() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

// RejectionPolicy defines how to handle rejected tasks
type RejectionPolicy int

const (
	// DiscardPolicy discards rejected tasks
	DiscardPolicy RejectionPolicy = iota
	// CallerRunsPolicy runs rejected tasks in caller's goroutine
	CallerRunsPolicy
	// AbortPolicy panics on rejected tasks
	AbortPolicy
)

// ThreadPoolConfig holds configuration for thread pool
type ThreadPoolConfig struct {
	Name            string
	Workers         int
	QueueSize       int
	Metrics         *ThreadPoolMetrics
	RejectionPolicy RejectionPolicy
}

// ThreadPool manages a pool of worker goroutines
type ThreadPool struct {
	name            string
	workers         int
	taskQueue       chan Task
	wg              sync.WaitGroup
	quit            chan struct{}
	once            sync.Once
	activeWorkerMu  sync.Mutex
	activeCount     int
	metrics         *ThreadPoolMetrics
	rejectionPolicy RejectionPolicy
	rejectedTasks   []Task
	rejectedMu      sync.Mutex
}

// NewThreadPool creates a new thread pool with the specified configuration
func NewThreadPool(config ThreadPoolConfig) *ThreadPool {
	if config.Workers <= 0 {
		config.Workers = 1
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 100
	}

	tp := &ThreadPool{
		name:            config.Name,
		workers:         config.Workers,
		taskQueue:       make(chan Task, config.QueueSize),
		quit:            make(chan struct{}),
		metrics:         config.Metrics,
		rejectionPolicy: config.RejectionPolicy,
		rejectedTasks:   make([]Task, 0),
	}

	// Initialize metrics
	if tp.metrics != nil {
		tp.metrics.SetWorkerCount(config.Name, config.Workers)
		tp.metrics.SetQueueSize(config.Name, 0)
		tp.metrics.SetActiveWorkers(config.Name, 0)
	}

	// Start worker goroutines
	tp.wg.Add(config.Workers)
	for i := 0; i < config.Workers; i++ {
		go tp.worker(i)
	}

	// Start queue size monitor
	go tp.monitorQueueSize()

	return tp
}

// worker is the goroutine that processes tasks
func (tp *ThreadPool) worker(id int) {
	defer tp.wg.Done()

	for {
		select {
		case task, ok := <-tp.taskQueue:
			if !ok {
				return
			}

			tp.incrementActiveWorkers()

			start := time.Now()
			func() {
				defer func() {
					duration := time.Since(start).Seconds()
					if tp.metrics != nil {
						tp.metrics.ObserveTaskDuration(tp.name, duration)
					}

					if r := recover(); r != nil {
						if tp.metrics != nil {
							tp.metrics.RecordTaskFailed(tp.name)
							tp.metrics.RecordTaskCompleted(tp.name, "failed")
						}
						fmt.Printf("[%s] Task panicked: %v\n", tp.name, r)
					} else {
						if tp.metrics != nil {
							tp.metrics.RecordTaskCompleted(tp.name, "success")
						}
					}

					tp.decrementActiveWorkers()
				}()
				task()
			}()

		case <-tp.quit:
			return
		}
	}
}

// incrementActiveWorkers increments the active worker count
func (tp *ThreadPool) incrementActiveWorkers() {
	tp.activeWorkerMu.Lock()
	tp.activeCount++
	if tp.metrics != nil {
		tp.metrics.SetActiveWorkers(tp.name, tp.activeCount)
	}
	tp.activeWorkerMu.Unlock()
}

// decrementActiveWorkers decrements the active worker count
func (tp *ThreadPool) decrementActiveWorkers() {
	tp.activeWorkerMu.Lock()
	tp.activeCount--
	if tp.metrics != nil {
		tp.metrics.SetActiveWorkers(tp.name, tp.activeCount)
	}
	tp.activeWorkerMu.Unlock()
}

// monitorQueueSize monitors and reports queue size
func (tp *ThreadPool) monitorQueueSize() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if tp.metrics != nil {
				tp.metrics.SetQueueSize(tp.name, len(tp.taskQueue))
			}
		case <-tp.quit:
			return
		}
	}
}

// handleRejection handles a rejected task based on the rejection policy
func (tp *ThreadPool) handleRejection(task Task) {
	switch tp.rejectionPolicy {
	case CallerRunsPolicy:
		// Run in caller's goroutine
		task()
	case AbortPolicy:
		panic(fmt.Sprintf("[%s] Task rejected: queue full", tp.name))
	case DiscardPolicy:
		// Save to rejected tasks list
		tp.rejectedMu.Lock()
		tp.rejectedTasks = append(tp.rejectedTasks, task)
		tp.rejectedMu.Unlock()
	}
}

// Submit adds a task to the thread pool (non-blocking)
// Returns false if queue is full or pool is shutting down
func (tp *ThreadPool) Submit(task Task) bool {
	if tp.metrics != nil {
		tp.metrics.RecordTaskSubmitted(tp.name)
	}

	select {
	case tp.taskQueue <- task:
		return true
	case <-tp.quit:
		if tp.metrics != nil {
			tp.metrics.RecordTaskRejected(tp.name)
		}
		tp.handleRejection(task)
		return false
	default:
		if tp.metrics != nil {
			tp.metrics.RecordTaskRejected(tp.name)
		}
		tp.handleRejection(task)
		return false
	}
}

// SubmitWait adds a task and blocks until it can be queued
// This guarantees the task will be executed (unless pool shuts down)
func (tp *ThreadPool) SubmitWait(task Task) bool {
	if tp.metrics != nil {
		tp.metrics.RecordTaskSubmitted(tp.name)
	}

	select {
	case tp.taskQueue <- task:
		return true
	case <-tp.quit:
		if tp.metrics != nil {
			tp.metrics.RecordTaskRejected(tp.name)
		}
		return false
	}
}

// SubmitWithTimeout tries to submit a task with a timeout
// Returns true if task was queued, false if timeout or shutdown
func (tp *ThreadPool) SubmitWithTimeout(task Task, timeout time.Duration) bool {
	if tp.metrics != nil {
		tp.metrics.RecordTaskSubmitted(tp.name)
	}

	select {
	case tp.taskQueue <- task:
		return true
	case <-tp.quit:
		if tp.metrics != nil {
			tp.metrics.RecordTaskRejected(tp.name)
		}
		return false
	case <-time.After(timeout):
		if tp.metrics != nil {
			tp.metrics.RecordTaskTimedOut(tp.name)
		}
		tp.handleRejection(task)
		return false
	}
}

// SubmitWithRetry attempts to submit a task with retries
// Retries 'attempts' times with 'delay' between each attempt
// Returns true if task was queued, false if all attempts failed
func (tp *ThreadPool) SubmitWithRetry(task Task, attempts int, delay time.Duration) bool {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			if tp.metrics != nil {
				tp.metrics.RecordTaskRetried(tp.name)
			}
			time.Sleep(delay)
		}
		if tp.Submit(task) {
			return true
		}
	}
	return false
}

// SubmitWithContext submits a task that can be cancelled via context
func (tp *ThreadPool) SubmitWithContext(ctx context.Context, fn TaskWithContext) bool {
	task := func() {
		fn(ctx)
	}
	return tp.Submit(task)
}

// SubmitWithResult submits a task that returns a result and error
func (tp *ThreadPool) SubmitWithResult(fn func() (interface{}, error)) *Future {
	future := &Future{
		done: make(chan struct{}),
	}

	task := func() {
		defer func() {
			if r := recover(); r != nil {
				future.err = fmt.Errorf("panic: %v", r)
			}
			future.once.Do(func() {
				close(future.done)
			})
		}()

		future.result, future.err = fn()
	}

	tp.SubmitWait(task)
	return future
}

// SubmitWithResultNonBlocking submits a task that returns a result (non-blocking)
func (tp *ThreadPool) SubmitWithResultNonBlocking(fn func() (interface{}, error)) *Future {
	future := &Future{
		done: make(chan struct{}),
	}

	task := func() {
		defer func() {
			if r := recover(); r != nil {
				future.err = fmt.Errorf("panic: %v", r)
			}
			future.once.Do(func() {
				close(future.done)
			})
		}()

		future.result, future.err = fn()
	}

	if tp.Submit(task) {
		return future
	}
	return nil
}

// GetRejectedTasks returns and clears the list of rejected tasks
func (tp *ThreadPool) GetRejectedTasks() []Task {
	tp.rejectedMu.Lock()
	defer tp.rejectedMu.Unlock()

	tasks := make([]Task, len(tp.rejectedTasks))
	copy(tasks, tp.rejectedTasks)
	tp.rejectedTasks = tp.rejectedTasks[:0]
	return tasks
}

// RetryRejectedTasks attempts to resubmit all rejected tasks
func (tp *ThreadPool) RetryRejectedTasks() int {
	tasks := tp.GetRejectedTasks()
	successCount := 0

	for _, task := range tasks {
		if tp.SubmitWait(task) {
			successCount++
		}
	}

	return successCount
}

// GetQueueSize returns the current number of tasks in the queue
func (tp *ThreadPool) GetQueueSize() int {
	return len(tp.taskQueue)
}

// GetActiveWorkers returns the current number of active workers
func (tp *ThreadPool) GetActiveWorkers() int {
	tp.activeWorkerMu.Lock()
	defer tp.activeWorkerMu.Unlock()
	return tp.activeCount
}

// Shutdown gracefully shuts down the thread pool
func (tp *ThreadPool) Shutdown() {
	tp.once.Do(func() {
		close(tp.taskQueue)
		tp.wg.Wait()
		close(tp.quit)
	})
}

// ShutdownNow immediately stops the thread pool
func (tp *ThreadPool) ShutdownNow() {
	tp.once.Do(func() {
		close(tp.quit)
		tp.wg.Wait()
		close(tp.taskQueue)
	})
}

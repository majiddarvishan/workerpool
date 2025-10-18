package workerpool

import (
	"fmt"
	"sync"
	"time"
)

// Task represents a unit of work
type Task func()

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

// ThreadPool manages a pool of worker goroutines
type ThreadPool struct {
	workers   int
	taskQueue chan Task
	wg        sync.WaitGroup
	quit      chan struct{}
	once      sync.Once
}

// NewThreadPool creates a new thread pool with the specified number of workers
func NewThreadPool(workers int, queueSize int) *ThreadPool {
	if workers <= 0 {
		workers = 1
	}
	if queueSize <= 0 {
		queueSize = 100
	}

	tp := &ThreadPool{
		workers:   workers,
		taskQueue: make(chan Task, queueSize),
		quit:      make(chan struct{}),
	}

	// Start worker goroutines
	tp.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go tp.worker(i)
	}

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
			task()
		case <-tp.quit:
			return
		}
	}
}

// Submit adds a task to the thread pool (non-blocking)
func (tp *ThreadPool) Submit(task Task) bool {
	select {
	case tp.taskQueue <- task:
		return true
	case <-tp.quit:
		return false
	default:
		return false
	}
}

// SubmitWait adds a task and blocks until it can be queued
func (tp *ThreadPool) SubmitWait(task Task) bool {
	select {
	case tp.taskQueue <- task:
		return true
	case <-tp.quit:
		return false
	}
}

// SubmitWithResult submits a task that returns a result and error
// Returns a Future to retrieve the result later
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
// Returns nil if queue is full or shutting down
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

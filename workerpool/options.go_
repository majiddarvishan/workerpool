package workerpool

// Hooks let you observe pool lifecycle events.
type Hooks[T any] struct {
    OnSubmit func()
    OnStart  func()
    OnFinish func(Result[T])
    OnError  func(error)
}

// Options configure the pool.
type Options[T any] struct {
    NumWorkers      int
    QueueSize       int
    Hooks           Hooks[T]
    DrainOnShutdown bool
    Metrics         *Metrics
}

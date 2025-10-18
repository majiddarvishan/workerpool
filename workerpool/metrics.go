package workerpool

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ThreadPoolMetrics holds all Prometheus metrics for thread pools
type ThreadPoolMetrics struct {
	TasksSubmitted  *prometheus.CounterVec
	TasksCompleted  *prometheus.CounterVec
	TasksFailed     *prometheus.CounterVec
	TasksRejected   *prometheus.CounterVec
	TaskDuration    *prometheus.HistogramVec
	QueueSize       *prometheus.GaugeVec
	ActiveWorkers   *prometheus.GaugeVec
	WorkerCount     *prometheus.GaugeVec
}

// NewThreadPoolMetrics creates and registers all thread pool metrics
func NewThreadPoolMetrics() *ThreadPoolMetrics {
	return &ThreadPoolMetrics{
		TasksSubmitted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "threadpool_tasks_submitted_total",
				Help: "Total number of tasks submitted to the thread pool",
			},
			[]string{"pool_name"},
		),
		TasksCompleted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "threadpool_tasks_completed_total",
				Help: "Total number of tasks completed by the thread pool",
			},
			[]string{"pool_name", "status"},
		),
		TasksFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "threadpool_tasks_failed_total",
				Help: "Total number of tasks that failed",
			},
			[]string{"pool_name"},
		),
		TasksRejected: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "threadpool_tasks_rejected_total",
				Help: "Total number of tasks rejected (queue full)",
			},
			[]string{"pool_name"},
		),
		TaskDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "threadpool_task_duration_seconds",
				Help:    "Duration of task execution in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"pool_name"},
		),
		QueueSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "threadpool_queue_size",
				Help: "Current number of tasks in the queue",
			},
			[]string{"pool_name"},
		),
		ActiveWorkers: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "threadpool_active_workers",
				Help: "Current number of active workers",
			},
			[]string{"pool_name"},
		),
		WorkerCount: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "threadpool_worker_count",
				Help: "Total number of workers in the pool",
			},
			[]string{"pool_name"},
		),
	}
}

// RecordTaskSubmitted increments the submitted tasks counter
func (m *ThreadPoolMetrics) RecordTaskSubmitted(poolName string) {
	m.TasksSubmitted.WithLabelValues(poolName).Inc()
}

// RecordTaskCompleted increments the completed tasks counter
func (m *ThreadPoolMetrics) RecordTaskCompleted(poolName string, status string) {
	m.TasksCompleted.WithLabelValues(poolName, status).Inc()
}

// RecordTaskFailed increments the failed tasks counter
func (m *ThreadPoolMetrics) RecordTaskFailed(poolName string) {
	m.TasksFailed.WithLabelValues(poolName).Inc()
}

// RecordTaskRejected increments the rejected tasks counter
func (m *ThreadPoolMetrics) RecordTaskRejected(poolName string) {
	m.TasksRejected.WithLabelValues(poolName).Inc()
}

// ObserveTaskDuration records task execution duration
func (m *ThreadPoolMetrics) ObserveTaskDuration(poolName string, duration float64) {
	m.TaskDuration.WithLabelValues(poolName).Observe(duration)
}

// SetQueueSize sets the current queue size
func (m *ThreadPoolMetrics) SetQueueSize(poolName string, size int) {
	m.QueueSize.WithLabelValues(poolName).Set(float64(size))
}

// SetActiveWorkers sets the current number of active workers
func (m *ThreadPoolMetrics) SetActiveWorkers(poolName string, count int) {
	m.ActiveWorkers.WithLabelValues(poolName).Set(float64(count))
}

// SetWorkerCount sets the total number of workers
func (m *ThreadPoolMetrics) SetWorkerCount(poolName string, count int) {
	m.WorkerCount.WithLabelValues(poolName).Set(float64(count))
}

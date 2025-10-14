package workerpool

import (
    "github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
    TasksSubmitted prometheus.Counter
    TasksCompleted prometheus.Counter
    TasksFailed    prometheus.Counter
    ActiveWorkers  prometheus.Gauge
    TaskLatency    prometheus.Histogram
}

func NewMetrics(namespace, subsystem string) *Metrics {
    m := &Metrics{
        TasksSubmitted: prometheus.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "tasks_submitted_total",
            Help:      "Total number of tasks submitted to the pool",
        }),
        TasksCompleted: prometheus.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "tasks_completed_total",
            Help:      "Total number of tasks completed successfully",
        }),
        TasksFailed: prometheus.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "tasks_failed_total",
            Help:      "Total number of tasks that failed",
        }),
        ActiveWorkers: prometheus.NewGauge(prometheus.GaugeOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "active_workers",
            Help:      "Current number of active workers",
        }),
        TaskLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "task_latency_seconds",
            Help:      "Histogram of task execution latency",
            Buckets:   prometheus.DefBuckets,
        }),
    }

    // Register metrics with Prometheus default registry
    prometheus.MustRegister(
        m.TasksSubmitted,
        m.TasksCompleted,
        m.TasksFailed,
        m.ActiveWorkers,
        m.TaskLatency,
    )

    return m
}

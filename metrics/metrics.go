package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// path and method are the only labels here (never identifier/client_id) to
// keep label cardinality bounded to the small, fixed set of real routes.
var (
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "throttle_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method", "status"},
	)

	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "throttle_http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"path", "method", "status"},
	)

	UsageJobsDropped = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "throttle_usage_jobs_dropped_total",
			Help: "Total usage/stats jobs dropped because the worker queue was full",
		},
	)

	UsageQueueLength = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "throttle_usage_queue_length",
			Help: "Current number of queued usage jobs waiting to be processed",
		},
	)
)

package handlers

import (
	"log/slog"

	"github.com/Alokxk/Throttle/metrics"
)

const (
	usageWorkerCount = 10
	usageQueueSize   = 1000
)

type usageJob struct {
	clientID   string
	identifier string
	algorithm  string
	allowed    bool
	requestID  string
}

func (h *Handler) startUsageWorkers() {
	for i := 0; i < usageWorkerCount; i++ {
		go h.usageWorker()
	}
}

func (h *Handler) usageWorker() {
	for job := range h.usageJobs {
		h.logUsage(job.clientID, job.identifier, job.algorithm, job.allowed, job.requestID)
		h.incrementStats(job.clientID, job.algorithm, job.allowed, job.requestID)
	}
}

// Non-blocking: dropping a stats job under saturation beats blocking
// the rate-limit check that the response actually depends on.
func (h *Handler) enqueueUsage(job usageJob) {
	select {
	case h.usageJobs <- job:
	default:
		metrics.UsageJobsDropped.Inc()
		slog.Warn("usage job queue full, dropping usage record", "client_id", job.clientID, "identifier", job.identifier, "request_id", job.requestID)
	}
	metrics.UsageQueueLength.Set(float64(len(h.usageJobs)))
}

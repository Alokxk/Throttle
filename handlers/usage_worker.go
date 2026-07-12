package handlers

import "log"

const (
	usageWorkerCount = 10
	usageQueueSize   = 1000
)

type usageJob struct {
	clientID   string
	identifier string
	algorithm  string
	allowed    bool
}

func (h *Handler) startUsageWorkers() {
	for i := 0; i < usageWorkerCount; i++ {
		go h.usageWorker()
	}
}

func (h *Handler) usageWorker() {
	for job := range h.usageJobs {
		h.logUsage(job.clientID, job.identifier, job.algorithm, job.allowed)
		h.incrementStats(job.clientID, job.algorithm, job.allowed)
	}
}

// Non-blocking: dropping a stats job under saturation beats blocking
// the rate-limit check that the response actually depends on.
func (h *Handler) enqueueUsage(job usageJob) {
	select {
	case h.usageJobs <- job:
	default:
		log.Printf("usage job queue full, dropping usage record for client=%s identifier=%s", job.clientID, job.identifier)
	}
}

package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Alokxk/Throttle/metrics"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Metrics(path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		status := strconv.Itoa(rec.status)
		metrics.RequestDuration.WithLabelValues(path, r.Method, status).Observe(time.Since(start).Seconds())
		metrics.RequestsTotal.WithLabelValues(path, r.Method, status).Inc()
	}
}

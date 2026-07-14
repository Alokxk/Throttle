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

// path is passed explicitly (the route pattern, e.g. "/rules/") rather than
// read from r.URL.Path (e.g. "/rules/api_default") — using the real URL
// would give the path label unbounded cardinality, one series per rule name.
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

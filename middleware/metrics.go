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

// Metrics takes the route pattern explicitly (rather than reading
// r.URL.Path) so labels stay a small fixed set of real routes, not one
// label value per identifier/rule name a caller happens to send.
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

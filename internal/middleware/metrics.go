package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/steviee/github-workflow-article/internal/handler"
)

// metricsResponseWriter wraps http.ResponseWriter to capture the status code for metrics
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code before writing
func (mrw *metricsResponseWriter) WriteHeader(code int) {
	if !mrw.written {
		mrw.statusCode = code
		mrw.written = true
		mrw.ResponseWriter.WriteHeader(code)
	}
}

// Write ensures status code is set even if WriteHeader is not called explicitly
func (mrw *metricsResponseWriter) Write(b []byte) (int, error) {
	if !mrw.written {
		mrw.WriteHeader(http.StatusOK)
	}
	return mrw.ResponseWriter.Write(b)
}

// Metrics middleware collects HTTP request metrics for Prometheus
// It tracks request counts by method, path, and status code, as well as request duration
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record start time
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			written:        false,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Record metrics
		method := r.Method
		path := r.URL.Path
		status := strconv.Itoa(wrapped.statusCode)

		// Increment request counter with labels
		handler.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()

		// Observe request duration
		handler.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
	})
}

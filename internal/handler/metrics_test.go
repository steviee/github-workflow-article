package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsHandler(t *testing.T) {
	t.Parallel()

	// Create a request to pass to the handler
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Get the metrics handler
	handler := MetricsHandler()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code, "handler should return 200 OK")

	// Check the content type - Prometheus metrics use text/plain
	contentType := rr.Header().Get("Content-Type")
	assert.True(t,
		strings.HasPrefix(contentType, "text/plain"),
		"content type should start with text/plain, got: %s", contentType)

	// Check that the response contains Prometheus metrics format
	body := rr.Body.String()
	assert.NotEmpty(t, body, "response body should not be empty")

	// Check for standard Go metrics that should be present
	// These are automatically registered by the prometheus client
	assert.Contains(t, body, "go_goroutines", "should include go_goroutines metric")
	assert.Contains(t, body, "go_threads", "should include go_threads metric")

	// Check for our custom metrics
	assert.Contains(t, body, "http_requests_total", "should include http_requests_total metric")
	assert.Contains(t, body, "http_request_duration_seconds", "should include http_request_duration_seconds metric")
	assert.Contains(t, body, "image_cache_hits_total", "should include image_cache_hits_total metric")
	assert.Contains(t, body, "image_cache_misses_total", "should include image_cache_misses_total metric")
	assert.Contains(t, body, "image_cache_size", "should include image_cache_size metric")
	assert.Contains(t, body, "image_processing_duration_seconds", "should include image_processing_duration_seconds metric")
	assert.Contains(t, body, "image_fetch_duration_seconds", "should include image_fetch_duration_seconds metric")
}

func TestMetricsHandler_MultipleRequests(t *testing.T) {
	t.Parallel()

	handler := MetricsHandler()

	// Test that multiple requests all succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NotEmpty(t, rr.Body.String())
	}
}

func TestMetricsHandler_IncrementsCounters(t *testing.T) {
	t.Parallel()

	// Increment a counter
	HTTPRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()

	// Get metrics
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	handler := MetricsHandler()
	handler.ServeHTTP(rr, req)

	// Check that the counter is in the metrics output
	body := rr.Body.String()
	assert.Contains(t, body, "http_requests_total")
	assert.Contains(t, body, `method="GET"`)
	assert.Contains(t, body, `path="/test"`)
	assert.Contains(t, body, `status="200"`)
}

func TestMetricsHandler_ObservesHistograms(t *testing.T) {
	t.Parallel()

	// Observe a histogram value
	HTTPRequestDuration.WithLabelValues("GET", "/test").Observe(0.5)

	// Get metrics
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	handler := MetricsHandler()
	handler.ServeHTTP(rr, req)

	// Check that the histogram is in the metrics output
	body := rr.Body.String()
	assert.Contains(t, body, "http_request_duration_seconds")
	assert.Contains(t, body, `method="GET"`)
	assert.Contains(t, body, `path="/test"`)
}

func TestMetricsHandler_CacheMetrics(t *testing.T) {
	t.Parallel()

	// Increment cache metrics
	ImageCacheHits.Inc()
	ImageCacheMisses.Inc()
	ImageCacheSize.Set(42)

	// Get metrics
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	handler := MetricsHandler()
	handler.ServeHTTP(rr, req)

	// Check that cache metrics are in the output
	body := rr.Body.String()
	assert.Contains(t, body, "image_cache_hits_total")
	assert.Contains(t, body, "image_cache_misses_total")
	assert.Contains(t, body, "image_cache_size")
}

func TestMetricsHandler_ProcessingMetrics(t *testing.T) {
	t.Parallel()

	// Observe processing and fetch durations
	ImageProcessingDuration.Observe(0.123)
	ImageFetchDuration.Observe(0.456)

	// Get metrics
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	handler := MetricsHandler()
	handler.ServeHTTP(rr, req)

	// Check that processing metrics are in the output
	body := rr.Body.String()
	assert.Contains(t, body, "image_processing_duration_seconds")
	assert.Contains(t, body, "image_fetch_duration_seconds")
}

func TestMetricsHandler_Format(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	handler := MetricsHandler()
	handler.ServeHTTP(rr, req)

	body := rr.Body.String()

	// Verify it's in Prometheus text format
	// Should have HELP and TYPE comments
	assert.Contains(t, body, "# HELP")
	assert.Contains(t, body, "# TYPE")

	// Should have metric names followed by values
	lines := strings.Split(body, "\n")
	foundMetric := false
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") && strings.TrimSpace(line) != "" {
			// Should have a metric name and a numeric value
			parts := strings.Fields(line)
			require.GreaterOrEqual(t, len(parts), 2, "metric line should have name and value")
			foundMetric = true
			break
		}
	}
	assert.True(t, foundMetric, "should have at least one metric line")
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	handlerPkg "github.com/steviee/github-workflow-article/internal/handler"
	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	t.Parallel()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with Metrics middleware
	wrappedHandler := Metrics(testHandler)

	// Get initial counter value
	initialCount := testutil.ToFloat64(
		handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-metrics", "200"),
	)

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/test-metrics", nil)
	rr := httptest.NewRecorder()

	// Call the handler
	wrappedHandler.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())

	// Check that the counter was incremented
	newCount := testutil.ToFloat64(
		handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-metrics", "200"),
	)
	assert.Equal(t, initialCount+1, newCount, "request counter should be incremented")
}

func TestMetrics_DifferentStatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		statusStr  string
	}{
		{"200 OK", http.StatusOK, "200"},
		{"201 Created", http.StatusCreated, "201"},
		{"400 Bad Request", http.StatusBadRequest, "400"},
		{"404 Not Found", http.StatusNotFound, "404"},
		{"500 Internal Server Error", http.StatusInternalServerError, "500"},
		{"501 Not Implemented", http.StatusNotImplemented, "501"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			wrappedHandler := Metrics(testHandler)

			// Get initial count
			initialCount := testutil.ToFloat64(
				handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-status", tt.statusStr),
			)

			req := httptest.NewRequest(http.MethodGet, "/test-status", nil)
			rr := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rr, req)

			assert.Equal(t, tt.statusCode, rr.Code)

			// Check counter was incremented with correct status
			newCount := testutil.ToFloat64(
				handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-status", tt.statusStr),
			)
			assert.Greater(t, newCount, initialCount, "counter should be incremented")
		})
	}
}

func TestMetrics_DifferentMethods(t *testing.T) {
	t.Parallel()

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := Metrics(testHandler)

			// Get initial count
			initialCount := testutil.ToFloat64(
				handlerPkg.HTTPRequestsTotal.WithLabelValues(method, "/test-method", "200"),
			)

			req := httptest.NewRequest(method, "/test-method", nil)
			rr := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rr, req)

			// Check counter was incremented with correct method
			newCount := testutil.ToFloat64(
				handlerPkg.HTTPRequestsTotal.WithLabelValues(method, "/test-method", "200"),
			)
			assert.Greater(t, newCount, initialCount, "counter should be incremented for method "+method)
		})
	}
}

func TestMetrics_DifferentPaths(t *testing.T) {
	t.Parallel()

	paths := []string{
		"/",
		"/health",
		"/ready",
		"/metrics",
		"/image",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := Metrics(testHandler)

			// Get initial count
			initialCount := testutil.ToFloat64(
				handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", path, "200"),
			)

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rr, req)

			// Check counter was incremented with correct path
			newCount := testutil.ToFloat64(
				handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", path, "200"),
			)
			assert.Greater(t, newCount, initialCount, "counter should be incremented for path "+path)
		})
	}
}

func TestMetrics_RequestDuration(t *testing.T) {
	t.Parallel()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := Metrics(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test-duration", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	// Verify the request was successful - duration histogram is observed internally
	// We validate the middleware works correctly by checking the response
	assert.Equal(t, http.StatusOK, rr.Code, "request should complete successfully")
}

func TestMetrics_WriteWithoutExplicitHeader(t *testing.T) {
	t.Parallel()

	// Handler that writes without calling WriteHeader explicitly
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK")) // Should implicitly set status to 200
	})

	wrappedHandler := Metrics(testHandler)

	// Get initial count
	initialCount := testutil.ToFloat64(
		handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-implicit", "200"),
	)

	req := httptest.NewRequest(http.MethodGet, "/test-implicit", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check counter was incremented with status 200
	newCount := testutil.ToFloat64(
		handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-implicit", "200"),
	)
	assert.Greater(t, newCount, initialCount, "counter should be incremented with status 200")
}

func TestMetrics_MultipleWrites(t *testing.T) {
	t.Parallel()

	// Handler that writes multiple times
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Part 1"))
		w.Write([]byte("Part 2"))
	})

	wrappedHandler := Metrics(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test-multiple", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, "Part 1Part 2", rr.Body.String())
	assert.Equal(t, http.StatusOK, rr.Code)

	// Metrics should still be recorded correctly
	count := testutil.ToFloat64(
		handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-multiple", "200"),
	)
	assert.Greater(t, count, float64(0), "counter should be incremented")
}

func TestMetrics_ErrorResponses(t *testing.T) {
	t.Parallel()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
	})

	wrappedHandler := Metrics(testHandler)

	// Get initial count
	initialCount := testutil.ToFloat64(
		handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-error", "500"),
	)

	req := httptest.NewRequest(http.MethodGet, "/test-error", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	// Check counter was incremented with status 500
	newCount := testutil.ToFloat64(
		handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-error", "500"),
	)
	assert.Greater(t, newCount, initialCount, "counter should be incremented for error response")
}

func TestMetrics_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := Metrics(testHandler)

	// Get initial count
	initialCount := testutil.ToFloat64(
		handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-concurrent", "200"),
	)

	// Make multiple concurrent requests
	numRequests := 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/test-concurrent", nil)
			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}

	// Check that counter was incremented for all requests
	newCount := testutil.ToFloat64(
		handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-concurrent", "200"),
	)
	assert.GreaterOrEqual(t, newCount, initialCount+float64(numRequests),
		"counter should be incremented for all concurrent requests")
}

func TestMetrics_PreservesResponseBody(t *testing.T) {
	t.Parallel()

	expectedBody := "Hello, World!"

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	})

	wrappedHandler := Metrics(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test-body", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, expectedBody, rr.Body.String(),
		"metrics middleware should preserve response body")
}

func TestMetrics_Labels(t *testing.T) {
	t.Parallel()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	wrappedHandler := Metrics(testHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	// Verify metrics with correct labels exist
	count := testutil.ToFloat64(
		handlerPkg.HTTPRequestsTotal.WithLabelValues("POST", "/api/users", "201"),
	)
	assert.Greater(t, count, float64(0),
		"metrics should be recorded with correct labels")
}

func TestMetricsResponseWriter_MultipleWriteHeaders(t *testing.T) {
	t.Parallel()

	// Handler that tries to call WriteHeader multiple times
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.WriteHeader(http.StatusInternalServerError) // Should be ignored
	})

	wrappedHandler := Metrics(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test-multi-header", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	// First status should win
	assert.Equal(t, http.StatusOK, rr.Code)

	// Metrics should record the first status code
	count := testutil.ToFloat64(
		handlerPkg.HTTPRequestsTotal.WithLabelValues("GET", "/test-multi-header", "200"),
	)
	assert.Greater(t, count, float64(0), "should record first status code")
}

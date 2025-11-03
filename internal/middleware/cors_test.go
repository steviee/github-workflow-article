package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testHandler is a simple handler that returns 200 OK
func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

func TestCORS(t *testing.T) {
	t.Parallel()

	// Create a test handler wrapped with CORS middleware
	handler := CORS(testHandler())

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check CORS headers
	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"),
		"should set Access-Control-Allow-Origin to *")

	assert.Equal(t, "GET, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"),
		"should set Access-Control-Allow-Methods")

	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Headers"),
		"should set Access-Control-Allow-Headers to *")

	assert.Equal(t, "3600", rr.Header().Get("Access-Control-Max-Age"),
		"should set Access-Control-Max-Age to 3600")

	// Check that the underlying handler was called
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}

func TestCORS_Preflight(t *testing.T) {
	t.Parallel()

	// Create a test handler wrapped with CORS middleware
	handler := CORS(testHandler())

	// Create a preflight OPTIONS request
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check CORS headers are present
	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "3600", rr.Header().Get("Access-Control-Max-Age"))

	// Preflight should return 200 OK
	assert.Equal(t, http.StatusOK, rr.Code, "preflight request should return 200 OK")

	// Preflight should not call the next handler (body should be empty)
	assert.Empty(t, rr.Body.String(), "preflight should not call next handler")
}

func TestCORS_DifferentMethods(t *testing.T) {
	t.Parallel()

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			handler := CORS(testHandler())
			req := httptest.NewRequest(method, "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// All methods should have CORS headers
			assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "GET, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Headers"))
			assert.Equal(t, "3600", rr.Header().Get("Access-Control-Max-Age"))

			// Non-OPTIONS methods should call the next handler
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "OK", rr.Body.String())
		})
	}
}

func TestCORS_MultiplePaths(t *testing.T) {
	t.Parallel()

	paths := []string{"/", "/health", "/ready", "/metrics", "/image"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			handler := CORS(testHandler())
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// All paths should have CORS headers
			assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "GET, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Headers"))
			assert.Equal(t, "3600", rr.Header().Get("Access-Control-Max-Age"))
		})
	}
}

func TestCORS_WithOriginHeader(t *testing.T) {
	t.Parallel()

	handler := CORS(testHandler())

	// Create a request with an Origin header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should still return * (allow all origins)
	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightWithHeaders(t *testing.T) {
	t.Parallel()

	handler := CORS(testHandler())

	// Create a preflight request with requested headers
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should allow all headers (*)
	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCORS_ChainedWithOtherHandlers(t *testing.T) {
	t.Parallel()

	// Create a handler that sets a custom header
	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Custom"))
	})

	// Wrap with CORS
	handler := CORS(customHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Both CORS headers and custom headers should be present
	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "test-value", rr.Header().Get("X-Custom-Header"))
	assert.Equal(t, "Custom", rr.Body.String())
}

func TestCORS_MaxAge(t *testing.T) {
	t.Parallel()

	handler := CORS(testHandler())
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Max age should be set to 3600 (1 hour in seconds)
	assert.Equal(t, "3600", rr.Header().Get("Access-Control-Max-Age"),
		"max age should be 3600 seconds")
}

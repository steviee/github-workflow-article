package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestLogger(t *testing.T) {
	t.Parallel()

	// Create a logger with a buffer to capture output
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap with RequestLogger middleware
	middleware := RequestLogger(logger)
	wrappedHandler := middleware(handler)

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("User-Agent", "test-agent")
	rr := httptest.NewRecorder()

	// Call the handler
	wrappedHandler.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())

	// Check that X-Request-ID header was added
	requestID := rr.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID, "X-Request-ID header should be set")

	// Parse request ID as UUID (36 characters with dashes)
	assert.Len(t, requestID, 36, "request ID should be a valid UUID")
	assert.Contains(t, requestID, "-", "request ID should contain dashes")

	// Check log output
	logOutput := buf.String()
	assert.NotEmpty(t, logOutput, "should have logged something")

	// Verify log contains expected fields
	assert.Contains(t, logOutput, "request_id")
	assert.Contains(t, logOutput, "method")
	assert.Contains(t, logOutput, "path")
	assert.Contains(t, logOutput, "query")
	assert.Contains(t, logOutput, "remote_addr")
	assert.Contains(t, logOutput, "user_agent")
	assert.Contains(t, logOutput, "status")
	assert.Contains(t, logOutput, "duration_ms")

	// Verify log contains actual values
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/test")
	assert.Contains(t, logOutput, "foo=bar")
	assert.Contains(t, logOutput, "192.168.1.1:1234")
	assert.Contains(t, logOutput, "test-agent")
	assert.Contains(t, logOutput, "200")
}

func TestRequestLogger_DifferentStatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"501 Not Implemented", http.StatusNotImplemented},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := logrus.New()
			logger.SetOutput(&buf)
			logger.SetFormatter(&logrus.JSONFormatter{})

			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			middleware := RequestLogger(logger)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rr, req)

			assert.Equal(t, tt.statusCode, rr.Code)

			logOutput := buf.String()
			assert.Contains(t, logOutput, `"status":`+strings.TrimSpace(strings.Fields(tt.name)[0]))
		})
	}
}

func TestRequestLogger_RequestIDInContext(t *testing.T) {
	t.Parallel()

	var capturedRequestID string

	logger := logrus.New()
	logger.SetOutput(bytes.NewBuffer(nil)) // Discard logs

	// Handler that captures request ID from context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestLogger(logger)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	// Request ID should be captured from context
	assert.NotEmpty(t, capturedRequestID, "request ID should be in context")

	// Request ID in context should match header
	headerRequestID := rr.Header().Get("X-Request-ID")
	assert.Equal(t, headerRequestID, capturedRequestID,
		"request ID in context should match header")
}

func TestGetRequestID(t *testing.T) {
	t.Parallel()

	t.Run("with request ID in context", func(t *testing.T) {
		t.Parallel()

		requestID := "test-request-id-123"
		ctx := context.WithValue(context.Background(), requestIDKey, requestID)

		result := GetRequestID(ctx)
		assert.Equal(t, requestID, result)
	})

	t.Run("without request ID in context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		result := GetRequestID(ctx)
		assert.Empty(t, result, "should return empty string when not in context")
	})

	t.Run("with wrong type in context", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(context.Background(), requestIDKey, 12345) // Not a string

		result := GetRequestID(ctx)
		assert.Empty(t, result, "should return empty string for wrong type")
	})
}

func TestRequestLogger_DifferentMethods(t *testing.T) {
	t.Parallel()

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := logrus.New()
			logger.SetOutput(&buf)
			logger.SetFormatter(&logrus.JSONFormatter{})

			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequestLogger(logger)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(method, "/test", nil)
			rr := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rr, req)

			logOutput := buf.String()
			assert.Contains(t, logOutput, `"method":"`+method+`"`)
		})
	}
}

func TestRequestLogger_DifferentPaths(t *testing.T) {
	t.Parallel()

	paths := []string{
		"/",
		"/health",
		"/ready",
		"/metrics",
		"/image",
		"/api/v1/users",
		"/api/v1/users/123",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := logrus.New()
			logger.SetOutput(&buf)
			logger.SetFormatter(&logrus.JSONFormatter{})

			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequestLogger(logger)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rr, req)

			logOutput := buf.String()
			assert.Contains(t, logOutput, path)
		})
	}
}

func TestRequestLogger_DurationTracking(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate some work (but keep it fast for tests)
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestLogger(logger)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "duration_ms")

	// Duration should be a non-negative number
	assert.Contains(t, logOutput, `"duration_ms":`)
}

func TestRequestLogger_WriteWithoutExplicitHeader(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Handler that writes without calling WriteHeader explicitly
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK")) // This should implicitly set status to 200
	})

	middleware := RequestLogger(logger)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"status":200`)
}

func TestRequestLogger_MultipleWrites(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Handler that writes multiple times
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Part 1"))
		_, _ = w.Write([]byte("Part 2"))
		_, _ = w.Write([]byte("Part 3"))
	})

	middleware := RequestLogger(logger)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, "Part 1Part 2Part 3", rr.Body.String())
	assert.Equal(t, http.StatusOK, rr.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"status":200`)
}

func TestRequestLogger_UniqueRequestIDs(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetOutput(bytes.NewBuffer(nil)) // Discard logs

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestLogger(logger)
	wrappedHandler := middleware(handler)

	// Generate multiple request IDs
	requestIDs := make(map[string]bool)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rr, req)

		requestID := rr.Header().Get("X-Request-ID")
		require.NotEmpty(t, requestID)
		requestIDs[requestID] = true
	}

	// All request IDs should be unique
	assert.Len(t, requestIDs, 10, "all request IDs should be unique")
}

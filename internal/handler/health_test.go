package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	// Create a request to pass to the handler
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	HealthHandler(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code, "handler should return 200 OK")

	// Check the content type
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "content type should be application/json")

	// Check the response body
	var response HealthResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "response should be valid JSON")

	assert.Equal(t, "healthy", response.Status, "status should be 'healthy'")
}

func TestHealthHandler_MultipleRequests(t *testing.T) {
	t.Parallel()

	// Test that multiple requests all return the same result
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()

		HealthHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response HealthResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "healthy", response.Status)
	}
}

func TestReadyHandler(t *testing.T) {
	t.Parallel()

	// Create a request to pass to the handler
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	ReadyHandler(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code, "handler should return 200 OK")

	// Check the content type
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "content type should be application/json")

	// Check the response body
	var response HealthResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "response should be valid JSON")

	assert.Equal(t, "ready", response.Status, "status should be 'ready'")
}

func TestReadyHandler_MultipleRequests(t *testing.T) {
	t.Parallel()

	// Test that multiple requests all return the same result
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rr := httptest.NewRecorder()

		ReadyHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response HealthResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "ready", response.Status)
	}
}

func TestHealthHandler_DifferentMethods(t *testing.T) {
	t.Parallel()

	// Health handler should work with any HTTP method (it doesn't check)
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(method, "/health", nil)
			rr := httptest.NewRecorder()

			HealthHandler(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)

			var response HealthResponse
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "healthy", response.Status)
		})
	}
}

func TestReadyHandler_DifferentMethods(t *testing.T) {
	t.Parallel()

	// Ready handler should work with any HTTP method (it doesn't check)
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(method, "/ready", nil)
			rr := httptest.NewRecorder()

			ReadyHandler(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)

			var response HealthResponse
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "ready", response.Status)
		})
	}
}

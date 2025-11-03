package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageHandler(t *testing.T) {
	t.Parallel()

	// Create a request to pass to the handler
	req := httptest.NewRequest(http.MethodGet, "/image?url=https://example.com/image.jpg", nil)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	ImageHandler(rr, req)

	// Check the status code - should be 501 Not Implemented
	assert.Equal(t, http.StatusNotImplemented, rr.Code, "handler should return 501 Not Implemented")

	// Check the content type
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "content type should be application/json")

	// Check the response body
	var response ImageErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "response should be valid JSON")

	assert.NotEmpty(t, response.Error, "error field should not be empty")
	assert.NotEmpty(t, response.Message, "message field should not be empty")
	assert.Contains(t, response.Error, "not yet implemented", "error should mention not implemented")
}

func TestImageHandler_ErrorMessage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/image", nil)
	rr := httptest.NewRecorder()

	ImageHandler(rr, req)

	var response ImageErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify error message content
	assert.Equal(t, "Image processing not yet implemented", response.Error)
	assert.Equal(t, "Coming soon in Issue #6-#10", response.Message)
}

func TestImageHandler_MultipleRequests(t *testing.T) {
	t.Parallel()

	// Test that multiple requests all return the same result
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/image", nil)
		rr := httptest.NewRecorder()

		ImageHandler(rr, req)

		assert.Equal(t, http.StatusNotImplemented, rr.Code)

		var response ImageErrorResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Error)
		assert.NotEmpty(t, response.Message)
	}
}

func TestImageHandler_DifferentMethods(t *testing.T) {
	t.Parallel()

	// Image handler should work with any HTTP method (it doesn't check)
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(method, "/image", nil)
			rr := httptest.NewRecorder()

			ImageHandler(rr, req)

			assert.Equal(t, http.StatusNotImplemented, rr.Code)

			var response ImageErrorResponse
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.NotEmpty(t, response.Error)
		})
	}
}

func TestImageHandler_WithQueryParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "with url parameter",
			query: "?url=https://example.com/image.jpg",
		},
		{
			name:  "with url and operations",
			query: "?url=https://example.com/image.jpg&operations=rotate-90,resize-200x200",
		},
		{
			name:  "with multiple parameters",
			query: "?url=https://example.com/image.jpg&operations=rotate-90&format=jpg",
		},
		{
			name:  "no parameters",
			query: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/image"+tt.query, nil)
			rr := httptest.NewRecorder()

			ImageHandler(rr, req)

			// All should return 501 regardless of parameters
			assert.Equal(t, http.StatusNotImplemented, rr.Code)

			var response ImageErrorResponse
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.NotEmpty(t, response.Error)
		})
	}
}

func TestImageHandler_JSONFormat(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/image", nil)
	rr := httptest.NewRecorder()

	ImageHandler(rr, req)

	// Verify the response is valid JSON with correct structure
	var response ImageErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "response should be valid JSON")

	// Verify fields are present and non-empty
	assert.NotEmpty(t, response.Error, "error field should be present")
	assert.NotEmpty(t, response.Message, "message field should be present")

	// Re-marshal to verify structure
	jsonBytes, err := json.Marshal(response)
	require.NoError(t, err)

	// Should be able to unmarshal back
	var response2 ImageErrorResponse
	err = json.Unmarshal(jsonBytes, &response2)
	require.NoError(t, err)

	assert.Equal(t, response.Error, response2.Error)
	assert.Equal(t, response.Message, response2.Message)
}

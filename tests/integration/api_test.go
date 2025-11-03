// Package integration provides end-to-end integration tests for the image processing API.
// These tests start a real HTTP server and make actual HTTP requests to verify the entire
// system works correctly from client perspective.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/steviee/github-workflow-article/internal/handler"
	appMiddleware "github.com/steviee/github-workflow-article/internal/middleware"
)

const (
	// Test image URLs from picsum.photos - a reliable test image service
	testImageURL800x600  = "https://picsum.photos/800/600"
	testImageURL400x300  = "https://picsum.photos/400/300"
	testImageURL1200x800 = "https://picsum.photos/1200/800"

	// PNG magic bytes for format verification
	pngMagicNumber = "\x89PNG\r\n\x1a\n"
)

// setupTestServer creates and returns an httptest server with all routes configured.
// The server should be closed by the caller using defer server.Close().
func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	// Initialize cache with shorter TTL for testing
	ctx := context.Background()
	handler.InitializeCache(ctx, 5*time.Minute, 10*time.Second)

	// Create chi router with middleware
	router := chi.NewRouter()
	router.Use(appMiddleware.CORS)
	router.Use(appMiddleware.Metrics)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(30 * time.Second))

	// Register routes
	router.Get("/health", handler.HealthHandler)
	router.Get("/ready", handler.ReadyHandler)
	router.Handle("/metrics", handler.MetricsHandler())
	router.Get("/image", handler.ImageHandler)

	// Create and return test server
	server := httptest.NewServer(router)
	return server
}

// verifyPNGFormat checks if the data starts with PNG magic bytes
func verifyPNGFormat(t *testing.T, data []byte) {
	t.Helper()
	require.GreaterOrEqual(t, len(data), 8, "data too short to be a valid PNG")
	assert.Equal(t, pngMagicNumber, string(data[:8]), "data does not start with PNG magic bytes")
}

// makeRequest is a helper function to make HTTP GET requests to the test server.
// The response body is read and closed before returning.
func makeRequest(t *testing.T, reqURL string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	require.NoError(t, err, "failed to create request")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	require.NoError(t, err, "failed to make request")
	defer func() { _ = resp.Body.Close() }() //nolint:bodyclose // Body is read before function returns

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")

	return resp, body
}

// TestImageEndpoint_NoOperations tests fetching an image without any operations.
// It should return the original image converted to PNG format.
func TestImageEndpoint_NoOperations(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	reqURL := server.URL + "/image?url=" + url.QueryEscape(testImageURL800x600)

	resp, body := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, body)

	// Verify response headers
	assert.Contains(t, resp.Header.Get("Content-Type"), "image/")
	assert.Equal(t, "SKIP", resp.Header.Get("X-Cache"), "images without operations should not be cached")
	assert.Equal(t, testImageURL800x600, resp.Header.Get("X-Original-URL"))
}

// TestImageEndpoint_SingleRotation tests rotating an image by 90 degrees.
func TestImageEndpoint_SingleRotation(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	testCases := []struct {
		name      string
		operation string
	}{
		{"rotate 90 degrees", "rotate-90"},
		{"rotate 180 degrees", "rotate-180"},
		{"rotate 270 degrees", "rotate-270"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := server.URL + "/image?url=" + url.QueryEscape(testImageURL800x600) + "&op=" + tc.operation

			resp, body := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.NotEmpty(t, body)

			// Verify PNG format
			verifyPNGFormat(t, body)

			// Verify response headers
			assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
			assert.Equal(t, tc.operation, resp.Header.Get("X-Operations-Applied"))
			assert.Equal(t, testImageURL800x600, resp.Header.Get("X-Original-URL"))
			assert.NotEmpty(t, resp.Header.Get("X-Original-Format"))
		})
	}
}

// TestImageEndpoint_SingleResize tests resizing an image to specific dimensions.
func TestImageEndpoint_SingleResize(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	reqURL := server.URL + "/image?url=" + url.QueryEscape(testImageURL800x600) + "&op=resize-400x300"

	resp, body := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, body)

	// Verify PNG format
	verifyPNGFormat(t, body)

	// Verify response headers
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
	assert.Equal(t, "resize-400x300", resp.Header.Get("X-Operations-Applied"))
	assert.Equal(t, testImageURL800x600, resp.Header.Get("X-Original-URL"))
}

// TestImageEndpoint_MultipleOperations tests chaining multiple operations together.
func TestImageEndpoint_MultipleOperations(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	testCases := []struct {
		name       string
		operations string
	}{
		{"rotate then resize", "rotate-90,resize-400x300"},
		{"resize then rotate", "resize-800x600,rotate-180"},
		{"multiple rotations", "rotate-90,rotate-90,rotate-90"},
		{"complex chain", "rotate-90,resize-600x400,rotate-180"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := server.URL + "/image?url=" + url.QueryEscape(testImageURL800x600) + "&op=" + tc.operations

			resp, body := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.NotEmpty(t, body)

			// Verify PNG format
			verifyPNGFormat(t, body)

			// Verify response headers
			assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
			assert.Equal(t, tc.operations, resp.Header.Get("X-Operations-Applied"))
		})
	}
}

// TestImageEndpoint_InvalidURL tests handling of invalid or unreachable image URLs.
// Should return 502 Bad Gateway with an error placeholder image.
func TestImageEndpoint_InvalidURL(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	testCases := []struct {
		name        string
		imageURL    string
		description string
	}{
		{
			name:        "non-existent domain",
			imageURL:    "https://this-domain-does-not-exist-12345.com/image.jpg",
			description: "domain that does not exist",
		},
		{
			name:        "invalid URL format",
			imageURL:    "not-a-valid-url",
			description: "malformed URL",
		},
		{
			name:        "404 not found",
			imageURL:    "https://httpbin.org/status/404",
			description: "URL returning 404",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := server.URL + "/image?url=" + url.QueryEscape(tc.imageURL) + "&op=rotate-90"

			resp, body := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper

			// Should return Bad Gateway
			assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
			assert.NotEmpty(t, body)

			// Should still be a PNG (placeholder image)
			verifyPNGFormat(t, body)

			// Verify placeholder header
			assert.Equal(t, "true", resp.Header.Get("X-Placeholder"))
			assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
		})
	}
}

// TestImageEndpoint_InvalidOperation tests handling of invalid operation strings.
// Should return 400 Bad Request with an error placeholder image.
func TestImageEndpoint_InvalidOperation(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	testCases := []struct {
		name      string
		operation string
	}{
		{"unknown operation", "flip-horizontal"},
		{"invalid resize format", "resize-800"},
		{"invalid resize dimensions", "resize-abcxdef"},
		{"negative dimensions", "resize--100x200"},
		{"zero dimensions", "resize-0x0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := server.URL + "/image?url=" + url.QueryEscape(testImageURL800x600) + "&op=" + tc.operation

			resp, body := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper

			// Should return Bad Request
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.NotEmpty(t, body)

			// Should still be a PNG (placeholder image)
			verifyPNGFormat(t, body)

			// Verify placeholder header
			assert.Equal(t, "true", resp.Header.Get("X-Placeholder"))
			assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
		})
	}
}

// TestImageEndpoint_MissingURL tests that missing URL parameter returns 400 Bad Request.
func TestImageEndpoint_MissingURL(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Request without url parameter
	reqURL := server.URL + "/image"

	resp, body := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper

	// Should return Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.NotEmpty(t, body)

	// Should be a PNG placeholder
	verifyPNGFormat(t, body)

	// Verify placeholder header
	assert.Equal(t, "true", resp.Header.Get("X-Placeholder"))
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
}

// TestImageEndpoint_Cache tests cache hit and miss behavior.
// First request should be a cache miss, second request should be a cache hit.
func TestImageEndpoint_Cache(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	reqURL := server.URL + "/image?url=" + url.QueryEscape(testImageURL400x300) + "&op=rotate-90"

	// First request - should be cache MISS
	resp1, body1 := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Equal(t, "MISS", resp1.Header.Get("X-Cache"), "first request should be cache miss")
	verifyPNGFormat(t, body1)

	// Second request - should be cache HIT
	resp2, body2 := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Equal(t, "HIT", resp2.Header.Get("X-Cache"), "second request should be cache hit")
	verifyPNGFormat(t, body2)

	// Cached response should be identical to original
	assert.Equal(t, len(body1), len(body2), "cached response size should match original")
	assert.Equal(t, body1, body2, "cached response content should match original")
}

// TestImageEndpoint_CacheDifferentOperations tests that different operations
// result in different cache entries.
func TestImageEndpoint_CacheDifferentOperations(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	baseURL := server.URL + "/image?url=" + url.QueryEscape(testImageURL400x300)

	// Request with rotate-90
	url1 := baseURL + "&op=rotate-90"
	resp1, body1 := makeRequest(t, url1) //nolint:bodyclose // Body is closed in helper
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Equal(t, "MISS", resp1.Header.Get("X-Cache"))

	// Request with rotate-180 - should be different cache entry
	url2 := baseURL + "&op=rotate-180"
	resp2, body2 := makeRequest(t, url2) //nolint:bodyclose // Body is closed in helper
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Equal(t, "MISS", resp2.Header.Get("X-Cache"), "different operation should not hit cache")

	// Images should be different
	assert.NotEqual(t, body1, body2, "different operations should produce different results")

	// Repeat first request - should now hit cache
	resp3, body3 := makeRequest(t, url1) //nolint:bodyclose // Body is closed in helper
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
	assert.Equal(t, "HIT", resp3.Header.Get("X-Cache"))
	assert.Equal(t, body1, body3, "cached image should match original")
}

// TestHealthEndpoint verifies the /health endpoint returns correct status.
func TestHealthEndpoint(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, body := makeRequest(t, server.URL+"/health") //nolint:bodyclose // Body is closed in helper

	// Verify status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify content type
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Parse JSON response
	var healthResp struct {
		Status string `json:"status"`
	}
	err := json.Unmarshal(body, &healthResp)
	require.NoError(t, err, "failed to parse health response JSON")

	// Verify response content
	assert.Equal(t, "healthy", healthResp.Status)
}

// TestReadyEndpoint verifies the /ready endpoint returns correct status.
func TestReadyEndpoint(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, body := makeRequest(t, server.URL+"/ready") //nolint:bodyclose // Body is closed in helper

	// Verify status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify content type
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Parse JSON response
	var readyResp struct {
		Status string `json:"status"`
	}
	err := json.Unmarshal(body, &readyResp)
	require.NoError(t, err, "failed to parse ready response JSON")

	// Verify response content
	assert.Equal(t, "ready", readyResp.Status)
}

// TestMetricsEndpoint verifies the /metrics endpoint returns Prometheus metrics.
func TestMetricsEndpoint(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, body := makeRequest(t, server.URL+"/metrics") //nolint:bodyclose // Body is closed in helper

	// Verify status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify content type
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, contentType, "text/plain", "metrics should be text/plain format")

	// Verify response contains Prometheus metrics
	bodyStr := string(body)
	assert.NotEmpty(t, bodyStr)

	// Check for expected metric names
	expectedMetrics := []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"image_cache_hits_total",
		"image_cache_misses_total",
		"image_cache_size",
		"image_processing_duration_seconds",
		"image_fetch_duration_seconds",
	}

	for _, metric := range expectedMetrics {
		assert.Contains(t, bodyStr, metric, "metrics response should contain %s", metric)
	}

	// Verify HELP and TYPE comments are present
	assert.Contains(t, bodyStr, "# HELP")
	assert.Contains(t, bodyStr, "# TYPE")
}

// TestCORSHeaders verifies that CORS headers are properly set on all endpoints.
func TestCORSHeaders(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	testCases := []struct {
		name     string
		endpoint string
	}{
		{"health endpoint", "/health"},
		{"ready endpoint", "/ready"},
		{"metrics endpoint", "/metrics"},
		{"image endpoint", "/image?url=" + url.QueryEscape(testImageURL400x300)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, _ := makeRequest(t, server.URL+tc.endpoint) //nolint:bodyclose // Body is closed in helper

			// Verify CORS headers
			assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"),
				"CORS should allow all origins")
		})
	}
}

// TestImageEndpoint_ConcurrentRequests tests that the API handles concurrent requests correctly.
func TestImageEndpoint_ConcurrentRequests(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	const numRequests = 10
	reqURL := server.URL + "/image?url=" + url.QueryEscape(testImageURL400x300) + "&op=rotate-90"

	// Channel to collect results
	results := make(chan int, numRequests)

	// Launch concurrent requests
	for i := 0; i < numRequests; i++ {
		go func() {
			resp, body := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper
			results <- resp.StatusCode

			// Verify it's a valid PNG
			if resp.StatusCode == http.StatusOK {
				assert.GreaterOrEqual(t, len(body), 8)
			}
		}()
	}

	// Collect all results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		statusCode := <-results
		if statusCode == http.StatusOK {
			successCount++
		}
	}

	// All requests should succeed
	assert.Equal(t, numRequests, successCount, "all concurrent requests should succeed")
}

// TestImageEndpoint_LargeImage tests handling of larger images to ensure proper processing.
func TestImageEndpoint_LargeImage(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Use a larger image
	largeImageURL := "https://picsum.photos/2000/1500"
	reqURL := server.URL + "/image?url=" + url.QueryEscape(largeImageURL) + "&op=resize-800x600"

	resp, body := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, body)

	// Verify PNG format
	verifyPNGFormat(t, body)

	// Verify response headers
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
	assert.Equal(t, "resize-800x600", resp.Header.Get("X-Operations-Applied"))
}

// TestImageEndpoint_EdgeCases tests various edge cases and boundary conditions.
func TestImageEndpoint_EdgeCases(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	testCases := []struct {
		name           string
		buildURL       func(string) string
		expectedStatus int
		requiresPNG    bool
		description    string
	}{
		{
			name: "empty operation string",
			buildURL: func(baseURL string) string {
				return baseURL + "/image?url=" + url.QueryEscape(testImageURL400x300) + "&op="
			},
			expectedStatus: http.StatusOK,
			requiresPNG:    false, // Returns original format when no operations
			description:    "empty op param returns original format",
		},
		{
			name: "whitespace in operations",
			buildURL: func(baseURL string) string {
				// URL-encode the operation string which includes a space
				return baseURL + "/image?url=" + url.QueryEscape(testImageURL400x300) + "&op=" + url.QueryEscape("rotate-90, resize-400x300")
			},
			expectedStatus: http.StatusOK,
			requiresPNG:    true, // Has valid operations despite whitespace
			description:    "operations with whitespace should still work",
		},
		{
			name: "trailing comma in operations",
			buildURL: func(baseURL string) string {
				return baseURL + "/image?url=" + url.QueryEscape(testImageURL400x300) + "&op=rotate-90,"
			},
			expectedStatus: http.StatusOK,
			requiresPNG:    true, // Has valid rotate-90 operation
			description:    "trailing comma should be ignored",
		},
		{
			name: "valid small dimensions",
			buildURL: func(baseURL string) string {
				return baseURL + "/image?url=" + url.QueryEscape(testImageURL400x300) + "&op=resize-1x1"
			},
			expectedStatus: http.StatusOK,
			requiresPNG:    true,
			description:    "very small dimensions should work",
		},
		{
			name: "valid large dimensions",
			buildURL: func(baseURL string) string {
				return baseURL + "/image?url=" + url.QueryEscape(testImageURL400x300) + "&op=resize-1400x1400"
			},
			expectedStatus: http.StatusOK,
			requiresPNG:    true,
			description:    "large dimensions up to limit should work",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := tc.buildURL(server.URL)
			resp, body := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper

			assert.Equal(t, tc.expectedStatus, resp.StatusCode, tc.description)
			assert.NotEmpty(t, body)

			// Verify response is an image
			assert.Contains(t, resp.Header.Get("Content-Type"), "image/")

			// Only verify PNG format if operations were applied
			if resp.StatusCode == http.StatusOK && tc.requiresPNG {
				verifyPNGFormat(t, body)
			}
		})
	}
}

// TestImageEndpoint_DifferentFormats tests fetching images in different formats.
func TestImageEndpoint_DifferentFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping format test in short mode")
	}

	server := setupTestServer(t)
	defer server.Close()

	// Note: picsum.photos returns JPEG by default
	// We'll test that our API converts it to PNG when operations are applied
	testCases := []struct {
		name     string
		imageURL string
	}{
		{
			name:     "JPEG image",
			imageURL: "https://picsum.photos/400/300.jpg",
		},
		{
			name:     "WebP image",
			imageURL: "https://picsum.photos/400/300.webp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := server.URL + "/image?url=" + url.QueryEscape(tc.imageURL) + "&op=rotate-90"

			resp, body := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper

			if resp.StatusCode == http.StatusOK {
				// Verify output is PNG regardless of input format
				verifyPNGFormat(t, body)
				assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
			}
		})
	}
}

// TestImageEndpoint_OperationOrder tests that operation order matters.
func TestImageEndpoint_OperationOrder(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	baseURL := server.URL + "/image?url=" + url.QueryEscape(testImageURL800x600)

	// Different operation orders should produce different results
	url1 := baseURL + "&op=rotate-90,resize-400x300"
	url2 := baseURL + "&op=resize-400x300,rotate-90"

	resp1, body1 := makeRequest(t, url1) //nolint:bodyclose // Body is closed in helper
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	resp2, body2 := makeRequest(t, url2) //nolint:bodyclose // Body is closed in helper
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// Results should be different because operation order matters
	// After rotate-90, dimensions swap, so resize will have different effect
	assert.NotEqual(t, body1, body2, "different operation orders should produce different results")
}

// TestServerGracefulHandling verifies the server doesn't panic on edge cases.
func TestServerGracefulHandling(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	testCases := []struct {
		name     string
		buildURL func(string) string
	}{
		{
			name: "multiple question marks",
			buildURL: func(baseURL string) string {
				return baseURL + "/image?url=" + url.QueryEscape(testImageURL400x300+"?extra=param")
			},
		},
		{
			name: "special characters in operation",
			buildURL: func(baseURL string) string {
				return baseURL + "/image?url=" + url.QueryEscape(testImageURL400x300) + "&op=" + url.QueryEscape("rotate-90 ")
			},
		},
		{
			name: "very long URL",
			buildURL: func(baseURL string) string {
				longURL := "https://picsum.photos/400/300" + string(bytes.Repeat([]byte("a"), 100))
				return baseURL + "/image?url=" + url.QueryEscape(longURL)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := tc.buildURL(server.URL)
			// Should not panic
			resp, _ := makeRequest(t, reqURL) //nolint:bodyclose // Body is closed in helper

			// Should return some valid HTTP status
			assert.NotEqual(t, 0, resp.StatusCode, "should return valid status code")
		})
	}
}

package handler

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageHandler_MissingURL(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/image", nil)
	rr := httptest.NewRecorder()

	ImageHandler(rr, req)

	// Should return 400 Bad Request with placeholder
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
	assert.Equal(t, "true", rr.Header().Get("X-Placeholder"))

	// Verify it's a valid PNG
	img, err := png.Decode(bytes.NewReader(rr.Body.Bytes()))
	assert.NoError(t, err)
	assert.NotNil(t, img)
}

func TestImageHandler_WithValidImage(t *testing.T) {
	t.Parallel()

	// Create a test server that serves a fake image
	imageData := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL, nil)
	rr := httptest.NewRecorder()

	ImageHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
	assert.Equal(t, testServer.URL, rr.Header().Get("X-Original-URL"))
	assert.Equal(t, imageData, rr.Body.Bytes())
}

func TestImageHandler_WithInvalidURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		url  string
	}{
		{"non-existent domain", "http://this-domain-definitely-does-not-exist-12345.com/image.png"},
		{"invalid scheme", "ftp://example.com/image.png"},
		{"malformed URL", "http://[::1]:namedport"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/image?url="+tc.url, nil)
			rr := httptest.NewRecorder()

			ImageHandler(rr, req)

			// Should return 502 Bad Gateway with placeholder
			assert.Equal(t, http.StatusBadGateway, rr.Code)
			assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
			assert.Equal(t, "true", rr.Header().Get("X-Placeholder"))

			// Verify it's a valid PNG
			img, err := png.Decode(bytes.NewReader(rr.Body.Bytes()))
			assert.NoError(t, err)
			assert.NotNil(t, img)
		})
	}
}

func TestImageHandler_WithHTTPError(t *testing.T) {
	t.Parallel()

	// Create a test server that returns 404
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL, nil)
	rr := httptest.NewRecorder()

	ImageHandler(rr, req)

	// Should return 502 Bad Gateway with placeholder
	assert.Equal(t, http.StatusBadGateway, rr.Code)
	assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
	assert.Equal(t, "true", rr.Header().Get("X-Placeholder"))

	// Verify it's a valid PNG placeholder
	img, err := png.Decode(bytes.NewReader(rr.Body.Bytes()))
	assert.NoError(t, err)
	assert.NotNil(t, img)
}

func TestImageHandler_WithNonImageContent(t *testing.T) {
	t.Parallel()

	// Create a test server that returns HTML instead of image
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>Not an image</html>"))
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL, nil)
	rr := httptest.NewRecorder()

	ImageHandler(rr, req)

	// Should return 502 with placeholder (invalid content type)
	assert.Equal(t, http.StatusBadGateway, rr.Code)
	assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
	assert.Equal(t, "true", rr.Header().Get("X-Placeholder"))

	// Verify it's a valid PNG placeholder
	img, err := png.Decode(bytes.NewReader(rr.Body.Bytes()))
	assert.NoError(t, err)
	assert.NotNil(t, img)
}

func TestImageHandler_WithCustomDimensions(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/image?url=invalid&width=300&height=200", nil)
	rr := httptest.NewRecorder()

	ImageHandler(rr, req)

	// Should return placeholder with requested dimensions
	assert.Equal(t, http.StatusBadGateway, rr.Code)
	assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))

	// Decode and check dimensions
	img, err := png.Decode(bytes.NewReader(rr.Body.Bytes()))
	assert.NoError(t, err)

	bounds := img.Bounds()
	assert.Equal(t, 400, bounds.Dx(), "width should be 400")
	assert.Equal(t, 300, bounds.Dy(), "height should be 300")
}

func TestImageHandler_WithDifferentImageTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		contentType string
		data        []byte
	}{
		{"JPEG", "image/jpeg", []byte("\xFF\xD8\xFF\xE0")},
		{"GIF", "image/gif", []byte("GIF89a")},
		{"WebP", "image/webp", []byte("RIFF")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create test server with specific image type
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(tc.data)
			}))
			defer testServer.Close()

			req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL, nil)
			rr := httptest.NewRecorder()

			ImageHandler(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, tc.contentType, rr.Header().Get("Content-Type"))
			assert.Equal(t, tc.data, rr.Body.Bytes())
		})
	}
}

func TestImageHandler_MultipleRequests(t *testing.T) {
	t.Parallel()

	// Create test server
	imageData := []byte("test-image-data")
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer testServer.Close()

	// Make multiple requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL, nil)
		rr := httptest.NewRecorder()

		ImageHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
		assert.Equal(t, imageData, rr.Body.Bytes())
	}
}

func TestImageHandler_ImageTooLarge(t *testing.T) {
	t.Parallel()

	// Create test server that returns image larger than 50MB
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", "52428801") // 50MB + 1 byte
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL, nil)
	rr := httptest.NewRecorder()

	ImageHandler(rr, req)

	// Should return 502 with placeholder (size exceeded)
	assert.Equal(t, http.StatusBadGateway, rr.Code)
	assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
	assert.Equal(t, "true", rr.Header().Get("X-Placeholder"))
}

func TestSendPlaceholder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		width      int
		height     int
	}{
		{"400 with default dimensions", 400, 0, 0},
		{"404 with custom dimensions", 404, 200, 150},
		{"500 with large dimensions", 500, 800, 600},
		{"502 with small dimensions", 502, 50, 50},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			sendPlaceholder(rr, tc.statusCode, tc.width, tc.height)

			assert.Equal(t, tc.statusCode, rr.Code)
			assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
			assert.Equal(t, "true", rr.Header().Get("X-Placeholder"))
			assert.Greater(t, len(rr.Body.Bytes()), 0)

			// Verify it's a valid PNG
			img, err := png.Decode(bytes.NewReader(rr.Body.Bytes()))
			assert.NoError(t, err)
			assert.NotNil(t, img)
		})
	}
}

// createTestPNG creates a simple test PNG image
func createTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with blue color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{0, 0, 255, 255})
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestImageHandler_WithResizeOperation(t *testing.T) {
	t.Parallel()

	// Create a test server that serves a test image
	imageData := createTestPNG(800, 600)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL+"&op=resize-400x300", nil)
	rr := httptest.NewRecorder()

	ImageHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
	assert.Equal(t, testServer.URL, rr.Header().Get("X-Original-URL"))

	// Decode and verify dimensions
	img, err := png.Decode(bytes.NewReader(rr.Body.Bytes()))
	require.NoError(t, err)
	bounds := img.Bounds()
	assert.Equal(t, 400, bounds.Dx(), "width should be 400")
	assert.Equal(t, 300, bounds.Dy(), "height should be 300")
}

func TestImageHandler_WithRotateOperation(t *testing.T) {
	t.Parallel()

	// Create a test server that serves a test image
	imageData := createTestPNG(400, 300)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer testServer.Close()

	testCases := []struct {
		name           string
		operation      string
		expectedWidth  int
		expectedHeight int
	}{
		{"rotate-90", "rotate-90", 300, 400},   // Dimensions swapped
		{"rotate-180", "rotate-180", 400, 300}, // Same dimensions
		{"rotate-270", "rotate-270", 300, 400}, // Dimensions swapped
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL+"&op="+tc.operation, nil)
			rr := httptest.NewRecorder()

			ImageHandler(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))

			// Decode and verify dimensions
			img, err := png.Decode(bytes.NewReader(rr.Body.Bytes()))
			require.NoError(t, err)
			bounds := img.Bounds()
			assert.Equal(t, tc.expectedWidth, bounds.Dx(), "width should match expected")
			assert.Equal(t, tc.expectedHeight, bounds.Dy(), "height should match expected")
		})
	}
}

func TestImageHandler_WithChainedOperations(t *testing.T) {
	t.Parallel()

	// Create a test server that serves a test image
	imageData := createTestPNG(800, 600)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer testServer.Close()

	testCases := []struct {
		name           string
		operations     string
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "rotate-90 then resize",
			operations:     "rotate-90,resize-300x400",
			expectedWidth:  300,
			expectedHeight: 400,
		},
		{
			name:           "resize then rotate-90",
			operations:     "resize-400x300,rotate-90",
			expectedWidth:  300,
			expectedHeight: 400,
		},
		{
			name:           "multiple rotations and resize",
			operations:     "rotate-90,rotate-90,resize-200x200",
			expectedWidth:  200,
			expectedHeight: 200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL+"&op="+tc.operations, nil)
			rr := httptest.NewRecorder()

			ImageHandler(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))

			// Decode and verify dimensions
			img, err := png.Decode(bytes.NewReader(rr.Body.Bytes()))
			require.NoError(t, err)
			bounds := img.Bounds()
			assert.Equal(t, tc.expectedWidth, bounds.Dx(), "width should match expected")
			assert.Equal(t, tc.expectedHeight, bounds.Dy(), "height should match expected")
		})
	}
}

func TestImageHandler_WithInvalidResizeFormat(t *testing.T) {
	t.Parallel()

	// Create a test server that serves a test image
	imageData := createTestPNG(400, 300)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer testServer.Close()

	testCases := []struct {
		name      string
		operation string
	}{
		{"missing dimensions", "resize-"},
		{"invalid format - no x", "resize-400"},
		{"invalid width", "resize-abcx300"},
		{"invalid height", "resize-400xabc"},
		{"zero width", "resize-0x300"},
		{"zero height", "resize-400x0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL+"&op="+tc.operation, nil)
			rr := httptest.NewRecorder()

			ImageHandler(rr, req)

			// Should return 400 Bad Request with placeholder
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
			assert.Equal(t, "true", rr.Header().Get("X-Placeholder"))
		})
	}
}

func TestImageHandler_WithInvalidOperation(t *testing.T) {
	t.Parallel()

	// Create a test server that serves a test image
	imageData := createTestPNG(400, 300)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL+"&op=invalid-operation", nil)
	rr := httptest.NewRecorder()

	ImageHandler(rr, req)

	// Should return 400 Bad Request with placeholder
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
	assert.Equal(t, "true", rr.Header().Get("X-Placeholder"))
}

func TestImageHandler_WithMaxDimensionExceeding(t *testing.T) {
	t.Parallel()

	// Create a test server that serves a test image
	imageData := createTestPNG(400, 300)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer testServer.Close()

	// Request dimensions that exceed 1400x1400
	req := httptest.NewRequest(http.MethodGet, "/image?url="+testServer.URL+"&op=resize-2000x1500", nil)
	rr := httptest.NewRecorder()

	ImageHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))

	// Decode and verify dimensions are clamped
	img, err := png.Decode(bytes.NewReader(rr.Body.Bytes()))
	require.NoError(t, err)
	bounds := img.Bounds()
	// Should be clamped to 1400x1050 (maintaining aspect ratio)
	assert.Equal(t, 1400, bounds.Dx(), "width should be clamped to 1400")
	assert.Equal(t, 1050, bounds.Dy(), "height should be proportionally scaled")
}

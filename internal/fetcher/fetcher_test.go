package fetcher

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewFetcher(t *testing.T) {
	t.Parallel()

	fetcher := NewFetcher(1024*1024, 30*time.Second)

	assert.NotNil(t, fetcher)
	assert.NotNil(t, fetcher.client)
	assert.Equal(t, int64(1024*1024), fetcher.maxSize)
	assert.Equal(t, "ImageProcessingAPI/1.0", fetcher.userAgent)
	assert.Equal(t, 10, fetcher.maxRedirect)
}

func TestFetch_Success(t *testing.T) {
	t.Parallel()

	// Create test server that returns a small PNG
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "ImageProcessingAPI/1.0", r.Header.Get("User-Agent"))
		assert.Equal(t, "image/*", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake-png-data"))
	}))
	defer server.Close()

	fetcher := NewFetcher(1024, 30*time.Second)
	result, err := fetcher.Fetch(server.URL)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []byte("fake-png-data"), result.Data)
	assert.Equal(t, "image/png", result.ContentType)
	assert.Equal(t, int64(13), result.Size)
	assert.Equal(t, server.URL, result.URL)
}

func TestFetch_InvalidScheme(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		url  string
	}{
		{"ftp scheme", "ftp://example.com/image.png"},
		{"file scheme", "file:///etc/passwd"},
		{"no scheme", "example.com/image.png"},
		{"relative path", "/images/test.png"},
	}

	fetcher := NewFetcher(1024, 30*time.Second)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := fetcher.Fetch(tc.url)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "invalid URL scheme")
		})
	}
}

func TestFetch_InvalidContentType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		contentType string
	}{
		{"text/html", "text/html"},
		{"application/json", "application/json"},
		{"text/plain", "text/plain"},
		{"video/mp4", "video/mp4"},
		{"empty", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("not an image"))
			}))
			defer server.Close()

			fetcher := NewFetcher(1024, 30*time.Second)
			result, err := fetcher.Fetch(server.URL)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "invalid content type")
		})
	}
}

func TestFetch_ContentLengthTooLarge(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", "2000") // Larger than maxSize
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytes.Repeat([]byte("X"), 2000))
	}))
	defer server.Close()

	fetcher := NewFetcher(1024, 30*time.Second) // maxSize = 1024
	result, err := fetcher.Fetch(server.URL)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "image too large")
	assert.Contains(t, err.Error(), "2000 bytes")
}

func TestFetch_ActualSizeTooLarge(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		// Don't set Content-Length, force reading to detect size
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytes.Repeat([]byte("X"), 2000))
	}))
	defer server.Close()

	fetcher := NewFetcher(1024, 30*time.Second) // maxSize = 1024
	result, err := fetcher.Fetch(server.URL)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "image too large")
}

func TestFetch_HTTPErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
	}{
		{"404 Not Found", http.StatusNotFound},
		{"403 Forbidden", http.StatusForbidden},
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			fetcher := NewFetcher(1024, 30*time.Second)
			result, err := fetcher.Fetch(server.URL)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "HTTP error")
		})
	}
}

func TestFetch_EmptyResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		// Don't write any data
	}))
	defer server.Close()

	fetcher := NewFetcher(1024, 30*time.Second)
	result, err := fetcher.Fetch(server.URL)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "empty response body")
}

func TestFetch_Timeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate slow server
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	// Create fetcher with very short timeout
	fetcher := NewFetcher(1024, 100*time.Millisecond)
	result, err := fetcher.Fetch(server.URL)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to fetch URL")
}

func TestFetch_TooManyRedirects(t *testing.T) {
	t.Parallel()

	// Create server that redirects to itself
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+"/redirect", http.StatusFound)
	}))
	defer server.Close()

	fetcher := NewFetcher(1024, 30*time.Second)
	result, err := fetcher.Fetch(server.URL)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "too many redirects")
}

func TestFetch_ValidRedirect(t *testing.T) {
	t.Parallel()

	// Create final destination server
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/gif")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("GIF89a"))
	}))
	defer finalServer.Close()

	// Create redirect server
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	fetcher := NewFetcher(1024, 30*time.Second)
	result, err := fetcher.Fetch(redirectServer.URL)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []byte("GIF89a"), result.Data)
	assert.Equal(t, "image/gif", result.ContentType)
	assert.Equal(t, int64(6), result.Size)
}

func TestFetch_DifferentImageTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		contentType string
		data        []byte
	}{
		{"PNG", "image/png", []byte("\x89PNG\r\n\x1a\n")},
		{"JPEG", "image/jpeg", []byte("\xFF\xD8\xFF")},
		{"GIF", "image/gif", []byte("GIF89a")},
		{"WebP", "image/webp", []byte("RIFF")},
		{"SVG", "image/svg+xml", []byte("<svg>")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(tc.data)
			}))
			defer server.Close()

			fetcher := NewFetcher(1024, 30*time.Second)
			result, err := fetcher.Fetch(server.URL)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tc.data, result.Data)
			assert.Equal(t, tc.contentType, result.ContentType)
			assert.Equal(t, int64(len(tc.data)), result.Size)
		})
	}
}

func TestFetch_InvalidURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		url  string
	}{
		{"invalid characters", "http://[::1]:namedport"},
		{"malformed", "http://foo bar.com"},
	}

	fetcher := NewFetcher(1024, 30*time.Second)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := fetcher.Fetch(tc.url)

			assert.Error(t, err)
			assert.Nil(t, result)
			// Error could be from URL validation or request creation
			assert.True(t, strings.Contains(err.Error(), "failed to create request") ||
				strings.Contains(err.Error(), "failed to fetch URL"))
		})
	}
}

// Package fetcher provides functionality to fetch images from remote URLs.
package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Fetcher handles fetching images from remote URLs with validation
type Fetcher struct {
	client      *http.Client
	userAgent   string
	maxSize     int64
	maxRedirect int
}

// FetchResult contains the fetched image data and metadata
type FetchResult struct {
	Data        []byte
	ContentType string
	URL         string
	Size        int64
}

// NewFetcher creates a new image fetcher with the given configuration
func NewFetcher(maxSize int64, timeout time.Duration) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		maxSize:     maxSize,
		userAgent:   "ImageProcessingAPI/1.0",
		maxRedirect: 10,
	}
}

// Fetch retrieves an image from the given URL with validation
func (f *Fetcher) Fetch(url string) (*FetchResult, error) {
	// Validate URL scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("invalid URL scheme: must be http or https")
	}

	// Create request with custom headers
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "image/*")

	// Execute request
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error but don't override the main error
			_ = closeErr
		}
	}()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Validate Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("invalid content type: %s (expected image/*)", contentType)
	}

	// Check Content-Length if provided
	if resp.ContentLength > f.maxSize {
		return nil, fmt.Errorf("image too large: %d bytes (max %d bytes)", resp.ContentLength, f.maxSize)
	}

	// Read body with size limit
	limitedReader := io.LimitReader(resp.Body, f.maxSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Verify actual size doesn't exceed limit
	size := int64(len(data))
	if size > f.maxSize {
		return nil, fmt.Errorf("image too large: %d bytes (max %d bytes)", size, f.maxSize)
	}

	// Verify we actually got some data
	if size == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	return &FetchResult{
		Data:        data,
		ContentType: contentType,
		URL:         url,
		Size:        size,
	}, nil
}

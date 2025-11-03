package handler

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTPRequestsTotal tracks the total number of HTTP requests by method, path, and status
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests by method, path, and status code",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration tracks the duration of HTTP requests
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// ImageCacheHits tracks the number of cache hits
	ImageCacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "image_cache_hits_total",
			Help: "Total number of image cache hits",
		},
	)

	// ImageCacheMisses tracks the number of cache misses
	ImageCacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "image_cache_misses_total",
			Help: "Total number of image cache misses",
		},
	)

	// ImageCacheSize tracks the current number of items in the cache
	ImageCacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "image_cache_size",
			Help: "Current number of images stored in cache",
		},
	)

	// ImageProcessingDuration tracks the time spent processing images
	ImageProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "image_processing_duration_seconds",
			Help:    "Time spent processing images in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// ImageFetchDuration tracks the time spent fetching images from source URLs
	ImageFetchDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "image_fetch_duration_seconds",
			Help:    "Time spent fetching images from source URLs in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
)

// MetricsHandler returns the Prometheus metrics HTTP handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// +build tools

// Package tools tracks build-time dependencies that are used in the project
// but not imported in the source code.
package tools

import (
	_ "github.com/disintegration/imaging"
	_ "github.com/go-chi/chi/v5"
	_ "github.com/prometheus/client_golang/prometheus"
	_ "github.com/prometheus/client_golang/prometheus/promhttp"
)

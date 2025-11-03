// Package placeholder provides functionality to generate placeholder images for errors.
package placeholder

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	// DefaultWidth is the default placeholder width if not specified
	DefaultWidth = 400
	// DefaultHeight is the default placeholder height if not specified
	DefaultHeight = 300
	// MaxDimension is the maximum allowed dimension for placeholders
	MaxDimension = 1400
)

// ColorScheme defines the colors for different error types
type ColorScheme int

const (
	// Orange color scheme for 4xx client errors
	Orange ColorScheme = iota
	// Red color scheme for 5xx server errors
	Red
	// Gray color scheme for other errors
	Gray
)

// Generate creates a placeholder image with the given status code and dimensions
func Generate(statusCode, width, height int) ([]byte, error) {
	// Validate and apply defaults for dimensions
	if width <= 0 {
		width = DefaultWidth
	}
	if height <= 0 {
		height = DefaultHeight
	}

	// Enforce maximum dimensions
	if width > MaxDimension {
		width = MaxDimension
	}
	if height > MaxDimension {
		height = MaxDimension
	}

	// Determine color scheme based on status code
	scheme := getColorScheme(statusCode)

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill background
	bgColor := getBackgroundColor(scheme)
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Add status code text
	textColor := color.RGBA{255, 255, 255, 255} // White text
	addLabel(img, fmt.Sprintf("%d", statusCode), textColor)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode placeholder image: %w", err)
	}

	return buf.Bytes(), nil
}

// getColorScheme determines the color scheme based on HTTP status code
func getColorScheme(statusCode int) ColorScheme {
	switch {
	case statusCode >= 400 && statusCode < 500:
		return Orange
	case statusCode >= 500 && statusCode < 600:
		return Red
	default:
		return Gray
	}
}

// getBackgroundColor returns the background color for a given scheme
func getBackgroundColor(scheme ColorScheme) color.RGBA {
	switch scheme {
	case Orange:
		return color.RGBA{255, 165, 0, 255} // Orange
	case Red:
		return color.RGBA{220, 53, 69, 255} // Red
	case Gray:
	default:
		return color.RGBA{128, 128, 128, 255} // Gray
	}
	return color.RGBA{128, 128, 128, 255}
}

// addLabel draws text centered in the image
func addLabel(img *image.RGBA, label string, col color.Color) {
	bounds := img.Bounds()

	// Use basic font
	face := basicfont.Face7x13

	// Calculate text width (approximate)
	textWidth := len(label) * 7

	// Calculate centered position
	x := (bounds.Dx() - textWidth) / 2
	y := bounds.Dy()/2 + 6 // Offset for baseline

	if x < 0 {
		x = 10
	}
	if y < 20 {
		y = 20
	}

	point := fixed.Point26_6{
		X: fixed.I(x),
		Y: fixed.I(y),
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  point,
	}
	d.DrawString(label)
}

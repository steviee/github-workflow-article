// Package processor provides image processing operations including rotation and resizing.
package processor

import (
	"image"

	"github.com/disintegration/imaging"
)

// Rotate90 rotates an image 90 degrees clockwise while preserving transparency.
// The output dimensions are swapped (width becomes height, height becomes width).
func Rotate90(img image.Image) image.Image {
	return imaging.Rotate90(img)
}

// Rotate180 rotates an image 180 degrees clockwise while preserving transparency.
// The output dimensions remain the same as the input.
func Rotate180(img image.Image) image.Image {
	return imaging.Rotate180(img)
}

// Rotate270 rotates an image 270 degrees clockwise (or 90 degrees counter-clockwise)
// while preserving transparency.
// The output dimensions are swapped (width becomes height, height becomes width).
func Rotate270(img image.Image) image.Image {
	return imaging.Rotate270(img)
}

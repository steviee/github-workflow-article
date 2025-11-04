// Package processor provides image processing operations including rotation and resizing.
package processor

import (
	"image"
	"math"

	"github.com/disintegration/imaging"
)

const (
	// MaxOutputDimension is the maximum allowed width or height for output images
	MaxOutputDimension = 1400
)

// Resize resizes an image to exact target dimensions while preserving aspect ratio.
// It uses a zoom-to-fill algorithm with center cropping:
// 1. Calculate scale factor to cover target dimensions (zoom to fill)
// 2. Scale image using high-quality Lanczos resampling
// 3. Crop to exact target size using center crop
//
// The function enforces maximum output dimensions of 1400x1400 pixels.
// If either dimension exceeds this limit, both dimensions are scaled
// proportionally to fit within the constraint.
//
// Transparency is preserved throughout the operation.
func Resize(img image.Image, width, height int) image.Image {
	if img == nil {
		return nil
	}

	// Enforce maximum output dimensions
	if width > MaxOutputDimension || height > MaxOutputDimension {
		width, height = enforceMaxDimensions(width, height)
	}

	// Handle edge cases
	if width <= 0 || height <= 0 {
		width = 1
		height = 1
	}

	bounds := img.Bounds()
	srcWidth := float64(bounds.Dx())
	srcHeight := float64(bounds.Dy())

	// If source is already the target size, return as-is
	if int(srcWidth) == width && int(srcHeight) == height {
		return img
	}

	// Calculate scale factor to cover target dimensions (zoom to fill)
	// We use max() to ensure the image covers the entire target area
	scaleX := float64(width) / srcWidth
	scaleY := float64(height) / srcHeight
	scale := math.Max(scaleX, scaleY)

	// Calculate scaled dimensions
	scaledWidth := int(math.Round(srcWidth * scale))
	scaledHeight := int(math.Round(srcHeight * scale))

	// Scale the image using high-quality Lanczos resampling filter
	scaled := imaging.Resize(img, scaledWidth, scaledHeight, imaging.Lanczos)

	// Calculate crop offset for center crop
	cropX := (scaledWidth - width) / 2
	cropY := (scaledHeight - height) / 2

	// Ensure crop offsets are not negative
	if cropX < 0 {
		cropX = 0
	}
	if cropY < 0 {
		cropY = 0
	}

	// Crop to exact target dimensions from center
	cropped := imaging.Crop(scaled, image.Rect(cropX, cropY, cropX+width, cropY+height))

	return cropped
}

// enforceMaxDimensions scales dimensions proportionally to fit within MaxOutputDimension constraint.
// It maintains the aspect ratio while ensuring neither dimension exceeds the maximum.
func enforceMaxDimensions(width, height int) (int, int) {
	if width <= 0 || height <= 0 {
		return 1, 1
	}

	// Calculate which dimension is the limiting factor
	scaleWidth := float64(MaxOutputDimension) / float64(width)
	scaleHeight := float64(MaxOutputDimension) / float64(height)

	// Use the smaller scale factor to ensure both dimensions fit
	scale := math.Min(scaleWidth, scaleHeight)

	// Apply scale if needed
	if scale < 1.0 {
		newWidth := int(math.Round(float64(width) * scale))
		newHeight := int(math.Round(float64(height) * scale))
		return newWidth, newHeight
	}

	return width, height
}

// Rotate90 rotates an image 90 degrees clockwise while preserving transparency.
// Note: The imaging library's Rotate270 function rotates counter-clockwise,
// so 270° CCW equals 90° CW.
func Rotate90(img image.Image) image.Image {
	if img == nil {
		return nil
	}
	return imaging.Rotate270(img)
}

// Rotate180 rotates an image 180 degrees clockwise while preserving transparency.
// Note: 180° rotation is the same in both clockwise and counter-clockwise directions.
func Rotate180(img image.Image) image.Image {
	if img == nil {
		return nil
	}
	return imaging.Rotate180(img)
}

// Rotate270 rotates an image 270 degrees clockwise while preserving transparency.
// Note: The imaging library's Rotate90 function rotates counter-clockwise,
// so 90° CCW equals 270° CW.
func Rotate270(img image.Image) image.Image {
	if img == nil {
		return nil
	}
	return imaging.Rotate90(img)
}

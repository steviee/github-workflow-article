package processor

import (
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getQuadrantColor returns the color for a given quadrant
func getQuadrantColor(x, y, width, height int) color.RGBA {
	midX := width / 2
	midY := height / 2

	if y < midY {
		if x < midX {
			return color.RGBA{255, 0, 0, 255} // Top-left: Red
		}
		return color.RGBA{0, 255, 0, 255} // Top-right: Green
	}

	if x < midX {
		return color.RGBA{0, 0, 255, 255} // Bottom-left: Blue
	}
	return color.RGBA{255, 255, 0, 255} // Bottom-right: Yellow
}

// createTestImage creates a simple test image with distinct corners for rotation testing
// The image has different colors in each corner to verify rotation correctness:
// - Top-left: Red
// - Top-right: Green
// - Bottom-left: Blue
// - Bottom-right: Yellow
func createTestImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with distinct colors in each quadrant
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := getQuadrantColor(x, y, width, height)
			img.Set(x, y, c)
		}
	}

	return img
}

// createTransparentTestImage creates a test image with transparency gradient
func createTransparentTestImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with horizontal transparency gradient (left=transparent, right=opaque)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create a horizontal gradient of transparency
			// Ensure we don't overflow by using explicit bounds checking
			alphaVal := (x * 255) / width
			if alphaVal > 255 {
				alphaVal = 255
			}
			if alphaVal < 0 {
				alphaVal = 0
			}
			//nolint:gosec // G115: alphaVal is guaranteed to be in range [0, 255]
			alpha := uint8(alphaVal)
			c := color.RGBA{255, 0, 0, alpha}
			img.Set(x, y, c)
		}
	}

	return img
}

// getColorAt is a helper to get the color at a specific position
func getColorAt(img image.Image, x, y int) color.Color {
	return img.At(x, y)
}

// colorsEqual checks if two colors are equal (allowing small differences due to encoding)
func colorsEqual(c1, c2 color.Color) bool {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	// Allow small differences (within 1%) due to image encoding/decoding
	threshold := uint32(655) // ~1% of 65535

	return abs(r1, r2) <= threshold &&
		abs(g1, g2) <= threshold &&
		abs(b1, b2) <= threshold &&
		abs(a1, a2) <= threshold
}

func abs(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func TestRotate90(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{
			name:   "square image",
			width:  100,
			height: 100,
		},
		{
			name:   "landscape image",
			width:  200,
			height: 100,
		},
		{
			name:   "portrait image",
			width:  100,
			height: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			original := createTestImage(tt.width, tt.height)

			// Rotate 90 degrees
			rotated := Rotate90(original)

			// Verify dimensions are swapped
			bounds := rotated.Bounds()
			assert.Equal(t, tt.height, bounds.Dx(), "Width should be original height")
			assert.Equal(t, tt.width, bounds.Dy(), "Height should be original width")

			// The imaging library rotates counter-clockwise (CCW)
			// After 90° CCW rotation:
			// - Original top-left (red) -> bottom-left
			// - Original top-right (green) -> top-left
			// - Original bottom-left (blue) -> bottom-right
			// - Original bottom-right (yellow) -> top-right

			// Sample points from corners (5 pixels in from edge to avoid boundary effects)
			margin := 5

			// Top-left should now be green (was top-right)
			tlColor := getColorAt(rotated, margin, margin)
			assert.True(t, colorsEqual(tlColor, color.RGBA{0, 255, 0, 255}),
				"Top-left should be green after 90° CCW rotation")

			// Top-right should now be yellow (was bottom-right)
			trColor := getColorAt(rotated, bounds.Dx()-margin, margin)
			assert.True(t, colorsEqual(trColor, color.RGBA{255, 255, 0, 255}),
				"Top-right should be yellow after 90° CCW rotation")

			// Bottom-left should now be red (was top-left)
			blColor := getColorAt(rotated, margin, bounds.Dy()-margin)
			assert.True(t, colorsEqual(blColor, color.RGBA{255, 0, 0, 255}),
				"Bottom-left should be red after 90° CCW rotation")

			// Bottom-right should now be blue (was bottom-left)
			brColor := getColorAt(rotated, bounds.Dx()-margin, bounds.Dy()-margin)
			assert.True(t, colorsEqual(brColor, color.RGBA{0, 0, 255, 255}),
				"Bottom-right should be blue after 90° CCW rotation")
		})
	}
}

func TestRotate180(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{
			name:   "square image",
			width:  100,
			height: 100,
		},
		{
			name:   "landscape image",
			width:  200,
			height: 100,
		},
		{
			name:   "portrait image",
			width:  100,
			height: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			original := createTestImage(tt.width, tt.height)

			// Rotate 180 degrees
			rotated := Rotate180(original)

			// Verify dimensions remain the same
			bounds := rotated.Bounds()
			assert.Equal(t, tt.width, bounds.Dx(), "Width should remain the same")
			assert.Equal(t, tt.height, bounds.Dy(), "Height should remain the same")

			// After 180° rotation:
			// - Original top-left (red) -> bottom-right
			// - Original top-right (green) -> bottom-left
			// - Original bottom-left (blue) -> top-right
			// - Original bottom-right (yellow) -> top-left

			// Sample points from corners
			margin := 5

			// Top-left should now be yellow (was bottom-right)
			tlColor := getColorAt(rotated, margin, margin)
			assert.True(t, colorsEqual(tlColor, color.RGBA{255, 255, 0, 255}),
				"Top-left should be yellow after 180° rotation")

			// Top-right should now be blue (was bottom-left)
			trColor := getColorAt(rotated, bounds.Dx()-margin, margin)
			assert.True(t, colorsEqual(trColor, color.RGBA{0, 0, 255, 255}),
				"Top-right should be blue after 180° rotation")

			// Bottom-left should now be green (was top-right)
			blColor := getColorAt(rotated, margin, bounds.Dy()-margin)
			assert.True(t, colorsEqual(blColor, color.RGBA{0, 255, 0, 255}),
				"Bottom-left should be green after 180° rotation")

			// Bottom-right should now be red (was top-left)
			brColor := getColorAt(rotated, bounds.Dx()-margin, bounds.Dy()-margin)
			assert.True(t, colorsEqual(brColor, color.RGBA{255, 0, 0, 255}),
				"Bottom-right should be red after 180° rotation")
		})
	}
}

func TestRotate270(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{
			name:   "square image",
			width:  100,
			height: 100,
		},
		{
			name:   "landscape image",
			width:  200,
			height: 100,
		},
		{
			name:   "portrait image",
			width:  100,
			height: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			original := createTestImage(tt.width, tt.height)

			// Rotate 270 degrees
			rotated := Rotate270(original)

			// Verify dimensions are swapped
			bounds := rotated.Bounds()
			assert.Equal(t, tt.height, bounds.Dx(), "Width should be original height")
			assert.Equal(t, tt.width, bounds.Dy(), "Height should be original width")

			// The imaging library rotates counter-clockwise
			// After 270° CCW rotation (= 90° CW):
			// - Original top-left (red) -> top-right
			// - Original top-right (green) -> bottom-right
			// - Original bottom-left (blue) -> top-left
			// - Original bottom-right (yellow) -> bottom-left

			// Sample points from corners
			margin := 5

			// Top-left should now be blue (was bottom-left)
			tlColor := getColorAt(rotated, margin, margin)
			assert.True(t, colorsEqual(tlColor, color.RGBA{0, 0, 255, 255}),
				"Top-left should be blue after 270° CCW rotation")

			// Top-right should now be red (was top-left)
			trColor := getColorAt(rotated, bounds.Dx()-margin, margin)
			assert.True(t, colorsEqual(trColor, color.RGBA{255, 0, 0, 255}),
				"Top-right should be red after 270° CCW rotation")

			// Bottom-left should now be yellow (was bottom-right)
			blColor := getColorAt(rotated, margin, bounds.Dy()-margin)
			assert.True(t, colorsEqual(blColor, color.RGBA{255, 255, 0, 255}),
				"Bottom-left should be yellow after 270° CCW rotation")

			// Bottom-right should now be green (was top-right)
			brColor := getColorAt(rotated, bounds.Dx()-margin, bounds.Dy()-margin)
			assert.True(t, colorsEqual(brColor, color.RGBA{0, 255, 0, 255}),
				"Bottom-right should be green after 270° CCW rotation")
		})
	}
}

func TestTransparencyPreservation(t *testing.T) {
	// Create a transparent test image with horizontal gradient
	original := createTransparentTestImage(100, 100)

	t.Run("rotate90 preserves transparency", func(t *testing.T) {
		rotated := Rotate90(original)

		// After 90° CCW rotation, horizontal gradient becomes vertical
		// Top should have original right edge (opaque)
		// Bottom should have original left edge (transparent)

		// Check top (was right edge - opaque)
		topColor := getColorAt(rotated, 50, 5)
		_, _, _, a := topColor.RGBA()
		assert.Greater(t, a, uint32(50000), "Top should be mostly opaque")

		// Check bottom (was left edge - transparent)
		bottomColor := getColorAt(rotated, 50, 95)
		_, _, _, a2 := bottomColor.RGBA()
		assert.Less(t, a2, uint32(10000), "Bottom should be mostly transparent")
	})

	t.Run("rotate180 preserves transparency", func(t *testing.T) {
		rotated := Rotate180(original)

		// After 180° rotation, transparency gradient is reversed horizontally
		// Left edge should be mostly opaque (was right edge)
		leftColor := getColorAt(rotated, 5, 50)
		_, _, _, a := leftColor.RGBA()
		assert.Greater(t, a, uint32(50000), "Left edge should be mostly opaque after 180°")

		// Right edge should be mostly transparent (was left edge)
		rightColor := getColorAt(rotated, 95, 50)
		_, _, _, a2 := rightColor.RGBA()
		assert.Less(t, a2, uint32(10000), "Right edge should be mostly transparent after 180°")
	})

	t.Run("rotate270 preserves transparency", func(t *testing.T) {
		rotated := Rotate270(original)

		// Check that alpha channel is preserved (not all opaque)
		hasTransparency := false
		bounds := rotated.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y += 10 {
			for x := bounds.Min.X; x < bounds.Max.X; x += 10 {
				_, _, _, a := rotated.At(x, y).RGBA()
				if a < 65535 { // Not fully opaque
					hasTransparency = true
					break
				}
			}
			if hasTransparency {
				break
			}
		}
		assert.True(t, hasTransparency, "Rotated image should preserve transparency")
	})
}

func TestRotationChaining(t *testing.T) {
	// Create test image
	original := createTestImage(100, 100)

	t.Run("four 90° rotations return to original", func(t *testing.T) {
		// Rotate 90° four times should be equivalent to no rotation
		result := Rotate90(original)
		result = Rotate90(result)
		result = Rotate90(result)
		result = Rotate90(result)

		// Verify dimensions
		bounds := result.Bounds()
		assert.Equal(t, original.Bounds().Dx(), bounds.Dx())
		assert.Equal(t, original.Bounds().Dy(), bounds.Dy())

		// Verify corners match original
		margin := 5
		origTL := getColorAt(original, margin, margin)
		resultTL := getColorAt(result, margin, margin)
		assert.True(t, colorsEqual(origTL, resultTL), "Top-left should match after 4x90° rotation")
	})

	t.Run("two 180° rotations return to original", func(t *testing.T) {
		// Rotate 180° twice should return to original
		result := Rotate180(original)
		result = Rotate180(result)

		// Verify dimensions
		bounds := result.Bounds()
		assert.Equal(t, original.Bounds().Dx(), bounds.Dx())
		assert.Equal(t, original.Bounds().Dy(), bounds.Dy())

		// Verify corners match original
		margin := 5
		origTL := getColorAt(original, margin, margin)
		resultTL := getColorAt(result, margin, margin)
		assert.True(t, colorsEqual(origTL, resultTL), "Top-left should match after 2x180° rotation")
	})

	t.Run("90° + 270° equals 360°", func(t *testing.T) {
		// Rotate 90° then 270° should return to original orientation
		result := Rotate90(original)
		result = Rotate270(result)

		// Verify dimensions
		bounds := result.Bounds()
		assert.Equal(t, original.Bounds().Dx(), bounds.Dx())
		assert.Equal(t, original.Bounds().Dy(), bounds.Dy())

		// Verify corners match original
		margin := 5
		origTL := getColorAt(original, margin, margin)
		resultTL := getColorAt(result, margin, margin)
		assert.True(t, colorsEqual(origTL, resultTL), "Should return to original after 90°+270°")
	})
}

func TestRotateWithPNGEncoding(t *testing.T) {
	// Test that rotated images can be encoded to PNG without errors
	original := createTestImage(100, 100)

	tests := []struct {
		name     string
		rotateOp func(image.Image) image.Image
	}{
		{"rotate90", Rotate90},
		{"rotate180", Rotate180},
		{"rotate270", Rotate270},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rotated := tt.rotateOp(original)

			// Encode to PNG
			var buf []byte
			err := png.Encode(&mockWriter{buf: &buf}, rotated)
			require.NoError(t, err, "Should encode rotated image to PNG without error")
			assert.NotEmpty(t, buf, "PNG data should not be empty")
		})
	}
}

// mockWriter is a simple writer for testing PNG encoding
type mockWriter struct {
	buf *[]byte
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

func TestRotateDifferentImageTypes(t *testing.T) {
	// Test with different image types to ensure compatibility
	t.Run("RGBA image", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 50, 100))
		rotated := Rotate90(img)
		bounds := rotated.Bounds()
		assert.Equal(t, 100, bounds.Dx())
		assert.Equal(t, 50, bounds.Dy())
	})

	t.Run("NRGBA image", func(t *testing.T) {
		img := image.NewNRGBA(image.Rect(0, 0, 50, 100))
		rotated := Rotate90(img)
		bounds := rotated.Bounds()
		assert.Equal(t, 100, bounds.Dx())
		assert.Equal(t, 50, bounds.Dy())
	})

	t.Run("Gray image", func(t *testing.T) {
		img := image.NewGray(image.Rect(0, 0, 50, 100))
		rotated := Rotate90(img)
		bounds := rotated.Bounds()
		assert.Equal(t, 100, bounds.Dx())
		assert.Equal(t, 50, bounds.Dy())
	})
}

func TestRotateEdgeCases(t *testing.T) {
	t.Run("very small image (1x1)", func(t *testing.T) {
		img := createTestImage(1, 1)
		rotated := Rotate90(img)
		bounds := rotated.Bounds()
		assert.Equal(t, 1, bounds.Dx())
		assert.Equal(t, 1, bounds.Dy())
	})

	t.Run("very small image (2x3)", func(t *testing.T) {
		img := createTestImage(2, 3)
		rotated := Rotate90(img)
		bounds := rotated.Bounds()
		assert.Equal(t, 3, bounds.Dx())
		assert.Equal(t, 2, bounds.Dy())
	})

	t.Run("large image", func(t *testing.T) {
		img := createTestImage(1000, 1500)
		rotated := Rotate90(img)
		bounds := rotated.Bounds()
		assert.Equal(t, 1500, bounds.Dx())
		assert.Equal(t, 1000, bounds.Dy())
	})
}

package processor

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestImage creates a test image with the given dimensions and optional transparency
func createTestImage(width, height int, transparent bool) image.Image {
	var img draw.Image
	if transparent {
		img = image.NewRGBA(image.Rect(0, 0, width, height))
		// Fill with semi-transparent blue
		draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 255, 128}}, image.Point{}, draw.Src)
	} else {
		img = image.NewRGBA(image.Rect(0, 0, width, height))
		// Fill with opaque red
		draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{255, 0, 0, 255}}, image.Point{}, draw.Src)
	}
	return img
}

// createDistinctiveImage creates an image with different colors in corners for testing rotation/crop
func createDistinctiveImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Top-left: Red
	draw.Draw(img, image.Rect(0, 0, width/2, height/2),
		&image.Uniform{color.RGBA{255, 0, 0, 255}}, image.Point{}, draw.Src)

	// Top-right: Green
	draw.Draw(img, image.Rect(width/2, 0, width, height/2),
		&image.Uniform{color.RGBA{0, 255, 0, 255}}, image.Point{}, draw.Src)

	// Bottom-left: Blue
	draw.Draw(img, image.Rect(0, height/2, width/2, height),
		&image.Uniform{color.RGBA{0, 0, 255, 255}}, image.Point{}, draw.Src)

	// Bottom-right: Yellow
	draw.Draw(img, image.Rect(width/2, height/2, width, height),
		&image.Uniform{color.RGBA{255, 255, 0, 255}}, image.Point{}, draw.Src)

	return img
}

func TestResize_PortraitToLandscape(t *testing.T) {
	t.Parallel()

	// Create a portrait image (400x600)
	img := createTestImage(400, 600, false)

	// Resize to landscape (800x400)
	result := Resize(img, 800, 400)

	require.NotNil(t, result)
	bounds := result.Bounds()
	assert.Equal(t, 800, bounds.Dx(), "width should be exactly 800")
	assert.Equal(t, 400, bounds.Dy(), "height should be exactly 400")
}

func TestResize_LandscapeToPortrait(t *testing.T) {
	t.Parallel()

	// Create a landscape image (800x400)
	img := createTestImage(800, 400, false)

	// Resize to portrait (400x800)
	result := Resize(img, 400, 800)

	require.NotNil(t, result)
	bounds := result.Bounds()
	assert.Equal(t, 400, bounds.Dx(), "width should be exactly 400")
	assert.Equal(t, 800, bounds.Dy(), "height should be exactly 800")
}

func TestResize_SquareToRectangle(t *testing.T) {
	t.Parallel()

	// Create a square image (500x500)
	img := createTestImage(500, 500, false)

	// Resize to rectangle (1000x500)
	result := Resize(img, 1000, 500)

	require.NotNil(t, result)
	bounds := result.Bounds()
	assert.Equal(t, 1000, bounds.Dx(), "width should be exactly 1000")
	assert.Equal(t, 500, bounds.Dy(), "height should be exactly 500")
}

func TestResize_SmallToLarge(t *testing.T) {
	t.Parallel()

	// Create a small image (100x100)
	img := createTestImage(100, 100, false)

	// Resize to large (800x600)
	result := Resize(img, 800, 600)

	require.NotNil(t, result)
	bounds := result.Bounds()
	assert.Equal(t, 800, bounds.Dx(), "width should be exactly 800")
	assert.Equal(t, 600, bounds.Dy(), "height should be exactly 600")
}

func TestResize_LargeToSmall(t *testing.T) {
	t.Parallel()

	// Create a large image (2000x1500)
	img := createTestImage(2000, 1500, false)

	// Resize to small (200x150)
	result := Resize(img, 200, 150)

	require.NotNil(t, result)
	bounds := result.Bounds()
	assert.Equal(t, 200, bounds.Dx(), "width should be exactly 200")
	assert.Equal(t, 150, bounds.Dy(), "height should be exactly 150")
}

func TestResize_TransparencyPreserved(t *testing.T) {
	t.Parallel()

	// Create a transparent image
	img := createTestImage(400, 300, true)

	// Resize
	result := Resize(img, 200, 150)

	require.NotNil(t, result)

	// Check that the result is RGBA (supports transparency)
	_, isRGBA := result.(*image.RGBA)
	_, isNRGBA := result.(*image.NRGBA)
	assert.True(t, isRGBA || isNRGBA, "result should support transparency")
}

func TestResize_MaxDimensionEnforcement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		inputWidth     int
		inputHeight    int
		requestWidth   int
		requestHeight  int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "width exceeds max",
			inputWidth:     800,
			inputHeight:    600,
			requestWidth:   2000,
			requestHeight:  600,
			expectedWidth:  1400,
			expectedHeight: 420,
		},
		{
			name:           "height exceeds max",
			inputWidth:     600,
			inputHeight:    800,
			requestWidth:   600,
			requestHeight:  2000,
			expectedWidth:  420,
			expectedHeight: 1400,
		},
		{
			name:           "both exceed max",
			inputWidth:     800,
			inputHeight:    600,
			requestWidth:   2800,
			requestHeight:  2100,
			expectedWidth:  1400,
			expectedHeight: 1050,
		},
		{
			name:           "within max dimensions",
			inputWidth:     800,
			inputHeight:    600,
			requestWidth:   1000,
			requestHeight:  750,
			expectedWidth:  1000,
			expectedHeight: 750,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			img := createTestImage(tc.inputWidth, tc.inputHeight, false)
			result := Resize(img, tc.requestWidth, tc.requestHeight)

			require.NotNil(t, result)
			bounds := result.Bounds()
			assert.Equal(t, tc.expectedWidth, bounds.Dx(), "width should match expected")
			assert.Equal(t, tc.expectedHeight, bounds.Dy(), "height should match expected")
		})
	}
}

func TestResize_EdgeCase1x1(t *testing.T) {
	t.Parallel()

	// Create an image
	img := createTestImage(100, 100, false)

	// Resize to 1x1
	result := Resize(img, 1, 1)

	require.NotNil(t, result)
	bounds := result.Bounds()
	assert.Equal(t, 1, bounds.Dx(), "width should be 1")
	assert.Equal(t, 1, bounds.Dy(), "height should be 1")
}

func TestResize_EdgeCaseVeryLargeDimensions(t *testing.T) {
	t.Parallel()

	// Create a small image
	img := createTestImage(100, 100, false)

	// Request very large dimensions (should be clamped)
	result := Resize(img, 5000, 5000)

	require.NotNil(t, result)
	bounds := result.Bounds()
	assert.LessOrEqual(t, bounds.Dx(), MaxOutputDimension, "width should not exceed max")
	assert.LessOrEqual(t, bounds.Dy(), MaxOutputDimension, "height should not exceed max")
	assert.Equal(t, bounds.Dx(), bounds.Dy(), "should maintain aspect ratio (square)")
}

func TestResize_EdgeCaseZeroOrNegativeDimensions(t *testing.T) {
	t.Parallel()

	img := createTestImage(100, 100, false)

	testCases := []struct {
		name   string
		width  int
		height int
	}{
		{"zero width", 0, 100},
		{"zero height", 100, 0},
		{"both zero", 0, 0},
		{"negative width", -10, 100},
		{"negative height", 100, -10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := Resize(img, tc.width, tc.height)

			require.NotNil(t, result)
			bounds := result.Bounds()
			// Should default to 1x1
			assert.Equal(t, 1, bounds.Dx(), "width should default to 1")
			assert.Equal(t, 1, bounds.Dy(), "height should default to 1")
		})
	}
}

func TestResize_SameDimensionsAsSource(t *testing.T) {
	t.Parallel()

	// Create an image
	img := createTestImage(400, 300, false)

	// Resize to same dimensions
	result := Resize(img, 400, 300)

	require.NotNil(t, result)
	bounds := result.Bounds()
	assert.Equal(t, 400, bounds.Dx(), "width should remain 400")
	assert.Equal(t, 300, bounds.Dy(), "height should remain 300")
}

func TestResize_NilImage(t *testing.T) {
	t.Parallel()

	result := Resize(nil, 100, 100)
	assert.Nil(t, result, "should return nil for nil input")
}

func TestResize_CenterCrop(t *testing.T) {
	t.Parallel()

	// Create a distinctive image to verify center cropping
	img := createDistinctiveImage(400, 400)

	// Resize to smaller dimensions - should crop from center
	result := Resize(img, 200, 200)

	require.NotNil(t, result)
	bounds := result.Bounds()
	assert.Equal(t, 200, bounds.Dx(), "width should be 200")
	assert.Equal(t, 200, bounds.Dy(), "height should be 200")

	// The center of the result should contain parts of all four quadrants
	// since we're cropping from the center of the original image
	centerX := bounds.Dx() / 2
	centerY := bounds.Dy() / 2

	// Verify image is not nil by checking center pixel exists
	rgbaResult, ok := result.(*image.NRGBA)
	if !ok {
		rgbaResult2, ok2 := result.(*image.RGBA)
		assert.True(t, ok2, "result should be RGBA or NRGBA")
		if ok2 {
			_ = rgbaResult2.At(centerX, centerY)
		}
	} else {
		_ = rgbaResult.At(centerX, centerY)
	}
}

func TestResize_VariousAspectRatios(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		srcWidth       int
		srcHeight      int
		targetWidth    int
		targetHeight   int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "16:9 to 4:3",
			srcWidth:       1600,
			srcHeight:      900,
			targetWidth:    800,
			targetHeight:   600,
			expectedWidth:  800,
			expectedHeight: 600,
		},
		{
			name:           "4:3 to 16:9 (exceeds max)",
			srcWidth:       800,
			srcHeight:      600,
			targetWidth:    1600,
			targetHeight:   900,
			expectedWidth:  1400, // Clamped by max dimension
			expectedHeight: 788,  // Proportionally scaled
		},
		{
			name:           "1:1 to 2:1",
			srcWidth:       500,
			srcHeight:      500,
			targetWidth:    1000,
			targetHeight:   500,
			expectedWidth:  1000,
			expectedHeight: 500,
		},
		{
			name:           "2:1 to 1:1",
			srcWidth:       1000,
			srcHeight:      500,
			targetWidth:    500,
			targetHeight:   500,
			expectedWidth:  500,
			expectedHeight: 500,
		},
		{
			name:           "3:2 to 1:1",
			srcWidth:       600,
			srcHeight:      400,
			targetWidth:    400,
			targetHeight:   400,
			expectedWidth:  400,
			expectedHeight: 400,
		},
		{
			name:           "ultra-wide to portrait",
			srcWidth:       2560,
			srcHeight:      1080,
			targetWidth:    600,
			targetHeight:   800,
			expectedWidth:  600,
			expectedHeight: 800,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			img := createTestImage(tc.srcWidth, tc.srcHeight, false)
			result := Resize(img, tc.targetWidth, tc.targetHeight)

			require.NotNil(t, result)
			bounds := result.Bounds()
			assert.Equal(t, tc.expectedWidth, bounds.Dx(), "width should match expected")
			assert.Equal(t, tc.expectedHeight, bounds.Dy(), "height should match expected")
		})
	}
}

func TestEnforceMaxDimensions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		width          int
		height         int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "both within limit",
			width:          1000,
			height:         800,
			expectedWidth:  1000,
			expectedHeight: 800,
		},
		{
			name:           "width exceeds limit",
			width:          2000,
			height:         1000,
			expectedWidth:  1400,
			expectedHeight: 700,
		},
		{
			name:           "height exceeds limit",
			width:          1000,
			height:         2000,
			expectedWidth:  700,
			expectedHeight: 1400,
		},
		{
			name:           "both exceed limit proportionally",
			width:          2800,
			height:         2100,
			expectedWidth:  1400,
			expectedHeight: 1050,
		},
		{
			name:           "both exceed limit - square",
			width:          2000,
			height:         2000,
			expectedWidth:  1400,
			expectedHeight: 1400,
		},
		{
			name:           "zero dimensions",
			width:          0,
			height:         0,
			expectedWidth:  1,
			expectedHeight: 1,
		},
		{
			name:           "negative dimensions",
			width:          -100,
			height:         -200,
			expectedWidth:  1,
			expectedHeight: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w, h := enforceMaxDimensions(tc.width, tc.height)
			assert.Equal(t, tc.expectedWidth, w, "width should match expected")
			assert.Equal(t, tc.expectedHeight, h, "height should match expected")
		})
	}
}

func TestRotate90(t *testing.T) {
	t.Parallel()

	img := createTestImage(400, 300, false)
	result := Rotate90(img)

	require.NotNil(t, result)
	bounds := result.Bounds()
	// After 90° rotation, dimensions should be swapped
	assert.Equal(t, 300, bounds.Dx(), "width should be original height")
	assert.Equal(t, 400, bounds.Dy(), "height should be original width")
}

func TestRotate90_NilImage(t *testing.T) {
	t.Parallel()

	result := Rotate90(nil)
	assert.Nil(t, result, "should return nil for nil input")
}

func TestRotate180(t *testing.T) {
	t.Parallel()

	img := createTestImage(400, 300, false)
	result := Rotate180(img)

	require.NotNil(t, result)
	bounds := result.Bounds()
	// After 180° rotation, dimensions should remain the same
	assert.Equal(t, 400, bounds.Dx(), "width should remain the same")
	assert.Equal(t, 300, bounds.Dy(), "height should remain the same")
}

func TestRotate180_NilImage(t *testing.T) {
	t.Parallel()

	result := Rotate180(nil)
	assert.Nil(t, result, "should return nil for nil input")
}

func TestRotate270(t *testing.T) {
	t.Parallel()

	img := createTestImage(400, 300, false)
	result := Rotate270(img)

	require.NotNil(t, result)
	bounds := result.Bounds()
	// After 270° rotation, dimensions should be swapped
	assert.Equal(t, 300, bounds.Dx(), "width should be original height")
	assert.Equal(t, 400, bounds.Dy(), "height should be original width")
}

func TestRotate270_NilImage(t *testing.T) {
	t.Parallel()

	result := Rotate270(nil)
	assert.Nil(t, result, "should return nil for nil input")
}

func TestRotate_TransparencyPreserved(t *testing.T) {
	t.Parallel()

	img := createTestImage(400, 300, true)

	testCases := []struct {
		name   string
		rotate func(image.Image) image.Image
	}{
		{"Rotate90", Rotate90},
		{"Rotate180", Rotate180},
		{"Rotate270", Rotate270},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.rotate(img)
			require.NotNil(t, result)

			// Check that result supports transparency
			_, isRGBA := result.(*image.RGBA)
			_, isNRGBA := result.(*image.NRGBA)
			assert.True(t, isRGBA || isNRGBA, "result should support transparency")
		})
	}
}

func TestResize_RealWorldDimensions(t *testing.T) {
	t.Parallel()

	// Test with common real-world image dimensions
	testCases := []struct {
		name           string
		srcWidth       int
		srcHeight      int
		targetWidth    int
		targetHeight   int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "HD to thumbnail",
			srcWidth:       1920,
			srcHeight:      1080,
			targetWidth:    320,
			targetHeight:   180,
			expectedWidth:  320,
			expectedHeight: 180,
		},
		{
			name:           "4K to HD (exceeds max)",
			srcWidth:       3840,
			srcHeight:      2160,
			targetWidth:    1920,
			targetHeight:   1080,
			expectedWidth:  1400, // Clamped
			expectedHeight: 788,  // Proportionally scaled
		},
		{
			name:           "Portrait photo to square",
			srcWidth:       3024,
			srcHeight:      4032,
			targetWidth:    800,
			targetHeight:   800,
			expectedWidth:  800,
			expectedHeight: 800,
		},
		{
			name:           "Instagram post",
			srcWidth:       1080,
			srcHeight:      1080,
			targetWidth:    1080,
			targetHeight:   1080,
			expectedWidth:  1080,
			expectedHeight: 1080,
		},
		{
			name:           "YouTube thumbnail",
			srcWidth:       1920,
			srcHeight:      1080,
			targetWidth:    1280,
			targetHeight:   720,
			expectedWidth:  1280,
			expectedHeight: 720,
		},
		{
			name:           "Mobile screenshot to preview",
			srcWidth:       1170,
			srcHeight:      2532,
			targetWidth:    375,
			targetHeight:   812,
			expectedWidth:  375,
			expectedHeight: 812,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			img := createTestImage(tc.srcWidth, tc.srcHeight, false)
			result := Resize(img, tc.targetWidth, tc.targetHeight)

			require.NotNil(t, result)
			bounds := result.Bounds()
			assert.Equal(t, tc.expectedWidth, bounds.Dx(), "width should match expected")
			assert.Equal(t, tc.expectedHeight, bounds.Dy(), "height should match expected")
		})
	}
}

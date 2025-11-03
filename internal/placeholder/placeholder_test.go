package placeholder

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate_Success(t *testing.T) {
	t.Parallel()

	data, err := Generate(404, 400, 300)

	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Greater(t, len(data), 0)

	// Verify it's a valid PNG
	img, err := png.Decode(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.NotNil(t, img)

	// Verify dimensions
	bounds := img.Bounds()
	assert.Equal(t, 400, bounds.Dx())
	assert.Equal(t, 300, bounds.Dy())
}

func TestGenerate_DifferentStatusCodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		scheme     ColorScheme
	}{
		{"400 Bad Request", 400, Orange},
		{"404 Not Found", 404, Orange},
		{"403 Forbidden", 403, Orange},
		{"499 Client Error", 499, Orange},
		{"500 Internal Server Error", 500, Red},
		{"502 Bad Gateway", 502, Red},
		{"503 Service Unavailable", 503, Red},
		{"599 Server Error", 599, Red},
		{"300 Redirect", 300, Gray},
		{"200 OK", 200, Gray},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := Generate(tc.statusCode, 200, 150)

			assert.NoError(t, err)
			assert.NotNil(t, data)

			// Verify it's a valid PNG
			img, err := png.Decode(bytes.NewReader(data))
			assert.NoError(t, err)
			assert.NotNil(t, img)

			// Verify color scheme
			scheme := getColorScheme(tc.statusCode)
			assert.Equal(t, tc.scheme, scheme)
		})
	}
}

func TestGenerate_DifferentDimensions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		width          int
		height         int
		expectedWidth  int
		expectedHeight int
	}{
		{"default dimensions", 0, 0, DefaultWidth, DefaultHeight},
		{"only width default", 0, 200, DefaultWidth, 200},
		{"only height default", 300, 0, 300, DefaultHeight},
		{"small image", 100, 100, 100, 100},
		{"large image", 1200, 800, 1200, 800},
		{"max width enforced", 2000, 600, MaxDimension, 600},
		{"max height enforced", 600, 2000, 600, MaxDimension},
		{"both dimensions exceed max", 2000, 2000, MaxDimension, MaxDimension},
		{"negative dimensions use defaults", -100, -200, DefaultWidth, DefaultHeight},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := Generate(404, tc.width, tc.height)

			assert.NoError(t, err)
			assert.NotNil(t, data)

			// Decode and verify dimensions
			img, err := png.Decode(bytes.NewReader(data))
			assert.NoError(t, err)

			bounds := img.Bounds()
			assert.Equal(t, tc.expectedWidth, bounds.Dx(), "width mismatch")
			assert.Equal(t, tc.expectedHeight, bounds.Dy(), "height mismatch")
		})
	}
}

func TestGenerate_ValidPNGFormat(t *testing.T) {
	t.Parallel()

	data, err := Generate(500, 300, 200)

	assert.NoError(t, err)
	assert.NotNil(t, data)

	// Verify PNG signature
	assert.True(t, len(data) > 8, "data too short to be valid PNG")
	assert.Equal(t, []byte{137, 80, 78, 71, 13, 10, 26, 10}, data[0:8], "invalid PNG signature")

	// Decode to ensure it's valid
	img, err := png.Decode(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.NotNil(t, img)
}

func TestGetColorScheme(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		statusCode     int
		expectedScheme ColorScheme
	}{
		{400, Orange},
		{404, Orange},
		{451, Orange},
		{499, Orange},
		{500, Red},
		{502, Red},
		{599, Red},
		{200, Gray},
		{300, Gray},
		{100, Gray},
		{600, Gray},
	}

	for _, tc := range testCases {
		scheme := getColorScheme(tc.statusCode)
		assert.Equal(t, tc.expectedScheme, scheme, "status code %d", tc.statusCode)
	}
}

func TestGetBackgroundColor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scheme        ColorScheme
		expectedColor string
	}{
		{Orange, "orange"},
		{Red, "red"},
		{Gray, "gray"},
	}

	for _, tc := range testCases {
		color := getBackgroundColor(tc.scheme)
		assert.NotNil(t, color, "scheme %v", tc.scheme)

		// Verify it's a valid RGB color
		switch tc.scheme {
		case Orange:
			assert.Equal(t, uint8(255), color.R, "orange R value")
			assert.Equal(t, uint8(165), color.G, "orange G value")
			assert.Equal(t, uint8(0), color.B, "orange B value")
		case Red:
			assert.Equal(t, uint8(220), color.R, "red R value")
			assert.Equal(t, uint8(53), color.G, "red G value")
			assert.Equal(t, uint8(69), color.B, "red B value")
		case Gray:
			assert.Equal(t, uint8(128), color.R, "gray R value")
			assert.Equal(t, uint8(128), color.G, "gray G value")
			assert.Equal(t, uint8(128), color.B, "gray B value")
		}
	}
}

func TestGenerate_ImageContainsText(t *testing.T) {
	t.Parallel()

	statusCode := 404
	data, err := Generate(statusCode, 400, 300)

	assert.NoError(t, err)
	assert.NotNil(t, data)

	img, err := png.Decode(bytes.NewReader(data))
	assert.NoError(t, err)

	// Verify image is mostly the background color (orange for 404)
	// but has some white pixels (text)
	rgbaImg, ok := img.(*image.RGBA)
	if !ok {
		// Convert to RGBA if needed
		bounds := img.Bounds()
		rgbaImg = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgbaImg.Set(x, y, img.At(x, y))
			}
		}
	}

	// Count white pixels (text)
	whitePix := 0
	totalPix := 0
	bounds := rgbaImg.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := rgbaImg.At(x, y).RGBA()
			totalPix++
			// Check for white pixels (text color)
			if r > 60000 && g > 60000 && b > 60000 && a > 60000 {
				whitePix++
			}
		}
	}

	// There should be some white pixels (text)
	assert.Greater(t, whitePix, 0, "no text pixels found")
	// But not too many (most should be background)
	assert.Less(t, whitePix, totalPix/10, "too many white pixels")
}

func TestGenerate_ConsistentOutput(t *testing.T) {
	t.Parallel()

	// Generate same placeholder twice
	data1, err1 := Generate(500, 300, 200)
	data2, err2 := Generate(500, 300, 200)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Should produce identical output
	assert.Equal(t, data1, data2, "placeholders should be deterministic")
}

func TestGenerate_DifferentStatusCodesProduceDifferentImages(t *testing.T) {
	t.Parallel()

	data404, err := Generate(404, 300, 200)
	assert.NoError(t, err)

	data500, err := Generate(500, 300, 200)
	assert.NoError(t, err)

	// Different status codes should produce different images
	assert.NotEqual(t, data404, data500, "404 and 500 should produce different images")
}

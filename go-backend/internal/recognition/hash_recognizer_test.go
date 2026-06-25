package recognition

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestDescriptorVariantsFromImageBytesReturnsScaleVariants(t *testing.T) {
	imageBytes := testFaceLikeJPEG(t, 160, 160)

	vectors, bounds, err := DescriptorVariantsFromImageBytes(imageBytes)
	if err != nil {
		t.Fatalf("DescriptorVariantsFromImageBytes failed: %v", err)
	}
	if bounds.Dx() != 160 || bounds.Dy() != 160 {
		t.Fatalf("unexpected bounds: %v", bounds)
	}
	if len(vectors) < 2 {
		t.Fatalf("expected at least two descriptor variants, got %d", len(vectors))
	}
}

func TestDescriptorFromImageBytesIsNormalized(t *testing.T) {
	imageBytes := testFaceLikeJPEG(t, 160, 160)

	vector, _, err := DescriptorFromImageBytes(imageBytes)
	if err != nil {
		t.Fatalf("DescriptorFromImageBytes failed: %v", err)
	}

	var norm float64
	for _, value := range vector {
		norm += float64(value * value)
	}
	if norm < 0.98 || norm > 1.02 {
		t.Fatalf("expected unit normalized descriptor, got norm squared %.4f", norm)
	}
}

func testFaceLikeJPEG(t *testing.T, width int, height int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			shade := uint8(180 + (x+y)%35)
			img.SetRGBA(x, y, color.RGBA{R: shade, G: shade, B: shade, A: 255})
		}
	}
	for y := height / 4; y < height*3/4; y++ {
		for x := width / 4; x < width*3/4; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 110 + uint8((x+y)%60), G: 90, B: 80, A: 255})
		}
	}

	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode test image: %v", err)
	}
	return buffer.Bytes()
}

package recognition

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
)

const ActiveDescriptorName = "cpu_hybrid_grid_v2_8x8_intensity_gradient"

type HashRecognizer struct{}

type descriptorVariant struct {
	regionScale   float64
	verticalShift float64
	blurRadius    int
}

type grayMatrix struct {
	width  int
	height int
	values []float64
}

var descriptorVariants = []descriptorVariant{
	{regionScale: 0.92, verticalShift: -0.02, blurRadius: 0},
	{regionScale: 0.82, verticalShift: -0.02, blurRadius: 0},
	{regionScale: 0.92, verticalShift: -0.02, blurRadius: 1},
	{regionScale: 0.82, verticalShift: -0.02, blurRadius: 2},
	{regionScale: 1.00, verticalShift: -0.03, blurRadius: 1},
}

func NewHashRecognizer() *HashRecognizer {
	return &HashRecognizer{}
}

func (r *HashRecognizer) Recognize(ctx context.Context, imageBytes []byte) ([]FaceObservation, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	vector, bounds, err := DescriptorFromImageBytes(imageBytes)
	if err != nil {
		return nil, err
	}
	return []FaceObservation{
		{
			Descriptor: vector,
			Box: Box{
				X:      0,
				Y:      0,
				Width:  bounds.Dx(),
				Height: bounds.Dy(),
			},
			Quality: 1,
		},
	}, nil
}

func (r *HashRecognizer) Close() error {
	return nil
}

func DescriptorFromImageBytes(imageBytes []byte) ([128]float32, image.Rectangle, error) {
	img, bounds, err := decodeImage(imageBytes)
	if err != nil {
		return [128]float32{}, image.Rectangle{}, err
	}
	return hybridGridVector(toGrayMatrix(img), descriptorVariants[0]), bounds, nil
}

func DescriptorVariantsFromImageBytes(imageBytes []byte) ([][128]float32, image.Rectangle, error) {
	img, bounds, err := decodeImage(imageBytes)
	if err != nil {
		return nil, image.Rectangle{}, err
	}

	gray := toGrayMatrix(img)
	vectors := make([][128]float32, 0, len(descriptorVariants))
	for _, variant := range descriptorVariants {
		vector := hybridGridVector(gray, variant)
		if !hasSimilarVector(vectors, vector, 0.015) {
			vectors = append(vectors, vector)
		}
	}
	return vectors, bounds, nil
}

func decodeImage(imageBytes []byte) (image.Image, image.Rectangle, error) {
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, image.Rectangle{}, err
	}
	return img, img.Bounds(), nil
}

func hybridGridVector(gray grayMatrix, variant descriptorVariant) [128]float32 {
	var vector [128]float32
	if gray.width <= 0 || gray.height <= 0 {
		return vector
	}
	if variant.blurRadius > 0 {
		gray = blurGrayMatrix(gray, variant.blurRadius)
	}

	roi := descriptorRegion(gray.width, gray.height, variant)
	index := 0
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			x0 := roi.Min.X + (col * roi.Dx() / 8)
			x1 := roi.Min.X + ((col + 1) * roi.Dx() / 8)
			y0 := roi.Min.Y + (row * roi.Dy() / 8)
			y1 := roi.Min.Y + ((row + 1) * roi.Dy() / 8)

			intensity, gradient := cellStats(gray, x0, y0, x1, y1)
			vector[index] = float32(intensity)
			vector[64+index] = float32(gradient)
			index++
		}
	}

	normalizeDescriptor(&vector)
	return vector
}

func blurGrayMatrix(gray grayMatrix, radius int) grayMatrix {
	if radius <= 0 || gray.width == 0 || gray.height == 0 {
		return gray
	}

	values := make([]float64, len(gray.values))
	for y := 0; y < gray.height; y++ {
		for x := 0; x < gray.width; x++ {
			var sum float64
			var count float64
			for yy := y - radius; yy <= y+radius; yy++ {
				for xx := x - radius; xx <= x+radius; xx++ {
					sum += gray.at(xx, yy)
					count++
				}
			}
			values[y*gray.width+x] = sum / count
		}
	}
	return grayMatrix{width: gray.width, height: gray.height, values: values}
}

func descriptorRegion(width int, height int, variant descriptorVariant) image.Rectangle {
	scale := variant.regionScale
	if scale <= 0 || scale > 1 {
		scale = 1
	}
	regionWidth := math.Max(8, float64(width)*scale)
	regionHeight := math.Max(8, float64(height)*scale)
	centerX := float64(width) / 2
	centerY := float64(height)/2 + float64(height)*variant.verticalShift

	x0 := int(math.Round(centerX - regionWidth/2))
	y0 := int(math.Round(centerY - regionHeight/2))
	x1 := x0 + int(math.Round(regionWidth))
	y1 := y0 + int(math.Round(regionHeight))

	if x0 < 0 {
		x1 -= x0
		x0 = 0
	}
	if y0 < 0 {
		y1 -= y0
		y0 = 0
	}
	if x1 > width {
		x0 -= x1 - width
		x1 = width
	}
	if y1 > height {
		y0 -= y1 - height
		y1 = height
	}
	x0 = clampInt(x0, 0, width-1)
	y0 = clampInt(y0, 0, height-1)
	x1 = clampInt(x1, x0+1, width)
	y1 = clampInt(y1, y0+1, height)
	return image.Rect(x0, y0, x1, y1)
}

func cellStats(gray grayMatrix, x0 int, y0 int, x1 int, y1 int) (float64, float64) {
	var intensitySum float64
	var gradientSum float64
	var count float64

	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			intensity := gray.at(x, y)
			gx := gray.at(x+1, y) - gray.at(x-1, y)
			gy := gray.at(x, y+1) - gray.at(x, y-1)
			intensitySum += intensity
			gradientSum += math.Sqrt(gx*gx + gy*gy)
			count++
		}
	}

	if count == 0 {
		return 0, 0
	}
	return intensitySum / count, gradientSum / count
}

func toGrayMatrix(img image.Image) grayMatrix {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	values := make([]float64, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			gray := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535.0
			values[y*width+x] = gray
		}
	}
	return grayMatrix{width: width, height: height, values: values}
}

func (g grayMatrix) at(x int, y int) float64 {
	if g.width == 0 || g.height == 0 {
		return 0
	}
	x = clampInt(x, 0, g.width-1)
	y = clampInt(y, 0, g.height-1)
	return g.values[y*g.width+x]
}

func normalizeDescriptor(vector *[128]float32) {
	var intensityMean float64
	var gradientMean float64
	for i := 0; i < 64; i++ {
		intensityMean += float64(vector[i])
		gradientMean += float64(vector[64+i])
	}
	intensityMean /= 64
	gradientMean /= 64

	var intensityVariance float64
	var gradientVariance float64
	for i := 0; i < 64; i++ {
		intensityCentered := float64(vector[i]) - intensityMean
		gradientCentered := float64(vector[64+i]) - gradientMean
		vector[i] = float32(intensityCentered)
		vector[64+i] = float32(gradientCentered)
		intensityVariance += intensityCentered * intensityCentered
		gradientVariance += gradientCentered * gradientCentered
	}

	intensityStdDev := math.Sqrt(intensityVariance / 64)
	gradientStdDev := math.Sqrt(gradientVariance / 64)
	for i := 0; i < 64; i++ {
		if intensityStdDev > 0.000001 {
			vector[i] = float32(float64(vector[i]) / intensityStdDev)
		}
		if gradientStdDev > 0.000001 {
			vector[64+i] = float32(float64(vector[64+i]) / gradientStdDev)
		}
	}

	var norm float64
	for _, value := range vector {
		norm += float64(value * value)
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return
	}
	for i := range vector {
		vector[i] = float32(float64(vector[i]) / norm)
	}
}

func hasSimilarVector(vectors [][128]float32, candidate [128]float32, threshold float64) bool {
	for _, vector := range vectors {
		if euclidean(vector, candidate) <= threshold {
			return true
		}
	}
	return false
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

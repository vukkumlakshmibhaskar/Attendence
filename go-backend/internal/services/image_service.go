package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"strings"

	"classroom-attendance/go-backend/internal/recognition"
)

func decodeBase64Image(raw string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, fmt.Errorf("image_base64 is empty")
	}
	if comma := strings.Index(value, ","); comma >= 0 {
		value = value[comma+1:]
	}
	return base64.StdEncoding.DecodeString(value)
}

func cropJPEG(imageBytes []byte, box recognition.Box) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	rect := image.Rect(box.X, box.Y, box.X+box.Width, box.Y+box.Height)
	if box.Width <= 0 || box.Height <= 0 {
		rect = bounds
	}
	rect = rect.Intersect(bounds)
	if rect.Empty() {
		rect = bounds
	}

	rgba := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, rect.Min, draw.Src)

	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, rgba, &jpeg.Options{Quality: 82}); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

package recognition

import (
	"fmt"
	"strings"
)

func NewRecognizer(mode string, dlibModelDir string) (Recognizer, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "mock", "hash", "":
		return NewHashRecognizer(), nil
	case "dlib", "go-face", "goface":
		return NewDlibRecognizer(dlibModelDir)
	default:
		return nil, fmt.Errorf("unsupported face recognizer mode %q", mode)
	}
}

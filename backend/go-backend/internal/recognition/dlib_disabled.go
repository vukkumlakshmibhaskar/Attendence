//go:build !dlib

package recognition

import "fmt"

func NewDlibRecognizer(modelDir string) (Recognizer, error) {
	return nil, fmt.Errorf("dlib recognizer is not compiled; build with -tags dlib and set DLIB_MODEL_DIR, requested dir: %s", modelDir)
}

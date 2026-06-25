package cognitive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type LabelResult struct {
	Label         string    `json:"label"`
	Confidence    float64   `json:"confidence"`
	Probabilities []float64 `json:"probabilities"`
}

type LivenessResult struct {
	IsLive        bool      `json:"is_live"`
	Confidence    float64   `json:"confidence"`
	Probabilities []float64 `json:"probabilities"`
}

type AnalysisResult struct {
	Emotion        LabelResult    `json:"emotion"`
	Gaze           LabelResult    `json:"gaze"`
	Liveness       LivenessResult `json:"liveness"`
	ProcessingMS   int            `json:"processing_ms"`
	ServiceVersion string         `json:"service_version"`
	ModelMode      string         `json:"model_mode"`
	Features       map[string]any `json:"features,omitempty"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Analyze(ctx context.Context, jpegBytes []byte) (AnalysisResult, []byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "face.jpg")
	if err != nil {
		return AnalysisResult{}, nil, err
	}
	if _, err := part.Write(jpegBytes); err != nil {
		return AnalysisResult{}, nil, err
	}
	if err := writer.Close(); err != nil {
		return AnalysisResult{}, nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/analyze", &body)
	if err != nil {
		return AnalysisResult{}, nil, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := c.http.Do(request)
	if err != nil {
		return AnalysisResult{}, nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return AnalysisResult{}, nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return AnalysisResult{}, responseBody, fmt.Errorf("cognitive service status %d: %s", response.StatusCode, string(responseBody))
	}

	var result AnalysisResult
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return AnalysisResult{}, responseBody, err
	}
	return result, responseBody, nil
}

func (c *Client) Health(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("cognitive health returned %d", response.StatusCode)
	}
	return nil
}

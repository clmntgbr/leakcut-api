package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go-api/internal/domain/port"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type recognizeRequest struct {
	Images []recognizeImage `json:"images"`
	Lang   string           `json:"lang"`
}

type recognizeImage struct {
	FrameID     string `json:"frame_id"`
	ImageBase64 string `json:"image_base64"`
}

type recognizeResponse struct {
	Results []recognizeResult `json:"results"`
}

type recognizeResult struct {
	FrameID    string  `json:"frame_id"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Status     string  `json:"status"`
}

func (c *Client) Recognize(ctx context.Context, images []port.OCRImage, lang string) ([]port.OCRItemResult, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("ocr url is not configured")
	}

	payload := recognizeRequest{Lang: lang, Images: make([]recognizeImage, 0, len(images))}
	for _, image := range images {
		payload.Images = append(payload.Images, recognizeImage{
			FrameID:     image.FrameID,
			ImageBase64: base64.StdEncoding.EncodeToString(image.Data),
		})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/ocr", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ocr service returned %d", resp.StatusCode)
	}

	var decoded recognizeResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}

	out := make([]port.OCRItemResult, 0, len(decoded.Results))
	for _, item := range decoded.Results {
		out = append(out, port.OCRItemResult{
			FrameID:    item.FrameID,
			Text:       item.Text,
			Confidence: item.Confidence,
			Status:     item.Status,
		})
	}
	return out, nil
}

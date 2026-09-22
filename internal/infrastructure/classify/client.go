package classify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-api/internal/domain/port"
)

var jevQuestions = map[string]jevQuestion{
	"has_email": {
		Type:         "boolean",
		Instructions: "Does the on-screen text contain an email address?",
	},
	"has_iban": {
		Type:         "boolean",
		Instructions: "Does the text contain a bank IBAN or bank account number?",
	},
	"has_api_key": {
		Type:         "boolean",
		Instructions: "Does the text contain an API key, access token, client secret, or similar credential?",
	},
	"has_password": {
		Type:         "boolean",
		Instructions: "Does the text contain a password, passphrase, or login credential?",
	},
	"has_credit_card": {
		Type:         "boolean",
		Instructions: "Does the text contain a payment card number?",
	},
	"has_phone": {
		Type:         "boolean",
		Instructions: "Does the text contain a personal phone number?",
	},
	"has_private_key": {
		Type:         "boolean",
		Instructions: "Does the text contain a private cryptographic key or certificate?",
	},
	"has_personal_id": {
		Type:         "boolean",
		Instructions: "Does the text contain a national ID, social security number, or similar personal identifier?",
	},
}

var categoryNames = map[string]string{
	"has_email":       "email",
	"has_iban":        "iban",
	"has_api_key":     "api_key",
	"has_password":    "password",
	"has_credit_card": "credit_card",
	"has_phone":       "phone",
	"has_private_key": "private_key",
	"has_personal_id": "personal_id",
}

type Client struct {
	baseURL     string
	apiKey      string
	model       string
	concurrency int
	httpClient  *http.Client
}

func NewClient(baseURL, apiKey, model string, timeout time.Duration, concurrency int) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if concurrency <= 0 {
		concurrency = 4
	}
	if model == "" {
		model = "typesafe-ai/jev"
	}
	if baseURL == "" {
		baseURL = "https://ai-gateway.vercel.sh"
	}

	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		apiKey:      strings.TrimSpace(apiKey),
		model:       model,
		concurrency: concurrency,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type jevQuestion struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
}

type evaluateRequest struct {
	Model     string                 `json:"model"`
	State     map[string]string      `json:"state"`
	Questions map[string]jevQuestion `json:"questions"`
}

type evaluateResponse struct {
	Answers map[string]jevAnswer `json:"answers"`
}

type jevAnswer struct {
	Type        string  `json:"type"`
	Probability float64 `json:"probability"`
	Noul        float64 `json:"noul"`
}

func (c *Client) Classify(ctx context.Context, frames []port.ClassifyFrame, threshold float64) ([]port.ClassifyItemResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("AI_GATEWAY_API_KEY is not configured")
	}
	if threshold <= 0 {
		threshold = 0.7
	}
	log.Printf("classify: calling jev model=%s url=%s/v1/evaluate frames=%d threshold=%.2f key=%t",
		c.model, c.baseURL, len(frames), threshold, c.apiKey != "")

	out := make([]port.ClassifyItemResult, len(frames))
	sem := make(chan struct{}, c.concurrency)
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

	for i, frame := range frames {
		if strings.TrimSpace(frame.Text) == "" {
			out[i] = port.ClassifyItemResult{FrameID: frame.FrameID, Status: "skipped"}
			continue
		}

		wg.Add(1)
		go func(index int, frame port.ClassifyFrame) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				once.Do(func() { firstErr = ctx.Err() })
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			item, err := c.evaluateFrame(ctx, frame, threshold)
			if err != nil {
				once.Do(func() { firstErr = err })
				return
			}
			out[index] = item
		}(i, frame)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

func (c *Client) evaluateFrame(ctx context.Context, frame port.ClassifyFrame, threshold float64) (port.ClassifyItemResult, error) {
	body, err := json.Marshal(evaluateRequest{
		Model:     c.model,
		State:     map[string]string{"source": "ocr_frame", "text": frame.Text},
		Questions: jevQuestions,
	})
	if err != nil {
		return port.ClassifyItemResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/evaluate", bytes.NewReader(body))
	if err != nil {
		return port.ClassifyItemResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("classify: frame %s jev request: %v", frame.FrameID, err)
		return port.ClassifyItemResult{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("classify: frame %s read body: %v", frame.FrameID, err)
		return port.ClassifyItemResult{}, err
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		log.Printf("classify: frame %s jev retryable status=%d body=%s", frame.FrameID, resp.StatusCode, truncate(raw, 500))
		return port.ClassifyItemResult{}, fmt.Errorf("jev gateway returned %d", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("classify: frame %s jev failed status=%d body=%s", frame.FrameID, resp.StatusCode, truncate(raw, 500))
		return port.ClassifyItemResult{
			FrameID: frame.FrameID,
			Status:  "failed",
		}, nil
	}

	var decoded evaluateResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		log.Printf("classify: frame %s decode answers: %v body=%s", frame.FrameID, err, truncate(raw, 500))
		return port.ClassifyItemResult{FrameID: frame.FrameID, Status: "failed"}, nil
	}
	if len(decoded.Answers) == 0 {
		log.Printf("classify: frame %s jev 200 but empty answers body=%s", frame.FrameID, truncate(raw, 500))
	}

	categories := make([]port.ClassifyCategory, 0, len(categoryNames))
	probability := 0.0
	for key, name := range categoryNames {
		answer, ok := decoded.Answers[key]
		if !ok {
			continue
		}
		score := answer.Probability
		if score == 0 && answer.Noul > 0 {
			score = answer.Noul
		}
		if score > probability {
			probability = score
		}
		categories = append(categories, port.ClassifyCategory{Name: name, Probability: score})
	}

	log.Printf("classify: frame %s jev ok status=%d categories=%d max=%.3f confidential=%t",
		frame.FrameID, resp.StatusCode, len(categories), probability, probability >= threshold)

	return port.ClassifyItemResult{
		FrameID:      frame.FrameID,
		Confidential: probability >= threshold,
		Probability:  probability,
		Categories:   categories,
		Status:       "success",
	}, nil
}

func truncate(raw []byte, limit int) string {
	text := strings.TrimSpace(string(raw))
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

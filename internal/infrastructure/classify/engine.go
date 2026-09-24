package classify

import (
	"strings"
	"time"

	"go-api/internal/domain/port"
)

const (
	EngineJev   = "jev"
	EngineLocal = "local"
)

func NewClassifier(
	engine, baseURL, apiKey, model string,
	timeout time.Duration,
	concurrency int,
) port.Classifier {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case EngineLocal, "rules", "regex":
		return NewLocalDetector()
	default:
		return NewClient(baseURL, apiKey, model, timeout, concurrency)
	}
}

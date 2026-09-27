package frame

import (
	"testing"

	"github.com/google/uuid"
)

func TestDecideRetention_ConfidentialAlwaysKept(t *testing.T) {
	id := uuid.New()
	out := DecideRetention([]RetentionCandidate{{
		FrameID: id, TimestampMs: 1000, StorageKey: "a.jpg", OCRStatus: "success", Confidential: true,
	}}, RetentionPolicy{RatioNeutral: 0, RatioEmpty: 0, ContextWindowSeconds: 0})
	if len(out) != 1 || !out[0].Retained || out[0].PruneReason != PruneReasonKeptConfidential {
		t.Fatalf("got %+v", out)
	}
}

func TestDecideRetention_ContextWindow(t *testing.T) {
	alarm := uuid.New()
	before := uuid.New()
	far := uuid.New()
	out := DecideRetention([]RetentionCandidate{
		{FrameID: before, TimestampMs: 1000, StorageKey: "b.jpg", OCRStatus: "success", Confidential: false},
		{FrameID: alarm, TimestampMs: 3000, StorageKey: "a.jpg", OCRStatus: "success", Confidential: true},
		{FrameID: far, TimestampMs: 20000, StorageKey: "f.jpg", OCRStatus: "success", Confidential: false},
	}, RetentionPolicy{RatioNeutral: 0, RatioEmpty: 0, ContextWindowSeconds: 4})

	byID := map[uuid.UUID]RetentionDecision{}
	for _, d := range out {
		byID[d.FrameID] = d
	}
	if !byID[before].Retained || byID[before].PruneReason != PruneReasonKeptContext {
		t.Fatalf("before: %+v", byID[before])
	}
	if byID[far].Retained {
		t.Fatalf("far should be pruned: %+v", byID[far])
	}
}

func TestDecideRetention_EmptyVsNeutralRatios(t *testing.T) {
	empty := make([]RetentionCandidate, 0, 10)
	for i := 0; i < 10; i++ {
		empty = append(empty, RetentionCandidate{
			FrameID: uuid.New(), TimestampMs: int64(i * 1000), StorageKey: "e.jpg", OCRStatus: "empty",
		})
	}
	out := DecideRetention(empty, RetentionPolicy{RatioNeutral: 0.5, RatioEmpty: 0.1, ContextWindowSeconds: 4})
	kept := 0
	for _, d := range out {
		if d.Retained {
			kept++
			if d.PruneReason != PruneReasonKeptSampledEmpty {
				t.Fatalf("reason: %s", d.PruneReason)
			}
		}
	}
	if kept != 1 {
		t.Fatalf("empty 10%% of 10: kept %d want 1", kept)
	}

	neutral := make([]RetentionCandidate, 0, 8)
	for i := 0; i < 8; i++ {
		neutral = append(neutral, RetentionCandidate{
			FrameID: uuid.New(), TimestampMs: int64(i * 1000), StorageKey: "n.jpg", OCRStatus: "success",
		})
	}
	out = DecideRetention(neutral, RetentionPolicy{RatioNeutral: 0.5, RatioEmpty: 0.1, ContextWindowSeconds: 4})
	kept = 0
	for _, d := range out {
		if d.Retained {
			kept++
		}
	}
	if kept != 4 {
		t.Fatalf("neutral 50%% of 8: kept %d want 4", kept)
	}
}

func TestDecideRetention_RatioZeroPrunesAllNonAlarm(t *testing.T) {
	out := DecideRetention([]RetentionCandidate{{
		FrameID: uuid.New(), TimestampMs: 0, StorageKey: "n.jpg", OCRStatus: "success",
	}}, RetentionPolicy{RatioNeutral: 0, RatioEmpty: 0, ContextWindowSeconds: 0})
	if out[0].Retained {
		t.Fatalf("expected prune: %+v", out[0])
	}
}

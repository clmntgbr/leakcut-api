package frame

import (
	"math"
	"sort"

	"github.com/google/uuid"
)

const (
	PruneReasonKeptConfidential       = "kept_confidential"
	PruneReasonKeptContext            = "kept_context"
	PruneReasonKeptSampledNeutral     = "kept_sampled_neutral"
	PruneReasonKeptSampledEmpty       = "kept_sampled_empty"
	PruneReasonPrunedRedundantNeutral = "pruned_redundant_neutral"
	PruneReasonPrunedEmpty            = "pruned_empty"
)

type RetentionPolicy struct {
	RatioNeutral           float64
	RatioEmpty             float64
	ContextWindowSeconds   int
}

func NormalizeRetentionPolicy(p RetentionPolicy) RetentionPolicy {
	if p.RatioNeutral < 0 {
		p.RatioNeutral = 0
	}
	if p.RatioNeutral > 1 {
		p.RatioNeutral = 1
	}
	if p.RatioEmpty < 0 {
		p.RatioEmpty = 0
	}
	if p.RatioEmpty > 1 {
		p.RatioEmpty = 1
	}
	if p.ContextWindowSeconds < 0 {
		p.ContextWindowSeconds = 0
	}
	return p
}

type RetentionCandidate struct {
	FrameID      uuid.UUID
	TimestampMs  int64
	StorageKey   string
	OCRStatus    string
	Confidential bool
}

type RetentionDecision struct {
	FrameID     uuid.UUID
	StorageKey  string
	Retained    bool
	PruneReason string
}

// DecideRetention applies option B (context window around confidential frames)
// then option A (regular temporal sampling) with distinct ratios for empty vs neutral.
func DecideRetention(candidates []RetentionCandidate, policy RetentionPolicy) []RetentionDecision {
	policy = NormalizeRetentionPolicy(policy)
	if len(candidates) == 0 {
		return nil
	}

	sorted := append([]RetentionCandidate(nil), candidates...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].TimestampMs == sorted[j].TimestampMs {
			return sorted[i].FrameID.String() < sorted[j].FrameID.String()
		}
		return sorted[i].TimestampMs < sorted[j].TimestampMs
	})

	alarmTimestamps := make([]int64, 0)
	for _, c := range sorted {
		if c.Confidential {
			alarmTimestamps = append(alarmTimestamps, c.TimestampMs)
		}
	}
	windowMs := int64(policy.ContextWindowSeconds) * 1000

	decisions := make([]RetentionDecision, len(sorted))
	emptyIdx := 0
	neutralIdx := 0

	for i, c := range sorted {
		d := RetentionDecision{
			FrameID:    c.FrameID,
			StorageKey: c.StorageKey,
		}
		switch {
		case c.Confidential:
			d.Retained = true
			d.PruneReason = PruneReasonKeptConfidential
		case withinContextWindow(c.TimestampMs, alarmTimestamps, windowMs):
			d.Retained = true
			d.PruneReason = PruneReasonKeptContext
		case c.OCRStatus == "empty":
			if sampleKeep(emptyIdx, policy.RatioEmpty) {
				d.Retained = true
				d.PruneReason = PruneReasonKeptSampledEmpty
			} else {
				d.Retained = false
				d.PruneReason = PruneReasonPrunedEmpty
			}
			emptyIdx++
		default:
			if sampleKeep(neutralIdx, policy.RatioNeutral) {
				d.Retained = true
				d.PruneReason = PruneReasonKeptSampledNeutral
			} else {
				d.Retained = false
				d.PruneReason = PruneReasonPrunedRedundantNeutral
			}
			neutralIdx++
		}
		decisions[i] = d
	}
	return decisions
}

func withinContextWindow(ts int64, alarms []int64, windowMs int64) bool {
	if windowMs < 0 || len(alarms) == 0 {
		return false
	}
	for _, alarm := range alarms {
		delta := ts - alarm
		if delta < 0 {
			delta = -delta
		}
		if delta <= windowMs {
			return true
		}
	}
	return false
}

func sampleKeep(indexInCategory int, ratio float64) bool {
	if ratio >= 1 {
		return true
	}
	if ratio <= 0 {
		return false
	}
	step := int(math.Round(1.0 / ratio))
	if step < 1 {
		step = 1
	}
	return indexInCategory%step == 0
}

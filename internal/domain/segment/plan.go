package segment

import (
	"math"
)

type PlanConfig struct {
	SegmentDurationSeconds          int
	MinVideoDurationForSplitSeconds int
	MaxSegments                     int
}

func NormalizePlanConfig(cfg PlanConfig) PlanConfig {
	if cfg.SegmentDurationSeconds <= 0 {
		cfg.SegmentDurationSeconds = DefaultSegmentDurationSeconds
	}
	if cfg.MinVideoDurationForSplitSeconds <= 0 {
		cfg.MinVideoDurationForSplitSeconds = DefaultMinVideoDurationForSplitSeconds
	}
	if cfg.MaxSegments <= 0 {
		cfg.MaxSegments = DefaultMaxSegments
	}
	return cfg
}

type Plan struct {
	Count                    int
	EffectiveDurationSeconds float64
	SkipSplit                bool // single segment reuses original object
}

// PlanSegments decides how many segments to produce for a video duration.
func PlanSegments(durationSeconds float64, cfg PlanConfig) Plan {
	cfg = NormalizePlanConfig(cfg)
	if durationSeconds <= 0 {
		durationSeconds = float64(cfg.SegmentDurationSeconds)
	}
	if durationSeconds < float64(cfg.MinVideoDurationForSplitSeconds) {
		return Plan{
			Count:                    1,
			EffectiveDurationSeconds: durationSeconds,
			SkipSplit:                true,
		}
	}

	count := int(math.Ceil(durationSeconds / float64(cfg.SegmentDurationSeconds)))
	effective := float64(cfg.SegmentDurationSeconds)
	if count > cfg.MaxSegments {
		count = cfg.MaxSegments
		effective = math.Ceil(durationSeconds / float64(count))
	}
	if count < 1 {
		count = 1
	}
	return Plan{
		Count:                    count,
		EffectiveDurationSeconds: effective,
		SkipSplit:                count == 1,
	}
}

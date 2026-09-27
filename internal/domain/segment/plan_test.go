package segment

import (
	"testing"
)

func TestPlanSegments_BelowThreshold_SingleNoSplit(t *testing.T) {
	p := PlanSegments(60, PlanConfig{})
	if p.Count != 1 || !p.SkipSplit {
		t.Fatalf("got %+v", p)
	}
}

func TestPlanSegments_FiveMinutes(t *testing.T) {
	p := PlanSegments(5*60, PlanConfig{})
	if p.Count != 3 || p.SkipSplit {
		t.Fatalf("got %+v want count=3", p)
	}
}

func TestPlanSegments_CapsAtMax(t *testing.T) {
	p := PlanSegments(2*60*60, PlanConfig{})
	if p.Count != DefaultMaxSegments {
		t.Fatalf("count=%d want %d", p.Count, DefaultMaxSegments)
	}
	if p.EffectiveDurationSeconds < 180 {
		t.Fatalf("effective duration too small: %v", p.EffectiveDurationSeconds)
	}
}

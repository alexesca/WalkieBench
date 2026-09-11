package benchmark

import (
	"testing"

	"walkiebench/telemetry"
)

func TestProfilesAreStableAndFullCoversV2(t *testing.T) {
	profiles := ProfileNames()
	if len(profiles) < 10 {
		t.Fatalf("profiles = %v", profiles)
	}
	full, ok := Profile("full")
	if !ok || !full.RequiresV2 {
		t.Fatal("full profile must require V2")
	}
	for _, category := range []string{"server", "permissions", "groups", "forums", "efficiency", "declarative", "transport", "browser"} {
		found := false
		for _, item := range full.Categories {
			if item == category {
				found = true
			}
		}
		if !found {
			t.Fatalf("full profile omitted %q", category)
		}
	}
}

func TestEfficiencyScoreRequiresHardGates(t *testing.T) {
	base := telemetry.Scorecard{
		Passed:     true,
		Efficiency: telemetry.Efficiency{Operations: 4, RoundTrips: 2, EstimatedTotalTokens: 100, SuccessfulActions: 4},
		Metrics:    map[string]float64{"time_to_first_collaboration_ms": 100},
	}
	if score := EfficiencyScore(base); score <= 0 || score > 100 {
		t.Fatalf("score = %v", score)
	}
	base.Reliability.MessageLoss = 1
	if score := EfficiencyScore(base); score != 0 {
		t.Fatalf("lossy run received score %v", score)
	}
}

func TestCollectorAggregatesCategoryFailures(t *testing.T) {
	c := telemetry.New()
	c.SetCategory("security", true, true, 100, "passed")
	c.SetCategory("security", false, true, 0, "failed")
	s := c.Snapshot()
	if len(s.Categories) != 1 || s.Categories[0].Passed || s.Categories[0].Score != 0 {
		t.Fatalf("categories = %#v", s.Categories)
	}
}

func TestScenarioLabelsAreDeterministicForSeed(t *testing.T) {
	cfg := Config{Profile: "core", Seed: 42}
	a := New(nil, cfg, telemetry.New(), nil)
	b := New(nil, cfg, telemetry.New(), nil)
	for i := 0; i < 5; i++ {
		if got, want := a.label("probe"), b.label("probe"); got != want {
			t.Fatalf("label %d = %q, want %q", i, got, want)
		}
	}
}

package telemetry

import (
	"testing"
	"time"
)

func TestCollectorProducesPercentilesAndEncryptionGate(t *testing.T) {
	c := New()
	c.BeginScenario("test")
	c.RegisterPlaintext("secret-marker")
	c.Observe("", "SendDM", 10*time.Millisecond, true, nil)
	c.Observe("", "ReceiveDMs", 20*time.Millisecond, true, nil)
	c.ObserveWire("SendDM", 20, 40, 3*time.Millisecond, nil, c.ContainsPlaintext([]byte(`{"content":"secret-marker"}`)))
	c.EndScenario(true, nil)
	s := c.Snapshot()
	if s.Passed || s.Reliability.EncryptionViolations != 1 {
		t.Fatalf("expected encryption gate failure: %+v", s)
	}
	if s.OperationSummary["SendDM"].P95MS != 10 {
		t.Fatalf("unexpected percentile: %+v", s.OperationSummary["SendDM"])
	}
	if len(s.Wire) != 1 || !s.Wire[0].PlaintextDetected {
		t.Fatalf("wire sample missing plaintext detection: %+v", s.Wire)
	}
}

func TestEstimateTokensIsDeterministic(t *testing.T) {
	for _, tc := range []struct {
		bytes int64
		want  int64
	}{
		{0, 0}, {1, 1}, {4, 1}, {5, 2}, {100, 25},
	} {
		if got := EstimateTokens(tc.bytes); got != tc.want {
			t.Fatalf("EstimateTokens(%d) = %d, want %d", tc.bytes, got, tc.want)
		}
	}
}

func TestPlaintextProbeIgnoresJSONFieldNames(t *testing.T) {
	c := New()
	c.RegisterPlaintext("collaboration")
	if c.ContainsPlaintext([]byte(`{"examples":["collaboration"]}`)) {
		t.Fatal("public help text was treated as protected content")
	}
	if c.ContainsPlaintext([]byte(`{"collaboration_topics":["ht1:encrypted"]}`)) {
		t.Fatal("field name was treated as plaintext content")
	}
	if !c.ContainsPlaintext([]byte(`{"collaboration_topics":["collaboration"]}`)) {
		t.Fatal("plaintext value was not detected")
	}
}

func TestAgentEfficiencyExcludesBenchmarkLoadTraffic(t *testing.T) {
	c := New()
	c.ObserveWire("setup", 1000, 2000, time.Millisecond, nil, false)
	c.Observe("", "setup", time.Millisecond, true, nil)
	c.BeginAgentJob("find-and-message")
	c.ObserveWire("DiscoverServers", 40, 80, time.Millisecond, nil, false)
	c.Observe("", "DiscoverServers", time.Millisecond, true, nil)
	c.ObserveWire("Heartbeat", 20, 20, time.Millisecond, nil, false)
	c.Observe("", "Heartbeat", time.Millisecond, true, nil)
	c.EndAgentJob(true)
	s := c.Snapshot()
	if s.Efficiency.Operations != 2 || s.Efficiency.RoundTrips != 2 {
		t.Fatalf("agent efficiency included non-job traffic: %+v", s.Efficiency)
	}
	if s.Efficiency.TotalWireBytes != 160 || s.Efficiency.MaintenanceOperations != 1 {
		t.Fatalf("agent efficiency counters = %+v", s.Efficiency)
	}
	if len(s.AgentJobs) != 1 || !s.AgentJobs[0].Passed || s.AgentJobs[0].EstimatedTokens != 40 {
		t.Fatalf("agent job = %+v", s.AgentJobs)
	}
	if s.Metrics["overall_wire_bytes"] != 3160 {
		t.Fatalf("overall wire bytes = %v", s.Metrics["overall_wire_bytes"])
	}
}

func TestCategoryFailureCannotRetainPassingScore(t *testing.T) {
	c := New()
	c.SetCategory("security", true, true, 100, "passed")
	c.SetCategory("security", false, true, 0, "failed")
	c.SetCategory("security", true, true, 100, "passed")
	s := c.Snapshot()
	if len(s.Categories) != 1 || s.Categories[0].Passed || s.Categories[0].Score != 0 {
		t.Fatalf("failed category retained passing state: %+v", s.Categories)
	}
}

func TestSemanticOutcomeCanFailCompletedAgentJob(t *testing.T) {
	c := New()
	c.BeginAgentJob("manifest")
	c.Observe("", "ApplyManifest", time.Millisecond, true, nil)
	c.EndAgentJob(true)
	c.FailAgentJob("manifest")
	s := c.Snapshot()
	if len(s.AgentJobs) != 1 || s.AgentJobs[0].Passed {
		t.Fatalf("semantic failure was not recorded: %+v", s.AgentJobs)
	}
}

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

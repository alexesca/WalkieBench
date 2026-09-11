package benchmark

import "walkiebench/telemetry"

// EfficiencyScore is transparent and bounded. It is meaningful only after
// correctness and security gates pass; lower effort for equivalent successful
// work scores higher, while omitted data cannot improve the result.
func EfficiencyScore(s telemetry.Scorecard) float64 {
	if !s.Passed || s.Reliability.MessageLoss > 0 || s.Reliability.OrderingViolations > 0 || s.Reliability.AccessControlViolations > 0 || s.Reliability.EncryptionViolations > 0 {
		return 0
	}
	e := s.Efficiency
	if e.SuccessfulActions == 0 {
		return 0
	}
	cap := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}
	ops := float64(e.Operations)
	if ops == 0 {
		ops = 1
	}
	rounds := float64(e.RoundTrips)
	if rounds == 0 {
		rounds = 1
	}
	tokens := float64(e.EstimatedTotalTokens)
	if tokens == 0 {
		tokens = 1
	}
	ms := s.Metrics["time_to_first_collaboration_ms"]
	if ms <= 0 {
		ms = 1
	}
	return 100 * (0.30*cap(40/ops) + 0.20*cap(10/rounds) + 0.25*cap(4000/tokens) + 0.25*cap(5000/ms))
}

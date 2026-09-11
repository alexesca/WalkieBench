package telemetry

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Operation struct {
	Scenario      string    `json:"scenario"`
	Name          string    `json:"name"`
	LatencyMS     float64   `json:"latency_ms"`
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"`
	RequestBytes  int64     `json:"request_bytes,omitempty"`
	ResponseBytes int64     `json:"response_bytes,omitempty"`
	At            time.Time `json:"at"`
}
type WireSample struct {
	Operation         string  `json:"operation"`
	RequestBytes      int64   `json:"request_bytes"`
	ResponseBytes     int64   `json:"response_bytes"`
	LatencyMS         float64 `json:"latency_ms"`
	Success           bool    `json:"success"`
	PlaintextDetected bool    `json:"plaintext_detected"`
}
type Metric struct {
	Count  int     `json:"count"`
	MinMS  float64 `json:"min_ms"`
	MaxMS  float64 `json:"max_ms"`
	MeanMS float64 `json:"mean_ms"`
	P50MS  float64 `json:"p50_ms"`
	P95MS  float64 `json:"p95_ms"`
	P99MS  float64 `json:"p99_ms"`
}
type Scenario struct {
	Name        string    `json:"name"`
	Category    string    `json:"category,omitempty"`
	Status      string    `json:"status"`
	HardGate    bool      `json:"hard_gate"`
	Passed      bool      `json:"passed"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	WallClockMS float64   `json:"wall_clock_ms"`
	Operations  int       `json:"operations"`
	Errors      []string  `json:"errors,omitempty"`
}
type Resource struct {
	At               time.Time `json:"at"`
	ClientAllocBytes uint64    `json:"client_alloc_bytes"`
	ClientHeapBytes  uint64    `json:"client_heap_bytes"`
	ClientCPUSeconds float64   `json:"client_cpu_seconds"`
	ServerRSSBytes   uint64    `json:"server_rss_bytes,omitempty"`
	ServerCPUSeconds float64   `json:"server_cpu_seconds,omitempty"`
	ServerReadBytes  uint64    `json:"server_read_bytes,omitempty"`
	ServerWriteBytes uint64    `json:"server_write_bytes,omitempty"`
	StorageBytes     int64     `json:"storage_bytes,omitempty"`
}
type Reliability struct {
	MessageLoss             int `json:"message_loss"`
	OrderingViolations      int `json:"ordering_violations"`
	FailedResumes           int `json:"failed_resumes"`
	AccessControlViolations int `json:"access_control_violations"`
	EncryptionViolations    int `json:"encryption_violations"`
}
type Efficiency struct {
	Operations                 int     `json:"operations"`
	RoundTrips                 int     `json:"round_trips"`
	RequestBytes               int64   `json:"request_bytes"`
	ResponseBytes              int64   `json:"response_bytes"`
	TotalWireBytes             int64   `json:"total_wire_bytes"`
	AgentInputBytes            int64   `json:"agent_input_bytes"`
	AgentOutputBytes           int64   `json:"agent_output_bytes"`
	EstimatedInputTokens       int64   `json:"estimated_input_tokens"`
	EstimatedOutputTokens      int64   `json:"estimated_output_tokens"`
	EstimatedTotalTokens       int64   `json:"estimated_total_tokens"`
	Retries                    int     `json:"retries"`
	MaintenanceOperations      int     `json:"maintenance_operations"`
	SuccessfulActions          int     `json:"successful_actions"`
	TimeToFirstCollaborationMS float64 `json:"time_to_first_collaboration_ms,omitempty"`
	EfficiencyScore            float64 `json:"efficiency_score,omitempty"`
}
type AgentJob struct {
	Name                  string  `json:"name"`
	Passed                bool    `json:"passed"`
	ElapsedMS             float64 `json:"elapsed_ms"`
	Operations            int     `json:"operations"`
	RoundTrips            int     `json:"round_trips"`
	RequestBytes          int64   `json:"request_bytes"`
	ResponseBytes         int64   `json:"response_bytes"`
	TotalWireBytes        int64   `json:"total_wire_bytes"`
	EstimatedTokens       int64   `json:"estimated_tokens"`
	Retries               int     `json:"retries"`
	FailedOperations      int     `json:"failed_operations"`
	MaintenanceOperations int     `json:"maintenance_operations"`
}
type Scorecard struct {
	Benchmark        string             `json:"benchmark"`
	Version          string             `json:"version"`
	StartedAt        time.Time          `json:"started_at"`
	FinishedAt       time.Time          `json:"finished_at"`
	Passed           bool               `json:"passed"`
	InvalidReasons   []string           `json:"invalid_reasons,omitempty"`
	Operations       []Operation        `json:"operations"`
	Wire             []WireSample       `json:"wire"`
	OperationSummary map[string]Metric  `json:"operation_summary"`
	Scenarios        []Scenario         `json:"scenarios"`
	Metrics          map[string]float64 `json:"metrics"`
	Reliability      Reliability        `json:"reliability"`
	Resources        []Resource         `json:"resources,omitempty"`
	Profile          string             `json:"profile,omitempty"`
	Seed             int64              `json:"seed"`
	Efficiency       Efficiency         `json:"efficiency"`
	AgentJobs        []AgentJob         `json:"agent_jobs,omitempty"`
	Categories       []Category         `json:"categories,omitempty"`
}

type Category struct {
	Name     string  `json:"name"`
	Passed   bool    `json:"passed"`
	HardGate bool    `json:"hard_gate"`
	Score    float64 `json:"score"`
	Details  string  `json:"details,omitempty"`
}

type Collector struct {
	mu            sync.Mutex
	score         Scorecard
	values        map[string][]float64
	plaintext     []string
	scenario      string
	scenarioStart time.Time
	scenarioOps   int
	activeJob     int
	jobStarted    time.Time
	overallWire   Efficiency
}

func New() *Collector {
	return &Collector{score: Scorecard{Benchmark: "WalkieBench", Version: "2.0", StartedAt: time.Now(), OperationSummary: map[string]Metric{}, Metrics: map[string]float64{}}, values: map[string][]float64{}, activeJob: -1}
}
func (c *Collector) BeginAgentJob(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.score.AgentJobs = append(c.score.AgentJobs, AgentJob{Name: name})
	c.activeJob = len(c.score.AgentJobs) - 1
	c.jobStarted = time.Now()
}
func (c *Collector) EndAgentJob(passed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.activeJob < 0 {
		return
	}
	job := &c.score.AgentJobs[c.activeJob]
	job.Passed = passed
	job.ElapsedMS = float64(time.Since(c.jobStarted).Microseconds()) / 1000
	c.activeJob = -1
}
func (c *Collector) SetRunMetadata(profile string, seed int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.score.Profile = profile
	c.score.Seed = seed
}
func (c *Collector) SetEfficiencyScore(score float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.score.Efficiency.EfficiencyScore = score
}
func (c *Collector) SetTimeToFirstCollaboration(ms float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.score.Efficiency.TimeToFirstCollaborationMS = ms
}
func (c *Collector) SetCategory(name string, passed, hardGate bool, score float64, details string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.score.Categories {
		if c.score.Categories[i].Name == name {
			c.score.Categories[i].Passed = c.score.Categories[i].Passed && passed
			c.score.Categories[i].HardGate = c.score.Categories[i].HardGate || hardGate
			if score > c.score.Categories[i].Score {
				c.score.Categories[i].Score = score
			}
			if details != "passed" {
				c.score.Categories[i].Details = details
			}
			return
		}
	}
	c.score.Categories = append(c.score.Categories, Category{Name: name, Passed: passed, HardGate: hardGate, Score: score, Details: details})
}
func (c *Collector) BeginScenario(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scenario = name
	c.scenarioStart = time.Now()
	c.scenarioOps = 0
}
func (c *Collector) EndScenario(passed bool, errs []error) {
	status := "passed"
	if !passed {
		status = "failed"
	}
	c.EndScenarioStatus(status, false, errs)
}
func (c *Collector) EndScenarioStatus(status string, hardGate bool, errs []error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	s := Scenario{Name: c.scenario, Status: status, Passed: status == "passed", HardGate: hardGate, StartedAt: c.scenarioStart, FinishedAt: now, WallClockMS: float64(now.Sub(c.scenarioStart).Microseconds()) / 1000, Operations: c.scenarioOps}
	for _, e := range errs {
		s.Errors = append(s.Errors, e.Error())
	}
	c.score.Scenarios = append(c.score.Scenarios, s)
}
func (c *Collector) Observe(scenario, name string, d time.Duration, ok bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if scenario != "" {
		c.scenario = scenario
	}
	c.scenarioOps++
	o := Operation{Scenario: c.scenario, Name: name, LatencyMS: float64(d.Microseconds()) / 1000, Success: ok, At: time.Now()}
	if err != nil {
		o.Error = err.Error()
	}
	c.score.Operations = append(c.score.Operations, o)
	c.values[name] = append(c.values[name], o.LatencyMS)
	if c.activeJob >= 0 {
		job := &c.score.AgentJobs[c.activeJob]
		job.Operations++
		if !ok {
			job.FailedOperations++
		}
		switch name {
		case "Heartbeat", "Resume", "WaitForEvents", "Sync":
			job.MaintenanceOperations++
		}
		if strings.Contains(strings.ToLower(name), "retry") {
			job.Retries++
		}
	}
}
func (c *Collector) RegisterPlaintext(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if value != "" {
		c.plaintext = append(c.plaintext, value)
	}
}
func (c *Collector) ContainsPlaintext(body []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	var value any
	if json.Unmarshal(body, &value) == nil {
		return containsPlaintextValue(value, c.plaintext, false)
	}
	for _, v := range c.plaintext {
		if bytes.Contains(body, []byte(v)) {
			return true
		}
	}
	return false
}

func containsPlaintextValue(value any, probes []string, protected bool) bool {
	switch x := value.(type) {
	case map[string]any:
		for key, child := range x {
			if containsPlaintextValue(child, probes, protectedWireKey(key)) {
				return true
			}
		}
	case []any:
		for _, child := range x {
			if containsPlaintextValue(child, probes, protected) {
				return true
			}
		}
	case string:
		if protected {
			for _, probe := range probes {
				// Probes are the exact values written by the benchmark. Exact
				// matching prevents a common metadata word such as
				// "collaboration" in public help text from matching a probe
				// registered for a protected capability value.
				if probe != "" && x == probe {
					return true
				}
			}
		}
	}
	return false
}

// protectedWireKey mirrors the contract's content-bearing fields. Public
// routing metadata, query filters, help text, and JSON object keys must not
// turn an unrelated occurrence of a probe into an encryption violation.
func protectedWireKey(key string) bool {
	switch key {
	case "content", "title", "summary", "bio", "description", "purpose", "topics", "topic", "tags", "rules", "current_work", "limitations", "reason", "interests", "capabilities", "collaboration_topics":
		return true
	default:
		return false
	}
}
func (c *Collector) ObserveWire(op string, req, resp int64, d time.Duration, err error, plaintext bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.score.Wire = append(c.score.Wire, WireSample{Operation: op, RequestBytes: req, ResponseBytes: resp, LatencyMS: float64(d.Microseconds()) / 1000, Success: err == nil, PlaintextDetected: plaintext})
	c.overallWire.RoundTrips++
	c.overallWire.RequestBytes += req
	c.overallWire.ResponseBytes += resp
	c.overallWire.TotalWireBytes += req + resp
	if c.activeJob >= 0 {
		job := &c.score.AgentJobs[c.activeJob]
		job.RoundTrips++
		job.RequestBytes += req
		job.ResponseBytes += resp
		job.TotalWireBytes += req + resp
		job.EstimatedTokens += EstimateTokens(req) + EstimateTokens(resp)
	}
	if plaintext {
		c.score.Reliability.EncryptionViolations++
		c.score.InvalidReasons = append(c.score.InvalidReasons, "plaintext content observed on contract wire")
	}
}
func (c *Collector) Metric(name string, value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.score.Metrics[name] = value
}
func (c *Collector) AddReliability(fn func(*Reliability)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn(&c.score.Reliability)
}
func (c *Collector) AddInvalid(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.score.InvalidReasons = append(c.score.InvalidReasons, reason)
}
func (c *Collector) AddResource(r Resource) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.score.Resources = append(c.score.Resources, r)
}
func (c *Collector) Snapshot() Scorecard {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.score
	s.FinishedAt = time.Now()
	s.OperationSummary = map[string]Metric{}
	successful := 0
	for _, o := range s.Operations {
		if o.Success {
			successful++
		}
	}
	s.Metrics["successful_operations"] = float64(successful)
	s.Metrics["overall_round_trips"] = float64(c.overallWire.RoundTrips)
	s.Metrics["overall_wire_bytes"] = float64(c.overallWire.TotalWireBytes)
	s.Efficiency = Efficiency{EfficiencyScore: s.Efficiency.EfficiencyScore, TimeToFirstCollaborationMS: s.Efficiency.TimeToFirstCollaborationMS}
	for _, job := range s.AgentJobs {
		s.Efficiency.Operations += job.Operations
		s.Efficiency.RoundTrips += job.RoundTrips
		s.Efficiency.RequestBytes += job.RequestBytes
		s.Efficiency.ResponseBytes += job.ResponseBytes
		s.Efficiency.TotalWireBytes += job.TotalWireBytes
		s.Efficiency.EstimatedTotalTokens += job.EstimatedTokens
		s.Efficiency.Retries += job.Retries
		s.Efficiency.MaintenanceOperations += job.MaintenanceOperations
		s.Efficiency.SuccessfulActions += job.Operations - job.FailedOperations
	}
	s.Efficiency.AgentInputBytes = s.Efficiency.RequestBytes
	s.Efficiency.AgentOutputBytes = s.Efficiency.ResponseBytes
	s.Efficiency.EstimatedInputTokens = EstimateTokens(s.Efficiency.RequestBytes)
	s.Efficiency.EstimatedOutputTokens = EstimateTokens(s.Efficiency.ResponseBytes)
	if s.Efficiency.SuccessfulActions > 0 {
		s.Metrics["bytes_per_successful_action"] = float64(s.Efficiency.TotalWireBytes) / float64(s.Efficiency.SuccessfulActions)
		s.Metrics["tokens_per_successful_action"] = float64(s.Efficiency.EstimatedTotalTokens) / float64(s.Efficiency.SuccessfulActions)
	}
	if v, ok := s.Metrics["run_wall_clock_ms"]; ok && successful > 0 {
		s.Metrics["resource_cost_ms_per_successful_operation"] = v / float64(successful)
	}
	for n, vs := range c.values {
		s.OperationSummary[n] = summarize(vs)
	}
	s.Passed = len(s.InvalidReasons) == 0
	for _, v := range s.Scenarios {
		if !v.Passed && (v.Status != "unsupported" || v.HardGate) {
			s.Passed = false
		}
	}
	return s
}
func EstimateTokens(bytes int64) int64 {
	if bytes <= 0 {
		return 0
	}
	return (bytes + 3) / 4
}
func (c *Collector) WriteJSON(path string) error {
	b, e := json.MarshalIndent(c.Snapshot(), "", "  ")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0750); e != nil {
		return e
	}
	return os.WriteFile(path, b, 0600)
}
func summarize(v []float64) Metric {
	m := Metric{Count: len(v)}
	if len(v) == 0 {
		return m
	}
	cp := append([]float64(nil), v...)
	for i := 1; i < len(cp); i++ {
		x := cp[i]
		j := i - 1
		for j >= 0 && cp[j] > x {
			cp[j+1] = cp[j]
			j--
		}
		cp[j+1] = x
	}
	sum := 0.0
	for _, x := range cp {
		sum += x
	}
	m.MinMS = cp[0]
	m.MaxMS = cp[len(cp)-1]
	m.MeanMS = sum / float64(len(cp))
	m.P50MS = cp[(len(cp)-1)*50/100]
	m.P95MS = cp[(len(cp)-1)*95/100]
	m.P99MS = cp[(len(cp)-1)*99/100]
	return m
}

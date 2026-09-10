package telemetry

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
}

type Collector struct {
	mu            sync.Mutex
	score         Scorecard
	values        map[string][]float64
	plaintext     []string
	scenario      string
	scenarioStart time.Time
	scenarioOps   int
}

func New() *Collector {
	return &Collector{score: Scorecard{Benchmark: "WalkieBench", Version: "1.0", StartedAt: time.Now(), OperationSummary: map[string]Metric{}, Metrics: map[string]float64{}}, values: map[string][]float64{}}
}
func (c *Collector) BeginScenario(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scenario = name
	c.scenarioStart = time.Now()
	c.scenarioOps = 0
}
func (c *Collector) EndScenario(passed bool, errs []error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	s := Scenario{Name: c.scenario, Passed: passed, StartedAt: c.scenarioStart, FinishedAt: now, WallClockMS: float64(now.Sub(c.scenarioStart).Microseconds()) / 1000, Operations: c.scenarioOps}
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
	for _, v := range c.plaintext {
		if bytes.Contains(body, []byte(v)) {
			return true
		}
	}
	return false
}
func (c *Collector) ObserveWire(op string, req, resp int64, d time.Duration, err error, plaintext bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.score.Wire = append(c.score.Wire, WireSample{Operation: op, RequestBytes: req, ResponseBytes: resp, LatencyMS: float64(d.Microseconds()) / 1000, Success: err == nil, PlaintextDetected: plaintext})
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
	if v, ok := s.Metrics["run_wall_clock_ms"]; ok && successful > 0 {
		s.Metrics["resource_cost_ms_per_successful_operation"] = v / float64(successful)
	}
	for n, vs := range c.values {
		s.OperationSummary[n] = summarize(vs)
	}
	s.Passed = len(s.InvalidReasons) == 0
	for _, v := range s.Scenarios {
		if !v.Passed {
			s.Passed = false
		}
	}
	return s
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

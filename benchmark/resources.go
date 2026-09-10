package benchmark

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"walkiebench/telemetry"
)

func startResourceSampler(ctx context.Context, c Config, t *telemetry.Collector) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	sample := func() {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		r := telemetry.Resource{At: time.Now(), ClientAllocBytes: m.TotalAlloc, ClientHeapBytes: m.HeapAlloc, ClientCPUSeconds: procCPU(os.Getpid())}
		if c.ResourceCommand != "" {
			if b, e := exec.CommandContext(ctx, c.ResourceCommand).Output(); e == nil {
				var ext telemetry.Resource
				if json.Unmarshal(b, &ext) == nil {
					r.ServerRSSBytes = ext.ServerRSSBytes
					r.ServerCPUSeconds = ext.ServerCPUSeconds
					r.ServerReadBytes = ext.ServerReadBytes
					r.ServerWriteBytes = ext.ServerWriteBytes
					r.StorageBytes = ext.StorageBytes
				}
			}
		}
		t.AddResource(r)
	}
	sample()
	go func() {
		defer close(done)
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				sample()
			case <-stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() { close(stop); <-done; sample() }
}
func procCPU(pid int) float64 {
	b, e := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if e != nil {
		return 0
	}
	line := string(b)
	i := strings.LastIndex(line, ") ")
	if i < 0 {
		return 0
	}
	p := strings.Fields(line[i+2:])
	if len(p) < 13 {
		return 0
	}
	u, _ := strconv.ParseFloat(p[11], 64)
	s, _ := strconv.ParseFloat(p[12], 64)
	return (u + s) / 100
}

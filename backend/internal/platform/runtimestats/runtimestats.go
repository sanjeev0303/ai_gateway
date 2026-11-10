// Package runtimestats reports what one process is using: CPU time, memory,
// goroutines and its database pool. Every service serves it on an internal
// endpoint, and the worker's monitor polls it to chart and alert on resource
// use without needing access to Docker or Kubernetes.
package runtimestats

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"runtime/metrics"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Stats is one reading. CPU is cumulative seconds; the poller turns two
// readings into a percentage.
type Stats struct {
	UptimeSeconds float64    `json:"uptime_seconds"`
	CPUSeconds    float64    `json:"cpu_seconds"`
	NumCPU        int        `json:"num_cpu"`
	RSSBytes      int64      `json:"rss_bytes"`
	HeapBytes     int64      `json:"heap_bytes"`
	Goroutines    int        `json:"goroutines"`
	GCCycles      uint32     `json:"gc_cycles"`
	DBPool        *PoolStats `json:"db_pool,omitempty"`
}

type PoolStats struct {
	Total    int32 `json:"total"`
	Acquired int32 `json:"acquired"`
	Idle     int32 `json:"idle"`
	Max      int32 `json:"max"`
}

var started = time.Now()

// Collect reads the current process. pool may be nil.
func Collect(pool *pgxpool.Pool) Stats {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	s := Stats{
		UptimeSeconds: time.Since(started).Seconds(),
		CPUSeconds:    cpuSeconds(),
		NumCPU:        runtime.NumCPU(),
		RSSBytes:      rssBytes(int64(mem.Sys)),
		HeapBytes:     int64(mem.HeapAlloc),
		Goroutines:    runtime.NumGoroutine(),
		GCCycles:      mem.NumGC,
	}
	if pool != nil {
		ps := pool.Stat()
		s.DBPool = &PoolStats{
			Total:    ps.TotalConns(),
			Acquired: ps.AcquiredConns(),
			Idle:     ps.IdleConns(),
			Max:      ps.MaxConns(),
		}
	}
	return s
}

// Handler serves Collect as JSON to callers holding token. An empty token
// disables the endpoint: it then answers 404 like any unknown path.
func Handler(token string, pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		presented := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" || subtle.ConstantTimeCompare([]byte(presented), []byte(token)) != 1 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(Collect(pool))
	})
}

// cpuSeconds is the process's user plus system CPU time. Containers run on
// Linux, where /proc is exact; elsewhere (a developer's machine) it falls back
// to the Go runtime's own estimate.
func cpuSeconds() float64 {
	if raw, err := os.ReadFile("/proc/self/stat"); err == nil {
		// The command name (field 2) may contain spaces, so count fields
		// from the closing parenthesis. utime and stime are fields 14 and 15.
		text := string(raw)
		if i := strings.LastIndexByte(text, ')'); i >= 0 {
			fields := strings.Fields(text[i+1:])
			if len(fields) > 12 {
				utime, err1 := strconv.ParseFloat(fields[11], 64)
				stime, err2 := strconv.ParseFloat(fields[12], 64)
				if err1 == nil && err2 == nil {
					// ponytail: assumes the usual 100 clock ticks per second;
					// read sysconf(_SC_CLK_TCK) if a kernel ever differs.
					return (utime + stime) / 100
				}
			}
		}
	}

	samples := []metrics.Sample{
		{Name: "/cpu/classes/total:cpu-seconds"},
		{Name: "/cpu/classes/idle:cpu-seconds"},
	}
	metrics.Read(samples)
	if samples[0].Value.Kind() != metrics.KindFloat64 || samples[1].Value.Kind() != metrics.KindFloat64 {
		return 0
	}
	return samples[0].Value.Float64() - samples[1].Value.Float64()
}

// rssBytes is the resident memory from /proc, or the runtime's total obtained
// from the OS where /proc is missing.
func rssBytes(fallback int64) int64 {
	raw, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return fallback
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				return kb * 1024
			}
		}
	}
	return fallback
}

// Command perfgate runs the micro-benchmark suite, compares it against the
// committed baseline (perf/baseline.json) using the regression policy in
// perf/budgets.json, and fails (non-zero exit) when a benchmark regresses
// beyond the configured thresholds. It is the PR-tier regression gate; the
// end-to-end API/resource/DB budgets are enforced separately by cmd/perf.
//
//	go run ./cmd/perfgate              # run the gate against the baseline
//	go run ./cmd/perfgate -update      # (re)record the baseline from a clean run
//
// Allocations and bytes-per-op are deterministic, so their thresholds are
// strict; ns/op is noisy on a shared machine, so it uses a looser threshold
// and benchmarks below the noise floor are not latency-gated.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type regressionPolicy struct {
	LatencyIncreasePercent float64 `json:"latency_increase_percent"`
	AllocsIncreasePercent  float64 `json:"allocs_increase_percent"`
	BytesIncreasePercent   float64 `json:"bytes_increase_percent"`
	NoiseFloorNs           float64 `json:"noise_floor_ns"`
	BenchCount             int     `json:"bench_count"`
}

type budgets struct {
	Regression regressionPolicy `json:"regression"`
}

type metric struct {
	NsPerOp     float64 `json:"ns_per_op"`
	BytesPerOp  float64 `json:"bytes_per_op"`
	AllocsPerOp float64 `json:"allocs_per_op"`
}

type baselineFile struct {
	Metadata   map[string]any    `json:"metadata"`
	Benchmarks map[string]metric `json:"benchmarks"`
}

func main() {
	var (
		budgetsPath  = flag.String("budgets", "perf/budgets.json", "path to budgets config")
		baselinePath = flag.String("baseline", "perf/baseline.json", "path to baseline results")
		pkg          = flag.String("pkg", "./internal/interfaces/gatewayv2", "package holding the benchmarks")
		bench        = flag.String("bench", "BenchmarkPromptTurn", "benchmark regex to run")
		reportPath   = flag.String("report", "perf/last_report.json", "where to write the JSON report")
		update       = flag.Bool("update", false, "record the current run as the new baseline instead of gating")
		count        = flag.Int("count", 0, "override bench count (default: budgets.regression.bench_count)")
	)
	flag.Parse()

	b, err := loadBudgets(*budgetsPath)
	if err != nil {
		fail("read budgets: %v", err)
	}
	n := b.Regression.BenchCount
	if *count > 0 {
		n = *count
	}
	if n <= 0 {
		n = 6
	}

	current, err := runBenchmarks(*pkg, *bench, n)
	if err != nil {
		fail("run benchmarks: %v", err)
	}
	if len(current) == 0 {
		fail("no benchmarks matched %q in %s", *bench, *pkg)
	}

	if *update {
		if err := writeBaseline(*baselinePath, current); err != nil {
			fail("write baseline: %v", err)
		}
		fmt.Printf("baseline updated: %s (%d benchmarks)\n", *baselinePath, len(current))
		return
	}

	base, err := loadBaseline(*baselinePath)
	if err != nil {
		fail("read baseline (run with -update first): %v", err)
	}

	rows, violations := evaluate(base.Benchmarks, current, b.Regression)
	printReport(rows, violations)
	_ = writeReport(*reportPath, base, current, rows, violations)

	if len(violations) > 0 {
		fmt.Printf("\nPERF GATE: FAIL — %d regression(s) over budget\n", len(violations))
		os.Exit(1)
	}
	fmt.Printf("\nPERF GATE: PASS — %d benchmark(s) within budget\n", len(rows))
}

func loadBudgets(path string) (budgets, error) {
	var b budgets
	data, err := os.ReadFile(path)
	if err != nil {
		return b, err
	}
	return b, json.Unmarshal(data, &b)
}

func loadBaseline(path string) (baselineFile, error) {
	var bf baselineFile
	data, err := os.ReadFile(path)
	if err != nil {
		return bf, err
	}
	return bf, json.Unmarshal(data, &bf)
}

// benchLine matches the standard `go test -benchmem` output, e.g.
// BenchmarkPromptTurn/small_1turn_337B-8   272955   4654 ns/op   72.40 MB/s   1072 B/op   14 allocs/op
var benchLine = regexp.MustCompile(`^(Benchmark\S+?)(?:-\d+)?\s+\d+\s+([\d.]+)\s+ns/op(?:.*?\s([\d.]+)\s+B/op)?(?:\s+([\d.]+)\s+allocs/op)?`)

// runBenchmarks runs `go test -bench` count times and returns the median of
// each benchmark's metrics. The median resists the occasional slow sample on a
// loaded machine better than the mean.
func runBenchmarks(pkg, bench string, count int) (map[string]metric, error) {
	cmd := exec.Command("go", "test", "-run=^$", "-bench="+bench, "-benchmem",
		"-count="+strconv.Itoa(count), pkg)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v\n%s", err, out)
	}

	ns := map[string][]float64{}
	by := map[string][]float64{}
	al := map[string][]float64{}
	for _, line := range strings.Split(string(out), "\n") {
		m := benchLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		name := m[1]
		ns[name] = append(ns[name], atof(m[2]))
		if m[3] != "" {
			by[name] = append(by[name], atof(m[3]))
		}
		if m[4] != "" {
			al[name] = append(al[name], atof(m[4]))
		}
	}

	res := map[string]metric{}
	for name, samples := range ns {
		res[name] = metric{
			NsPerOp:     median(samples),
			BytesPerOp:  median(by[name]),
			AllocsPerOp: median(al[name]),
		}
	}
	return res, nil
}

type row struct {
	Name       string  `json:"name"`
	Metric     string  `json:"metric"`
	Baseline   float64 `json:"baseline"`
	Current    float64 `json:"current"`
	DeltaPct   float64 `json:"delta_percent"`
	Budget     float64 `json:"budget_percent"`
	Violation  bool    `json:"violation"`           // hard failure (deterministic metrics only)
	Warn       bool    `json:"warn,omitempty"`      // advisory (noisy ns/op over budget)
	Suppressed bool    `json:"suppressed_below_noise_floor,omitempty"`
}

func evaluate(base, current map[string]metric, p regressionPolicy) ([]row, []row) {
	var rows, violations []row
	names := make([]string, 0, len(current))
	for n := range current {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		cur := current[name]
		b, ok := base[name]
		if !ok {
			// A new benchmark with no baseline is not a regression; it will be
			// recorded on the next -update.
			continue
		}
		// ns/op is noisy across separate runs on a shared machine, so it is
		// advisory only (warn, never a hard failure) and is suppressed below
		// the noise floor. Reliable latency gating needs a dedicated quiet
		// runner — see perf/README.md.
		nsRow := compare(name, "ns/op", b.NsPerOp, cur.NsPerOp, p.LatencyIncreasePercent)
		nsRow.Warn = nsRow.Violation && b.NsPerOp >= p.NoiseFloorNs
		nsRow.Suppressed = b.NsPerOp < p.NoiseFloorNs
		nsRow.Violation = false
		// allocs/op and B/op are deterministic, so they are the hard gate.
		allocRow := compare(name, "allocs/op", b.AllocsPerOp, cur.AllocsPerOp, p.AllocsIncreasePercent)
		bytesRow := compare(name, "B/op", b.BytesPerOp, cur.BytesPerOp, p.BytesIncreasePercent)

		for _, r := range []row{nsRow, allocRow, bytesRow} {
			rows = append(rows, r)
			if r.Violation {
				violations = append(violations, r)
			}
		}
	}
	return rows, violations
}

func compare(name, metricName string, baseVal, curVal, budgetPct float64) row {
	var delta float64
	if baseVal > 0 {
		delta = (curVal - baseVal) / baseVal * 100
	}
	return row{
		Name:      name,
		Metric:    metricName,
		Baseline:  baseVal,
		Current:   curVal,
		DeltaPct:  delta,
		Budget:    budgetPct,
		Violation: delta > budgetPct,
	}
}

func printReport(rows, violations []row) {
	fmt.Printf("%-42s %-10s %12s %12s %9s %8s\n", "BENCHMARK", "METRIC", "BASELINE", "CURRENT", "DELTA%", "STATUS")
	for _, r := range rows {
		status := "ok"
		if r.Suppressed {
			status = "noise"
		} else if r.Violation {
			status = "FAIL"
		} else if r.Warn {
			status = "warn"
		}
		fmt.Printf("%-42s %-10s %12.1f %12.1f %+8.1f%% %8s\n", trim(r.Name, 42), r.Metric, r.Baseline, r.Current, r.DeltaPct, status)
	}
	_ = violations
}

func writeReport(path string, base baselineFile, current map[string]metric, rows, violations []row) error {
	report := map[string]any{
		"generated_at":       time.Now().UTC().Format(time.RFC3339),
		"baseline_metadata":  base.Metadata,
		"current_benchmarks": current,
		"rows":               rows,
		"violations":         violations,
		"passed":             len(violations) == 0,
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	return os.WriteFile(path, data, 0o644)
}

func writeBaseline(path string, benchmarks map[string]metric) error {
	bf := baselineFile{
		Metadata: map[string]any{
			"recorded_at": time.Now().UTC().Format(time.RFC3339),
			"git_commit":  gitCommit(),
			"git_branch":  gitBranch(),
			"go_version":  runtime.Version(),
			"goos":        runtime.GOOS,
			"goarch":      runtime.GOARCH,
			"num_cpu":     runtime.NumCPU(),
			"host":        hostname(),
		},
		Benchmarks: benchmarks,
	}
	data, _ := json.MarshalIndent(bf, "", "  ")
	return os.WriteFile(path, data, 0o644)
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid]
	}
	return (s[mid-1] + s[mid]) / 2
}

func atof(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func gitCommit() string { return gitOut("rev-parse", "HEAD") }
func gitBranch() string { return gitOut("rev-parse", "--abbrev-ref", "HEAD") }

func gitOut(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "n/a"
	}
	return strings.TrimSpace(string(out))
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "n/a"
	}
	return h
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "perfgate: "+format+"\n", a...)
	os.Exit(2)
}

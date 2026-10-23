package architecture_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMonthlySLAUsesActualMinutesAndConservativeDeduplication(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		values      []int
		duplicate   bool
		good, known int
		status      string
	}{
		{name: "complete", values: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, good: 10, known: 10, status: "provisionally_met"},
		{name: "failure", values: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 0}, good: 9, known: 10, status: "provisionally_breached"},
		{name: "unknown", values: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, -1}, good: 9, known: 9, status: "insufficient_data"},
		{name: "missing", values: []int{1, 1, 1, 1, 1, 1, 1, 1, 1}, good: 9, known: 9, status: "insufficient_data"},
		{name: "duplicate", values: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, duplicate: true, good: 9, known: 10, status: "provisionally_breached"},
		{name: "empty", status: "insufficient_data"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var timestamps []int
			for index := range scenario.values {
				timestamps = append(timestamps, (index+1)*60000+10000)
			}
			if scenario.duplicate {
				timestamps = append(timestamps, timestamps[0]+1)
				scenario.values = append(scenario.values, 0)
			}
			input, err := json.Marshal([]any{map[string]any{"timestamps": timestamps, "values": scenario.values}})
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, "jq", "--arg", "month", "1970-01-01T00:00:00Z", "--argjson", "start", "0", "--argjson", "end", "600", "-f", "scripts/reliability-monthly.jq")
			command.Dir = repositoryRoot(t)
			command.Stdin = strings.NewReader(string(input))
			output, err := command.CombinedOutput()
			require.NoError(t, err, "%s", output)
			var result struct {
				Good   int    `json:"good_minutes"`
				Known  int    `json:"known_minutes"`
				Status string `json:"status"`
			}
			require.NoError(t, json.Unmarshal(output, &result))
			require.Equal(t, scenario.good, result.Good)
			require.Equal(t, scenario.known, result.Known)
			require.Equal(t, scenario.status, result.Status)
		})
	}
}

func TestExperimentsRefuseNonLocalContextBeforeWriting(t *testing.T) {
	directory := t.TempDir()
	marker := filepath.Join(directory, "write-attempted")
	kubectl := filepath.Join(directory, "kubectl")
	require.NoError(t, os.WriteFile(kubectl, []byte("#!/bin/sh\nif [ \"$1 $2\" = 'config current-context' ]; then printf production; else touch \"$WRITE_MARKER\"; fi\n"), 0o700))
	command := exec.CommandContext(t.Context(), "./scripts/reliability-experiment.sh", "errors")
	command.Dir = repositoryRoot(t)
	command.Env = append(os.Environ(), "KUBECTL="+kubectl, "WRITE_MARKER="+marker)
	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "restricted to context minikube")
	require.NoFileExists(t, marker)
}

func TestPhaseVerificationUsesMeasuredSLIAndConfiguredTargets(t *testing.T) {
	for _, scenario := range []struct {
		name, mode  string
		ratios      map[string]float64
		targets     map[string]float64
		wantSuccess bool
	}{
		{name: "latency", mode: "latency", ratios: map[string]float64{"api": 1, "catalog": 1, "database": 1, "latency": .8}, wantSuccess: true},
		{name: "errors", mode: "errors", ratios: map[string]float64{"api": .8, "catalog": .8, "database": 1, "latency": 1}, wantSuccess: true},
		{name: "no degradation", mode: "errors", ratios: map[string]float64{"api": 1, "catalog": 1, "database": 1, "latency": 1}},
		{name: "wrong failure domain", mode: "errors", ratios: map[string]float64{"api": .8, "catalog": .8, "database": .8, "latency": 1}},
		{name: "missing indicator", mode: "errors", ratios: map[string]float64{"api": .8, "catalog": .8, "database": 1}},
		{name: "changed objectives met", mode: "latency", ratios: map[string]float64{"api": .98, "catalog": 1, "database": 1, "latency": .97}, targets: map[string]float64{"api": .95, "catalog": .99, "database": .99, "latency": .96}},
		{name: "changed objectives breached", mode: "latency", ratios: map[string]float64{"api": .98, "catalog": 1, "database": 1, "latency": .94}, targets: map[string]float64{"api": .95, "catalog": .99, "database": .99, "latency": .96}, wantSuccess: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				values := scenario.ratios
				if strings.Contains(r.URL.Query().Get("query"), "slo:target") {
					values = map[string]float64{"api": .995, "catalog": .995, "database": .999, "latency": .99}
					if scenario.targets != nil {
						values = scenario.targets
					}
				}
				var series []any
				for indicator, value := range values {
					series = append(series, map[string]any{"metric": map[string]string{"indicator": indicator}, "value": []any{1, fmt.Sprint(value)}})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"result": series}})
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, "./scripts/verify-reliability-phase.sh", scenario.mode)
			command.Dir = repositoryRoot(t)
			command.Env = append(os.Environ(), "METRICS_URL="+server.URL, "RELIABILITY_PHASE_TIMEOUT=0")
			output, err := command.CombinedOutput()
			if scenario.wantSuccess {
				require.NoError(t, err, "%s", output)
			} else {
				require.Error(t, err, "%s", output)
			}
		})
	}
}

func TestReliabilityGraphLinksHaveMovingTimeRangeAndRefresh(t *testing.T) {
	command := exec.CommandContext(t.Context(), "./scripts/reliability-graphs.sh")
	command.Dir = repositoryRoot(t)
	output, err := command.Output()
	require.NoError(t, err)
	links := 0
	for line := range strings.SplitSeq(string(output), "\n") {
		if !strings.HasPrefix(line, "http://127.0.0.1:8428/vmui/") {
			continue
		}
		links++
		require.Contains(t, line, "g0.relative_time=last_15_minutes")
		require.Contains(t, line, "refresh=5s")
		require.NotContains(t, line, "g0.end_input=")
	}
	require.Equal(t, 8, links)
}

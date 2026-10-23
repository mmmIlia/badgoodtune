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

func TestMetricsCheckRejectsIncompleteOrStaleData(t *testing.T) {
	for _, scenario := range []string{"healthy", "missing metric", "duplicate metric", "stale pod", "database down"} {
		t.Run(scenario, func(t *testing.T) {
			kubectl := filepath.Join(t.TempDir(), "kubectl")
			require.NoError(t, os.WriteFile(kubectl, []byte(`#!/bin/sh
case "$2" in
  deployment/badgoodtune-backend) printf 2 ;;
  configmap/badgoodtune-config) printf catalog_test ;;
  pods) printf '%s' '{"items":[{"metadata":{"name":"backend-a"},"status":{"phase":"Running"}},{"metadata":{"name":"backend-b"},"status":{"phase":"Running"}}]}' ;;
  *) exit 2 ;;
esac
`), 0o700))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var result any
				if r.URL.Path == "/api/v1/targets" {
					pods := []string{"backend-a", "backend-b"}
					if scenario == "stale pod" {
						pods[1] = "backend-old"
					}
					result = map[string]any{"activeTargets": []any{
						map[string]any{"labels": map[string]string{"job": "badgoodtune-backend", "pod": pods[0]}, "health": "up"},
						map[string]any{"labels": map[string]string{"job": "badgoodtune-backend", "pod": pods[1]}, "health": "up"},
						map[string]any{"labels": map[string]string{"job": "postgres", "pod": "exporter"}, "health": "up"},
					}}
				} else {
					query := r.URL.Query().Get("query")
					result = map[string]any{"result": metricSeriesForCheck(t, scenario, query)}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": result})
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, "./scripts/check-metrics.sh")
			command.Dir = repositoryRoot(t)
			command.Env = append(os.Environ(), "KUBECTL="+kubectl, "METRICS_URL="+server.URL, "METRICS_WAIT_TIMEOUT=0")
			output, err := command.CombinedOutput()
			if scenario == "healthy" {
				require.NoError(t, err, "%s", output)
			} else {
				require.Error(t, err, "%s", output)
				require.Contains(t, string(output), "Timed out:")
			}
		})
	}
}

func metricSeriesForCheck(t *testing.T, scenario, query string) []map[string]any {
	t.Helper()
	series := func(name, pod, value string) map[string]any {
		return map[string]any{"metric": map[string]string{"__name__": name, "pod": pod}, "value": []any{1, value}}
	}
	if strings.HasPrefix(query, "pg_up{") {
		value := "1"
		if scenario == "database down" {
			value = "0"
		}
		return []map[string]any{series("pg_up", "exporter", value)}
	}
	if strings.HasPrefix(query, "pg_stat_database_numbackends{") {
		if !strings.Contains(query, `datname="catalog_test"`) || !strings.Contains(query, `pod="exporter"`) {
			t.Errorf("database statistics query ignores deployed configuration or current target: %s", query)
		}
		return []map[string]any{series("pg_stat_database_numbackends", "exporter", "3")}
	}
	names := []string{"go_goroutines", "process_cpu_seconds_total", "badgoodtune_users_registered_total", "badgoodtune_logins_total", "badgoodtune_albums_created_total", "badgoodtune_reviews_created_total", "badgoodtune_review_score_count", "badgoodtune_review_score_sum"}
	var result []map[string]any
	for _, pod := range []string{"backend-a", "backend-b"} {
		for index, name := range names {
			if scenario == "missing metric" && pod == "backend-b" && index == 0 {
				continue
			}
			value := "1"
			if scenario == "duplicate metric" && pod == "backend-b" && index == 0 {
				value = "2"
			}
			result = append(result, series(name, pod, value))
		}
	}
	return result
}

func TestPortForwardReturnsCommandFailureAndStopsProcess(t *testing.T) {
	directory := t.TempDir()
	kubectl := filepath.Join(directory, "kubectl")
	stopped := filepath.Join(directory, "stopped")
	script := fmt.Sprintf("#!/bin/sh\ntrap 'touch %q; exit 0' TERM\nprintf 'Forwarding from 127.0.0.1:18428 -> 8428\\n'\nwhile :; do sleep 1; done\n", stopped)
	require.NoError(t, os.WriteFile(kubectl, []byte(script), 0o700))
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "./scripts/port-forward.sh", "observability", "service/victoria-metrics", "18428:8428", "sh", "-c", "exit 7")
	command.Dir = repositoryRoot(t)
	command.Env = append(os.Environ(), "KUBECTL="+kubectl)
	output, err := command.CombinedOutput()
	var failure *exec.ExitError
	require.ErrorAs(t, err, &failure, "%s", output)
	require.Equal(t, 7, failure.ExitCode())
	require.FileExists(t, stopped, "temporary port-forward must be stopped even when the command fails")
}

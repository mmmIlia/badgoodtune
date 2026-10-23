package metrics

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsExportsOpenMetricsAndRuntime(t *testing.T) {
	m := New()
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	r.Header.Set("Accept", "application/openmetrics-text; version=1.0.0")
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/openmetrics-text")
	assert.Contains(t, w.Body.String(), "go_goroutines ")
	assert.Contains(t, w.Body.String(), "process_start_time_seconds ")
	assert.Contains(t, w.Body.String(), "badgoodtune_reviews_created_total 0")
	assert.Contains(t, w.Body.String(), "# EOF\n")
	assert.Zero(t, testutil.ToFloat64(New().users), "registries must be isolated")
}

func TestMetricsHandlesConcurrentEvents(t *testing.T) {
	m := New()
	var workers sync.WaitGroup
	for range 100 {
		workers.Go(func() {
			m.UserRegistered()
			m.LoginSucceeded()
			m.AlbumCreated()
			m.ReviewCreated(9)
			m.RequestStarted()
			m.RequestCompleted("createReview", 201, 20*time.Millisecond)
		})
	}
	workers.Wait()
	assert.Equal(t, float64(100), testutil.ToFloat64(m.users))
	assert.Equal(t, float64(100), testutil.ToFloat64(m.logins))
	assert.Equal(t, float64(100), testutil.ToFloat64(m.albums))
	assert.Equal(t, float64(100), testutil.ToFloat64(m.reviews))
	assert.Zero(t, testutil.ToFloat64(m.inFlight))
	assert.Equal(t, float64(100), testutil.ToFloat64(m.requests.WithLabelValues("createReview", "201")))
	families, err := m.registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == "badgoodtune_review_score" {
			histogram := family.Metric[0].GetHistogram()
			assert.Equal(t, uint64(100), histogram.GetSampleCount())
			assert.Equal(t, float64(900), histogram.GetSampleSum())
			return
		}
	}
	t.Fatal("review score histogram is missing")
}

func TestLatencySeparatesSuccessfulAndFailedRequests(t *testing.T) {
	m := New()
	for _, request := range []struct {
		status   int
		duration time.Duration
	}{{200, 100 * time.Millisecond}, {200, time.Second}, {500, time.Millisecond}} {
		m.RequestStarted()
		m.RequestCompleted("ListAlbums", request.status, request.duration)
	}
	families, err := m.registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "badgoodtune_http_request_duration_seconds" {
			continue
		}
		require.Len(t, family.Metric, 2)
		for _, metric := range family.Metric {
			code := ""
			for _, label := range metric.Label {
				if label.GetName() == "code" {
					code = label.GetValue()
				}
			}
			if code == "200" {
				assert.Equal(t, uint64(2), metric.Histogram.GetSampleCount())
				for _, bucket := range metric.Histogram.Bucket {
					if bucket.GetUpperBound() == 0.5 {
						assert.Equal(t, uint64(1), bucket.GetCumulativeCount())
					}
				}
			} else {
				assert.Equal(t, "500", code)
				assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())
			}
		}
		return
	}
	t.Fatal("HTTP duration histogram is missing")
}

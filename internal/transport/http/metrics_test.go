package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type httpObserverStub struct {
	started    int
	operations []string
	statuses   []int
}

func (o *httpObserverStub) RequestStarted() { o.started++ }
func (o *httpObserverStub) RequestCompleted(operation string, status int, _ time.Duration) {
	o.operations = append(o.operations, operation)
	o.statuses = append(o.statuses, status)
}

func TestHTTPMetricsExcludeProbesAndKeepUnknownRoutesBounded(t *testing.T) {
	o := &httpObserverStub{}
	h := newTestHandler(t, testDependencies{observer: o})
	for _, target := range []string{"/health/live", "/health/ready", "/missing/user-1", "/missing/user-2"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))
	}
	assert.Equal(t, 2, o.started)
	assert.Equal(t, []string{"unmatched", "unmatched"}, o.operations)
	assert.Equal(t, []int{404, 404}, o.statuses)
}

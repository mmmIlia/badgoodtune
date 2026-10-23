package metrics

import (
	"strconv"
	"time"
)

func (m *Metrics) RequestStarted() { m.inFlight.Inc() }

func (m *Metrics) RequestCompleted(operation string, status int, duration time.Duration) {
	m.inFlight.Dec()
	code := strconv.Itoa(status)
	m.requests.WithLabelValues(operation, code).Inc()
	m.duration.WithLabelValues(operation, code).Observe(duration.Seconds())
}

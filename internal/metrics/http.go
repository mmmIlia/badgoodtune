package metrics

import (
	"strconv"
	"time"
)

func (m *Metrics) RequestStarted() { m.inFlight.Inc() }

func (m *Metrics) RequestCompleted(operation string, status int, duration time.Duration) {
	m.inFlight.Dec()
	m.requests.WithLabelValues(operation, strconv.Itoa(status)).Inc()
	m.duration.WithLabelValues(operation).Observe(duration.Seconds())
}

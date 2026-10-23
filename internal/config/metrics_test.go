package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricsValidation(t *testing.T) {
	m := Metrics{Address: ":9090", ReadHeaderTimeout: time.Second, WriteTimeout: time.Second, IdleTimeout: time.Second}
	assert.Empty(t, m.problems())
	for _, address := range []string{"", "9090", ":0", ":65536", ":invalid"} {
		m.Address = address
		assert.NotEmpty(t, m.problems(), address)
	}
}

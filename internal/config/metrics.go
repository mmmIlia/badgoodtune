package config

import (
	"net"
	"strconv"
	"time"
)

type Metrics struct {
	Address           string        `env:"METRICS_ADDRESS" env-default:":9090"`
	ReadHeaderTimeout time.Duration `env:"METRICS_READ_HEADER_TIMEOUT" env-default:"5s"`
	WriteTimeout      time.Duration `env:"METRICS_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout       time.Duration `env:"METRICS_IDLE_TIMEOUT" env-default:"30s"`
}

func (m Metrics) problems() []string {
	var problems []string
	_, port, err := net.SplitHostPort(m.Address)
	number, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || number < 1 || number > 65535 {
		problems = append(problems, "METRICS_ADDRESS must contain a host and a port between 1 and 65535")
	}
	if m.ReadHeaderTimeout <= 0 {
		problems = append(problems, "METRICS_READ_HEADER_TIMEOUT must be positive")
	}
	if m.WriteTimeout <= 0 {
		problems = append(problems, "METRICS_WRITE_TIMEOUT must be positive")
	}
	if m.IdleTimeout <= 0 {
		problems = append(problems, "METRICS_IDLE_TIMEOUT must be positive")
	}
	return problems
}

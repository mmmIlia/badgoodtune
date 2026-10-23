package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

type Metrics struct {
	registry    *prometheus.Registry
	users       prometheus.Counter
	logins      prometheus.Counter
	albums      prometheus.Counter
	reviews     prometheus.Counter
	reviewScore prometheus.Histogram
	requests    *prometheus.CounterVec
	duration    *prometheus.HistogramVec
	inFlight    prometheus.Gauge
}

func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		users:    newCounter("users_registered_total", "Number of successful user registrations."),
		logins:   newCounter("logins_total", "Number of successful explicit logins, excluding registration."),
		albums:   newCounter("albums_created_total", "Number of successfully created albums."),
		reviews:  newCounter("reviews_created_total", "Number of successfully created reviews."),
		reviewScore: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "badgoodtune",
			Name:      "review_score",
			Help:      "Distribution of review scores at initial publication, excluding edits.",
			Buckets:   prometheus.LinearBuckets(domain.ReviewScoreMin, 1, domain.ReviewScoreMax-domain.ReviewScoreMin+1),
		}),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "badgoodtune", Name: "http_requests_total", Help: "Number of completed API requests.",
		}, []string{"operation", "code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "badgoodtune", Name: "http_request_duration_seconds", Help: "API request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "badgoodtune", Name: "http_requests_in_flight", Help: "Number of API requests currently being served.",
		}),
	}
	m.registry.MustRegister(
		collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.users, m.logins, m.albums, m.reviews, m.reviewScore, m.requests, m.duration, m.inFlight,
	)
	return m
}

func (m *Metrics) Handler() http.Handler {
	routes := http.NewServeMux()
	routes.Handle("GET /metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	return routes
}

func newCounter(name, help string) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{Namespace: "badgoodtune", Name: name, Help: help})
}

var (
	_ usecase.AuthObserver   = (*Metrics)(nil)
	_ usecase.AlbumObserver  = (*Metrics)(nil)
	_ usecase.ReviewObserver = (*Metrics)(nil)
)

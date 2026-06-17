package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SensorDataReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacon_sensor_data_received_total",
		Help: "Total number of sensor data records received",
	})

	SignalReceptionReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacon_signal_reception_received_total",
		Help: "Total number of signal reception records received",
	})

	VisibilityCalculations = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacon_visibility_calculations_total",
		Help: "Total number of visibility calculations performed",
	})

	MonteCarloRuns = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacon_monte_carlo_runs_total",
		Help: "Total number of Monte Carlo simulation runs",
	})

	AlertsTriggered = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "beacon_alerts_triggered_total",
		Help: "Total number of alerts triggered by type",
	}, []string{"type", "severity"})

	HttpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "beacon_http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	HttpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "beacon_http_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	MonteCarloDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "beacon_monte_carlo_duration_seconds",
		Help:    "Monte Carlo simulation duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	VisibilityDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "beacon_visibility_duration_seconds",
		Help:    "Visibility calculation duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	ActiveBeacons = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_active_beacons",
		Help: "Number of active beacons",
	})

	NetworkConnectivity = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_network_connectivity_index",
		Help: "Current network connectivity index",
	})

	NetworkReliability = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_network_reliability",
		Help: "Current network reliability from Monte Carlo",
	})

	EventBusPublished = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "beacon_eventbus_published_total",
		Help: "Total number of events published by type",
	}, []string{"event_type"})

	ValidationFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "beacon_validation_failures_total",
		Help: "Total number of validation failures by field",
	}, []string{"field"})
)

package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	CommandsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kvstore_commands_total",
			Help: "Total number of commands processed.",
		},
		[]string{"command"},
	)

	KeysTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kvstore_keys_total",
			Help: "Current number of keys in store.",
		},
	)

	CommandDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kvstore_command_duration_seconds",
			Help:    "Latency of each command in seconds.",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"command"},
	)
)

func Register() {
	prometheus.MustRegister(CommandsTotal, KeysTotal, CommandDurationSeconds)
}

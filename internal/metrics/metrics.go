package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	NAMESPACE = "amazingchow"
	SUBSYSTEM = "infra-websocket-gateway-service"
)

var (
	WebsocketConnectionTotalCnt = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: NAMESPACE,
			Subsystem: SUBSYSTEM,
			Name:      "websocket_connection_total_cnt",
			Help:      "Total number of client websocket connections.",
		},
		[]string{"project"},
	)
	_MetricsList = []prometheus.Collector{
		WebsocketConnectionTotalCnt,
	}
)

var _RegisterMetricsOnce sync.Once

// Register all metrics.
func Register() {
	_RegisterMetricsOnce.Do(func() {
		for _, metrics := range _MetricsList {
			prometheus.MustRegister(metrics)
		}
	})
}

// SinceInSeconds gets the time since the specified start in seconds.
func SinceInSeconds(st time.Time) float64 {
	return time.Since(st).Seconds()
}

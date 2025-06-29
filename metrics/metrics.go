package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/philippe-berto/httpkit/utils"
	"github.com/philippe-berto/logger"
)

type Config struct {
	Port   int64 `env:"METRIC_PORT"   envDefault:"80"`
	Enable bool  `env:"METRIC_ENABLE" envDefault:"0"`
}

var (
	requestsTotalByEndpointAndStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total_by_endpoint_and_status",
			Help: "Total number of HTTP requests by endpoint",
		},
		[]string{"path", "method", "status"},
	)

	requestDurationByEndpointAndStatus = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds_by_endpoint_and_status",
			Help:    "Histogram of response latency (seconds) of HTTP requests by status and endpoint.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method", "status"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotalByEndpointAndStatus)

	prometheus.MustRegister(requestDurationByEndpointAndStatus)
}

func StartMetrics(port int64, enable bool, log *logger.Logger) {
	if !enable {
		return
	}

	http.Handle("/metrics", promhttp.Handler())
	log.Info("Starting Metrics Server on: %v", port)

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.WithFields(logger.Fields{"error": err}).Fatal("Failed to start serving metrics!")

		return
	}
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if utils.CheckInValidPath(r) {
			next.ServeHTTP(w, r)

			return
		}

		start := time.Now()

		ww := &utils.StatusWriter{ResponseWriter: w, StatusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		statusCode := fmt.Sprintf("%d", ww.StatusCode)

		path := chi.RouteContext(r.Context()).RoutePattern()
		method := r.Method

		duration := time.Since(start).Seconds()

		requestsTotalByEndpointAndStatus.WithLabelValues(path, method, statusCode).Inc()
		requestDurationByEndpointAndStatus.WithLabelValues(path, method, statusCode).Observe(duration)
	})
}

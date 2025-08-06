package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Registry = prometheus.NewRegistry()

	reqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "route", "status"},
	)

	reqInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_inflight",
			Help: "In-flight HTTP requests",
		},
	)

	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	cacheItems = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cache_items",
			Help: "Approximate number of items in cache",
		},
	)
)

func init() {
	Registry.MustRegister(reqTotal, reqInFlight, reqDuration, cacheItems)
}

// CacheSizer provides ability to get cache size
// Implemented by internal/cache MemoryCache via Size()
type CacheSizer interface { Size() int }

// UpdateCacheItems gauges current cache size
func UpdateCacheItems(c CacheSizer) {
	if c == nil { return }
	cacheItems.Set(float64(c.Size()))
}

// Middleware instruments HTTP requests
func Middleware(route string, next http.Handler, sizer CacheSizer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqInFlight.Inc()
		defer reqInFlight.Dec()

		// Capture status code
		rw := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)

		dur := time.Since(start).Seconds()
		reqDuration.WithLabelValues(r.Method, route).Observe(dur)
		reqTotal.WithLabelValues(r.Method, route, http.StatusText(rw.status)).Inc()

		// Update cache items gauge opportunistically
		UpdateCacheItems(sizer)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Handler returns a promhttp handler for the Registry
func Handler() http.Handler { return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{}) }

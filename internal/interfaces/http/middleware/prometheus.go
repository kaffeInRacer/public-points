package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status_code"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "Size of HTTP requests in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP responses in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint", "status_code"},
	)

	// Business metrics
	activeUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_users_total",
			Help: "Number of currently active users",
		},
	)

	ordersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_total",
			Help: "Total number of orders",
		},
		[]string{"status"},
	)

	orderValue = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "order_value_dollars",
			Help:    "Value of orders in dollars",
			Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
		[]string{"status"},
	)

	productsViewed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "products_viewed_total",
			Help: "Total number of product views",
		},
		[]string{"product_id", "category"},
	)

	cartOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cart_operations_total",
			Help: "Total number of cart operations",
		},
		[]string{"operation"}, // add, remove, update, clear
	)

	paymentAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_attempts_total",
			Help: "Total number of payment attempts",
		},
		[]string{"method", "status"},
	)

	// Database metrics
	dbConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_connections",
			Help: "Number of database connections",
		},
		[]string{"state"}, // active, idle, open
	)

	dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
		},
		[]string{"operation", "table"},
	)

	// Cache metrics
	cacheOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "result"}, // get/set/delete, hit/miss/error
	)

	cacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_size_bytes",
			Help: "Size of cache in bytes",
		},
		[]string{"cache_type"},
	)
)

// PrometheusMetrics returns a middleware that collects Prometheus metrics
func PrometheusMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Get request size
		requestSize := computeApproximateRequestSize(c)

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get response size
		responseSize := float64(c.Writer.Size())

		// Get labels
		method := c.Request.Method
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = "unknown"
		}
		statusCode := strconv.Itoa(c.Writer.Status())

		// Record metrics
		httpRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
		httpRequestDuration.WithLabelValues(method, endpoint, statusCode).Observe(duration)
		httpRequestSize.WithLabelValues(method, endpoint).Observe(float64(requestSize))
		httpResponseSize.WithLabelValues(method, endpoint, statusCode).Observe(responseSize)
	}
}

// Business metric helpers
func RecordActiveUser() {
	activeUsers.Inc()
}

func RecordUserLogout() {
	activeUsers.Dec()
}

func RecordOrder(status string, value float64) {
	ordersTotal.WithLabelValues(status).Inc()
	orderValue.WithLabelValues(status).Observe(value)
}

func RecordProductView(productID, category string) {
	productsViewed.WithLabelValues(productID, category).Inc()
}

func RecordCartOperation(operation string) {
	cartOperations.WithLabelValues(operation).Inc()
}

func RecordPaymentAttempt(method, status string) {
	paymentAttempts.WithLabelValues(method, status).Inc()
}

// Database metric helpers
func RecordDBConnections(active, idle, open int) {
	dbConnections.WithLabelValues("active").Set(float64(active))
	dbConnections.WithLabelValues("idle").Set(float64(idle))
	dbConnections.WithLabelValues("open").Set(float64(open))
}

func RecordDBQuery(operation, table string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// Cache metric helpers
func RecordCacheOperation(operation, result string) {
	cacheOperations.WithLabelValues(operation, result).Inc()
}

func RecordCacheSize(cacheType string, size int64) {
	cacheSize.WithLabelValues(cacheType).Set(float64(size))
}

// Helper function to compute approximate request size
func computeApproximateRequestSize(r *gin.Context) int {
	s := 0
	if r.Request.URL != nil {
		s = len(r.Request.URL.Path)
	}

	s += len(r.Request.Method)
	s += len(r.Request.Proto)
	for name, values := range r.Request.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Request.Host)

	// N.B. r.Request.Form and r.Request.MultipartForm are assumed to be included in r.Request.URL.

	if r.Request.ContentLength != -1 {
		s += int(r.Request.ContentLength)
	}
	return s
}
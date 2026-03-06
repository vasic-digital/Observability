package gin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.observability/pkg/health"
	"digital.vasic.observability/pkg/metrics"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestCollector creates a PrometheusCollector with an isolated registry
// so tests do not conflict with each other.
func newTestCollector(t *testing.T, namespace string) metrics.Collector {
	t.Helper()
	reg := prometheus.NewRegistry()
	cfg := &metrics.PrometheusConfig{
		Namespace:      namespace,
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       reg,
	}
	return metrics.NewPrometheusCollector(cfg)
}

func TestMetricsMiddleware_RecordsLatency(t *testing.T) {
	collector := newTestCollector(t, "mw_latency")

	router := gin.New()
	router.Use(MetricsMiddleware(collector))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestMetricsMiddleware_Records4xx(t *testing.T) {
	collector := newTestCollector(t, "mw_4xx")

	router := gin.New()
	router.Use(MetricsMiddleware(collector))
	// No route registered for /missing, so Gin returns 404.

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHealthHandler_ReturnsReport(t *testing.T) {
	agg := health.NewAggregator(nil)
	agg.Register("db", func(_ context.Context) error {
		return nil
	})

	router := gin.New()
	router.GET("/health", HealthHandler(agg))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var report health.Report
	err := json.Unmarshal(w.Body.Bytes(), &report)
	require.NoError(t, err)
	assert.Equal(t, health.StatusHealthy, report.Status)
	assert.Len(t, report.Components, 1)
	assert.Equal(t, "db", report.Components[0].Name)
}

func TestHealthHandler_UnhealthyReturns503(t *testing.T) {
	agg := health.NewAggregator(nil)
	agg.Register("db", func(_ context.Context) error {
		return errors.New("connection refused")
	})

	router := gin.New()
	router.GET("/health", HealthHandler(agg))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var report health.Report
	err := json.Unmarshal(w.Body.Bytes(), &report)
	require.NoError(t, err)
	assert.Equal(t, health.StatusUnhealthy, report.Status)
}

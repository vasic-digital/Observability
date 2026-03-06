// Package gin provides Gin framework middleware for metrics collection
// and a health check handler that integrates with the observability module.
package gin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"digital.vasic.observability/pkg/health"
	"digital.vasic.observability/pkg/metrics"
)

// MetricsMiddleware returns a Gin middleware that records request latency
// and increments a per-status-code counter using the provided Collector.
func MetricsMiddleware(collector metrics.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		status := c.Writer.Status()
		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		label := fmt.Sprintf("%s_%s_%d", method, path, status)
		collector.RecordLatency(label, duration, nil)
		collector.IncrementCounter(
			fmt.Sprintf("http_requests_%d", status), nil,
		)
	}
}

// HealthHandler returns a Gin handler that performs a health check using
// the provided Aggregator and returns the report as JSON. Returns 200 OK
// when healthy, or 503 Service Unavailable otherwise.
func HealthHandler(agg *health.Aggregator) gin.HandlerFunc {
	return func(c *gin.Context) {
		report := agg.Check(context.Background())
		status := http.StatusOK
		if report.Status != health.StatusHealthy {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, report)
	}
}

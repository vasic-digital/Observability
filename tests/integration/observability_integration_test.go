package integration

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.observability/pkg/analytics"
	"digital.vasic.observability/pkg/health"
	"digital.vasic.observability/pkg/logging"
	"digital.vasic.observability/pkg/metrics"
	"digital.vasic.observability/pkg/trace"
)

func TestMetricsCollectorCounterAndGauge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	registry := prometheus.NewRegistry()
	collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
		Namespace: "integration_test",
		Registry:  registry,
	})

	labels := map[string]string{"service": "test"}
	collector.IncrementCounter("requests_total", labels)
	collector.IncrementCounter("requests_total", labels)
	collector.AddCounter("requests_total", 3, labels)

	collector.SetGauge("active_connections", 42, labels)
	collector.SetGauge("active_connections", 38, labels)

	collector.RecordLatency("request_duration", 150*time.Millisecond, labels)
	collector.RecordValue("request_size", 1024, labels)

	gathered, err := registry.Gather()
	require.NoError(t, err)
	assert.True(t, len(gathered) > 0, "should have gathered metrics")
}

func TestMetricsRegisterAndUse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	registry := prometheus.NewRegistry()
	collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
		Namespace: "reg_test",
		Registry:  registry,
	})

	err := collector.RegisterCounter("http_requests", "HTTP request count", []string{"method", "status"})
	require.NoError(t, err)

	err = collector.RegisterHistogram("http_latency", "HTTP request latency", []string{"method"}, nil)
	require.NoError(t, err)

	err = collector.RegisterGauge("goroutines", "Active goroutines", []string{})
	require.NoError(t, err)

	collector.IncrementCounter("http_requests", map[string]string{"method": "GET", "status": "200"})
	collector.RecordLatency("http_latency", 50*time.Millisecond, map[string]string{"method": "GET"})
	collector.SetGauge("goroutines", 150, map[string]string{})
}

func TestHealthAggregatorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	agg := health.NewAggregator(&health.AggregatorConfig{
		Timeout: 2 * time.Second,
	})

	agg.Register("database", health.StaticCheck(nil))
	agg.Register("cache", health.StaticCheck(nil))
	agg.RegisterOptional("search", health.StaticCheck(fmt.Errorf("unavailable")))

	report := agg.Check(context.Background())
	assert.Equal(t, health.StatusDegraded, report.Status)
	assert.Len(t, report.Components, 3)

	for _, comp := range report.Components {
		if comp.Name == "database" || comp.Name == "cache" {
			assert.Equal(t, health.StatusHealthy, comp.Status)
		}
		if comp.Name == "search" {
			assert.Equal(t, health.StatusUnhealthy, comp.Status)
			assert.Contains(t, comp.Message, "unavailable")
		}
	}
}

func TestLoggerWithCorrelationID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	var buf bytes.Buffer
	logger := logging.NewLogrusAdapter(&logging.Config{
		Level:       logging.DebugLevel,
		Format:      "json",
		Output:      &buf,
		ServiceName: "integration-test",
	})

	logger.Info("basic message")
	logger.WithField("request_id", "req-123").Info("with field")
	logger.WithCorrelationID("corr-abc").Info("with correlation")
	logger.WithError(fmt.Errorf("test error")).Error("error occurred")
	logger.WithFields(map[string]interface{}{
		"user": "admin",
		"ip":   "192.168.1.1",
	}).Info("with fields")

	output := buf.String()
	assert.Contains(t, output, "basic message")
	assert.Contains(t, output, "req-123")
	assert.Contains(t, output, "corr-abc")
	assert.Contains(t, output, "test error")
}

func TestContextCorrelationID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	ctx := context.Background()
	assert.Empty(t, logging.CorrelationIDFromContext(ctx))

	ctx = logging.ContextWithCorrelationID(ctx, "trace-xyz")
	assert.Equal(t, "trace-xyz", logging.CorrelationIDFromContext(ctx))

	var buf bytes.Buffer
	logger := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.DebugLevel,
		Format: "json",
		Output: &buf,
	})

	enriched := logging.WithContext(logger, ctx)
	enriched.Info("correlated log")

	output := buf.String()
	assert.Contains(t, output, "trace-xyz")
}

func TestTracerInitAndShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	tracer, err := trace.InitTracer(&trace.TracerConfig{
		ServiceName:    "integration-test",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   trace.ExporterNone,
		SampleRate:     1.0,
	})
	require.NoError(t, err)
	require.NotNil(t, tracer)

	ctx, span := tracer.StartSpan(context.Background(), "test-operation")
	assert.NotNil(t, span)
	assert.NotNil(t, ctx)
	span.End()

	err = tracer.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestNoOpCollectors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	noopMetrics := &metrics.NoOpCollector{}
	noopMetrics.IncrementCounter("test", nil)
	noopMetrics.AddCounter("test", 1, nil)
	noopMetrics.RecordLatency("test", time.Second, nil)
	noopMetrics.RecordValue("test", 1.0, nil)
	noopMetrics.SetGauge("test", 1.0, nil)

	noopLogger := &logging.NoOpLogger{}
	noopLogger.Info("test")
	noopLogger.Warn("test")
	noopLogger.Error("test")
	noopLogger.Debug("test")
	_ = noopLogger.WithField("key", "val")
	_ = noopLogger.WithFields(nil)
	_ = noopLogger.WithCorrelationID("id")
	_ = noopLogger.WithError(nil)

	noopAnalytics := &analytics.NoOpCollector{}
	assert.NoError(t, noopAnalytics.Track(context.Background(), analytics.Event{}))
	assert.NoError(t, noopAnalytics.TrackBatch(context.Background(), nil))
	stats, err := noopAnalytics.Query(context.Background(), "t", "g", time.Hour)
	assert.NoError(t, err)
	assert.Nil(t, stats)
	assert.NoError(t, noopAnalytics.Close())
}

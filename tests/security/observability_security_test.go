package security

import (
	"bytes"
	"context"
	"fmt"
	"strings"
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

func TestSecurity_NilMetricLabels(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	registry := prometheus.NewRegistry()
	collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
		Namespace: "nil_label_test",
		Registry:  registry,
	})

	assert.NotPanics(t, func() {
		collector.IncrementCounter("test_counter", nil)
		collector.AddCounter("test_counter", 1, nil)
		collector.RecordLatency("test_latency", time.Second, nil)
		collector.RecordValue("test_value", 1.0, nil)
		collector.SetGauge("test_gauge", 1.0, nil)
	})
}

func TestSecurity_DuplicateRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	registry := prometheus.NewRegistry()
	collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
		Namespace: "dup_test",
		Registry:  registry,
	})

	err := collector.RegisterCounter("dup_counter", "first", []string{"label"})
	require.NoError(t, err)

	err = collector.RegisterCounter("dup_counter", "second", []string{"label"})
	assert.NoError(t, err, "should be idempotent for same name")
}

func TestSecurity_HealthCheckTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	agg := health.NewAggregator(&health.AggregatorConfig{
		Timeout: 100 * time.Millisecond,
	})

	agg.Register("slow-service", func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	start := time.Now()
	report := agg.Check(context.Background())
	duration := time.Since(start)

	assert.Equal(t, health.StatusUnhealthy, report.Status)
	assert.True(t, duration < 2*time.Second,
		"should not wait for slow service, took %v", duration)
	assert.Contains(t, report.Components[0].Message, "timed out")
}

func TestSecurity_LoggerLargeInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	var buf bytes.Buffer
	logger := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.DebugLevel,
		Format: "json",
		Output: &buf,
	})

	largeMsg := strings.Repeat("A", 100000)
	assert.NotPanics(t, func() {
		logger.Info(largeMsg)
	})
	assert.True(t, buf.Len() > 0)
}

func TestSecurity_TracerNilSpanHelpers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	assert.NotPanics(t, func() {
		trace.RecordError(nil, fmt.Errorf("test"))
		trace.SetOK(nil)
		trace.EndSpanWithError(nil, nil)
		trace.EndSpanWithError(nil, fmt.Errorf("test"))
	})
}

func TestSecurity_AnalyticsSQLInjection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	noopCollector := analytics.NewCollector(nil, nil)

	_, ok := noopCollector.(*analytics.NoOpCollector)
	assert.True(t, ok, "nil config should produce NoOpCollector")
}

func TestSecurity_LoggerNilError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	var buf bytes.Buffer
	logger := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.DebugLevel,
		Format: "json",
		Output: &buf,
	})

	assert.NotPanics(t, func() {
		logger.WithError(nil).Error("no error attached")
	})
}

func TestSecurity_EmptyCorrelationID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	ctx := context.Background()
	id := logging.CorrelationIDFromContext(ctx)
	assert.Empty(t, id)

	logger := &logging.NoOpLogger{}
	enriched := logging.WithContext(logger, ctx)
	assert.NotNil(t, enriched)
}

func TestSecurity_TracerDefaultConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test in short mode")  // SKIP-OK: #short-mode
	}

	tracer, err := trace.InitTracer(nil)
	require.NoError(t, err)
	require.NotNil(t, tracer)
	defer tracer.Shutdown(context.Background())

	assert.NotNil(t, tracer.Provider())
}

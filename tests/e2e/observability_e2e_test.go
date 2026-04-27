package e2e

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.observability/pkg/health"
	"digital.vasic.observability/pkg/logging"
	"digital.vasic.observability/pkg/metrics"
	"digital.vasic.observability/pkg/trace"
)

func TestE2E_FullObservabilityStack(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	registry := prometheus.NewRegistry()
	metricsCollector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
		Namespace: "e2e_svc",
		Registry:  registry,
	})

	var logBuf bytes.Buffer
	logger := logging.NewLogrusAdapter(&logging.Config{
		Level:       logging.DebugLevel,
		Format:      "json",
		Output:      &logBuf,
		ServiceName: "e2e-service",
	})

	tracer, err := trace.InitTracer(&trace.TracerConfig{
		ServiceName:  "e2e-service",
		ExporterType: trace.ExporterNone,
		SampleRate:   1.0,
	})
	require.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	healthAgg := health.NewAggregator(nil)
	healthAgg.Register("metrics", health.StaticCheck(nil))
	healthAgg.Register("logging", health.StaticCheck(nil))

	ctx := logging.ContextWithCorrelationID(context.Background(), "e2e-trace-001")
	reqLogger := logging.WithContext(logger, ctx)

	ctx, span := tracer.StartSpan(ctx, "e2e-request")
	reqLogger.Info("processing request")

	metricsCollector.IncrementCounter("requests_total",
		map[string]string{"method": "GET", "path": "/api/data"})
	metricsCollector.RecordLatency("request_duration_seconds",
		45*time.Millisecond,
		map[string]string{"method": "GET"})

	span.End()

	report := healthAgg.Check(ctx)
	assert.Equal(t, health.StatusHealthy, report.Status)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "processing request")
	assert.Contains(t, logOutput, "e2e-trace-001")
}

func TestE2E_TracerSpanHierarchy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	tracer, err := trace.InitTracer(&trace.TracerConfig{
		ServiceName:  "span-test",
		ExporterType: trace.ExporterNone,
		SampleRate:   1.0,
	})
	require.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	ctx, parentSpan := tracer.StartSpan(context.Background(), "parent-operation")

	ctx, childSpan := tracer.StartInternalSpan(ctx, "child-internal")
	childSpan.End()

	_, clientSpan := tracer.StartClientSpan(ctx, "external-call")
	clientSpan.End()

	parentSpan.End()
}

func TestE2E_TraceFuncWithError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	tracer, err := trace.InitTracer(&trace.TracerConfig{
		ServiceName:  "trace-func-test",
		ExporterType: trace.ExporterNone,
		SampleRate:   1.0,
	})
	require.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	err = tracer.TraceFunc(context.Background(), "successful-op",
		func(ctx context.Context) error {
			return nil
		})
	assert.NoError(t, err)

	err = tracer.TraceFunc(context.Background(), "failing-op",
		func(ctx context.Context) error {
			return fmt.Errorf("operation failed")
		})
	assert.Error(t, err)
}

func TestE2E_HealthCheckMixedComponents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	agg := health.NewAggregator(&health.AggregatorConfig{
		Timeout: 3 * time.Second,
	})

	agg.Register("postgresql", health.StaticCheck(nil))
	agg.Register("redis", health.StaticCheck(nil))
	agg.RegisterOptional("elasticsearch", health.StaticCheck(fmt.Errorf("timeout")))
	agg.RegisterOptional("prometheus", health.StaticCheck(nil))

	assert.Equal(t, 4, agg.ComponentCount())

	report := agg.Check(context.Background())
	assert.Equal(t, health.StatusDegraded, report.Status)
	assert.Len(t, report.Components, 4)
	assert.False(t, report.Timestamp.IsZero())

	healthy := 0
	unhealthy := 0
	for _, c := range report.Components {
		if c.Status == health.StatusHealthy {
			healthy++
		} else {
			unhealthy++
		}
		assert.True(t, c.Duration >= 0)
		assert.False(t, c.LastChecked.IsZero())
	}
	assert.Equal(t, 3, healthy)
	assert.Equal(t, 1, unhealthy)
}

func TestE2E_LoggerLevelFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	var buf bytes.Buffer
	logger := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.WarnLevel,
		Format: "text",
		Output: &buf,
	})

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestE2E_TimedSpan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	tracer, err := trace.InitTracer(&trace.TracerConfig{
		ServiceName:  "timed-span-test",
		ExporterType: trace.ExporterNone,
		SampleRate:   1.0,
	})
	require.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	ctx, finish := tracer.TimedSpan(context.Background(), "timed-operation")
	assert.NotNil(t, ctx)

	time.Sleep(10 * time.Millisecond)
	finish(nil)

	_, finish2 := tracer.TimedSpan(context.Background(), "failing-timed")
	finish2(fmt.Errorf("something failed"))
}

func TestE2E_HealthCheckRequiredFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")  // SKIP-OK: #short-mode
	}

	agg := health.NewAggregator(nil)
	agg.Register("critical-db", health.StaticCheck(fmt.Errorf("connection refused")))
	agg.RegisterOptional("optional-cache", health.StaticCheck(nil))

	report := agg.Check(context.Background())
	assert.Equal(t, health.StatusUnhealthy, report.Status)
}

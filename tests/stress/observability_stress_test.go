package stress

import (
	"bytes"
	"context"
	"fmt"
	"sync"
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

// Resource limit: GOMAXPROCS=2 recommended for stress tests

func TestStress_ConcurrentMetricsCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")  // SKIP-OK: #short-mode
	}

	registry := prometheus.NewRegistry()
	collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
		Namespace: "stress_test",
		Registry:  registry,
	})

	const goroutines = 100
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			labels := map[string]string{"worker": fmt.Sprintf("w%d", id%5)}
			collector.IncrementCounter("stress_requests", labels)
			collector.RecordLatency("stress_latency", time.Duration(id)*time.Millisecond, labels)
			collector.SetGauge("stress_gauge", float64(id), labels)
		}(i)
	}

	wg.Wait()

	gathered, err := registry.Gather()
	require.NoError(t, err)
	assert.True(t, len(gathered) > 0)
}

func TestStress_ConcurrentLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")  // SKIP-OK: #short-mode
	}

	var buf bytes.Buffer
	logger := logging.NewLogrusAdapter(&logging.Config{
		Level:       logging.DebugLevel,
		Format:      "json",
		Output:      &buf,
		ServiceName: "stress-test",
	})

	const goroutines = 80
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			l := logger.WithField("worker", id).
				WithCorrelationID(fmt.Sprintf("corr-%d", id))
			l.Info(fmt.Sprintf("message from worker %d", id))
			l.Debug("debug info")
		}(i)
	}

	wg.Wait()
	assert.True(t, buf.Len() > 0)
}

func TestStress_ConcurrentHealthChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")  // SKIP-OK: #short-mode
	}

	agg := health.NewAggregator(&health.AggregatorConfig{
		Timeout: 2 * time.Second,
	})

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("component-%d", i)
		if i%3 == 0 {
			agg.RegisterOptional(name, health.StaticCheck(fmt.Errorf("degraded")))
		} else {
			agg.Register(name, health.StaticCheck(nil))
		}
	}

	const goroutines = 50
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			report := agg.Check(context.Background())
			assert.NotNil(t, report)
			assert.Len(t, report.Components, 10)
		}()
	}

	wg.Wait()
}

func TestStress_ConcurrentTraceSpans(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")  // SKIP-OK: #short-mode
	}

	tracer, err := trace.InitTracer(&trace.TracerConfig{
		ServiceName:  "stress-tracer",
		ExporterType: trace.ExporterNone,
		SampleRate:   1.0,
	})
	require.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	const goroutines = 100
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx, span := tracer.StartSpan(
				context.Background(),
				fmt.Sprintf("stress-op-%d", id),
			)
			_, child := tracer.StartInternalSpan(ctx, "child-op")
			child.End()
			span.End()
		}(i)
	}

	wg.Wait()
}

func TestStress_ConcurrentContextCorrelation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")  // SKIP-OK: #short-mode
	}

	var buf bytes.Buffer
	logger := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.DebugLevel,
		Format: "json",
		Output: &buf,
	})

	const goroutines = 50
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			corrID := fmt.Sprintf("corr-%d", id)
			ctx := logging.ContextWithCorrelationID(context.Background(), corrID)

			retrieved := logging.CorrelationIDFromContext(ctx)
			assert.Equal(t, corrID, retrieved)

			enriched := logging.WithContext(logger, ctx)
			enriched.Info("correlated message")
		}(i)
	}

	wg.Wait()
}

func TestStress_MetricsGatherUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")  // SKIP-OK: #short-mode
	}

	registry := prometheus.NewRegistry()
	collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
		Namespace: "gather_stress",
		Registry:  registry,
	})

	const goroutines = 50
	var wg sync.WaitGroup

	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			labels := map[string]string{"id": fmt.Sprintf("%d", id)}
			collector.IncrementCounter("gather_counter", labels)
			collector.RecordValue("gather_hist", float64(id), labels)
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = registry.Gather()
		}()
	}

	wg.Wait()
}

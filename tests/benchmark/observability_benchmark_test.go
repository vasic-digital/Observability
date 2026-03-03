package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"digital.vasic.observability/pkg/health"
	"digital.vasic.observability/pkg/logging"
	"digital.vasic.observability/pkg/metrics"
	"digital.vasic.observability/pkg/trace"
)

func BenchmarkMetricsIncrementCounter(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	registry := prometheus.NewRegistry()
	collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
		Namespace: "bench",
		Registry:  registry,
	})
	labels := map[string]string{"method": "GET"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.IncrementCounter("requests", labels)
	}
}

func BenchmarkMetricsRecordLatency(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	registry := prometheus.NewRegistry()
	collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
		Namespace: "bench_lat",
		Registry:  registry,
	})
	labels := map[string]string{"method": "GET"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordLatency("latency", 50*time.Millisecond, labels)
	}
}

func BenchmarkMetricsSetGauge(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	registry := prometheus.NewRegistry()
	collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
		Namespace: "bench_gauge",
		Registry:  registry,
	})
	labels := map[string]string{"instance": "localhost"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.SetGauge("goroutines", float64(i), labels)
	}
}

func BenchmarkLoggerInfo(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	var buf bytes.Buffer
	logger := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.InfoLevel,
		Format: "json",
		Output: &buf,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLoggerWithFields(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	var buf bytes.Buffer
	logger := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.InfoLevel,
		Format: "json",
		Output: &buf,
	})

	fields := map[string]interface{}{
		"user_id":    "123",
		"request_id": "req-456",
		"method":     "GET",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(fields).Info("request processed")
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	agg := health.NewAggregator(nil)
	for i := 0; i < 5; i++ {
		agg.Register(fmt.Sprintf("svc-%d", i), health.StaticCheck(nil))
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agg.Check(ctx)
	}
}

func BenchmarkTracerStartSpan(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	tracer, _ := trace.InitTracer(&trace.TracerConfig{
		ServiceName:  "bench-tracer",
		ExporterType: trace.ExporterNone,
		SampleRate:   1.0,
	})
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := tracer.StartSpan(ctx, "bench-span")
		span.End()
	}
}

func BenchmarkCorrelationIDContext(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := logging.ContextWithCorrelationID(context.Background(), "bench-id")
		_ = logging.CorrelationIDFromContext(ctx)
	}
}

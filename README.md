# digital.vasic.observability

A generic, reusable Go module for application observability: distributed tracing, metrics, structured logging, health checks, and analytics.

## Installation

```bash
go get digital.vasic.observability
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    "digital.vasic.observability/pkg/trace"
    "digital.vasic.observability/pkg/metrics"
    "digital.vasic.observability/pkg/logging"
    "digital.vasic.observability/pkg/health"
)

func main() {
    // Initialize tracing
    tracer, _ := trace.InitTracer(&trace.TracerConfig{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        ExporterType:   trace.ExporterStdout,
        SampleRate:     1.0,
    })
    defer tracer.Shutdown(context.Background())

    // Create a span
    ctx, span := tracer.StartSpan(context.Background(), "main.operation")
    defer span.End()

    // Trace a function
    tracer.TraceFunc(ctx, "db.query", func(ctx context.Context) error {
        time.Sleep(10 * time.Millisecond)
        return nil
    })

    // Initialize metrics
    collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
        Namespace: "myapp",
    })
    collector.IncrementCounter("requests_total", map[string]string{
        "method": "GET",
        "status": "200",
    })
    collector.RecordLatency("request_duration", 150*time.Millisecond, map[string]string{
        "endpoint": "/api/v1/users",
    })

    // Initialize logging
    logger := logging.NewLogrusAdapter(&logging.Config{
        Level:       logging.InfoLevel,
        Format:      "json",
        ServiceName: "my-service",
    })
    logger.WithCorrelationID("req-123").Info("Request processed")

    // Initialize health checks
    agg := health.NewAggregator(nil)
    agg.Register("database", health.StaticCheck(nil))
    agg.RegisterOptional("cache", health.StaticCheck(nil))
    report := agg.Check(context.Background())
    fmt.Printf("System health: %s\n", report.Status)
}
```

## Features

- **Distributed tracing** with OpenTelemetry (OTLP, Jaeger, Zipkin, stdout exporters)
- **Prometheus metrics** with auto-created counters, histograms, and gauges
- **Structured logging** with correlation ID propagation via context
- **Health check aggregation** with required/optional component support
- **ClickHouse analytics** with graceful NoOp fallback
- **Thread-safe** concurrent access across all packages
- **Generic interfaces** for easy mocking and testing

## Packages

| Package | Description |
|---------|-------------|
| `pkg/trace` | OpenTelemetry tracing with multiple exporter backends |
| `pkg/metrics` | Prometheus metrics collection with auto-registration |
| `pkg/logging` | Structured logging with correlation IDs |
| `pkg/health` | Health check aggregation for multi-component systems |
| `pkg/analytics` | ClickHouse analytics with NoOp fallback |

## Tracing

```go
// Create tracer with OTLP exporter
tracer, _ := trace.InitTracer(&trace.TracerConfig{
    ServiceName:  "my-service",
    ExporterType: trace.ExporterOTLP,
    Endpoint:     "localhost:4318",
    SampleRate:   0.5,
})
defer tracer.Shutdown(context.Background())

// Simple span
ctx, span := tracer.StartSpan(ctx, "operation", attribute.String("key", "val"))
defer span.End()

// Client span (for outgoing calls)
ctx, span := tracer.StartClientSpan(ctx, "http.request")
defer trace.EndSpanWithError(span, err)

// Timed span with automatic duration recording
ctx, finish := tracer.TimedSpan(ctx, "slow.operation")
defer finish(nil)

// Trace a function
err := tracer.TraceFunc(ctx, "db.query", func(ctx context.Context) error {
    return db.QueryRow(ctx, "SELECT 1")
})

// Trace with result
result, err := trace.TraceFuncWithResult(tracer, ctx, "compute", func(ctx context.Context) (int, error) {
    return 42, nil
})
```

## Metrics

```go
// With custom Prometheus registry
reg := prometheus.NewRegistry()
collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
    Namespace: "myapp",
    Subsystem: "api",
    Registry:  reg,
})

// Pre-register metrics (optional, auto-created on use)
collector.RegisterCounter("requests_total", "Total requests", []string{"method"})
collector.RegisterHistogram("duration_seconds", "Duration", []string{"endpoint"}, nil)
collector.RegisterGauge("connections", "Active connections", []string{"pool"})

// Record metrics
collector.IncrementCounter("requests_total", map[string]string{"method": "POST"})
collector.RecordLatency("duration_seconds", 100*time.Millisecond, map[string]string{"endpoint": "/api"})
collector.SetGauge("connections", 42, map[string]string{"pool": "main"})

// No-op collector for tests
var c metrics.Collector = &metrics.NoOpCollector{}
```

## Logging

```go
// Create logger
logger := logging.NewLogrusAdapter(&logging.Config{
    Level:       logging.DebugLevel,
    Format:      "json",
    ServiceName: "my-service",
})

// Structured logging
logger.WithField("user_id", "123").Info("User logged in")
logger.WithFields(map[string]interface{}{
    "request_id": "abc",
    "latency_ms": 150,
}).Debug("Request completed")
logger.WithError(err).Error("Operation failed")

// Correlation ID propagation
ctx := logging.ContextWithCorrelationID(ctx, "corr-123")
enriched := logging.WithContext(logger, ctx)
enriched.Info("Request with correlation ID")

// Extract correlation ID
id := logging.CorrelationIDFromContext(ctx)
```

## Health Checks

```go
agg := health.NewAggregator(&health.AggregatorConfig{
    Timeout: 3 * time.Second,
})

// Required: failure = unhealthy
agg.Register("database", func(ctx context.Context) error {
    return db.PingContext(ctx)
})

// Optional: failure = degraded
agg.RegisterOptional("cache", func(ctx context.Context) error {
    return redis.Ping(ctx).Err()
})

// Check all components (runs in parallel)
report := agg.Check(ctx)
// report.Status: "healthy", "degraded", or "unhealthy"
// report.Components: individual results with durations
```

## Analytics

```go
// Auto-fallback to NoOp if ClickHouse unavailable
collector := analytics.NewCollector(&analytics.ClickHouseConfig{
    Host:     "localhost",
    Port:     9000,
    Database: "analytics",
    Table:    "events",
}, logger)
defer collector.Close()

// Track events
collector.Track(ctx, analytics.Event{
    Name:      "request.completed",
    Timestamp: time.Now(),
    Properties: map[string]interface{}{"duration_ms": 250},
    Tags:       map[string]string{"service": "api"},
})

// Batch tracking
collector.TrackBatch(ctx, events)

// Query aggregated stats
stats, _ := collector.Query(ctx, "events", "name", 24*time.Hour)
```

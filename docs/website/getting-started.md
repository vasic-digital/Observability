# Getting Started

## Installation

```bash
go get digital.vasic.observability
```

## Health Checks

Set up a health check aggregator with required and optional components:

```go
package main

import (
    "context"
    "fmt"

    "digital.vasic.observability/pkg/health"
)

func main() {
    agg := health.NewAggregator()

    // Required component -- if this fails, overall status is "unhealthy"
    agg.Register("database", func(ctx context.Context) error {
        // Ping your database
        return nil
    })

    // Optional component -- if this fails, overall status is "degraded"
    agg.RegisterOptional("cache", func(ctx context.Context) error {
        // Ping your cache
        return nil
    })

    report := agg.Check(context.Background())
    fmt.Printf("Status: %s\n", report.Status) // healthy, degraded, or unhealthy

    for _, comp := range report.Components {
        fmt.Printf("  %s: %s\n", comp.Name, comp.Status)
    }
}
```

## Structured Logging

Use the logging package with correlation IDs:

```go
package main

import (
    "digital.vasic.observability/pkg/logging"
)

func main() {
    logger := logging.NewLogrusAdapter(nil)

    // Basic logging
    logger.Info("Application started")
    logger.Warn("Cache miss rate high")

    // Add context fields
    reqLogger := logger.WithField("request_id", "abc-123").
        WithField("user_id", 42)
    reqLogger.Info("Processing request")

    // Correlation ID support
    corrLogger := logger.WithCorrelationID("corr-xyz-789")
    corrLogger.Info("Handling correlated operation")

    // Error logging
    logger.WithError(fmt.Errorf("connection refused")).Error("Database unreachable")
}
```

## Prometheus Metrics

Collect counters, histograms, and gauges:

```go
package main

import (
    "digital.vasic.observability/pkg/metrics"
)

func main() {
    collector := metrics.NewPrometheusCollector("myapp")

    // Increment a counter
    collector.IncrementCounter("http_requests_total", map[string]string{
        "method": "GET",
        "path":   "/api/users",
    })

    // Record latency
    collector.RecordLatency("http_request_duration_seconds", 0.045, map[string]string{
        "method": "GET",
    })

    // Set a gauge
    collector.SetGauge("active_connections", 42, nil)

    // Use NoOpCollector when metrics are disabled
    var c metrics.Collector = &metrics.NoOpCollector{}
    c.IncrementCounter("ignored", nil) // silently discarded
}
```

## Distributed Tracing

Initialize OpenTelemetry tracing:

```go
package main

import (
    "context"
    "digital.vasic.observability/pkg/trace"
)

func main() {
    cfg := &trace.TracerConfig{
        ServiceName:  "my-service",
        ExporterType: trace.ExporterOTLP,
        Endpoint:     "localhost:4318",
    }

    tracer, err := trace.InitTracer(cfg)
    if err != nil {
        panic(err)
    }
    defer tracer.Shutdown(context.Background())

    // Create a span
    ctx, span := tracer.StartSpan(context.Background(), "process-request")
    defer span.End()

    // Child span
    _, childSpan := tracer.StartSpan(ctx, "database-query")
    childSpan.End()
}
```

# User Guide

## Introduction

`digital.vasic.observability` is a generic Go module that provides five observability pillars: distributed tracing, metrics collection, structured logging, health check aggregation, and analytics. Each package exposes a clean interface with production-ready implementations and no-op fallbacks for testing or disabled subsystems.

## Installation

```bash
go get digital.vasic.observability
```

Requires Go 1.24 or later.

## Distributed Tracing (`pkg/trace`)

The `trace` package wraps OpenTelemetry with a simplified API for creating spans, recording errors, and tracing functions.

### Initialization

```go
import "digital.vasic.observability/pkg/trace"

// Create a tracer with stdout exporter (good for development)
tracer, err := trace.InitTracer(&trace.TracerConfig{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Environment:    "development",
    ExporterType:   trace.ExporterStdout,
    SampleRate:     1.0,
})
if err != nil {
    log.Fatal(err)
}
defer tracer.Shutdown(context.Background())
```

### Exporter Types

| Constant | Description |
|----------|-------------|
| `ExporterStdout` | Human-readable output to stdout |
| `ExporterOTLP` | OpenTelemetry Protocol (OTLP) over HTTP |
| `ExporterJaeger` | Jaeger backend via OTLP |
| `ExporterZipkin` | Zipkin backend via OTLP |
| `ExporterNone` | Disables tracing (NeverSample sampler) |

### Creating Spans

```go
// Basic span
ctx, span := tracer.StartSpan(ctx, "process.request",
    attribute.String("request.id", reqID),
)
defer span.End()

// Client span for outgoing calls
ctx, span := tracer.StartClientSpan(ctx, "http.call",
    attribute.String("http.url", url),
)
defer trace.EndSpanWithError(span, err)

// Internal span for in-process operations
ctx, span := tracer.StartInternalSpan(ctx, "validate.input")
defer span.End()
```

### Tracing Functions

```go
// Trace a function that returns an error
err := tracer.TraceFunc(ctx, "db.query", func(ctx context.Context) error {
    return db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
})

// Trace a function that returns a value and an error (generic)
user, err := trace.TraceFuncWithResult(tracer, ctx, "get.user",
    func(ctx context.Context) (*User, error) {
        return userRepo.FindByID(ctx, userID)
    },
)
```

### Timed Spans

```go
// Automatically records duration_seconds attribute
ctx, finish := tracer.TimedSpan(ctx, "slow.operation")
defer finish(nil)  // pass error if applicable

// The span will have a duration_seconds float64 attribute
```

### Error Recording

```go
ctx, span := tracer.StartSpan(ctx, "operation")
defer span.End()

if err := doWork(ctx); err != nil {
    trace.RecordError(span, err)  // Records error + sets span status
    return err
}
trace.SetOK(span)  // Explicitly mark success
```

### OTLP Configuration

```go
tracer, err := trace.InitTracer(&trace.TracerConfig{
    ServiceName:  "my-service",
    ExporterType: trace.ExporterOTLP,
    Endpoint:     "otel-collector:4318",
    SampleRate:   0.5,   // Sample 50% of traces
    Insecure:     true,  // Disable TLS (for local dev)
    Headers: map[string]string{
        "Authorization": "Bearer token",
    },
})
```

### Accessing the Underlying Provider

```go
// Get the OpenTelemetry TracerProvider for advanced use cases
provider := tracer.Provider()
```

---

## Metrics Collection (`pkg/metrics`)

The `metrics` package provides Prometheus-based counters, histograms, and gauges with automatic registration.

### Initialization

```go
import "digital.vasic.observability/pkg/metrics"

// Default Prometheus registry
collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
    Namespace: "myapp",
    Subsystem: "api",
})

// Custom registry (useful for testing)
reg := prometheus.NewRegistry()
collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
    Namespace: "myapp",
    Registry:  reg,
})
```

### Recording Metrics

Metrics are automatically created on first use. No pre-registration required.

```go
// Increment a counter by 1
collector.IncrementCounter("requests_total", map[string]string{
    "method": "GET",
    "status": "200",
})

// Add an arbitrary value to a counter
collector.AddCounter("bytes_sent_total", 1024, map[string]string{
    "endpoint": "/api/v1/data",
})

// Record a latency observation (converts to seconds)
collector.RecordLatency("request_duration_seconds", 150*time.Millisecond,
    map[string]string{"endpoint": "/api/v1/users"},
)

// Record a raw float64 value in a histogram
collector.RecordValue("response_size_bytes", 4096, map[string]string{
    "content_type": "application/json",
})

// Set a gauge value
collector.SetGauge("active_connections", 42, map[string]string{
    "pool": "primary",
})
```

### Pre-Registration

For better control over help text, label names, and histogram buckets:

```go
collector.RegisterCounter(
    "requests_total",
    "Total number of HTTP requests",
    []string{"method", "status"},
)

collector.RegisterHistogram(
    "request_duration_seconds",
    "HTTP request duration in seconds",
    []string{"endpoint"},
    []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
)

collector.RegisterGauge(
    "active_connections",
    "Number of active connections",
    []string{"pool"},
)
```

### No-Op Collector

For tests or when metrics are disabled:

```go
var c metrics.Collector = &metrics.NoOpCollector{}
c.IncrementCounter("anything", nil)  // silently discarded
```

---

## Structured Logging (`pkg/logging`)

The `logging` package provides a `Logger` interface backed by logrus with correlation ID propagation.

### Initialization

```go
import "digital.vasic.observability/pkg/logging"

logger := logging.NewLogrusAdapter(&logging.Config{
    Level:       logging.DebugLevel,
    Format:      "json",        // "json" or "text"
    ServiceName: "my-service",  // included in every entry
    Output:      os.Stdout,     // defaults to os.Stderr
})
```

### Log Levels

| Constant | Description |
|----------|-------------|
| `DebugLevel` | Most verbose, for development diagnostics |
| `InfoLevel` | Default, general operational messages |
| `WarnLevel` | Potentially problematic situations |
| `ErrorLevel` | Errors that need attention |

### Structured Fields

```go
// Single field
logger.WithField("user_id", "u-123").Info("User authenticated")

// Multiple fields
logger.WithFields(map[string]interface{}{
    "request_id": "req-abc",
    "latency_ms": 150,
    "method":     "POST",
}).Info("Request completed")

// Error field
logger.WithError(err).Error("Database connection failed")
```

### Correlation ID Propagation

Correlation IDs flow through `context.Context` for request tracing across service boundaries.

```go
// Store correlation ID in context
ctx := logging.ContextWithCorrelationID(ctx, "corr-abc-123")

// Create a logger enriched from context
enriched := logging.WithContext(logger, ctx)
enriched.Info("Processing request")
// Output: {"correlation_id": "corr-abc-123", "msg": "Processing request", ...}

// Extract correlation ID from context
id := logging.CorrelationIDFromContext(ctx)  // "corr-abc-123"

// Direct correlation ID on logger
logger.WithCorrelationID("corr-abc-123").Info("Direct usage")
```

### Chaining

Logger methods return new Logger instances, so they can be chained:

```go
logger.
    WithCorrelationID("corr-123").
    WithField("component", "auth").
    WithError(err).
    Error("Authentication failed")
```

### No-Op Logger

```go
var l logging.Logger = &logging.NoOpLogger{}
l.Info("silently discarded")
```

---

## Health Checks (`pkg/health`)

The `health` package aggregates health checks from multiple components into a single system status.

### Initialization

```go
import "digital.vasic.observability/pkg/health"

agg := health.NewAggregator(&health.AggregatorConfig{
    Timeout: 3 * time.Second,  // per-check timeout, defaults to 5s
})
```

### Registering Components

```go
// Required: failure makes the system "unhealthy"
agg.Register("database", func(ctx context.Context) error {
    return db.PingContext(ctx)
})

agg.Register("redis", func(ctx context.Context) error {
    return redis.Ping(ctx).Err()
})

// Optional: failure makes the system "degraded" (not "unhealthy")
agg.RegisterOptional("cache-l2", func(ctx context.Context) error {
    return memcached.Ping()
})

agg.RegisterOptional("search-index", func(ctx context.Context) error {
    resp, err := http.Get("http://elasticsearch:9200/_cluster/health")
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return fmt.Errorf("elasticsearch unhealthy: %d", resp.StatusCode)
    }
    return nil
})
```

### Running Health Checks

All checks run in parallel with the configured timeout:

```go
report := agg.Check(ctx)

fmt.Printf("Overall: %s\n", report.Status)
// "healthy", "degraded", or "unhealthy"

for _, comp := range report.Components {
    fmt.Printf("  %s: %s (%s)\n", comp.Name, comp.Status, comp.Duration)
    if comp.Message != "" {
        fmt.Printf("    Message: %s\n", comp.Message)
    }
}
```

### Status Aggregation Rules

| Scenario | Overall Status |
|----------|---------------|
| All components healthy | `healthy` |
| Required component fails | `unhealthy` |
| Only optional component fails | `degraded` |
| Multiple optional failures | `degraded` |
| Required + optional both fail | `unhealthy` |

### Utility Functions

```go
// Static check (always returns the given error, or nil for healthy)
agg.Register("always-ok", health.StaticCheck(nil))
agg.Register("always-fail", health.StaticCheck(fmt.Errorf("down")))

// Component count
count := agg.ComponentCount()
```

### JSON Serialization

Both `Report` and `ComponentResult` have JSON tags for HTTP endpoint integration:

```go
report := agg.Check(ctx)
json.NewEncoder(w).Encode(report)
```

Example output:

```json
{
  "status": "degraded",
  "components": [
    {"name": "database", "status": "healthy", "duration": 1200000, "last_checked": "..."},
    {"name": "cache", "status": "unhealthy", "message": "connection refused", "duration": 5000000000, "last_checked": "..."}
  ],
  "timestamp": "2025-01-15T10:30:00Z"
}
```

---

## Analytics (`pkg/analytics`)

The `analytics` package provides ClickHouse-backed event tracking with automatic NoOp fallback.

### Initialization

```go
import "digital.vasic.observability/pkg/analytics"

// Automatic fallback: tries ClickHouse, falls back to NoOp
collector := analytics.NewCollector(&analytics.ClickHouseConfig{
    Host:     "localhost",
    Port:     9000,
    Database: "analytics",
    Table:    "events",
    Username: "default",
    Password: "",
}, logger)
defer collector.Close()
```

If ClickHouse is not available, `NewCollector` logs a warning and returns a `NoOpCollector` that silently discards all events.

### Tracking Events

```go
// Single event
err := collector.Track(ctx, analytics.Event{
    Name:      "request.completed",
    Timestamp: time.Now(),
    Properties: map[string]interface{}{
        "duration_ms": 250,
        "status_code": 200,
    },
    Tags: map[string]string{
        "service":  "api",
        "endpoint": "/v1/users",
    },
})

// Batch tracking
events := []analytics.Event{
    {Name: "llm.request", Timestamp: time.Now(), Tags: map[string]string{"provider": "claude"}},
    {Name: "llm.request", Timestamp: time.Now(), Tags: map[string]string{"provider": "gemini"}},
}
err := collector.TrackBatch(ctx, events)
```

### Querying Aggregated Statistics

```go
stats, err := collector.Query(ctx, "events", "name", 24*time.Hour)
for _, s := range stats {
    fmt.Printf("%s: %d events\n", s.Group, s.TotalCount)
}
```

The `AggregatedStats` struct includes fields for `TotalCount`, `AvgDuration`, `P95Duration`, `P99Duration`, `ErrorRate`, `Period`, and `Extra`.

### Raw Read Queries

The `ClickHouseCollector` supports arbitrary SELECT queries:

```go
if chCollector, ok := collector.(*analytics.ClickHouseCollector); ok {
    results, err := chCollector.ExecuteReadQuery(ctx,
        "SELECT name, count() FROM events WHERE timestamp > ? GROUP BY name",
        time.Now().Add(-1*time.Hour),
    )
    // results is []map[string]interface{}
}
```

Only SELECT queries are permitted; other statements are rejected.

### No-Op Collector

```go
var c analytics.Collector = &analytics.NoOpCollector{}
c.Track(ctx, event)  // silently discarded
c.Query(ctx, "events", "name", time.Hour)  // returns nil, nil
```

---

## Putting It All Together

Here is a complete example combining all five packages:

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "digital.vasic.observability/pkg/analytics"
    "digital.vasic.observability/pkg/health"
    "digital.vasic.observability/pkg/logging"
    "digital.vasic.observability/pkg/metrics"
    "digital.vasic.observability/pkg/trace"
)

func main() {
    ctx := context.Background()

    // 1. Logging
    logger := logging.NewLogrusAdapter(&logging.Config{
        Level:       logging.InfoLevel,
        Format:      "json",
        ServiceName: "my-service",
    })

    // 2. Tracing
    tracer, err := trace.InitTracer(&trace.TracerConfig{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        ExporterType:   trace.ExporterOTLP,
        Endpoint:       "localhost:4318",
        SampleRate:     1.0,
        Insecure:       true,
    })
    if err != nil {
        logger.WithError(err).Error("Failed to init tracer")
        return
    }
    defer tracer.Shutdown(ctx)

    // 3. Metrics
    collector := metrics.NewPrometheusCollector(&metrics.PrometheusConfig{
        Namespace: "myapp",
    })

    // 4. Health checks
    healthAgg := health.NewAggregator(&health.AggregatorConfig{
        Timeout: 3 * time.Second,
    })
    healthAgg.Register("self", health.StaticCheck(nil))

    // 5. Analytics (falls back to NoOp if ClickHouse unavailable)
    analyticsCollector := analytics.NewCollector(&analytics.ClickHouseConfig{
        Host: "localhost", Port: 9000, Database: "analytics", Table: "events",
    }, nil)
    defer analyticsCollector.Close()

    // Use in a request handler
    http.HandleFunc("/api/v1/process", func(w http.ResponseWriter, r *http.Request) {
        ctx := logging.ContextWithCorrelationID(r.Context(), r.Header.Get("X-Correlation-ID"))
        reqLogger := logging.WithContext(logger, ctx)

        ctx, span := tracer.StartSpan(ctx, "handle.process")
        defer span.End()

        start := time.Now()
        reqLogger.Info("Processing request")

        // ... business logic ...

        duration := time.Since(start)
        collector.IncrementCounter("requests_total", map[string]string{
            "method": r.Method, "status": "200",
        })
        collector.RecordLatency("request_duration_seconds", duration, map[string]string{
            "endpoint": "/api/v1/process",
        })

        _ = analyticsCollector.Track(ctx, analytics.Event{
            Name:      "request.completed",
            Timestamp: time.Now(),
            Properties: map[string]interface{}{"duration_ms": duration.Milliseconds()},
        })

        fmt.Fprintln(w, "OK")
    })

    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        report := healthAgg.Check(r.Context())
        if report.Status == health.StatusUnhealthy {
            w.WriteHeader(503)
        }
        fmt.Fprintf(w, "status: %s\n", report.Status)
    })

    logger.Info("Server starting on :8080")
    http.ListenAndServe(":8080", nil)
}
```

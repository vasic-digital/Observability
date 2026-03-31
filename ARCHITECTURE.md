# Architecture -- Observability

## Purpose

Generic, reusable Go module for application observability: distributed tracing with OpenTelemetry (OTLP, Jaeger, Zipkin, stdout exporters), Prometheus metrics collection with auto-registration, structured logging with correlation ID propagation, health check aggregation with required/optional component support, and ClickHouse analytics with graceful NoOp fallback.

## Structure

```
pkg/
  trace/      OpenTelemetry tracing: InitTracer, StartSpan, TraceFunc, TimedSpan, multiple exporter backends
  metrics/    Prometheus metrics: counters, histograms, gauges with auto-creation on use
  logging/    Structured logging: LogrusAdapter, correlation ID via context, WithField/WithError
  health/     Health check aggregation: required (failure=unhealthy) and optional (failure=degraded) components
  analytics/  ClickHouse analytics: Track, TrackBatch, Query with NoOp fallback when unavailable
```

## Key Components

- **`trace.Tracer`** -- OpenTelemetry tracer with pluggable exporters (OTLP, Jaeger, Zipkin, Stdout, None). Provides StartSpan, StartClientSpan, TimedSpan, TraceFunc, TraceFuncWithResult
- **`metrics.PrometheusCollector`** -- Auto-registers counters, histograms, and gauges on first use. Thread-safe. NoOpCollector available for tests
- **`logging.LogrusAdapter`** -- Implements Logger interface with JSON/text formatting, correlation ID injection via context, and field-based structured logging
- **`health.Aggregator`** -- Runs health checks in parallel with timeout. Required checks determine healthy/unhealthy; optional checks determine degraded status
- **`analytics.Collector`** -- ClickHouse event tracking with batch support. Auto-falls back to NoOp if ClickHouse is unavailable

## Data Flow

```
Tracing: tracer.StartSpan(ctx, "op") -> span with trace/span IDs -> exporter (OTLP/Jaeger/stdout)
Metrics: collector.IncrementCounter("requests", labels) -> auto-register -> Prometheus registry
Logging: logger.WithCorrelationID("req-123").Info("msg") -> JSON output with correlation_id field
Health:  aggregator.Check(ctx) -> parallel health probes -> aggregate Report{Status, Components}
Analytics: collector.Track(ctx, event) -> ClickHouse INSERT (or NoOp)
```

## Dependencies

- `go.opentelemetry.io/otel` -- OpenTelemetry API and SDK
- `github.com/prometheus/client_golang` -- Prometheus metrics
- `github.com/sirupsen/logrus` -- Structured logging
- `github.com/stretchr/testify` -- Test assertions

## Testing Strategy

Table-driven tests with `testify` and race detection. Tracing tests use stdout exporter. Metrics tests use custom Prometheus registries. Health check tests use mock check functions. Analytics tests verify NoOp fallback. Correlation ID propagation tested via context round-trip.

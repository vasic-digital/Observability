# Observability Module

`digital.vasic.observability` is a generic, reusable Go module for application observability. It provides distributed tracing, Prometheus metrics collection, structured logging with correlation IDs, health check aggregation, and ClickHouse-backed analytics.

## Key Features

- **Distributed tracing** -- OpenTelemetry integration with pluggable exporters (OTLP, Jaeger, Zipkin, stdout, none)
- **Prometheus metrics** -- Counter, histogram, and gauge collection with double-checked locking for thread safety
- **Structured logging** -- Logrus adapter with correlation ID propagation via context
- **Health checks** -- Aggregator that runs checks in parallel with required/optional component distinction
- **Analytics** -- ClickHouse adapter with graceful degradation to NoOp when unavailable
- **Gin integration** -- Middleware adapters for Gin framework
- **Null objects** -- Every package provides a no-op implementation for disabled subsystems

## Package Overview

| Package | Purpose |
|---------|---------|
| `pkg/trace` | OpenTelemetry tracing with configurable exporters |
| `pkg/metrics` | Prometheus metrics (counters, histograms, gauges) |
| `pkg/logging` | Structured logging with correlation ID support |
| `pkg/health` | Health check aggregation with required/optional components |
| `pkg/analytics` | ClickHouse analytics with NoOp fallback |
| `pkg/gin` | Gin framework middleware adapters |
| `pkg/middleware` | Generic HTTP middleware for observability |

## Installation

```bash
go get digital.vasic.observability
```

Requires Go 1.24 or later.

## Dependencies

| Dependency | Used By | Purpose |
|-----------|---------|---------|
| `go.opentelemetry.io/otel` | `trace` | Distributed tracing SDK |
| `github.com/prometheus/client_golang` | `metrics` | Prometheus metrics |
| `github.com/sirupsen/logrus` | `logging`, `analytics` | Structured logging |
| `github.com/ClickHouse/clickhouse-go/v2` | `analytics` | ClickHouse driver |

The `health` package has zero external dependencies -- it uses only the Go standard library.

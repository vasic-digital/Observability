# CLAUDE.md - Observability Module

## Overview

`digital.vasic.observability` is a generic, reusable Go module for application observability. It provides distributed tracing (OpenTelemetry), Prometheus metrics collection, structured logging with correlation IDs, health check aggregation, and ClickHouse-backed analytics.

**Module**: `digital.vasic.observability` (Go 1.24+)

## Build & Test

```bash
go build ./...
go test ./... -count=1 -race
go test ./... -short              # Unit tests only
go test -tags=integration ./...   # Integration tests
go test -bench=. ./tests/benchmark/
```

## Code Style

- Standard Go conventions, `gofmt` formatting
- Imports grouped: stdlib, third-party, internal (blank line separated)
- Line length <= 100 chars
- Naming: `camelCase` private, `PascalCase` exported, acronyms all-caps
- Errors: always check, wrap with `fmt.Errorf("...: %w", err)`
- Tests: table-driven, `testify`, naming `Test<Struct>_<Method>_<Scenario>`

## Package Structure

| Package | Purpose |
|---------|---------|
| `pkg/trace` | OpenTelemetry tracing with OTLP/Jaeger/Zipkin/stdout exporters |
| `pkg/metrics` | Prometheus metrics collection (counters, histograms, gauges) |
| `pkg/logging` | Structured logging with correlation ID support (logrus) |
| `pkg/health` | Health check aggregation with required/optional components |
| `pkg/analytics` | ClickHouse analytics adapter with NoOp fallback |

## Key Interfaces

- `metrics.Collector` -- IncrementCounter, AddCounter, RecordLatency, RecordValue, SetGauge
- `logging.Logger` -- Info, Warn, Error, Debug, WithField, WithFields, WithCorrelationID, WithError
- `health.Checker` -- Check(ctx) returning aggregated Report
- `analytics.Collector` -- Track, TrackBatch, Query, Close

## Design Patterns

- **Strategy**: Pluggable exporters (OTLP, Jaeger, Zipkin, Stdout, None)
- **Adapter**: LogrusAdapter for Logger interface, PrometheusCollector for Collector
- **Null Object**: NoOpCollector, NoOpLogger for disabled subsystems
- **Aggregator**: Health check combining required/optional component results
- **Graceful Degradation**: Analytics falls back to NoOp when ClickHouse unavailable

## Commit Style

Conventional Commits: `feat(trace): add OTLP exporter support`

# Changelog

All notable changes to `digital.vasic.observability` will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-01-01

### Added

#### Package `trace`
- `Tracer` struct wrapping OpenTelemetry TracerProvider with convenience methods.
- `InitTracer` factory with configurable exporters (OTLP, Jaeger, Zipkin, Stdout, None).
- `StartSpan`, `StartClientSpan`, `StartInternalSpan` for creating spans of different kinds.
- `TraceFunc` for tracing a function that returns an error.
- `TraceFuncWithResult` generic function for tracing functions with return values.
- `TimedSpan` for automatic duration recording on spans.
- `RecordError`, `SetOK`, `EndSpanWithError` helper functions for span status management.
- `DefaultConfig` returning sensible tracer defaults.
- `TracerConfig` with fields for service name, version, environment, exporter type, endpoint, sample rate, headers, and TLS settings.

#### Package `metrics`
- `Collector` interface with `IncrementCounter`, `AddCounter`, `RecordLatency`, `RecordValue`, `SetGauge`.
- `PrometheusCollector` implementing `Collector` with automatic metric creation on first use.
- `RegisterCounter`, `RegisterHistogram`, `RegisterGauge` for explicit pre-registration.
- `PrometheusConfig` with namespace, subsystem, default buckets, and custom registry support.
- `NoOpCollector` null object implementation.
- `DefaultPrometheusConfig` returning sensible defaults.
- Thread-safe double-checked locking for lazy metric creation.

#### Package `logging`
- `Logger` interface with `Info`, `Warn`, `Error`, `Debug`, `WithField`, `WithFields`, `WithCorrelationID`, `WithError`.
- `LogrusAdapter` implementing `Logger` with logrus backend.
- `Config` with level, format (JSON/text), output writer, and service name.
- `ContextWithCorrelationID` and `CorrelationIDFromContext` for context-based correlation ID propagation.
- `WithContext` helper to enrich a Logger from context.
- `NoOpLogger` null object implementation.
- `Level` type with `DebugLevel`, `InfoLevel`, `WarnLevel`, `ErrorLevel` constants.
- `DefaultConfig` returning sensible defaults.

#### Package `health`
- `Checker` interface with `Check` method returning `*Report`.
- `Aggregator` implementing `Checker` with parallel check execution.
- `Register` for required components (failure = unhealthy).
- `RegisterOptional` for optional components (failure = degraded).
- `ComponentCount` for querying registered component count.
- `AggregatorConfig` with per-check timeout (default 5 seconds).
- `Report` and `ComponentResult` structs with JSON tags for HTTP endpoint integration.
- `Status` type with `StatusHealthy`, `StatusDegraded`, `StatusUnhealthy` constants.
- `CheckFunc` type for component health check functions.
- `StaticCheck` factory for fixed-status checks.
- `TCPCheck` placeholder documenting the TCP check pattern.
- `DefaultAggregatorConfig` returning sensible defaults.

#### Package `analytics`
- `Collector` interface with `Track`, `TrackBatch`, `Query`, `Close`.
- `ClickHouseCollector` implementing `Collector` with ClickHouse storage.
- `ExecuteReadQuery` for arbitrary read-only SQL queries (SELECT only).
- `NewCollector` factory with automatic NoOp fallback on ClickHouse failure.
- `NewClickHouseCollector` for direct ClickHouse collector creation.
- `ClickHouseConfig` with host, port, database, credentials, TLS, and table settings.
- `Event` struct with name, timestamp, properties, and tags.
- `AggregatedStats` struct with count, duration percentiles, error rate, and extras.
- `NoOpCollector` null object implementation.
- SQL injection prevention via `isValidIdentifier` validation.
- Transaction-based batch inserts.

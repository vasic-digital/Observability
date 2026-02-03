# Architecture

## Design Philosophy

The `digital.vasic.observability` module is designed around three principles:

1. **Independence** -- Each package stands alone with no inter-package dependencies. Consumers import only what they need.
2. **Interface-first** -- Every package defines a minimal interface. Production implementations, no-op fallbacks, and test doubles all satisfy the same contract.
3. **Zero mandatory infrastructure** -- Every package degrades gracefully when its backing infrastructure is unavailable, through no-op implementations or disabled samplers.

## Package Overview

```
digital.vasic.observability
  pkg/
    trace/       OpenTelemetry distributed tracing
    metrics/     Prometheus metrics collection
    logging/     Structured logging with correlation IDs
    health/      Health check aggregation
    analytics/   ClickHouse analytics with NoOp fallback
```

Each package is independent. There are no imports between packages. This is intentional: it allows consumers to use any subset of packages without pulling in unwanted dependencies.

## Design Patterns

### Strategy Pattern (trace)

The `trace` package uses the Strategy pattern for exporter selection. The `ExporterType` field in `TracerConfig` selects the concrete exporter at initialization time:

- `ExporterStdout` -- `stdouttrace.New()` for development
- `ExporterOTLP` / `ExporterJaeger` / `ExporterZipkin` -- `otlptracehttp.New()` with backend-specific configuration
- `ExporterNone` -- `NeverSample()` sampler, no exporter

The `Tracer` struct wraps all exporters behind the same `trace.Tracer` interface from OpenTelemetry, so callers are unaware of which exporter is active.

**Rationale**: Allows switching between development (stdout), testing (none), and production (OTLP) exporters with a single config change, no code changes.

### Adapter Pattern (logging, metrics)

**LogrusAdapter** adapts the third-party `logrus.Entry` to the module's `Logger` interface. This decouples consumers from logrus. If a future migration to `slog` or `zerolog` is needed, only the adapter changes.

**PrometheusCollector** adapts Prometheus counter/histogram/gauge vector types to the module's `Collector` interface. Consumers call `IncrementCounter(name, labels)` without knowing about `prometheus.CounterVec`.

**Rationale**: Isolates third-party library details behind stable interfaces. Consumers program against `Logger` and `Collector`, not logrus or Prometheus directly.

### Null Object Pattern (all packages)

Every package provides a no-op implementation of its interface:

| Package | Null Object | Behavior |
|---------|-------------|----------|
| `metrics` | `NoOpCollector` | Silently discards all metric recordings |
| `logging` | `NoOpLogger` | Discards all log output, returns itself on `WithField` |
| `health` | (not needed) | `Aggregator` with no components returns healthy |
| `analytics` | `NoOpCollector` | Discards events, returns empty results |
| `trace` | `ExporterNone` | Uses `NeverSample()`, creates no-op spans via OTel |

**Rationale**: Eliminates nil checks throughout consumer code. Consumers can always safely call interface methods regardless of whether the subsystem is enabled.

### Aggregator Pattern (health)

The `Aggregator` collects multiple `CheckFunc` registrations and executes them in parallel during `Check()`. Results are combined into a single `Report` with deterministic status aggregation:

1. If any **required** component is unhealthy, overall status is `unhealthy`.
2. If any **optional** component is unhealthy (and no required failures), overall status is `degraded`.
3. Otherwise, overall status is `healthy`.

Each check runs in its own goroutine with a per-check timeout enforced via `context.WithTimeout`.

**Rationale**: Services typically depend on multiple components (database, cache, message queue). The Aggregator provides a single health endpoint that reflects the true system state, distinguishing between critical failures (required) and reduced functionality (optional).

### Graceful Degradation Pattern (analytics)

The `NewCollector` factory function attempts to connect to ClickHouse. If the connection fails, it logs a warning and returns a `NoOpCollector` instead of an error:

```go
func NewCollector(config *ClickHouseConfig, logger *logrus.Logger) Collector {
    collector, err := NewClickHouseCollector(*config, logger)
    if err != nil {
        logger.WithError(err).Warn("ClickHouse unavailable, using no-op")
        return &NoOpCollector{}
    }
    return collector
}
```

**Rationale**: Analytics should never block application startup or cause failures in the critical path. If ClickHouse is down, the application continues normally -- analytics data is lost, but the service remains operational.

## Concurrency Model

All public types are safe for concurrent use from multiple goroutines:

### Tracer (`pkg/trace`)

- `sync.RWMutex` protects the `provider` and `tracer` fields.
- `StartSpan`, `StartClientSpan`, `StartInternalSpan` acquire a read lock.
- `Shutdown` acquires a write lock.
- The underlying OpenTelemetry `TracerProvider` is itself thread-safe.

### PrometheusCollector (`pkg/metrics`)

- `sync.RWMutex` protects the `counters`, `histograms`, and `gauges` maps.
- Uses **double-checked locking** in `getOrCreateX` methods: first attempt with `RLock`, then promote to `Lock` only if creation is needed.
- Prometheus metric vectors (`CounterVec`, `HistogramVec`, `GaugeVec`) are thread-safe once registered.

### LogrusAdapter (`pkg/logging`)

- `sync.RWMutex` protects the `logger` entry.
- `WithField` / `WithFields` / `WithCorrelationID` / `WithError` return **new** `LogrusAdapter` instances, so they do not mutate the original.
- The underlying logrus logger is itself thread-safe.

### Aggregator (`pkg/health`)

- `sync.RWMutex` protects the `components` slice.
- `Register` / `RegisterOptional` acquire a write lock.
- `Check` copies the component slice under a read lock, then launches goroutines for each check.
- `sync.WaitGroup` synchronizes parallel check completion.

### ClickHouseCollector (`pkg/analytics`)

- `sync.RWMutex` protects the config.
- `database/sql.DB` is thread-safe by design.
- Batch inserts use transactions for atomicity.

## Resource Management

### Shutdown Sequence

Consumers are responsible for calling shutdown/close methods:

1. `tracer.Shutdown(ctx)` -- flushes pending spans and shuts down the TracerProvider.
2. `analyticsCollector.Close()` -- closes the ClickHouse connection.
3. Metrics and logging have no explicit shutdown (Prometheus registry persists; logrus flushes on write).
4. Health aggregator has no resources to release.

### Context Propagation

- **Tracing**: Spans are propagated via `context.Context`. `StartSpan` embeds the new span in the returned context. Child operations that accept this context automatically become child spans.
- **Logging**: Correlation IDs are stored in `context.Context` via `ContextWithCorrelationID` and extracted via `CorrelationIDFromContext`. The `WithContext` helper enriches a logger from the context.
- **Health**: `Check` passes the caller's context to each `CheckFunc`, enabling cancellation propagation.
- **Analytics**: `Track` and `Query` accept `context.Context` for cancellation and deadline propagation to ClickHouse.

## Dependency Graph

```
trace       --> go.opentelemetry.io/otel (SDK, exporters)
metrics     --> github.com/prometheus/client_golang
logging     --> github.com/sirupsen/logrus
health      --> (stdlib only)
analytics   --> github.com/ClickHouse/clickhouse-go/v2, github.com/sirupsen/logrus
```

The `health` package has zero third-party dependencies -- it uses only the Go standard library. This makes it suitable for the most constrained environments.

## SQL Safety (analytics)

The `Query` method constructs SQL dynamically using the `groupBy` and `table` parameters. To prevent SQL injection, both parameters are validated by `isValidIdentifier`, which permits only alphanumeric characters and underscores. Invalid identifiers cause `Query` to return an error before any SQL is executed.

The `ExecuteReadQuery` method accepts arbitrary SQL but enforces a read-only constraint: only queries beginning with `SELECT` (case-insensitive) are permitted.

## Error Handling

- **trace**: `InitTracer` returns errors for unsupported exporter types or resource creation failures. Span operations (start, end, record) never return errors.
- **metrics**: Registration methods return errors on Prometheus registration failures. Recording methods (`IncrementCounter`, `RecordLatency`, etc.) silently skip if the metric cannot be created.
- **logging**: No methods return errors. Log output failures are handled by logrus internally.
- **health**: `CheckFunc` errors are captured in `ComponentResult.Message` and reflected in the status. The `Check` method never returns an error.
- **analytics**: `Track`, `TrackBatch`, `Query`, and `ExecuteReadQuery` return errors. `NewCollector` swallows connection errors and falls back to NoOp.

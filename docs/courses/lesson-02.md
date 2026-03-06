# Lesson 2: Prometheus Metrics and Structured Logging

## Learning Objectives

- Build a Prometheus metrics collector with double-checked locking for thread-safe metric registration
- Implement the Adapter pattern to decouple logging from a specific framework
- Use correlation IDs for request tracing across log entries

## Key Concepts

- **Adapter Pattern (Logging)**: `LogrusAdapter` wraps `logrus.Entry` behind a `Logger` interface. Consumers call interface methods, not logrus directly. Switching to `slog` or `zerolog` requires only a new adapter.
- **Adapter Pattern (Metrics)**: `PrometheusCollector` wraps Prometheus counter/histogram/gauge vector types behind a `Collector` interface. Consumers call `IncrementCounter(name, labels)` without knowing about `prometheus.CounterVec`.
- **Double-Checked Locking**: `getOrCreateCounter` first tries with `RLock`. If the metric does not exist, it promotes to `Lock` and checks again before creating. This minimizes write lock contention.
- **Null Object Pattern**: `NoOpCollector` and `NoOpLogger` silently discard all operations, eliminating nil checks in consumer code.

## Code Walkthrough

### Source: `pkg/metrics/metrics.go`

The `PrometheusCollector` manages three maps of Prometheus vectors:

```go
type PrometheusCollector struct {
    mu         sync.RWMutex
    counters   map[string]*prometheus.CounterVec
    histograms map[string]*prometheus.HistogramVec
    gauges     map[string]*prometheus.GaugeVec
}
```

Methods like `IncrementCounter`, `RecordLatency`, and `SetGauge` transparently create and register metrics on first use. The double-checked locking pattern avoids holding a write lock for every call.

### Source: `pkg/logging/logging.go`

The `Logger` interface and `LogrusAdapter`:

- `WithField`/`WithFields` return new adapter instances (immutable pattern)
- `WithCorrelationID` enriches logs with a request correlation ID
- Context integration via `ContextWithCorrelationID` and `CorrelationIDFromContext`

### Source: `pkg/metrics/metrics_test.go` and `pkg/logging/logging_test.go`

Tests verify metric creation and recording, concurrent metric updates, logger field immutability, and correlation ID propagation.

## Practice Exercise

1. Create a `PrometheusCollector` and register three metrics: a request counter, a latency histogram, and a connection gauge. Increment, record, and set values, then verify via the Prometheus registry.
2. Build a logging middleware that creates a `LogrusAdapter` with a correlation ID for each request. Log the request start and end, and verify the correlation ID appears in both log entries.
3. Write a test that creates 10 goroutines, each calling `IncrementCounter` 1000 times. Verify the final counter value is exactly 10000 (tests the double-checked locking correctness).

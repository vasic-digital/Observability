# API Reference

Complete reference for all exported types, functions, and methods in `digital.vasic.observability`.

---

## Package `trace`

```
import "digital.vasic.observability/pkg/trace"
```

OpenTelemetry-based distributed tracing with support for multiple exporters.

### Types

#### `ExporterType`

```go
type ExporterType string
```

Defines the type of trace exporter.

**Constants:**

| Constant | Value | Description |
|----------|-------|-------------|
| `ExporterStdout` | `"stdout"` | Human-readable output to stdout |
| `ExporterOTLP` | `"otlp"` | OpenTelemetry Protocol over HTTP |
| `ExporterJaeger` | `"jaeger"` | Jaeger backend via OTLP |
| `ExporterZipkin` | `"zipkin"` | Zipkin backend via OTLP |
| `ExporterNone` | `"none"` | Disables trace exporting (NeverSample) |

#### `TracerConfig`

```go
type TracerConfig struct {
    ServiceName    string
    ServiceVersion string
    Environment    string
    ExporterType   ExporterType
    Endpoint       string
    SampleRate     float64
    Headers        map[string]string
    Insecure       bool
}
```

Configures the tracer provider and exporter.

| Field | Type | Description |
|-------|------|-------------|
| `ServiceName` | `string` | Identifies the service in traces |
| `ServiceVersion` | `string` | Version of the service |
| `Environment` | `string` | Deployment environment (e.g., "production") |
| `ExporterType` | `ExporterType` | Which exporter to use |
| `Endpoint` | `string` | Collector endpoint (for OTLP/Jaeger/Zipkin) |
| `SampleRate` | `float64` | Fraction of traces sampled (0.0 to 1.0) |
| `Headers` | `map[string]string` | Additional HTTP headers for the exporter |
| `Insecure` | `bool` | Disables TLS for the exporter connection |

#### `Tracer`

```go
type Tracer struct {
    // unexported fields
}
```

Wraps an OpenTelemetry TracerProvider with convenience methods.

### Functions

#### `DefaultConfig`

```go
func DefaultConfig() *TracerConfig
```

Returns a `TracerConfig` with sensible defaults: service name `"service"`, version `"1.0.0"`, environment `"development"`, exporter `ExporterNone`, sample rate `1.0`.

#### `InitTracer`

```go
func InitTracer(config *TracerConfig) (*Tracer, error)
```

Creates and configures a new `Tracer`. Sets up the exporter, resource, sampler, and registers the provider as the global OpenTelemetry TracerProvider. If `config` is nil, uses `DefaultConfig()`.

#### `RecordError`

```go
func RecordError(span trace.Span, err error)
```

Records an error on the span and sets the span status to Error. No-op if `span` or `err` is nil.

#### `SetOK`

```go
func SetOK(span trace.Span)
```

Sets the span status to Ok. No-op if `span` is nil.

#### `EndSpanWithError`

```go
func EndSpanWithError(span trace.Span, err error)
```

Records an error (if non-nil) or sets status to OK, then ends the span. Intended for use with `defer`.

#### `TraceFuncWithResult`

```go
func TraceFuncWithResult[T any](
    t *Tracer,
    ctx context.Context,
    name string,
    fn func(ctx context.Context) (T, error),
    attrs ...attribute.KeyValue,
) (T, error)
```

Generic function. Executes `fn` within a traced span and returns the result. Records errors on the span.

### Methods on `*Tracer`

#### `Shutdown`

```go
func (t *Tracer) Shutdown(ctx context.Context) error
```

Gracefully flushes and shuts down the tracer provider.

#### `StartSpan`

```go
func (t *Tracer) StartSpan(
    ctx context.Context,
    name string,
    attrs ...attribute.KeyValue,
) (context.Context, trace.Span)
```

Creates a new span. The returned context carries the span for child operations.

#### `StartClientSpan`

```go
func (t *Tracer) StartClientSpan(
    ctx context.Context,
    name string,
    attrs ...attribute.KeyValue,
) (context.Context, trace.Span)
```

Creates a span with `SpanKindClient`, suitable for outgoing calls.

#### `StartInternalSpan`

```go
func (t *Tracer) StartInternalSpan(
    ctx context.Context,
    name string,
    attrs ...attribute.KeyValue,
) (context.Context, trace.Span)
```

Creates a span with `SpanKindInternal`, suitable for in-process operations.

#### `TraceFunc`

```go
func (t *Tracer) TraceFunc(
    ctx context.Context,
    name string,
    fn func(ctx context.Context) error,
    attrs ...attribute.KeyValue,
) error
```

Executes `fn` within a traced span. Records errors on the span.

#### `TimedSpan`

```go
func (t *Tracer) TimedSpan(
    ctx context.Context,
    name string,
    attrs ...attribute.KeyValue,
) (context.Context, func(err error))
```

Starts a span and returns a finish function that records `duration_seconds` as an attribute before ending the span.

#### `Provider`

```go
func (t *Tracer) Provider() *sdktrace.TracerProvider
```

Returns the underlying OpenTelemetry TracerProvider.

---

## Package `metrics`

```
import "digital.vasic.observability/pkg/metrics"
```

Prometheus-based metrics collection with a generic collector interface.

### Types

#### `Collector` (interface)

```go
type Collector interface {
    IncrementCounter(name string, labels map[string]string)
    AddCounter(name string, value float64, labels map[string]string)
    RecordLatency(name string, duration time.Duration, labels map[string]string)
    RecordValue(name string, value float64, labels map[string]string)
    SetGauge(name string, value float64, labels map[string]string)
}
```

Generic interface for recording application metrics.

| Method | Description |
|--------|-------------|
| `IncrementCounter` | Increments a named counter by 1 |
| `AddCounter` | Adds an arbitrary value to a counter |
| `RecordLatency` | Records a `time.Duration` in a histogram (converted to seconds) |
| `RecordValue` | Records a raw `float64` in a histogram |
| `SetGauge` | Sets the current value of a gauge |

#### `PrometheusConfig`

```go
type PrometheusConfig struct {
    Namespace      string
    Subsystem      string
    DefaultBuckets []float64
    Registry       prometheus.Registerer
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Namespace` | `string` | Prepended to all metric names |
| `Subsystem` | `string` | Inserted between namespace and name |
| `DefaultBuckets` | `[]float64` | Histogram bucket boundaries (nil = `prometheus.DefBuckets`) |
| `Registry` | `prometheus.Registerer` | Prometheus registry (nil = `DefaultRegisterer`) |

#### `PrometheusCollector`

```go
type PrometheusCollector struct {
    // unexported fields
}
```

Implements `Collector` using Prometheus. Thread-safe.

#### `NoOpCollector`

```go
type NoOpCollector struct{}
```

Implements `Collector`. Discards all metrics. Useful for tests and disabled environments.

### Functions

#### `DefaultPrometheusConfig`

```go
func DefaultPrometheusConfig() *PrometheusConfig
```

Returns a config with `prometheus.DefBuckets` and nil registry/namespace/subsystem.

#### `NewPrometheusCollector`

```go
func NewPrometheusCollector(config *PrometheusConfig) *PrometheusCollector
```

Creates a Prometheus-backed metrics collector. If `config` is nil, uses defaults.

### Methods on `*PrometheusCollector`

#### `RegisterCounter`

```go
func (c *PrometheusCollector) RegisterCounter(name, help string, labelNames []string) error
```

Pre-registers a counter. Returns nil if already registered. Returns error on Prometheus registration failure.

#### `RegisterHistogram`

```go
func (c *PrometheusCollector) RegisterHistogram(
    name, help string,
    labelNames []string,
    buckets []float64,
) error
```

Pre-registers a histogram. If `buckets` is empty, uses `DefaultBuckets` from config.

#### `RegisterGauge`

```go
func (c *PrometheusCollector) RegisterGauge(name, help string, labelNames []string) error
```

Pre-registers a gauge.

#### `IncrementCounter`

```go
func (c *PrometheusCollector) IncrementCounter(name string, labels map[string]string)
```

Increments a counter by 1. Auto-creates the counter if not pre-registered.

#### `AddCounter`

```go
func (c *PrometheusCollector) AddCounter(name string, value float64, labels map[string]string)
```

Adds a value to a counter. Auto-creates if needed.

#### `RecordLatency`

```go
func (c *PrometheusCollector) RecordLatency(
    name string, duration time.Duration, labels map[string]string,
)
```

Records a duration in seconds in a histogram. Delegates to `RecordValue`.

#### `RecordValue`

```go
func (c *PrometheusCollector) RecordValue(name string, value float64, labels map[string]string)
```

Records a float64 observation in a histogram. Auto-creates if needed.

#### `SetGauge`

```go
func (c *PrometheusCollector) SetGauge(name string, value float64, labels map[string]string)
```

Sets the current value of a gauge. Auto-creates if needed.

---

## Package `logging`

```
import "digital.vasic.observability/pkg/logging"
```

Structured logging with correlation ID support, backed by logrus.

### Types

#### `Logger` (interface)

```go
type Logger interface {
    Info(msg string)
    Warn(msg string)
    Error(msg string)
    Debug(msg string)
    WithField(key string, value interface{}) Logger
    WithFields(fields map[string]interface{}) Logger
    WithCorrelationID(id string) Logger
    WithError(err error) Logger
}
```

Structured logging interface. All implementations must be safe for concurrent use. `WithX` methods return new Logger instances without mutating the receiver.

#### `Level`

```go
type Level int
```

**Constants:**

| Constant | Value | Description |
|----------|-------|-------------|
| `DebugLevel` | `0` | Most verbose |
| `InfoLevel` | `1` | Default level |
| `WarnLevel` | `2` | Potentially problematic |
| `ErrorLevel` | `3` | Errors needing attention |

#### `Config`

```go
type Config struct {
    Level       Level
    Format      string
    Output      io.Writer
    ServiceName string
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Level` | `Level` | Minimum log level |
| `Format` | `string` | `"json"` or `"text"` |
| `Output` | `io.Writer` | Log output destination (default: `os.Stderr`) |
| `ServiceName` | `string` | Included as `"service"` field in every entry |

#### `LogrusAdapter`

```go
type LogrusAdapter struct {
    // unexported fields
}
```

Implements `Logger` using logrus. Thread-safe.

#### `NoOpLogger`

```go
type NoOpLogger struct{}
```

Implements `Logger`. Discards all output. `WithX` methods return the same instance.

### Functions

#### `DefaultConfig`

```go
func DefaultConfig() *Config
```

Returns a config with `InfoLevel` and `"json"` format.

#### `NewLogrusAdapter`

```go
func NewLogrusAdapter(config *Config) *LogrusAdapter
```

Creates a Logger backed by logrus. If `config` is nil, uses defaults.

#### `ContextWithCorrelationID`

```go
func ContextWithCorrelationID(ctx context.Context, id string) context.Context
```

Returns a new context with the correlation ID stored.

#### `CorrelationIDFromContext`

```go
func CorrelationIDFromContext(ctx context.Context) string
```

Extracts the correlation ID from the context. Returns empty string if not set.

#### `WithContext`

```go
func WithContext(logger Logger, ctx context.Context) Logger
```

Returns a Logger enriched with the correlation ID from the context, if present.

### Methods on `*LogrusAdapter`

#### `Info`

```go
func (a *LogrusAdapter) Info(msg string)
```

#### `Warn`

```go
func (a *LogrusAdapter) Warn(msg string)
```

#### `Error`

```go
func (a *LogrusAdapter) Error(msg string)
```

#### `Debug`

```go
func (a *LogrusAdapter) Debug(msg string)
```

#### `WithField`

```go
func (a *LogrusAdapter) WithField(key string, value interface{}) Logger
```

Returns a new Logger with the field added.

#### `WithFields`

```go
func (a *LogrusAdapter) WithFields(fields map[string]interface{}) Logger
```

Returns a new Logger with all fields added.

#### `WithCorrelationID`

```go
func (a *LogrusAdapter) WithCorrelationID(id string) Logger
```

Returns a new Logger with `"correlation_id"` set.

#### `WithError`

```go
func (a *LogrusAdapter) WithError(err error) Logger
```

Returns a new Logger with the error field set.

---

## Package `health`

```
import "digital.vasic.observability/pkg/health"
```

Health check aggregation for services with multiple dependencies.

### Types

#### `Status`

```go
type Status string
```

**Constants:**

| Constant | Value | Description |
|----------|-------|-------------|
| `StatusHealthy` | `"healthy"` | Fully operational |
| `StatusDegraded` | `"degraded"` | Operational with reduced capability |
| `StatusUnhealthy` | `"unhealthy"` | Not operational |

#### `CheckFunc`

```go
type CheckFunc func(ctx context.Context) error
```

Function that checks component health. Returns nil for healthy, error for unhealthy.

#### `ComponentResult`

```go
type ComponentResult struct {
    Name        string        `json:"name"`
    Status      Status        `json:"status"`
    Message     string        `json:"message,omitempty"`
    Duration    time.Duration `json:"duration"`
    LastChecked time.Time     `json:"last_checked"`
}
```

Health check result for a single component.

#### `Report`

```go
type Report struct {
    Status     Status            `json:"status"`
    Components []ComponentResult `json:"components"`
    Timestamp  time.Time         `json:"timestamp"`
}
```

Aggregated health check results with overall status.

#### `Checker` (interface)

```go
type Checker interface {
    Check(ctx context.Context) *Report
}
```

Contract for health check implementations.

#### `Aggregator`

```go
type Aggregator struct {
    // unexported fields
}
```

Combines multiple component health checks. Implements `Checker`. Thread-safe.

#### `AggregatorConfig`

```go
type AggregatorConfig struct {
    Timeout time.Duration
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Timeout` | `time.Duration` | Maximum duration per health check (default: 5s) |

### Functions

#### `DefaultAggregatorConfig`

```go
func DefaultAggregatorConfig() *AggregatorConfig
```

Returns a config with 5-second timeout.

#### `NewAggregator`

```go
func NewAggregator(config *AggregatorConfig) *Aggregator
```

Creates a new health check aggregator. If `config` is nil, uses defaults.

#### `StaticCheck`

```go
func StaticCheck(err error) CheckFunc
```

Returns a CheckFunc that always returns the given error (nil for healthy).

#### `TCPCheck`

```go
func TCPCheck(address string) CheckFunc
```

Returns a CheckFunc that documents the TCP check pattern. Note: this is a placeholder that returns an error directing callers to use `net.DialTimeout` in their own CheckFunc (to keep the health package dependency-free).

### Methods on `*Aggregator`

#### `Register`

```go
func (a *Aggregator) Register(name string, check CheckFunc)
```

Adds a **required** component. Failure makes overall status `unhealthy`.

#### `RegisterOptional`

```go
func (a *Aggregator) RegisterOptional(name string, check CheckFunc)
```

Adds an **optional** component. Failure makes overall status `degraded`.

#### `Check`

```go
func (a *Aggregator) Check(ctx context.Context) *Report
```

Runs all registered checks in parallel and returns an aggregated report.

#### `ComponentCount`

```go
func (a *Aggregator) ComponentCount() int
```

Returns the number of registered components.

---

## Package `analytics`

```
import "digital.vasic.observability/pkg/analytics"
```

ClickHouse analytics with NoOp fallback for event tracking and aggregation.

### Types

#### `Event`

```go
type Event struct {
    Name       string
    Timestamp  time.Time
    Properties map[string]interface{}
    Tags       map[string]string
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Event type identifier (e.g., `"request.completed"`) |
| `Timestamp` | `time.Time` | When the event occurred (auto-set to `time.Now()` if zero) |
| `Properties` | `map[string]interface{}` | Event-specific key-value data |
| `Tags` | `map[string]string` | String labels for filtering/grouping |

#### `AggregatedStats`

```go
type AggregatedStats struct {
    Group       string
    TotalCount  int64
    AvgDuration float64
    P95Duration float64
    P99Duration float64
    ErrorRate   float64
    Period      string
    Extra       map[string]interface{}
}
```

Aggregated statistics over a time window.

#### `Collector` (interface)

```go
type Collector interface {
    Track(ctx context.Context, event Event) error
    TrackBatch(ctx context.Context, events []Event) error
    Query(ctx context.Context, table string, groupBy string,
        window time.Duration) ([]AggregatedStats, error)
    Close() error
}
```

| Method | Description |
|--------|-------------|
| `Track` | Records a single event |
| `TrackBatch` | Records multiple events in one batch |
| `Query` | Retrieves aggregated stats grouped by a column within a time window |
| `Close` | Releases resources |

#### `ClickHouseConfig`

```go
type ClickHouseConfig struct {
    Host     string
    Port     int
    Database string
    Username string
    Password string
    TLS      bool
    Table    string
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Host` | `string` | ClickHouse server hostname |
| `Port` | `int` | ClickHouse server port |
| `Database` | `string` | Database name |
| `Username` | `string` | Authentication username |
| `Password` | `string` | Authentication password |
| `TLS` | `bool` | Enable encrypted connections |
| `Table` | `string` | Default table for event storage |

#### `ClickHouseCollector`

```go
type ClickHouseCollector struct {
    // unexported fields
}
```

Implements `Collector` using ClickHouse. Thread-safe.

#### `NoOpCollector`

```go
type NoOpCollector struct{}
```

Implements `Collector`. Discards all events and returns empty query results.

### Functions

#### `NewClickHouseCollector`

```go
func NewClickHouseCollector(
    config ClickHouseConfig,
    logger *logrus.Logger,
) (*ClickHouseCollector, error)
```

Creates a ClickHouse-backed collector. Returns error if connection or ping fails.

#### `NewCollector`

```go
func NewCollector(config *ClickHouseConfig, logger *logrus.Logger) Collector
```

Factory that tries ClickHouse first, falls back to `NoOpCollector` on failure. Returns `NoOpCollector` if `config` is nil.

### Methods on `*ClickHouseCollector`

#### `Track`

```go
func (c *ClickHouseCollector) Track(ctx context.Context, event Event) error
```

Records a single event. Delegates to `TrackBatch`.

#### `TrackBatch`

```go
func (c *ClickHouseCollector) TrackBatch(ctx context.Context, events []Event) error
```

Records multiple events in a single transaction. Returns nil for empty input.

#### `Query`

```go
func (c *ClickHouseCollector) Query(
    ctx context.Context,
    table string,
    groupBy string,
    window time.Duration,
) ([]AggregatedStats, error)
```

Retrieves aggregated stats. Validates `table` and `groupBy` as safe SQL identifiers.

#### `ExecuteReadQuery`

```go
func (c *ClickHouseCollector) ExecuteReadQuery(
    ctx context.Context,
    query string,
    args ...interface{},
) ([]map[string]interface{}, error)
```

Executes an arbitrary SELECT query. Rejects non-SELECT statements. Returns results as a slice of maps.

#### `Close`

```go
func (c *ClickHouseCollector) Close() error
```

Closes the ClickHouse connection.

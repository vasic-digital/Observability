# FAQ

## What happens if ClickHouse is unavailable?

The `analytics.NewCollector` factory attempts to connect to ClickHouse. If the connection fails, it logs a warning and returns a `NoOpCollector` that silently discards all events. Your application continues running normally -- analytics data is lost, but the service remains operational.

## Can I use individual packages without the others?

Yes. Each package is fully independent with no inter-package imports. You can use `pkg/health` for health checks without pulling in the tracing or metrics dependencies. The `health` package has zero external dependencies.

## How are correlation IDs propagated?

Correlation IDs are stored in `context.Context` via `logging.ContextWithCorrelationID(ctx, id)` and extracted via `logging.CorrelationIDFromContext(ctx)`. The `WithContext` helper enriches a logger from the context. Pass the context through your function call chain, and all loggers created from it will include the correlation ID.

## Is the Prometheus collector thread-safe?

Yes. The `PrometheusCollector` uses `sync.RWMutex` with double-checked locking for metric creation. Read operations (recording values) use read locks, and creation uses write locks only when a metric does not yet exist. The underlying Prometheus metric vectors are themselves thread-safe.

## How do I switch between tracing exporters?

Set the `ExporterType` field in `TracerConfig`:

- `trace.ExporterStdout` -- Development (prints to stdout)
- `trace.ExporterOTLP` -- Production (sends to OTLP collector)
- `trace.ExporterJaeger` -- Jaeger backend
- `trace.ExporterZipkin` -- Zipkin backend
- `trace.ExporterNone` -- Disabled (no-op spans, zero overhead)

No code changes are needed beyond the config value.

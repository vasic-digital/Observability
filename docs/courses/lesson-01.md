# Lesson 1: Distributed Tracing with OpenTelemetry

## Learning Objectives

- Configure OpenTelemetry tracing with multiple exporter backends
- Implement the Strategy pattern for exporter selection at initialization time
- Propagate trace context through `context.Context` for parent-child span relationships

## Key Concepts

- **Strategy Pattern for Exporters**: The `ExporterType` field selects the concrete exporter: `ExporterStdout` for development, `ExporterOTLP`/`ExporterJaeger`/`ExporterZipkin` for production, `ExporterNone` for testing.
- **Thread-Safe Tracer**: `sync.RWMutex` protects the `provider` and `tracer` fields. Span creation acquires a read lock; `Shutdown` acquires a write lock.
- **Context Propagation**: `StartSpan` embeds the new span in the returned context. Child operations using this context automatically become child spans in the trace.
- **No-Op Fallback**: `ExporterNone` uses `NeverSample()` sampler, creating no-op spans via OpenTelemetry. This eliminates nil checks throughout consumer code.

## Code Walkthrough

### Source: `pkg/trace/tracer.go`

The `Tracer` struct wraps the OpenTelemetry `TracerProvider` and exposes high-level span creation methods:

- `StartSpan(ctx, name)` -- general-purpose span
- `StartClientSpan(ctx, name)` -- marks span as client-side
- `StartInternalSpan(ctx, name)` -- marks span as internal

Each method returns a modified context containing the span and a `trace.Span` that must be ended by the caller (typically via `defer span.End()`).

The `InitTracer` function selects the exporter based on config and creates the `TracerProvider` with the appropriate sampler and resource attributes.

### Source: `pkg/trace/tracer_test.go`

Tests cover initialization with each exporter type, span creation and context propagation, and graceful shutdown behavior.

## Practice Exercise

1. Read `pkg/trace/tracer.go` and list all supported exporter types. For each, identify which OpenTelemetry package is used.
2. Write a small program that initializes a tracer with `ExporterStdout`, creates a parent span, then creates two child spans within it. Observe the stdout output showing the span hierarchy.
3. Configure `ExporterNone` and verify that all span methods can be called without error but produce no output.

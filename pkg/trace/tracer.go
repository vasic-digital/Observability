// Package trace provides OpenTelemetry-based distributed tracing with support
// for multiple exporters including OTLP, Jaeger, Zipkin, and stdout.
package trace

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// ExporterType defines the type of trace exporter.
type ExporterType string

const (
	// ExporterStdout writes traces to stdout in human-readable format.
	ExporterStdout ExporterType = "stdout"
	// ExporterOTLP sends traces via the OpenTelemetry Protocol.
	ExporterOTLP ExporterType = "otlp"
	// ExporterJaeger sends traces to a Jaeger backend via OTLP.
	ExporterJaeger ExporterType = "jaeger"
	// ExporterZipkin sends traces to a Zipkin backend via OTLP.
	ExporterZipkin ExporterType = "zipkin"
	// ExporterNone disables trace exporting (no-op sampler).
	ExporterNone ExporterType = "none"
)

// TracerConfig configures the tracer provider and exporter.
type TracerConfig struct {
	// ServiceName identifies the service in traces.
	ServiceName string
	// ServiceVersion is the version of the service.
	ServiceVersion string
	// Environment is the deployment environment (e.g., development, production).
	Environment string
	// ExporterType selects which exporter to use.
	ExporterType ExporterType
	// Endpoint is the collector endpoint (for OTLP, Jaeger, Zipkin).
	Endpoint string
	// SampleRate controls the fraction of traces sampled (0.0 to 1.0).
	SampleRate float64
	// Headers are additional HTTP headers for the exporter.
	Headers map[string]string
	// Insecure disables TLS for the exporter connection.
	Insecure bool
}

// DefaultConfig returns a TracerConfig with sensible defaults.
func DefaultConfig() *TracerConfig {
	return &TracerConfig{
		ServiceName:    "service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		ExporterType:   ExporterNone,
		SampleRate:     1.0,
	}
}

// Tracer wraps an OpenTelemetry TracerProvider with convenience methods
// for creating spans, recording errors, and shutting down.
type Tracer struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	config   *TracerConfig
	mu       sync.RWMutex
}

// InitTracer creates and configures a new Tracer with the given config.
// It sets up the exporter, resource, sampler, and registers the provider
// as the global OpenTelemetry TracerProvider.
func InitTracer(config *TracerConfig) (*Tracer, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var exporter sdktrace.SpanExporter
	var err error

	switch config.ExporterType {
	case ExporterOTLP, ExporterJaeger, ExporterZipkin:
		exporter, err = setupOTLPExporter(context.Background(), config)
	case ExporterStdout:
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	case ExporterNone, "":
		return setupNoOpTracer(config)
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", config.ExporterType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	res, err := buildResource(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	sampler := buildSampler(config.SampleRate)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)

	t := &Tracer{
		provider: tp,
		tracer: tp.Tracer(
			config.ServiceName,
			trace.WithInstrumentationVersion(config.ServiceVersion),
		),
		config: config,
	}

	return t, nil
}

// Shutdown gracefully flushes and shuts down the tracer provider.
func (t *Tracer) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.provider == nil {
		return nil
	}
	return t.provider.Shutdown(ctx)
}

// StartSpan creates a new span with the given name and optional attributes.
// The returned context carries the new span and should be used for child
// operations. The caller must call span.End() when the operation completes.
func (t *Tracer) StartSpan(
	ctx context.Context,
	name string,
	attrs ...attribute.KeyValue,
) (context.Context, trace.Span) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
	}
	return t.tracer.Start(ctx, name, opts...)
}

// StartClientSpan creates a span with SpanKindClient, suitable for
// outgoing calls to external services.
func (t *Tracer) StartClientSpan(
	ctx context.Context,
	name string,
	attrs ...attribute.KeyValue,
) (context.Context, trace.Span) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindClient),
	}
	return t.tracer.Start(ctx, name, opts...)
}

// StartInternalSpan creates a span with SpanKindInternal, suitable for
// in-process operations.
func (t *Tracer) StartInternalSpan(
	ctx context.Context,
	name string,
	attrs ...attribute.KeyValue,
) (context.Context, trace.Span) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindInternal),
	}
	return t.tracer.Start(ctx, name, opts...)
}

// RecordError records an error on the given span and sets the span status
// to Error.
func RecordError(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetOK sets the span status to Ok.
func SetOK(span trace.Span) {
	if span == nil {
		return
	}
	span.SetStatus(codes.Ok, "")
}

// EndSpanWithError is a convenience function that records an error if present,
// or sets the status to OK, then ends the span. It is intended to be used
// with defer:
//
//	ctx, span := tracer.StartSpan(ctx, "operation")
//	defer func() { trace.EndSpanWithError(span, err) }()
func EndSpanWithError(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		RecordError(span, err)
	} else {
		SetOK(span)
	}
	span.End()
}

// TraceFunc executes a function within a traced span. If the function returns
// an error, it is recorded on the span.
func (t *Tracer) TraceFunc(
	ctx context.Context,
	name string,
	fn func(ctx context.Context) error,
	attrs ...attribute.KeyValue,
) error {
	ctx, span := t.StartSpan(ctx, name, attrs...)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		RecordError(span, err)
	} else {
		SetOK(span)
	}
	return err
}

// TraceFuncWithResult executes a function within a traced span and returns
// the result. If the function returns an error, it is recorded on the span.
func TraceFuncWithResult[T any](
	t *Tracer,
	ctx context.Context,
	name string,
	fn func(ctx context.Context) (T, error),
	attrs ...attribute.KeyValue,
) (T, error) {
	ctx, span := t.StartSpan(ctx, name, attrs...)
	defer span.End()

	result, err := fn(ctx)
	if err != nil {
		RecordError(span, err)
	} else {
		SetOK(span)
	}
	return result, err
}

// TimedSpan starts a span and returns a finish function that records the
// duration as an attribute before ending the span.
func (t *Tracer) TimedSpan(
	ctx context.Context,
	name string,
	attrs ...attribute.KeyValue,
) (context.Context, func(err error)) {
	start := time.Now()
	ctx, span := t.StartSpan(ctx, name, attrs...)

	return ctx, func(err error) {
		duration := time.Since(start)
		span.SetAttributes(
			attribute.Float64("duration_seconds", duration.Seconds()),
		)
		EndSpanWithError(span, err)
	}
}

// Provider returns the underlying TracerProvider.
func (t *Tracer) Provider() *sdktrace.TracerProvider {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.provider
}

// setupOTLPExporter configures an OTLP HTTP trace exporter.
func setupOTLPExporter(
	ctx context.Context,
	config *TracerConfig,
) (*otlptrace.Exporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.Endpoint),
	}

	if config.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if len(config.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(config.Headers))
	}

	return otlptracehttp.New(ctx, opts...)
}

// setupNoOpTracer creates a tracer with NeverSample sampler.
func setupNoOpTracer(config *TracerConfig) (*Tracer, error) {
	res, err := buildResource(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.NeverSample()),
	)

	otel.SetTracerProvider(tp)

	return &Tracer{
		provider: tp,
		tracer: tp.Tracer(
			config.ServiceName,
			trace.WithInstrumentationVersion(config.ServiceVersion),
		),
		config: config,
	}, nil
}

// buildResource creates an OpenTelemetry resource describing the service.
func buildResource(config *TracerConfig) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		attribute.String("service.name", config.ServiceName),
		attribute.String("service.version", config.ServiceVersion),
	}

	if config.Environment != "" {
		attrs = append(attrs,
			attribute.String("deployment.environment", config.Environment),
		)
	}

	return resource.Merge(
		resource.Default(),
		resource.NewSchemaless(attrs...),
	)
}

// buildSampler creates a sampler based on the configured sample rate.
func buildSampler(rate float64) sdktrace.Sampler {
	switch {
	case rate <= 0:
		return sdktrace.NeverSample()
	case rate >= 1.0:
		return sdktrace.AlwaysSample()
	default:
		return sdktrace.TraceIDRatioBased(rate)
	}
}

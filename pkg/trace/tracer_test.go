package trace

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "service", cfg.ServiceName)
	assert.Equal(t, "1.0.0", cfg.ServiceVersion)
	assert.Equal(t, "development", cfg.Environment)
	assert.Equal(t, ExporterNone, cfg.ExporterType)
	assert.Equal(t, 1.0, cfg.SampleRate)
}

func TestInitTracer_NilConfig(t *testing.T) {
	tr, err := InitTracer(nil)
	require.NoError(t, err)
	require.NotNil(t, tr)
	assert.NotNil(t, tr.Provider())
	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestInitTracer_ExporterTypes(t *testing.T) {
	tests := []struct {
		name         string
		exporterType ExporterType
		expectError  bool
	}{
		{
			name:         "none exporter",
			exporterType: ExporterNone,
			expectError:  false,
		},
		{
			name:         "empty string exporter",
			exporterType: "",
			expectError:  false,
		},
		{
			name:         "stdout exporter",
			exporterType: ExporterStdout,
			expectError:  false,
		},
		{
			name:         "unsupported exporter",
			exporterType: "unsupported",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &TracerConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				ExporterType:   tt.exporterType,
				SampleRate:     1.0,
			}
			tr, err := InitTracer(cfg)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, tr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, tr)
				require.NoError(t, tr.Shutdown(context.Background()))
			}
		})
	}
}

func newTestTracer(t *testing.T) (*Tracer, *tracetest.InMemoryExporter) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	cfg := DefaultConfig()
	cfg.ServiceName = "test"

	tr := &Tracer{
		provider: tp,
		tracer:   tp.Tracer("test"),
		config:   cfg,
	}

	return tr, exporter
}

func TestTracer_StartSpan(t *testing.T) {
	tr, exporter := newTestTracer(t)
	defer func() { _ = tr.Shutdown(context.Background()) }()

	ctx, span := tr.StartSpan(
		context.Background(),
		"test-span",
		attribute.String("key", "value"),
	)
	span.End()

	assert.NotNil(t, ctx)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "test-span", spans[0].Name)
}

func TestTracer_StartClientSpan(t *testing.T) {
	tr, exporter := newTestTracer(t)
	defer func() { _ = tr.Shutdown(context.Background()) }()

	ctx, span := tr.StartClientSpan(
		context.Background(),
		"client-call",
		attribute.String("service", "external"),
	)
	span.End()

	assert.NotNil(t, ctx)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "client-call", spans[0].Name)
}

func TestTracer_StartInternalSpan(t *testing.T) {
	tr, exporter := newTestTracer(t)
	defer func() { _ = tr.Shutdown(context.Background()) }()

	ctx, span := tr.StartInternalSpan(
		context.Background(),
		"internal-op",
	)
	span.End()

	assert.NotNil(t, ctx)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "internal-op", spans[0].Name)
}

func TestRecordError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		expectCode codes.Code
	}{
		{
			name:       "with error",
			err:        errors.New("something broke"),
			expectCode: codes.Error,
		},
		{
			name:       "nil error",
			err:        nil,
			expectCode: codes.Unset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, exporter := newTestTracer(t)
			defer func() { _ = tr.Shutdown(context.Background()) }()

			_, span := tr.StartSpan(context.Background(), "test")
			RecordError(span, tt.err)
			span.End()

			spans := exporter.GetSpans()
			require.Len(t, spans, 1)
			assert.Equal(t, tt.expectCode, spans[0].Status.Code)
		})
	}
}

func TestRecordError_NilSpan(t *testing.T) {
	// Should not panic
	RecordError(nil, errors.New("error"))
}

func TestSetOK(t *testing.T) {
	tr, exporter := newTestTracer(t)
	defer func() { _ = tr.Shutdown(context.Background()) }()

	_, span := tr.StartSpan(context.Background(), "ok-span")
	SetOK(span)
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Ok, spans[0].Status.Code)
}

func TestSetOK_NilSpan(t *testing.T) {
	// Should not panic
	SetOK(nil)
}

func TestEndSpanWithError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		expectCode codes.Code
	}{
		{
			name:       "with error",
			err:        errors.New("fail"),
			expectCode: codes.Error,
		},
		{
			name:       "without error",
			err:        nil,
			expectCode: codes.Ok,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, exporter := newTestTracer(t)
			defer func() { _ = tr.Shutdown(context.Background()) }()

			_, span := tr.StartSpan(context.Background(), "test")
			EndSpanWithError(span, tt.err)

			spans := exporter.GetSpans()
			require.Len(t, spans, 1)
			assert.Equal(t, tt.expectCode, spans[0].Status.Code)
		})
	}
}

func TestEndSpanWithError_NilSpan(t *testing.T) {
	// Should not panic
	EndSpanWithError(nil, errors.New("error"))
}

func TestTracer_TraceFunc(t *testing.T) {
	tests := []struct {
		name       string
		fn         func(ctx context.Context) error
		expectErr  bool
		expectCode codes.Code
	}{
		{
			name:       "successful function",
			fn:         func(_ context.Context) error { return nil },
			expectErr:  false,
			expectCode: codes.Ok,
		},
		{
			name: "failing function",
			fn: func(_ context.Context) error {
				return errors.New("function failed")
			},
			expectErr:  true,
			expectCode: codes.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, exporter := newTestTracer(t)
			defer func() { _ = tr.Shutdown(context.Background()) }()

			err := tr.TraceFunc(
				context.Background(),
				"traced-func",
				tt.fn,
				attribute.String("test", "true"),
			)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			spans := exporter.GetSpans()
			require.Len(t, spans, 1)
			assert.Equal(t, "traced-func", spans[0].Name)
			assert.Equal(t, tt.expectCode, spans[0].Status.Code)
		})
	}
}

func TestTraceFuncWithResult(t *testing.T) {
	tests := []struct {
		name         string
		fn           func(ctx context.Context) (string, error)
		expectResult string
		expectErr    bool
	}{
		{
			name: "successful with result",
			fn: func(_ context.Context) (string, error) {
				return "hello", nil
			},
			expectResult: "hello",
			expectErr:    false,
		},
		{
			name: "failure with error",
			fn: func(_ context.Context) (string, error) {
				return "", errors.New("failed")
			},
			expectResult: "",
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, exporter := newTestTracer(t)
			defer func() { _ = tr.Shutdown(context.Background()) }()

			result, err := TraceFuncWithResult(
				tr,
				context.Background(),
				"result-func",
				tt.fn,
			)

			assert.Equal(t, tt.expectResult, result)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			spans := exporter.GetSpans()
			require.Len(t, spans, 1)
		})
	}
}

func TestTracer_TimedSpan(t *testing.T) {
	tr, exporter := newTestTracer(t)
	defer func() { _ = tr.Shutdown(context.Background()) }()

	ctx, finish := tr.TimedSpan(
		context.Background(),
		"timed-op",
		attribute.String("op", "test"),
	)
	assert.NotNil(t, ctx)
	finish(nil)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "timed-op", spans[0].Name)

	// Check duration attribute was set
	found := false
	for _, attr := range spans[0].Attributes {
		if string(attr.Key) == "duration_seconds" {
			found = true
			assert.GreaterOrEqual(t, attr.Value.AsFloat64(), 0.0)
		}
	}
	assert.True(t, found, "duration_seconds attribute should be set")
}

func TestTracer_TimedSpan_WithError(t *testing.T) {
	tr, exporter := newTestTracer(t)
	defer func() { _ = tr.Shutdown(context.Background()) }()

	_, finish := tr.TimedSpan(context.Background(), "timed-error")
	finish(errors.New("timed out"))

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status.Code)
}

func TestTracer_Shutdown_NilProvider(t *testing.T) {
	tr := &Tracer{}
	assert.NoError(t, tr.Shutdown(context.Background()))
}

func TestTracer_Provider(t *testing.T) {
	tr, _ := newTestTracer(t)
	defer func() { _ = tr.Shutdown(context.Background()) }()

	assert.NotNil(t, tr.Provider())
}

func TestBuildSampler(t *testing.T) {
	tests := []struct {
		name string
		rate float64
	}{
		{name: "never sample", rate: 0.0},
		{name: "always sample", rate: 1.0},
		{name: "ratio sample", rate: 0.5},
		{name: "negative rate", rate: -1.0},
		{name: "over 1.0 rate", rate: 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sampler := buildSampler(tt.rate)
			assert.NotNil(t, sampler)
		})
	}
}

func TestExporterType_Constants(t *testing.T) {
	assert.Equal(t, ExporterType("stdout"), ExporterStdout)
	assert.Equal(t, ExporterType("otlp"), ExporterOTLP)
	assert.Equal(t, ExporterType("jaeger"), ExporterJaeger)
	assert.Equal(t, ExporterType("zipkin"), ExporterZipkin)
	assert.Equal(t, ExporterType("none"), ExporterNone)
}

func TestTracer_NestedSpans(t *testing.T) {
	tr, exporter := newTestTracer(t)
	defer func() { _ = tr.Shutdown(context.Background()) }()

	ctx, parentSpan := tr.StartSpan(
		context.Background(),
		"parent",
	)

	_, childSpan := tr.StartSpan(ctx, "child")
	childSpan.End()
	parentSpan.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 2)

	// Child should reference parent's span context
	childStub := spans[0]
	parentStub := spans[1]

	assert.Equal(t, "child", childStub.Name)
	assert.Equal(t, "parent", parentStub.Name)
	assert.Equal(t,
		parentStub.SpanContext.TraceID(),
		childStub.SpanContext.TraceID(),
	)
	assert.Equal(t,
		parentStub.SpanContext.SpanID(),
		childStub.Parent.SpanID(),
	)
}

func TestInitTracer_OTLPExporterTypes(t *testing.T) {
	// Test OTLP, Jaeger, Zipkin exporter types
	// These will fail to connect but exercise the setupOTLPExporter code path
	tests := []struct {
		name         string
		exporterType ExporterType
		endpoint     string
		insecure     bool
		headers      map[string]string
	}{
		{
			name:         "OTLP exporter",
			exporterType: ExporterOTLP,
			endpoint:     "localhost:4318",
			insecure:     true,
			headers:      nil,
		},
		{
			name:         "Jaeger exporter via OTLP",
			exporterType: ExporterJaeger,
			endpoint:     "localhost:4317",
			insecure:     true,
			headers:      nil,
		},
		{
			name:         "Zipkin exporter via OTLP",
			exporterType: ExporterZipkin,
			endpoint:     "localhost:9411",
			insecure:     true,
			headers:      nil,
		},
		{
			name:         "OTLP with headers",
			exporterType: ExporterOTLP,
			endpoint:     "localhost:4318",
			insecure:     true,
			headers:      map[string]string{"Authorization": "Bearer token"},
		},
		{
			name:         "OTLP secure",
			exporterType: ExporterOTLP,
			endpoint:     "localhost:4318",
			insecure:     false,
			headers:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &TracerConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				ExporterType:   tt.exporterType,
				Endpoint:       tt.endpoint,
				SampleRate:     1.0,
				Insecure:       tt.insecure,
				Headers:        tt.headers,
			}

			tr, err := InitTracer(cfg)
			// These should succeed in creating the tracer even if the
			// endpoint is not reachable, because OTLP exporters batch
			// spans asynchronously
			require.NoError(t, err)
			require.NotNil(t, tr)
			require.NoError(t, tr.Shutdown(context.Background()))
		})
	}
}

func TestBuildResource_EmptyEnvironment(t *testing.T) {
	// Test buildResource with empty environment
	cfg := &TracerConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		Environment:    "", // Empty
		ExporterType:   ExporterNone,
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)
	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestSetupNoOpTracer_Direct(t *testing.T) {
	// Test setupNoOpTracer directly via the public API with ExporterNone
	cfg := &TracerConfig{
		ServiceName:    "test-noop",
		ServiceVersion: "2.0.0",
		Environment:    "production",
		ExporterType:   ExporterNone,
		SampleRate:     0.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)

	// Should still create spans but they won't be sampled
	ctx, span := tr.StartSpan(context.Background(), "test-span")
	span.End()
	assert.NotNil(t, ctx)

	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestSetupNoOpTracer_WithEnvironment(t *testing.T) {
	// Test setupNoOpTracer with environment set
	cfg := &TracerConfig{
		ServiceName:    "noop-env-test",
		ServiceVersion: "1.0.0",
		Environment:    "staging",
		ExporterType:   ExporterNone,
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)

	// Verify provider is created
	assert.NotNil(t, tr.Provider())
	assert.Equal(t, cfg, tr.config)

	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestSetupNoOpTracer_EmptyEnvironment(t *testing.T) {
	// Test setupNoOpTracer with empty environment
	cfg := &TracerConfig{
		ServiceName:    "noop-no-env",
		ServiceVersion: "1.0.0",
		Environment:    "",
		ExporterType:   ExporterNone,
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)

	// Verify it still works without environment
	ctx, span := tr.StartSpan(context.Background(), "test")
	span.End()
	assert.NotNil(t, ctx)

	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestInitTracer_EmptyStringExporterType(t *testing.T) {
	// Test that empty string exporter type is treated as ExporterNone
	cfg := &TracerConfig{
		ServiceName:    "empty-exporter",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   "",
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)
	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestBuildResource_WithEnvironment(t *testing.T) {
	// Test buildResource with environment attribute
	cfg := &TracerConfig{
		ServiceName:    "resource-test",
		ServiceVersion: "1.0.0",
		Environment:    "production",
		ExporterType:   ExporterNone,
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)
	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestBuildResource_WithoutEnvironment(t *testing.T) {
	// Test buildResource without environment attribute (empty string)
	cfg := &TracerConfig{
		ServiceName:    "resource-no-env",
		ServiceVersion: "2.0.0",
		Environment:    "",
		ExporterType:   ExporterNone,
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)
	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestBuildResourceForTesting(t *testing.T) {
	// Test BuildResourceForTesting helper
	cfg := &TracerConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	res, err := BuildResourceForTesting(cfg)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestBuildResourceForTesting_NoEnvironment(t *testing.T) {
	cfg := &TracerConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		Environment:    "",
	}

	res, err := BuildResourceForTesting(cfg)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestInitTracer_StdoutExporter_WithEnvironment(t *testing.T) {
	// Test stdout exporter with buildResource having environment
	cfg := &TracerConfig{
		ServiceName:    "stdout-env-test",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		ExporterType:   ExporterStdout,
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)
	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestInitTracer_StdoutExporter_WithoutEnvironment(t *testing.T) {
	// Test stdout exporter with buildResource without environment
	cfg := &TracerConfig{
		ServiceName:    "stdout-no-env",
		ServiceVersion: "1.0.0",
		Environment:    "",
		ExporterType:   ExporterStdout,
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)
	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestSetTestExporterFactory(t *testing.T) {
	// Test that SetTestExporterFactory sets and resets the factory
	defer SetTestExporterFactory(nil) // Clean up after test

	called := false
	factory := func(_ *TracerConfig) (sdktrace.SpanExporter, error) {
		called = true
		return tracetest.NewInMemoryExporter(), nil
	}

	SetTestExporterFactory(factory)

	cfg := &TracerConfig{
		ServiceName:    "factory-test",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   ExporterOTLP, // Would normally try OTLP
		Endpoint:       "localhost:4318",
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)
	assert.True(t, called, "custom exporter factory should have been called")
	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestSetTestResourceFactory(t *testing.T) {
	// Test that SetTestResourceFactory sets and resets the factory
	defer SetTestResourceFactory(nil) // Clean up after test

	called := false
	factory := func(cfg *TracerConfig) (*resource.Resource, error) {
		called = true
		return buildResource(cfg)
	}

	SetTestResourceFactory(factory)

	cfg := &TracerConfig{
		ServiceName:    "resource-factory-test",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   ExporterStdout,
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)
	assert.True(t, called, "custom resource factory should have been called")
	require.NoError(t, tr.Shutdown(context.Background()))
}

func TestInitTracer_ExporterFactoryError(t *testing.T) {
	// Test that exporter factory errors are properly returned
	defer SetTestExporterFactory(nil) // Clean up after test

	expectedErr := errors.New("exporter creation failed")
	SetTestExporterFactory(func(_ *TracerConfig) (sdktrace.SpanExporter, error) {
		return nil, expectedErr
	})

	cfg := &TracerConfig{
		ServiceName:    "error-test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterOTLP,
		Endpoint:       "localhost:4318",
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.Error(t, err)
	assert.Nil(t, tr)
	assert.Contains(t, err.Error(), "failed to create exporter")
}

func TestInitTracer_ResourceFactoryError(t *testing.T) {
	// Test that resource factory errors are properly returned
	defer SetTestExporterFactory(nil)  // Clean up after test
	defer SetTestResourceFactory(nil)  // Clean up after test

	// First set a working exporter factory
	SetTestExporterFactory(func(_ *TracerConfig) (sdktrace.SpanExporter, error) {
		return tracetest.NewInMemoryExporter(), nil
	})

	// Then set a failing resource factory
	expectedErr := errors.New("resource creation failed")
	SetTestResourceFactory(func(_ *TracerConfig) (*resource.Resource, error) {
		return nil, expectedErr
	})

	cfg := &TracerConfig{
		ServiceName:    "resource-error-test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterOTLP,
		Endpoint:       "localhost:4318",
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.Error(t, err)
	assert.Nil(t, tr)
	assert.Contains(t, err.Error(), "failed to create resource")
}

func TestSetupNoOpTracer_ResourceFactoryError(t *testing.T) {
	// Test that resource factory errors in setupNoOpTracer are properly returned
	defer SetTestResourceFactory(nil)  // Clean up after test

	// Set a failing resource factory
	expectedErr := errors.New("resource creation failed for noop")
	SetTestResourceFactory(func(_ *TracerConfig) (*resource.Resource, error) {
		return nil, expectedErr
	})

	cfg := &TracerConfig{
		ServiceName:    "noop-resource-error-test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterNone, // This uses setupNoOpTracer
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.Error(t, err)
	assert.Nil(t, tr)
	assert.Contains(t, err.Error(), "failed to create resource")
}

func TestSetupNoOpTracer_WithResourceFactory(t *testing.T) {
	// Test that setupNoOpTracer uses testResourceFactory when set
	defer SetTestResourceFactory(nil)  // Clean up after test

	called := false
	SetTestResourceFactory(func(cfg *TracerConfig) (*resource.Resource, error) {
		called = true
		return buildResource(cfg)
	})

	cfg := &TracerConfig{
		ServiceName:    "noop-factory-test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterNone,
		SampleRate:     1.0,
	}

	tr, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, tr)
	assert.True(t, called, "resource factory should have been called for noop tracer")
	require.NoError(t, tr.Shutdown(context.Background()))
}

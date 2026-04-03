package logging_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"

	"digital.vasic.observability/pkg/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoOpLogger_AllMethods(t *testing.T) {
	t.Parallel()

	var logger logging.Logger = &logging.NoOpLogger{}

	// None of these should panic
	assert.NotPanics(t, func() {
		logger.Info("info")
		logger.Warn("warn")
		logger.Error("error")
		logger.Debug("debug")
		_ = logger.WithField("key", "val")
		_ = logger.WithFields(map[string]interface{}{"a": 1})
		_ = logger.WithCorrelationID("abc-123")
		_ = logger.WithError(errors.New("fail"))
	})
}

func TestNoOpLogger_Chaining(t *testing.T) {
	t.Parallel()

	var logger logging.Logger = &logging.NoOpLogger{}

	// Chained calls should all return the same no-op logger
	chained := logger.WithField("k", "v").WithCorrelationID("id").WithError(errors.New("x"))
	assert.NotNil(t, chained)

	// Should still be a NoOpLogger
	_, ok := chained.(*logging.NoOpLogger)
	assert.True(t, ok, "chained logger should still be NoOpLogger")
}

func TestLogrusAdapter_NilConfig(t *testing.T) {
	t.Parallel()

	// nil config should use DefaultConfig
	adapter := logging.NewLogrusAdapter(nil)
	require.NotNil(t, adapter)

	// Should not panic
	adapter.Info("test message")
}

func TestLogrusAdapter_ConcurrentLogging(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	adapter := logging.NewLogrusAdapter(&logging.Config{
		Level:       logging.DebugLevel,
		Format:      "json",
		Output:      &buf,
		ServiceName: "concurrent-test",
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			adapter.Info("concurrent info")
		}()
		go func() {
			defer wg.Done()
			adapter.Warn("concurrent warn")
		}()
		go func() {
			defer wg.Done()
			adapter.Error("concurrent error")
		}()
		go func() {
			defer wg.Done()
			adapter.Debug("concurrent debug")
		}()
	}

	wg.Wait()
	assert.NotEmpty(t, buf.String(), "buffer should contain log output")
}

func TestLogrusAdapter_WithError_NilError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	adapter := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.DebugLevel,
		Format: "json",
		Output: &buf,
	})

	// WithError(nil) should not panic
	assert.NotPanics(t, func() {
		l := adapter.WithError(nil)
		l.Info("with nil error")
	})
}

func TestLogrusAdapter_EmptyServiceName(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	adapter := logging.NewLogrusAdapter(&logging.Config{
		Level:       logging.InfoLevel,
		Format:      "json",
		Output:      &buf,
		ServiceName: "",
	})

	adapter.Info("no service name")
	assert.NotEmpty(t, buf.String())
}

func TestLogrusAdapter_TextFormat(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	adapter := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.InfoLevel,
		Format: "text",
		Output: &buf,
	})

	adapter.Info("text format message")
	assert.NotEmpty(t, buf.String())
}

func TestLogrusAdapter_UnknownLevel(t *testing.T) {
	t.Parallel()

	// An unknown level should fall back to InfoLevel
	var buf bytes.Buffer
	adapter := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.Level(99),
		Format: "json",
		Output: &buf,
	})

	adapter.Info("should work")
	adapter.Debug("should be suppressed at info level")

	output := buf.String()
	assert.Contains(t, output, "should work")
}

func TestCorrelationID_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Empty context has no correlation ID
	id := logging.CorrelationIDFromContext(ctx)
	assert.Empty(t, id)

	// Set a correlation ID
	ctx = logging.ContextWithCorrelationID(ctx, "test-corr-123")
	id = logging.CorrelationIDFromContext(ctx)
	assert.Equal(t, "test-corr-123", id)
}

func TestCorrelationID_EmptyString(t *testing.T) {
	t.Parallel()

	ctx := logging.ContextWithCorrelationID(context.Background(), "")
	id := logging.CorrelationIDFromContext(ctx)
	assert.Equal(t, "", id)
}

func TestWithContext_NoCorrelationID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	adapter := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.DebugLevel,
		Format: "json",
		Output: &buf,
	})

	// WithContext with no correlation ID should return the same logger
	result := logging.WithContext(adapter, context.Background())
	assert.NotNil(t, result)
}

func TestWithContext_WithCorrelationID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	adapter := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.DebugLevel,
		Format: "json",
		Output: &buf,
	})

	ctx := logging.ContextWithCorrelationID(context.Background(), "ctx-corr-456")
	enriched := logging.WithContext(adapter, ctx)
	enriched.Info("enriched message")

	assert.Contains(t, buf.String(), "ctx-corr-456")
}

func TestLogrusAdapter_WithFields_EmptyMap(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	adapter := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.DebugLevel,
		Format: "json",
		Output: &buf,
	})

	l := adapter.WithFields(map[string]interface{}{})
	l.Info("empty fields")
	assert.NotEmpty(t, buf.String())
}

func TestLogrusAdapter_WithFields_NilMap(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	adapter := logging.NewLogrusAdapter(&logging.Config{
		Level:  logging.DebugLevel,
		Format: "json",
		Output: &buf,
	})

	assert.NotPanics(t, func() {
		l := adapter.WithFields(nil)
		l.Info("nil fields")
	})
}

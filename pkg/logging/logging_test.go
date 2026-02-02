package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, InfoLevel, cfg.Level)
	assert.Equal(t, "json", cfg.Format)
}

func TestNewLogrusAdapter_NilConfig(t *testing.T) {
	logger := NewLogrusAdapter(nil)
	assert.NotNil(t, logger)
}

func TestNewLogrusAdapter_Levels(t *testing.T) {
	tests := []struct {
		name  string
		level Level
	}{
		{name: "debug", level: DebugLevel},
		{name: "info", level: InfoLevel},
		{name: "warn", level: WarnLevel},
		{name: "error", level: ErrorLevel},
		{name: "unknown defaults to info", level: Level(99)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogrusAdapter(&Config{Level: tt.level})
			assert.NotNil(t, logger)
		})
	}
}

func TestNewLogrusAdapter_Formats(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{name: "json format", format: "json"},
		{name: "text format", format: "text"},
		{name: "unknown defaults to json", format: "xml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogrusAdapter(&Config{Format: tt.format})
			assert.NotNil(t, logger)
		})
	}
}

func TestLogrusAdapter_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogrusAdapter(&Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	})

	logger.Info("test info message")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "test info message", entry["msg"])
	assert.Equal(t, "info", entry["level"])
}

func TestLogrusAdapter_Warn(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogrusAdapter(&Config{
		Level:  WarnLevel,
		Format: "json",
		Output: &buf,
	})

	logger.Warn("test warn message")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "test warn message", entry["msg"])
	assert.Equal(t, "warning", entry["level"])
}

func TestLogrusAdapter_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogrusAdapter(&Config{
		Level:  ErrorLevel,
		Format: "json",
		Output: &buf,
	})

	logger.Error("test error message")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "test error message", entry["msg"])
	assert.Equal(t, "error", entry["level"])
}

func TestLogrusAdapter_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogrusAdapter(&Config{
		Level:  DebugLevel,
		Format: "json",
		Output: &buf,
	})

	logger.Debug("test debug message")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "test debug message", entry["msg"])
	assert.Equal(t, "debug", entry["level"])
}

func TestLogrusAdapter_WithField(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogrusAdapter(&Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	})

	logger.WithField("key", "value").Info("with field")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "value", entry["key"])
}

func TestLogrusAdapter_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogrusAdapter(&Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	})

	fields := map[string]interface{}{
		"request_id": "abc-123",
		"user_id":    "user-456",
	}
	logger.WithFields(fields).Info("with fields")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "abc-123", entry["request_id"])
	assert.Equal(t, "user-456", entry["user_id"])
}

func TestLogrusAdapter_WithCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogrusAdapter(&Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	})

	logger.WithCorrelationID("corr-789").Info("correlated")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "corr-789", entry["correlation_id"])
}

func TestLogrusAdapter_WithError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogrusAdapter(&Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	})

	logger.WithError(errors.New("test error")).Info("with error")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "test error", entry["error"])
}

func TestLogrusAdapter_WithServiceName(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogrusAdapter(&Config{
		Level:       InfoLevel,
		Format:      "json",
		Output:      &buf,
		ServiceName: "my-service",
	})

	logger.Info("service log")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "my-service", entry["service"])
}

func TestLogrusAdapter_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogrusAdapter(&Config{
		Level:  WarnLevel,
		Format: "json",
		Output: &buf,
	})

	logger.Debug("should not appear")
	logger.Info("should not appear")
	assert.Empty(t, buf.String())

	logger.Warn("should appear")
	assert.NotEmpty(t, buf.String())
}

func TestContextWithCorrelationID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{name: "set and get", id: "abc-123", expected: "abc-123"},
		{name: "empty string", id: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ContextWithCorrelationID(context.Background(), tt.id)
			result := CorrelationIDFromContext(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCorrelationIDFromContext_NotSet(t *testing.T) {
	result := CorrelationIDFromContext(context.Background())
	assert.Equal(t, "", result)
}

func TestWithContext(t *testing.T) {
	tests := []struct {
		name          string
		correlationID string
		expectField   bool
	}{
		{
			name:          "with correlation ID",
			correlationID: "ctx-001",
			expectField:   true,
		},
		{
			name:          "without correlation ID",
			correlationID: "",
			expectField:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogrusAdapter(&Config{
				Level:  InfoLevel,
				Format: "json",
				Output: &buf,
			})

			ctx := context.Background()
			if tt.correlationID != "" {
				ctx = ContextWithCorrelationID(ctx, tt.correlationID)
			}

			enriched := WithContext(logger, ctx)
			enriched.Info("context log")

			var entry map[string]interface{}
			require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))

			if tt.expectField {
				assert.Equal(t, tt.correlationID, entry["correlation_id"])
			} else {
				_, exists := entry["correlation_id"]
				assert.False(t, exists)
			}
		})
	}
}

func TestNoOpLogger(t *testing.T) {
	var l Logger = &NoOpLogger{}

	// None of these should panic
	l.Info("test")
	l.Warn("test")
	l.Error("test")
	l.Debug("test")

	assert.Equal(t, l, l.WithField("k", "v"))
	assert.Equal(t, l, l.WithFields(map[string]interface{}{"k": "v"}))
	assert.Equal(t, l, l.WithCorrelationID("id"))
	assert.Equal(t, l, l.WithError(errors.New("err")))
}

func TestNoOpLogger_ImplementsInterface(t *testing.T) {
	var _ Logger = &NoOpLogger{}
	var _ Logger = &LogrusAdapter{}
}

func TestLogrusAdapter_ConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogrusAdapter(&Config{
		Level:  DebugLevel,
		Format: "json",
		Output: &buf,
	})

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			logger.Info("concurrent")
			logger.WithField("k", "v").Debug("with field")
			logger.WithCorrelationID("id").Warn("correlated")
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

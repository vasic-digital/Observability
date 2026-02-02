// Package logging provides structured logging with correlation ID support
// and a pluggable Logger interface backed by logrus.
package logging

import (
	"context"
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

// contextKey is a private type for context keys in this package.
type contextKey string

const (
	// correlationIDKey is the context key for correlation IDs.
	correlationIDKey contextKey = "correlation_id"
)

// Logger defines the interface for structured logging with correlation ID
// support. Implementations must be safe for concurrent use.
type Logger interface {
	// Info logs a message at the info level.
	Info(msg string)
	// Warn logs a message at the warn level.
	Warn(msg string)
	// Error logs a message at the error level.
	Error(msg string)
	// Debug logs a message at the debug level.
	Debug(msg string)
	// WithField returns a Logger with a single field added.
	WithField(key string, value interface{}) Logger
	// WithFields returns a Logger with multiple fields added.
	WithFields(fields map[string]interface{}) Logger
	// WithCorrelationID returns a Logger with the correlation ID field set.
	WithCorrelationID(id string) Logger
	// WithError returns a Logger with the error field set.
	WithError(err error) Logger
}

// Level represents the log level.
type Level int

const (
	// DebugLevel is the most verbose level.
	DebugLevel Level = iota
	// InfoLevel is the default level.
	InfoLevel
	// WarnLevel is for potentially problematic situations.
	WarnLevel
	// ErrorLevel is for errors that should be addressed.
	ErrorLevel
)

// Config configures the logger.
type Config struct {
	// Level sets the minimum log level.
	Level Level
	// Format selects the output format ("json" or "text").
	Format string
	// Output is the writer for log output. Defaults to os.Stderr.
	Output io.Writer
	// ServiceName is included as a field in every log entry.
	ServiceName string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Level:  InfoLevel,
		Format: "json",
	}
}

// LogrusAdapter implements Logger using logrus.
type LogrusAdapter struct {
	logger *logrus.Entry
	mu     sync.RWMutex
}

// NewLogrusAdapter creates a Logger backed by logrus.
func NewLogrusAdapter(config *Config) *LogrusAdapter {
	if config == nil {
		config = DefaultConfig()
	}

	l := logrus.New()

	switch config.Level {
	case DebugLevel:
		l.SetLevel(logrus.DebugLevel)
	case InfoLevel:
		l.SetLevel(logrus.InfoLevel)
	case WarnLevel:
		l.SetLevel(logrus.WarnLevel)
	case ErrorLevel:
		l.SetLevel(logrus.ErrorLevel)
	default:
		l.SetLevel(logrus.InfoLevel)
	}

	switch config.Format {
	case "text":
		l.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	default:
		l.SetFormatter(&logrus.JSONFormatter{})
	}

	if config.Output != nil {
		l.SetOutput(config.Output)
	}

	entry := logrus.NewEntry(l)
	if config.ServiceName != "" {
		entry = entry.WithField("service", config.ServiceName)
	}

	return &LogrusAdapter{logger: entry}
}

// Info logs at info level.
func (a *LogrusAdapter) Info(msg string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	a.logger.Info(msg)
}

// Warn logs at warn level.
func (a *LogrusAdapter) Warn(msg string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	a.logger.Warn(msg)
}

// Error logs at error level.
func (a *LogrusAdapter) Error(msg string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	a.logger.Error(msg)
}

// Debug logs at debug level.
func (a *LogrusAdapter) Debug(msg string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	a.logger.Debug(msg)
}

// WithField returns a new Logger with the given field.
func (a *LogrusAdapter) WithField(key string, value interface{}) Logger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return &LogrusAdapter{logger: a.logger.WithField(key, value)}
}

// WithFields returns a new Logger with the given fields.
func (a *LogrusAdapter) WithFields(fields map[string]interface{}) Logger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return &LogrusAdapter{logger: a.logger.WithFields(logrus.Fields(fields))}
}

// WithCorrelationID returns a new Logger with correlation_id set.
func (a *LogrusAdapter) WithCorrelationID(id string) Logger {
	return a.WithField("correlation_id", id)
}

// WithError returns a new Logger with the error field set.
func (a *LogrusAdapter) WithError(err error) Logger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return &LogrusAdapter{logger: a.logger.WithError(err)}
}

// ContextWithCorrelationID returns a new context with the correlation ID set.
func ContextWithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// CorrelationIDFromContext extracts the correlation ID from the context.
// Returns an empty string if no correlation ID is set.
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// WithContext returns a Logger enriched with the correlation ID from
// the context, if present.
func WithContext(logger Logger, ctx context.Context) Logger {
	id := CorrelationIDFromContext(ctx)
	if id != "" {
		return logger.WithCorrelationID(id)
	}
	return logger
}

// NoOpLogger is a Logger that discards all log output.
type NoOpLogger struct{}

// Info is a no-op.
func (n *NoOpLogger) Info(_ string) {}

// Warn is a no-op.
func (n *NoOpLogger) Warn(_ string) {}

// Error is a no-op.
func (n *NoOpLogger) Error(_ string) {}

// Debug is a no-op.
func (n *NoOpLogger) Debug(_ string) {}

// WithField returns the same no-op logger.
func (n *NoOpLogger) WithField(_ string, _ interface{}) Logger { return n }

// WithFields returns the same no-op logger.
func (n *NoOpLogger) WithFields(_ map[string]interface{}) Logger { return n }

// WithCorrelationID returns the same no-op logger.
func (n *NoOpLogger) WithCorrelationID(_ string) Logger { return n }

// WithError returns the same no-op logger.
func (n *NoOpLogger) WithError(_ error) Logger { return n }

package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvent_Defaults(t *testing.T) {
	e := Event{
		Name:      "test.event",
		Timestamp: time.Now(),
	}
	assert.Equal(t, "test.event", e.Name)
	assert.False(t, e.Timestamp.IsZero())
}

func TestNoOpCollector_Track(t *testing.T) {
	c := &NoOpCollector{}
	err := c.Track(context.Background(), Event{Name: "test"})
	assert.NoError(t, err)
}

func TestNoOpCollector_TrackBatch(t *testing.T) {
	c := &NoOpCollector{}
	events := []Event{
		{Name: "event1"},
		{Name: "event2"},
	}
	err := c.TrackBatch(context.Background(), events)
	assert.NoError(t, err)
}

func TestNoOpCollector_Query(t *testing.T) {
	c := &NoOpCollector{}
	stats, err := c.Query(
		context.Background(), "events", "name", time.Hour,
	)
	assert.NoError(t, err)
	assert.Nil(t, stats)
}

func TestNoOpCollector_Close(t *testing.T) {
	c := &NoOpCollector{}
	assert.NoError(t, c.Close())
}

func TestNoOpCollector_ImplementsInterface(t *testing.T) {
	var _ Collector = &NoOpCollector{}
}

func TestNewCollector_NilConfig(t *testing.T) {
	c := NewCollector(nil, nil)
	require.NotNil(t, c)

	// Should be a NoOpCollector
	_, ok := c.(*NoOpCollector)
	assert.True(t, ok)
}

func TestNewCollector_InvalidConfig(t *testing.T) {
	cfg := &ClickHouseConfig{
		Host:     "invalid-host-that-does-not-exist",
		Port:     9999,
		Database: "test",
		Username: "test",
		Password: "test",
	}

	c := NewCollector(cfg, nil)
	require.NotNil(t, c)

	// Should fall back to NoOpCollector
	_, ok := c.(*NoOpCollector)
	assert.True(t, ok)
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "simple name", input: "events", expected: true},
		{name: "with underscore", input: "event_log", expected: true},
		{name: "with numbers", input: "events2024", expected: true},
		{name: "uppercase", input: "Events", expected: true},
		{name: "empty string", input: "", expected: false},
		{name: "with space", input: "event log", expected: false},
		{name: "with semicolon", input: "events;DROP", expected: false},
		{name: "with dash", input: "event-log", expected: false},
		{name: "with dot", input: "schema.table", expected: false},
		{name: "with quotes", input: "events'", expected: false},
		{name: "SQL injection", input: "1; DROP TABLE", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAggregatedStats_Fields(t *testing.T) {
	stats := AggregatedStats{
		Group:       "provider_a",
		TotalCount:  100,
		AvgDuration: 250.5,
		P95Duration: 450.0,
		P99Duration: 900.0,
		ErrorRate:   0.05,
		Period:      "last_24h",
		Extra: map[string]interface{}{
			"custom_metric": 42.0,
		},
	}

	assert.Equal(t, "provider_a", stats.Group)
	assert.Equal(t, int64(100), stats.TotalCount)
	assert.Equal(t, 250.5, stats.AvgDuration)
	assert.Equal(t, 450.0, stats.P95Duration)
	assert.Equal(t, 900.0, stats.P99Duration)
	assert.InDelta(t, 0.05, stats.ErrorRate, 0.001)
	assert.Equal(t, "last_24h", stats.Period)
	assert.Equal(t, 42.0, stats.Extra["custom_metric"])
}

func TestClickHouseConfig_Fields(t *testing.T) {
	cfg := ClickHouseConfig{
		Host:     "localhost",
		Port:     9000,
		Database: "analytics",
		Username: "user",
		Password: "pass",
		TLS:      true,
		Table:    "events",
	}

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 9000, cfg.Port)
	assert.Equal(t, "analytics", cfg.Database)
	assert.Equal(t, "user", cfg.Username)
	assert.Equal(t, "pass", cfg.Password)
	assert.True(t, cfg.TLS)
	assert.Equal(t, "events", cfg.Table)
}

func TestEvent_WithProperties(t *testing.T) {
	e := Event{
		Name:      "request.completed",
		Timestamp: time.Now(),
		Properties: map[string]interface{}{
			"duration_ms": 250.0,
			"status_code": 200,
		},
		Tags: map[string]string{
			"service": "api",
			"region":  "us-east",
		},
	}

	assert.Equal(t, "request.completed", e.Name)
	assert.Equal(t, 250.0, e.Properties["duration_ms"])
	assert.Equal(t, 200, e.Properties["status_code"])
	assert.Equal(t, "api", e.Tags["service"])
	assert.Equal(t, "us-east", e.Tags["region"])
}

func TestNoOpCollector_TrackBatch_Empty(t *testing.T) {
	c := &NoOpCollector{}
	err := c.TrackBatch(context.Background(), nil)
	assert.NoError(t, err)

	err = c.TrackBatch(context.Background(), []Event{})
	assert.NoError(t, err)
}

func TestNoOpCollector_ConcurrentAccess(t *testing.T) {
	c := &NoOpCollector{}

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			_ = c.Track(context.Background(), Event{Name: "test"})
			_, _ = c.Query(
				context.Background(), "events", "name", time.Hour,
			)
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

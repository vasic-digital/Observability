package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCollector(t *testing.T) (*PrometheusCollector, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	cfg := &PrometheusConfig{
		Namespace:      "test",
		Subsystem:      "unit",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       reg,
	}
	return NewPrometheusCollector(cfg), reg
}

func TestDefaultPrometheusConfig(t *testing.T) {
	cfg := DefaultPrometheusConfig()
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.DefaultBuckets)
}

func TestNewPrometheusCollector_NilConfig(t *testing.T) {
	c := NewPrometheusCollector(nil)
	assert.NotNil(t, c)
	assert.NotNil(t, c.counters)
	assert.NotNil(t, c.histograms)
	assert.NotNil(t, c.gauges)
}

func TestPrometheusCollector_RegisterCounter(t *testing.T) {
	tests := []struct {
		name        string
		metricName  string
		help        string
		labels      []string
		registerErr bool
	}{
		{
			name:       "register new counter",
			metricName: "requests_total",
			help:       "Total requests",
			labels:     []string{"method", "status"},
		},
		{
			name:       "register duplicate is ok",
			metricName: "requests_total",
			help:       "Total requests",
			labels:     []string{"method", "status"},
		},
	}

	c, _ := newTestCollector(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.RegisterCounter(tt.metricName, tt.help, tt.labels)
			if tt.registerErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrometheusCollector_RegisterHistogram(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		help       string
		labels     []string
		buckets    []float64
	}{
		{
			name:       "register with default buckets",
			metricName: "request_duration",
			help:       "Request duration",
			labels:     []string{"endpoint"},
			buckets:    nil,
		},
		{
			name:       "register with custom buckets",
			metricName: "response_size",
			help:       "Response size",
			labels:     []string{"type"},
			buckets:    []float64{100, 500, 1000, 5000},
		},
	}

	c, _ := newTestCollector(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.RegisterHistogram(
				tt.metricName, tt.help, tt.labels, tt.buckets,
			)
			assert.NoError(t, err)
		})
	}
}

func TestPrometheusCollector_RegisterGauge(t *testing.T) {
	c, _ := newTestCollector(t)

	err := c.RegisterGauge("connections", "Active connections", []string{"pool"})
	assert.NoError(t, err)

	// Re-register should be fine
	err = c.RegisterGauge("connections", "Active connections", []string{"pool"})
	assert.NoError(t, err)
}

func TestPrometheusCollector_IncrementCounter(t *testing.T) {
	c, reg := newTestCollector(t)

	labels := map[string]string{"method": "GET"}
	c.IncrementCounter("http_requests", labels)
	c.IncrementCounter("http_requests", labels)
	c.IncrementCounter("http_requests", labels)

	metrics, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, mf := range metrics {
		if mf.GetName() == "test_unit_http_requests" {
			found = true
			require.Len(t, mf.GetMetric(), 1)
			assert.Equal(t, 3.0, mf.GetMetric()[0].GetCounter().GetValue())
		}
	}
	assert.True(t, found, "metric test_unit_http_requests should exist")
}

func TestPrometheusCollector_AddCounter(t *testing.T) {
	c, reg := newTestCollector(t)

	labels := map[string]string{"type": "tokens"}
	c.AddCounter("processed", 100, labels)
	c.AddCounter("processed", 50, labels)

	metrics, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, mf := range metrics {
		if mf.GetName() == "test_unit_processed" {
			found = true
			assert.Equal(t, 150.0, mf.GetMetric()[0].GetCounter().GetValue())
		}
	}
	assert.True(t, found)
}

func TestPrometheusCollector_RecordLatency(t *testing.T) {
	c, reg := newTestCollector(t)

	labels := map[string]string{"endpoint": "/api"}
	c.RecordLatency("request_duration", 250*time.Millisecond, labels)
	c.RecordLatency("request_duration", 500*time.Millisecond, labels)

	metrics, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, mf := range metrics {
		if mf.GetName() == "test_unit_request_duration" {
			found = true
			assert.Equal(t,
				uint64(2),
				mf.GetMetric()[0].GetHistogram().GetSampleCount(),
			)
		}
	}
	assert.True(t, found)
}

func TestPrometheusCollector_RecordValue(t *testing.T) {
	c, reg := newTestCollector(t)

	labels := map[string]string{"source": "cache"}
	c.RecordValue("score", 0.95, labels)

	metrics, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, mf := range metrics {
		if mf.GetName() == "test_unit_score" {
			found = true
			assert.Equal(t,
				uint64(1),
				mf.GetMetric()[0].GetHistogram().GetSampleCount(),
			)
		}
	}
	assert.True(t, found)
}

func TestPrometheusCollector_SetGauge(t *testing.T) {
	c, reg := newTestCollector(t)

	labels := map[string]string{"pool": "main"}
	c.SetGauge("active_connections", 42, labels)

	metrics, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, mf := range metrics {
		if mf.GetName() == "test_unit_active_connections" {
			found = true
			assert.Equal(t, 42.0, mf.GetMetric()[0].GetGauge().GetValue())
		}
	}
	assert.True(t, found)

	// Update the gauge
	c.SetGauge("active_connections", 10, labels)
	metrics, err = reg.Gather()
	require.NoError(t, err)

	for _, mf := range metrics {
		if mf.GetName() == "test_unit_active_connections" {
			assert.Equal(t, 10.0, mf.GetMetric()[0].GetGauge().GetValue())
		}
	}
}

func TestPrometheusCollector_AutoCreateOnUse(t *testing.T) {
	c, reg := newTestCollector(t)

	// These should auto-create the metrics
	c.IncrementCounter("auto_counter", map[string]string{"k": "v"})
	c.RecordValue("auto_histogram", 1.5, map[string]string{"k": "v"})
	c.SetGauge("auto_gauge", 7.0, map[string]string{"k": "v"})

	metrics, err := reg.Gather()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(metrics), 3)
}

func TestPrometheusCollector_NilLabels(t *testing.T) {
	c, reg := newTestCollector(t)

	c.IncrementCounter("no_label_counter", nil)
	c.RecordLatency("no_label_hist", time.Second, nil)
	c.SetGauge("no_label_gauge", 1.0, nil)

	metrics, err := reg.Gather()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(metrics), 3)
}

func TestPrometheusCollector_ConcurrentAccess(t *testing.T) {
	c, _ := newTestCollector(t)

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			labels := map[string]string{"worker": "test"}
			c.IncrementCounter("concurrent_counter", labels)
			c.RecordLatency("concurrent_latency", time.Millisecond, labels)
			c.SetGauge("concurrent_gauge", 1.0, labels)
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestNoOpCollector(t *testing.T) {
	var c Collector = &NoOpCollector{}

	// All methods should not panic
	c.IncrementCounter("test", nil)
	c.AddCounter("test", 1.0, nil)
	c.RecordLatency("test", time.Second, nil)
	c.RecordValue("test", 1.0, nil)
	c.SetGauge("test", 1.0, nil)
}

func TestNoOpCollector_ImplementsInterface(t *testing.T) {
	var _ Collector = &NoOpCollector{}
	var _ Collector = &PrometheusCollector{}
}

func TestLabelKeys(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		count  int
	}{
		{name: "nil labels", labels: nil, count: 0},
		{name: "empty labels", labels: map[string]string{}, count: 0},
		{
			name:   "two labels",
			labels: map[string]string{"a": "1", "b": "2"},
			count:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := labelKeys(tt.labels)
			assert.Len(t, keys, tt.count)
		})
	}
}

func TestToPrometheusLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		count  int
	}{
		{name: "nil", labels: nil, count: 0},
		{
			name:   "with values",
			labels: map[string]string{"a": "1"},
			count:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pl := toPrometheusLabels(tt.labels)
			assert.Len(t, pl, tt.count)
		})
	}
}

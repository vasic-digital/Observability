package metrics_test

import (
	"math"
	"sync"
	"testing"
	"time"

	"digital.vasic.observability/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func newTestRegistry() (*prometheus.Registry, *metrics.PrometheusConfig) {
	reg := prometheus.NewRegistry()
	return reg, &metrics.PrometheusConfig{
		Namespace: "edge",
		Subsystem: "test",
		Registry:  reg,
	}
}

func TestPrometheusCollector_EmptyMetricName(t *testing.T) {
	t.Parallel()

	_, cfg := newTestRegistry()
	c := metrics.NewPrometheusCollector(cfg)

	// Empty name -- Prometheus may not register it on the custom registry
	// but should not panic
	assert.NotPanics(t, func() {
		c.IncrementCounter("", nil)
		c.SetGauge("", 42.0, nil)
		c.RecordValue("", 1.5, nil)
	})
}

func TestPrometheusCollector_ConcurrentMetricUpdates(t *testing.T) {
	t.Parallel()

	_, cfg := newTestRegistry()
	c := metrics.NewPrometheusCollector(cfg)

	labels := map[string]string{"method": "GET"}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			c.IncrementCounter("concurrent_counter", labels)
		}()
		go func() {
			defer wg.Done()
			c.SetGauge("concurrent_gauge", 1.0, labels)
		}()
		go func() {
			defer wg.Done()
			c.RecordLatency("concurrent_latency", 50*time.Millisecond, labels)
		}()
	}

	wg.Wait()
	// No panics or races means the test passes
}

func TestPrometheusCollector_ManyUniqueLabelValues(t *testing.T) {
	t.Parallel()

	_, cfg := newTestRegistry()
	c := metrics.NewPrometheusCollector(cfg)

	// Pre-register the counter with a known label name
	err := c.RegisterCounter("cardinality_counter", "test counter", []string{"user_id"})
	assert.NoError(t, err)

	// Create many unique label values (high cardinality)
	for i := 0; i < 200; i++ {
		labels := map[string]string{
			"user_id": "user_" + time.Now().Format("150405") + "_" + string(rune('A'+i%26)),
		}
		c.IncrementCounter("cardinality_counter", labels)
	}

	// Should not panic or error out even with high cardinality
}

func TestPrometheusCollector_VeryLargeValues(t *testing.T) {
	t.Parallel()

	reg, cfg := newTestRegistry()
	c := metrics.NewPrometheusCollector(cfg)

	labels := map[string]string{"tier": "heavy"}

	// Very large counter value
	c.AddCounter("large_counter", math.MaxFloat64/2, labels)

	// Very large gauge value
	c.SetGauge("large_gauge", math.MaxFloat64, labels)

	// Very large histogram observation
	c.RecordValue("large_histogram", math.MaxFloat64, labels)

	families, err := reg.Gather()
	assert.NoError(t, err)
	assert.NotEmpty(t, families)
}

func TestPrometheusCollector_NegativeValues(t *testing.T) {
	t.Parallel()

	_, cfg := newTestRegistry()
	c := metrics.NewPrometheusCollector(cfg)

	labels := map[string]string{"op": "test"}

	// Negative gauge value is valid
	c.SetGauge("negative_gauge", -42.0, labels)

	// Negative histogram observation is valid (records in underflow bucket)
	c.RecordValue("negative_hist", -1.0, labels)

	// Negative latency -- seconds() of a negative duration
	c.RecordLatency("negative_latency", -100*time.Millisecond, labels)
}

func TestPrometheusCollector_NilLabels(t *testing.T) {
	t.Parallel()

	_, cfg := newTestRegistry()
	c := metrics.NewPrometheusCollector(cfg)

	// All methods should handle nil labels without panicking
	assert.NotPanics(t, func() {
		c.IncrementCounter("nil_label_counter", nil)
	})
	assert.NotPanics(t, func() {
		c.SetGauge("nil_label_gauge", 1.0, nil)
	})
	assert.NotPanics(t, func() {
		c.RecordValue("nil_label_hist", 1.0, nil)
	})
	assert.NotPanics(t, func() {
		c.RecordLatency("nil_label_latency", time.Second, nil)
	})
}

func TestPrometheusCollector_DuplicateRegistration(t *testing.T) {
	t.Parallel()

	_, cfg := newTestRegistry()
	c := metrics.NewPrometheusCollector(cfg)

	err := c.RegisterCounter("dup_counter", "first", []string{"k"})
	assert.NoError(t, err)

	// Duplicate registration with same name should be a no-op
	err = c.RegisterCounter("dup_counter", "second", []string{"k"})
	assert.NoError(t, err)

	err = c.RegisterHistogram("dup_hist", "first", []string{"k"}, nil)
	assert.NoError(t, err)

	err = c.RegisterHistogram("dup_hist", "second", []string{"k"}, nil)
	assert.NoError(t, err)

	err = c.RegisterGauge("dup_gauge", "first", []string{"k"})
	assert.NoError(t, err)

	err = c.RegisterGauge("dup_gauge", "second", []string{"k"})
	assert.NoError(t, err)
}

func TestNoOpCollector_AllMethods(t *testing.T) {
	t.Parallel()

	var c metrics.Collector = &metrics.NoOpCollector{}

	// All methods should be no-ops and not panic
	assert.NotPanics(t, func() {
		c.IncrementCounter("x", nil)
		c.AddCounter("x", 1.0, nil)
		c.RecordLatency("x", time.Second, nil)
		c.RecordValue("x", 1.0, nil)
		c.SetGauge("x", 1.0, nil)
	})
}

func TestPrometheusCollector_NilConfig(t *testing.T) {
	t.Parallel()

	// nil config should use DefaultPrometheusConfig
	c := metrics.NewPrometheusCollector(nil)
	assert.NotNil(t, c)

	// Should still be functional
	c.IncrementCounter("nil_cfg_counter", map[string]string{"a": "1"})
}

func TestPrometheusCollector_ConcurrentAutoCreate(t *testing.T) {
	t.Parallel()

	_, cfg := newTestRegistry()
	c := metrics.NewPrometheusCollector(cfg)

	var wg sync.WaitGroup
	// Multiple goroutines auto-creating the same metric simultaneously
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.IncrementCounter("race_counter", map[string]string{"env": "test"})
		}()
	}
	wg.Wait()
}

func TestPrometheusCollector_ZeroValueGauge(t *testing.T) {
	t.Parallel()

	reg, cfg := newTestRegistry()
	c := metrics.NewPrometheusCollector(cfg)

	c.SetGauge("zero_gauge", 0.0, map[string]string{"kind": "zero"})

	families, err := reg.Gather()
	assert.NoError(t, err)

	found := false
	for _, f := range families {
		if f.GetName() == "edge_test_zero_gauge" {
			found = true
			break
		}
	}
	assert.True(t, found, "zero-value gauge should still be registered")
}

func TestPrometheusCollector_InfAndNaN(t *testing.T) {
	t.Parallel()

	_, cfg := newTestRegistry()
	c := metrics.NewPrometheusCollector(cfg)

	labels := map[string]string{"special": "true"}

	// Prometheus allows +Inf and NaN observations in histograms
	assert.NotPanics(t, func() {
		c.RecordValue("inf_hist", math.Inf(1), labels)
	})
	assert.NotPanics(t, func() {
		c.RecordValue("nan_hist", math.NaN(), labels)
	})
	assert.NotPanics(t, func() {
		c.SetGauge("inf_gauge", math.Inf(-1), labels)
	})
}

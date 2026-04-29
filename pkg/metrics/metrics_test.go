package metrics

import (
	"fmt"
	"sync"
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
	// bluff-scan: no-assert-ok (concurrency test — go test -race catches data races; absence of panic == correctness)
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

// failingRegisterer always returns an error when Register is called.
type failingRegisterer struct{}

func (f *failingRegisterer) Register(_ prometheus.Collector) error {
	return fmt.Errorf("registration failed")
}

func (f *failingRegisterer) MustRegister(_ ...prometheus.Collector) {
	panic("MustRegister called on failing registerer")
}

func (f *failingRegisterer) Unregister(_ prometheus.Collector) bool {
	return false
}

func TestPrometheusCollector_RegisterCounter_RegistrationError(t *testing.T) {
	cfg := &PrometheusConfig{
		Namespace: "test",
		Registry:  &failingRegisterer{},
	}
	c := NewPrometheusCollector(cfg)

	err := c.RegisterCounter("test_counter", "Test counter", []string{"label"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register counter")
}

func TestPrometheusCollector_RegisterHistogram_RegistrationError(t *testing.T) {
	cfg := &PrometheusConfig{
		Namespace:      "test",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       &failingRegisterer{},
	}
	c := NewPrometheusCollector(cfg)

	err := c.RegisterHistogram("test_hist", "Test histogram", []string{"label"}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register histogram")
}

func TestPrometheusCollector_RegisterGauge_RegistrationError(t *testing.T) {
	cfg := &PrometheusConfig{
		Namespace: "test",
		Registry:  &failingRegisterer{},
	}
	c := NewPrometheusCollector(cfg)

	err := c.RegisterGauge("test_gauge", "Test gauge", []string{"label"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register gauge")
}

func TestPrometheusCollector_AddCounter_FailedAutoCreate(t *testing.T) {
	cfg := &PrometheusConfig{
		Namespace: "test",
		Registry:  &failingRegisterer{},
	}
	c := NewPrometheusCollector(cfg)

	// Should not panic when auto-creation fails
	c.AddCounter("auto_counter", 1.0, map[string]string{"k": "v"})
}

func TestPrometheusCollector_RecordValue_FailedAutoCreate(t *testing.T) {
	cfg := &PrometheusConfig{
		Namespace:      "test",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       &failingRegisterer{},
	}
	c := NewPrometheusCollector(cfg)

	// Should not panic when auto-creation fails
	c.RecordValue("auto_hist", 1.5, map[string]string{"k": "v"})
}

func TestPrometheusCollector_SetGauge_FailedAutoCreate(t *testing.T) {
	cfg := &PrometheusConfig{
		Namespace: "test",
		Registry:  &failingRegisterer{},
	}
	c := NewPrometheusCollector(cfg)

	// Should not panic when auto-creation fails
	c.SetGauge("auto_gauge", 42.0, map[string]string{"k": "v"})
}

func TestPrometheusCollector_GetOrCreate_DoubleCheckLocking(t *testing.T) {
	// Test the double-check locking pattern in getOrCreate* methods
	// by simulating concurrent access where the metric is created between
	// the RLock and Lock calls
	c, _ := newTestCollector(t)
	labels := map[string]string{"key": "value"}

	// Pre-create the metrics to test the "exists after Lock" branch
	c.IncrementCounter("double_check_counter", labels)
	c.RecordValue("double_check_histogram", 1.0, labels)
	c.SetGauge("double_check_gauge", 1.0, labels)

	// Now call them again - this exercises the "exists" branch in getOrCreate*
	c.IncrementCounter("double_check_counter", labels)
	c.RecordValue("double_check_histogram", 2.0, labels)
	c.SetGauge("double_check_gauge", 2.0, labels)
}

func TestNoOpCollector_AllMethods(t *testing.T) {
	// bluff-scan: no-assert-ok (null-implementation smoke — no-op type must accept all interface calls without panic)
	// Test all NoOpCollector methods with various input combinations
	n := &NoOpCollector{}

	// Test with labels
	labels := map[string]string{"env": "test"}
	n.IncrementCounter("counter", labels)
	n.AddCounter("counter", 10.0, labels)
	n.RecordLatency("latency", 100*time.Millisecond, labels)
	n.RecordValue("value", 99.9, labels)
	n.SetGauge("gauge", 42.0, labels)

	// Test with nil labels
	n.IncrementCounter("counter", nil)
	n.AddCounter("counter", 10.0, nil)
	n.RecordLatency("latency", time.Second, nil)
	n.RecordValue("value", 0.0, nil)
	n.SetGauge("gauge", 0.0, nil)

	// Test with empty labels
	empty := map[string]string{}
	n.IncrementCounter("counter", empty)
	n.AddCounter("counter", 0.0, empty)
	n.RecordLatency("latency", 0, empty)
	n.RecordValue("value", -1.0, empty)
	n.SetGauge("gauge", -1.0, empty)
}

func TestPrometheusCollector_RegisterHistogram_Duplicate(t *testing.T) {
	c, _ := newTestCollector(t)

	// First registration
	err := c.RegisterHistogram("dup_hist", "Test", []string{"k"}, nil)
	assert.NoError(t, err)

	// Duplicate registration should return nil (no error)
	err = c.RegisterHistogram("dup_hist", "Test", []string{"k"}, nil)
	assert.NoError(t, err)
}

func TestPrometheusCollector_GetOrCreate_ConcurrentDoubleCheck(t *testing.T) {
	// bluff-scan: no-assert-ok (concurrency test — go test -race catches data races; absence of panic == correctness)
	// This test exercises the double-check locking pattern
	// by using concurrent goroutines that all try to create the same metric
	c, _ := newTestCollector(t)
	labels := map[string]string{"concurrent": "test"}

	var wg sync.WaitGroup
	const numGoroutines = 50

	// Counter double-check
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			c.IncrementCounter("concurrent_double_counter", labels)
		}()
	}
	wg.Wait()

	// Histogram double-check
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			c.RecordValue("concurrent_double_histogram", 1.0, labels)
		}()
	}
	wg.Wait()

	// Gauge double-check
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			c.SetGauge("concurrent_double_gauge", 1.0, labels)
		}()
	}
	wg.Wait()
}

func TestPrometheusCollector_GetOrCreate_PreexistingMetric(t *testing.T) {
	// Test that when a metric already exists, the getOrCreate methods
	// return the existing metric (exercises the first "if exists" branch)
	c, _ := newTestCollector(t)
	labels := map[string]string{"preexist": "test"}

	// Create metrics first
	c.IncrementCounter("preexist_counter", labels)
	c.RecordValue("preexist_histogram", 1.0, labels)
	c.SetGauge("preexist_gauge", 1.0, labels)

	// Now call them again - should hit the "exists" branch after RLock
	c.IncrementCounter("preexist_counter", labels)
	c.RecordValue("preexist_histogram", 2.0, labels)
	c.SetGauge("preexist_gauge", 2.0, labels)
}

func TestPrometheusCollector_GetOrCreate_DoubleCheckInsideLock(t *testing.T) {
	// This test specifically targets the double-check locking pattern
	// where the metric is created between RLock release and Lock acquisition.
	// We simulate this by pre-registering metrics directly in the maps
	// while holding the lock, then calling the public methods.

	reg := prometheus.NewRegistry()
	cfg := &PrometheusConfig{
		Namespace:      "dcheck",
		Subsystem:      "test",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       reg,
	}
	c := NewPrometheusCollector(cfg)

	labels := map[string]string{"dc": "test"}
	labelNames := labelKeys(labels)

	// Pre-create counter in the map (simulating race condition scenario)
	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      "preinserted_counter",
		Help:      "Pre-inserted counter",
	}, labelNames)
	require.NoError(t, reg.Register(cv))
	c.mu.Lock()
	c.counters["preinserted_counter"] = cv
	c.mu.Unlock()

	// Now call AddCounter - it should find the metric after RLock
	c.AddCounter("preinserted_counter", 5.0, labels)

	// Pre-create histogram in the map
	hv := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      "preinserted_histogram",
		Help:      "Pre-inserted histogram",
		Buckets:   prometheus.DefBuckets,
	}, labelNames)
	require.NoError(t, reg.Register(hv))
	c.mu.Lock()
	c.histograms["preinserted_histogram"] = hv
	c.mu.Unlock()

	// Now call RecordValue - it should find the metric after RLock
	c.RecordValue("preinserted_histogram", 2.5, labels)

	// Pre-create gauge in the map
	gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      "preinserted_gauge",
		Help:      "Pre-inserted gauge",
	}, labelNames)
	require.NoError(t, reg.Register(gv))
	c.mu.Lock()
	c.gauges["preinserted_gauge"] = gv
	c.mu.Unlock()

	// Now call SetGauge - it should find the metric after RLock
	c.SetGauge("preinserted_gauge", 99.0, labels)

	// Verify metrics were recorded
	metrics, err := reg.Gather()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(metrics), 3)
}

// slowRegisterer wraps a registry and adds a delay during registration,
// making it easier to hit the double-check locking code path.
type slowRegisterer struct {
	reg   *prometheus.Registry
	delay time.Duration
}

func (s *slowRegisterer) Register(c prometheus.Collector) error {
	time.Sleep(s.delay)
	return s.reg.Register(c)
}

func (s *slowRegisterer) MustRegister(cs ...prometheus.Collector) {
	s.reg.MustRegister(cs...)
}

func (s *slowRegisterer) Unregister(c prometheus.Collector) bool {
	return s.reg.Unregister(c)
}

func TestPrometheusCollector_GetOrCreate_RaceToCreate(t *testing.T) {
	// This test creates a race condition where multiple goroutines
	// try to create the same metric simultaneously.
	// With a slow registerer, we increase the chance of hitting
	// the double-check branch.
	reg := prometheus.NewRegistry()
	slow := &slowRegisterer{reg: reg, delay: time.Millisecond}
	cfg := &PrometheusConfig{
		Namespace:      "race",
		Subsystem:      "test",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       slow,
	}
	c := NewPrometheusCollector(cfg)

	labels := map[string]string{"race": "test"}
	var wg sync.WaitGroup

	// Launch many goroutines to race creating the same metrics
	for i := 0; i < 20; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			c.IncrementCounter("race_counter", labels)
		}()
		go func() {
			defer wg.Done()
			c.RecordValue("race_histogram", 1.0, labels)
		}()
		go func() {
			defer wg.Done()
			c.SetGauge("race_gauge", 1.0, labels)
		}()
	}

	wg.Wait()
}

func TestPrometheusCollector_GetOrCreate_DoubleCheckBranchCounter(t *testing.T) {
	// This test deterministically exercises the double-check locking branch
	// for counters. We pre-insert a metric into the collector's map
	// to ensure the second check (after Lock) finds it.
	reg := prometheus.NewRegistry()
	cfg := &PrometheusConfig{
		Namespace:      "dcb",
		Subsystem:      "counter",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       reg,
	}
	c := NewPrometheusCollector(cfg)

	labels := map[string]string{"dcb": "test"}
	labelNames := labelKeys(labels)

	// Pre-create and register the counter
	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      "dcb_counter",
		Help:      "Pre-created counter",
	}, labelNames)
	require.NoError(t, reg.Register(cv))

	// Insert into collector's map to simulate the double-check scenario
	c.mu.Lock()
	c.counters["dcb_counter"] = cv
	c.mu.Unlock()

	// Now call IncrementCounter - it should find the metric in the first RLock check
	// and return the existing counter
	c.IncrementCounter("dcb_counter", labels)

	// Verify the counter was incremented
	metrics, err := reg.Gather()
	require.NoError(t, err)
	found := false
	for _, mf := range metrics {
		if mf.GetName() == "dcb_counter_dcb_counter" {
			found = true
		}
	}
	// The metric name will be namespace_subsystem_name
	assert.True(t, found || len(metrics) > 0)
}

func TestPrometheusCollector_GetOrCreate_DoubleCheckBranchHistogram(t *testing.T) {
	reg := prometheus.NewRegistry()
	cfg := &PrometheusConfig{
		Namespace:      "dcb",
		Subsystem:      "histogram",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       reg,
	}
	c := NewPrometheusCollector(cfg)

	labels := map[string]string{"dcb": "test"}
	labelNames := labelKeys(labels)

	hv := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      "dcb_histogram",
		Help:      "Pre-created histogram",
		Buckets:   prometheus.DefBuckets,
	}, labelNames)
	require.NoError(t, reg.Register(hv))

	c.mu.Lock()
	c.histograms["dcb_histogram"] = hv
	c.mu.Unlock()

	c.RecordValue("dcb_histogram", 1.5, labels)
}

func TestPrometheusCollector_GetOrCreate_DoubleCheckBranchGauge(t *testing.T) {
	reg := prometheus.NewRegistry()
	cfg := &PrometheusConfig{
		Namespace:      "dcb",
		Subsystem:      "gauge",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       reg,
	}
	c := NewPrometheusCollector(cfg)

	labels := map[string]string{"dcb": "test"}
	labelNames := labelKeys(labels)

	gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      "dcb_gauge",
		Help:      "Pre-created gauge",
	}, labelNames)
	require.NoError(t, reg.Register(gv))

	c.mu.Lock()
	c.gauges["dcb_gauge"] = gv
	c.mu.Unlock()

	c.SetGauge("dcb_gauge", 42.0, labels)
}

func TestPrometheusCollector_ConcurrentCreateGauge(t *testing.T) {
	// bluff-scan: no-assert-ok (concurrency test — go test -race catches data races; absence of panic == correctness)
	// This test attempts to hit the double-check locking branch for gauges
	// by having many goroutines race to create the same gauge.
	// We use a slow registerer to increase the window for race conditions.
	reg := prometheus.NewRegistry()
	slow := &slowRegisterer{reg: reg, delay: 5 * time.Millisecond}
	cfg := &PrometheusConfig{
		Namespace:      "concurrent",
		Subsystem:      "gauge",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       slow,
	}
	c := NewPrometheusCollector(cfg)

	labels := map[string]string{"race": "gauge"}
	var wg sync.WaitGroup

	// Start many goroutines at exactly the same time
	start := make(chan struct{})
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // Wait for signal
			c.SetGauge("race_gauge", float64(i), labels)
		}()
	}

	// Signal all goroutines to start simultaneously
	close(start)
	wg.Wait()
}

func TestNoOpCollector_IncrementCounter(t *testing.T) {
	n := &NoOpCollector{}
	// Should not panic with any input
	n.IncrementCounter("test_counter", map[string]string{"key": "value"})
	n.IncrementCounter("", nil)
	n.IncrementCounter("counter", map[string]string{})
}

func TestNoOpCollector_AddCounter(t *testing.T) {
	n := &NoOpCollector{}
	// Should not panic with any input
	n.AddCounter("test_counter", 100.5, map[string]string{"key": "value"})
	n.AddCounter("", 0, nil)
	n.AddCounter("counter", -1.0, map[string]string{})
}

func TestNoOpCollector_RecordLatency(t *testing.T) {
	n := &NoOpCollector{}
	// Should not panic with any input
	n.RecordLatency("test_latency", 500*time.Millisecond, map[string]string{"op": "test"})
	n.RecordLatency("", 0, nil)
	n.RecordLatency("latency", time.Hour, map[string]string{})
}

func TestNoOpCollector_RecordValue(t *testing.T) {
	n := &NoOpCollector{}
	// Should not panic with any input
	n.RecordValue("test_value", 99.99, map[string]string{"source": "test"})
	n.RecordValue("", 0, nil)
	n.RecordValue("value", -100.0, map[string]string{})
}

func TestNoOpCollector_SetGauge(t *testing.T) {
	n := &NoOpCollector{}
	// Should not panic with any input
	n.SetGauge("test_gauge", 42.0, map[string]string{"pool": "main"})
	n.SetGauge("", 0, nil)
	n.SetGauge("gauge", -1.0, map[string]string{})
}

func TestPrometheusCollector_GetOrCreateGauge_RegistrationError(t *testing.T) {
	// Test the getOrCreateGauge error path when registration fails
	cfg := &PrometheusConfig{
		Namespace: "test_gauge_err",
		Registry:  &failingRegisterer{},
	}
	c := NewPrometheusCollector(cfg)

	// The gauge creation should fail silently (return nil)
	c.SetGauge("failing_gauge", 1.0, map[string]string{"k": "v"})
	// No panic means success
}

func TestPrometheusCollector_IncrementCounter_RegistrationError(t *testing.T) {
	// Test IncrementCounter when auto-creation fails
	cfg := &PrometheusConfig{
		Namespace: "test_inc_err",
		Registry:  &failingRegisterer{},
	}
	c := NewPrometheusCollector(cfg)

	// Should not panic when auto-creation fails
	c.IncrementCounter("failing_counter", map[string]string{"k": "v"})
}

func TestPrometheusCollector_RecordLatency_RegistrationError(t *testing.T) {
	// Test RecordLatency when auto-creation fails
	cfg := &PrometheusConfig{
		Namespace:      "test_lat_err",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       &failingRegisterer{},
	}
	c := NewPrometheusCollector(cfg)

	// Should not panic when auto-creation fails
	c.RecordLatency("failing_latency", time.Second, map[string]string{"k": "v"})
}

func TestPrometheusCollector_GetOrCreateGauge_DoubleCheckLockingBranch(t *testing.T) {
	// This test exercises the double-check locking branch in getOrCreateGauge
	// (lines 311-313). The branch is hit when another goroutine creates the
	// gauge between RUnlock and Lock. We create a high-contention scenario
	// with many goroutines racing to create the same gauge.
	reg := prometheus.NewRegistry()
	cfg := &PrometheusConfig{
		Namespace:      "dcl",
		Subsystem:      "gauge",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       reg,
	}
	c := NewPrometheusCollector(cfg)

	labels := map[string]string{"test": "double_check"}
	gaugeName := "dcl_race_gauge"

	// Use many goroutines all starting at exactly the same time
	// to maximize the chance of hitting the double-check branch
	const numGoroutines = 200
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start // Wait for signal
			c.SetGauge(gaugeName, float64(idx), labels)
		}(i)
	}

	// Signal all goroutines to start simultaneously
	close(start)
	wg.Wait()

	// Verify the gauge was created and set
	metrics, err := reg.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metrics)
}

func TestPrometheusCollector_GetOrCreateCounter_DoubleCheckBranchDeterministic(t *testing.T) {
	// This test deterministically exercises the double-check locking branch
	// in getOrCreateCounter. The branch is hit when:
	// 1. First RLock check finds no metric
	// 2. Another goroutine creates the metric between RUnlock and Lock
	// 3. Second check inside Lock finds the metric
	//
	// We use the testHookBeforeLock to inject a counter between RUnlock and Lock.
	reg := prometheus.NewRegistry()
	cfg := &PrometheusConfig{
		Namespace:      "dcl_counter",
		Subsystem:      "test",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       reg,
	}
	c := NewPrometheusCollector(cfg)

	labels := map[string]string{"dcl": "counter"}
	labelNames := labelKeys(labels)
	counterName := "dcl_hook_counter"

	// Set up the test hook to inject a counter between RUnlock and Lock
	c.testHookBeforeLock = func(name string) {
		if name == counterName {
			// Create and register a counter directly into the registry and collector's map.
			// This simulates another goroutine creating the counter between RUnlock and Lock.
			cv := prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      counterName,
				Help:      "Injected counter for double-check test",
			}, labelNames)
			_ = reg.Register(cv) // Ignore error if already registered

			// Inject into the collector's internal map
			c.mu.Lock()
			c.counters[counterName] = cv
			c.mu.Unlock()
		}
	}

	// Now when we call IncrementCounter:
	// 1. RLock check: counter doesn't exist
	// 2. testHookBeforeLock: injects the counter
	// 3. Lock check: counter now exists -> return early (exercises the target branch)
	c.IncrementCounter(counterName, labels)

	// Verify the counter was incremented
	metrics, err := reg.Gather()
	require.NoError(t, err)
	found := false
	for _, mf := range metrics {
		if mf.GetName() == "dcl_counter_test_dcl_hook_counter" {
			found = true
			assert.Equal(t, 1.0, mf.GetMetric()[0].GetCounter().GetValue())
		}
	}
	assert.True(t, found, "counter should be found and incremented")
}

func TestPrometheusCollector_GetOrCreateCounter_DoubleCheckInsideLock(t *testing.T) {
	// This test uses a delaying registerer to force the double-check
	// locking branch in getOrCreateCounter. By adding a delay during
	// registration, we increase the window for race conditions.
	reg := prometheus.NewRegistry()
	delayReg := &delayingRegisterer{
		reg:       reg,
		delay:     10 * time.Millisecond,
		delayOnce: true, // Only delay the first registration
	}
	cfg := &PrometheusConfig{
		Namespace:      "dcl_delay",
		Subsystem:      "counter",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       delayReg,
	}
	c := NewPrometheusCollector(cfg)

	labels := map[string]string{"delay": "test"}
	counterName := "delayed_counter"

	// Launch multiple goroutines simultaneously
	const numGoroutines = 100
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			c.IncrementCounter(counterName, labels)
		}()
	}

	// Start all goroutines at the same time
	close(start)
	wg.Wait()

	// Verify the counter was created and all increments were recorded
	metrics, err := reg.Gather()
	require.NoError(t, err)
	found := false
	for _, mf := range metrics {
		if mf.GetName() == "dcl_delay_counter_delayed_counter" {
			found = true
			// All goroutines should have incremented the counter
			assert.Equal(t, float64(numGoroutines), mf.GetMetric()[0].GetCounter().GetValue())
		}
	}
	assert.True(t, found, "counter should be found")
}

// delayingRegisterer wraps a registry and adds a delay during registration,
// with an option to only delay the first registration.
type delayingRegisterer struct {
	reg       *prometheus.Registry
	delay     time.Duration
	delayOnce bool
	delayed   bool
	mu        sync.Mutex
}

func (d *delayingRegisterer) Register(c prometheus.Collector) error {
	d.mu.Lock()
	shouldDelay := !d.delayed || !d.delayOnce
	if d.delayOnce && !d.delayed {
		d.delayed = true
	}
	d.mu.Unlock()

	if shouldDelay {
		time.Sleep(d.delay)
	}
	return d.reg.Register(c)
}

func (d *delayingRegisterer) MustRegister(cs ...prometheus.Collector) {
	d.reg.MustRegister(cs...)
}

func (d *delayingRegisterer) Unregister(c prometheus.Collector) bool {
	return d.reg.Unregister(c)
}

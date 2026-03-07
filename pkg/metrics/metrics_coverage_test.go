package metrics

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrometheusCollector_GetOrCreateHistogram_DoubleCheckBranch deterministically
// exercises the double-check locking branch in getOrCreateHistogram (lines 283-285).
// It uses the testHookBeforeLock mechanism to inject a histogram between the
// RUnlock and Lock calls.
func TestPrometheusCollector_GetOrCreateHistogram_DoubleCheckBranch(t *testing.T) {
	reg := prometheus.NewRegistry()
	cfg := &PrometheusConfig{
		Namespace:      "hist_dcl",
		Subsystem:      "coverage",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       reg,
	}
	c := NewPrometheusCollector(cfg)

	labels := map[string]string{"dcl": "hist"}
	labelNames := labelKeys(labels)
	histName := "dcl_hook_histogram"

	// Set up a test hook that injects a histogram into the collector's
	// internal map between RUnlock and Lock. This deterministically triggers
	// the second check (line 283: "if hv, exists = c.histograms[name]; exists").
	c.testHookBeforeLock = func(name string) {
		if name == histName {
			hv := prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      histName,
				Help:      "Injected histogram for double-check coverage",
				Buckets:   prometheus.DefBuckets,
			}, labelNames)
			_ = reg.Register(hv)

			c.mu.Lock()
			c.histograms[histName] = hv
			c.mu.Unlock()
		}
	}

	// When RecordValue is called:
	// 1. RLock check: histogram doesn't exist
	// 2. testHookBeforeLock: injects the histogram
	// 3. Lock check: histogram now exists -> returns early (covers line 283-285)
	c.RecordValue(histName, 42.0, labels)

	// Verify the histogram was recorded.
	metrics, err := reg.Gather()
	require.NoError(t, err)
	found := false
	for _, mf := range metrics {
		if mf.GetName() == "hist_dcl_coverage_dcl_hook_histogram" {
			found = true
			assert.Equal(t,
				uint64(1),
				mf.GetMetric()[0].GetHistogram().GetSampleCount(),
			)
		}
	}
	assert.True(t, found, "histogram should be found and have one observation")
}

// TestPrometheusCollector_GetOrCreateHistogram_ConcurrentRace uses high
// concurrency to exercise the double-check locking branch in getOrCreateHistogram.
func TestPrometheusCollector_GetOrCreateHistogram_ConcurrentRace(t *testing.T) {
	reg := prometheus.NewRegistry()
	cfg := &PrometheusConfig{
		Namespace:      "hist_race",
		Subsystem:      "cov",
		DefaultBuckets: prometheus.DefBuckets,
		Registry:       reg,
	}
	c := NewPrometheusCollector(cfg)

	labels := map[string]string{"race": "hist"}
	const numGoroutines = 200
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			c.RecordValue("race_hist", float64(idx), labels)
		}(i)
	}

	close(start)
	wg.Wait()

	metrics, err := reg.Gather()
	require.NoError(t, err)
	found := false
	for _, mf := range metrics {
		if mf.GetName() == "hist_race_cov_race_hist" {
			found = true
			assert.Equal(t,
				uint64(numGoroutines),
				mf.GetMetric()[0].GetHistogram().GetSampleCount(),
			)
		}
	}
	assert.True(t, found, "histogram should be found")
}

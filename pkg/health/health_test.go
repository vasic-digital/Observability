package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_Constants(t *testing.T) {
	assert.Equal(t, Status("healthy"), StatusHealthy)
	assert.Equal(t, Status("degraded"), StatusDegraded)
	assert.Equal(t, Status("unhealthy"), StatusUnhealthy)
}

func TestDefaultAggregatorConfig(t *testing.T) {
	cfg := DefaultAggregatorConfig()
	assert.Equal(t, 5*time.Second, cfg.Timeout)
}

func TestNewAggregator_NilConfig(t *testing.T) {
	agg := NewAggregator(nil)
	assert.NotNil(t, agg)
	assert.Equal(t, 5*time.Second, agg.timeout)
}

func TestNewAggregator_CustomConfig(t *testing.T) {
	agg := NewAggregator(&AggregatorConfig{Timeout: 10 * time.Second})
	assert.Equal(t, 10*time.Second, agg.timeout)
}

func TestAggregator_Register(t *testing.T) {
	agg := NewAggregator(nil)
	assert.Equal(t, 0, agg.ComponentCount())

	agg.Register("db", StaticCheck(nil))
	assert.Equal(t, 1, agg.ComponentCount())

	agg.RegisterOptional("cache", StaticCheck(nil))
	assert.Equal(t, 2, agg.ComponentCount())
}

func TestAggregator_Check_AllHealthy(t *testing.T) {
	agg := NewAggregator(nil)
	agg.Register("db", StaticCheck(nil))
	agg.Register("redis", StaticCheck(nil))
	agg.RegisterOptional("cache", StaticCheck(nil))

	report := agg.Check(context.Background())

	assert.Equal(t, StatusHealthy, report.Status)
	assert.Len(t, report.Components, 3)
	for _, comp := range report.Components {
		assert.Equal(t, StatusHealthy, comp.Status)
	}
	assert.False(t, report.Timestamp.IsZero())
}

func TestAggregator_Check_RequiredUnhealthy(t *testing.T) {
	agg := NewAggregator(nil)
	agg.Register("db", StaticCheck(errors.New("connection refused")))
	agg.Register("redis", StaticCheck(nil))

	report := agg.Check(context.Background())

	assert.Equal(t, StatusUnhealthy, report.Status)

	var dbResult *ComponentResult
	for i := range report.Components {
		if report.Components[i].Name == "db" {
			dbResult = &report.Components[i]
		}
	}
	require.NotNil(t, dbResult)
	assert.Equal(t, StatusUnhealthy, dbResult.Status)
	assert.Equal(t, "connection refused", dbResult.Message)
}

func TestAggregator_Check_OptionalUnhealthy(t *testing.T) {
	agg := NewAggregator(nil)
	agg.Register("db", StaticCheck(nil))
	agg.RegisterOptional("cache", StaticCheck(errors.New("cache down")))

	report := agg.Check(context.Background())

	assert.Equal(t, StatusDegraded, report.Status)
}

func TestAggregator_Check_RequiredAndOptionalUnhealthy(t *testing.T) {
	agg := NewAggregator(nil)
	agg.Register("db", StaticCheck(errors.New("db down")))
	agg.RegisterOptional("cache", StaticCheck(errors.New("cache down")))

	report := agg.Check(context.Background())

	// Required failure takes precedence
	assert.Equal(t, StatusUnhealthy, report.Status)
}

func TestAggregator_Check_Timeout(t *testing.T) {
	agg := NewAggregator(&AggregatorConfig{Timeout: 50 * time.Millisecond})
	agg.Register("slow", func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	report := agg.Check(context.Background())

	assert.Equal(t, StatusUnhealthy, report.Status)
	require.Len(t, report.Components, 1)
	assert.Contains(t, report.Components[0].Message, "timed out")
}

func TestAggregator_Check_EmptyComponents(t *testing.T) {
	agg := NewAggregator(nil)
	report := agg.Check(context.Background())

	assert.Equal(t, StatusHealthy, report.Status)
	assert.Empty(t, report.Components)
}

func TestAggregator_Check_Duration(t *testing.T) {
	agg := NewAggregator(nil)
	agg.Register("fast", func(_ context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	report := agg.Check(context.Background())
	require.Len(t, report.Components, 1)
	assert.Greater(t, report.Components[0].Duration, time.Duration(0))
}

func TestAggregator_Check_LastChecked(t *testing.T) {
	agg := NewAggregator(nil)
	agg.Register("comp", StaticCheck(nil))

	before := time.Now()
	report := agg.Check(context.Background())
	after := time.Now()

	require.Len(t, report.Components, 1)
	assert.True(t,
		report.Components[0].LastChecked.After(before) ||
			report.Components[0].LastChecked.Equal(before),
	)
	assert.True(t,
		report.Components[0].LastChecked.Before(after) ||
			report.Components[0].LastChecked.Equal(after),
	)
}

func TestAggregator_Check_Parallel(t *testing.T) {
	agg := NewAggregator(nil)

	// Add multiple checks that each take 50ms
	for i := 0; i < 5; i++ {
		agg.Register("comp", func(_ context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}

	start := time.Now()
	report := agg.Check(context.Background())
	duration := time.Since(start)

	// Should complete in ~50ms, not ~250ms (parallel execution)
	assert.Equal(t, StatusHealthy, report.Status)
	assert.Less(t, duration, 200*time.Millisecond)
}

func TestAggregator_ConcurrentAccess(t *testing.T) {
	agg := NewAggregator(nil)

	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			agg.Register("comp", StaticCheck(nil))
			agg.Check(context.Background())
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}

func TestStaticCheck(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		expectErr bool
	}{
		{name: "healthy", err: nil, expectErr: false},
		{
			name:      "unhealthy",
			err:       errors.New("down"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := StaticCheck(tt.err)
			err := check(context.Background())
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAggregator_MixedStatuses(t *testing.T) {
	tests := []struct {
		name           string
		required       []error
		optional       []error
		expectedStatus Status
	}{
		{
			name:           "all healthy",
			required:       []error{nil, nil},
			optional:       []error{nil},
			expectedStatus: StatusHealthy,
		},
		{
			name:           "required unhealthy",
			required:       []error{errors.New("fail"), nil},
			optional:       []error{nil},
			expectedStatus: StatusUnhealthy,
		},
		{
			name:           "optional unhealthy",
			required:       []error{nil},
			optional:       []error{errors.New("fail")},
			expectedStatus: StatusDegraded,
		},
		{
			name:           "both unhealthy",
			required:       []error{errors.New("fail")},
			optional:       []error{errors.New("fail")},
			expectedStatus: StatusUnhealthy,
		},
		{
			name:           "no components",
			required:       nil,
			optional:       nil,
			expectedStatus: StatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := NewAggregator(nil)
			for i, err := range tt.required {
				name := "required"
				if i > 0 {
					name = "required_" + string(rune('a'+i))
				}
				agg.Register(name, StaticCheck(err))
			}
			for i, err := range tt.optional {
				name := "optional"
				if i > 0 {
					name = "optional_" + string(rune('a'+i))
				}
				agg.RegisterOptional(name, StaticCheck(err))
			}

			report := agg.Check(context.Background())
			assert.Equal(t, tt.expectedStatus, report.Status)
		})
	}
}

func TestAggregator_ImplementsChecker(t *testing.T) {
	var _ Checker = &Aggregator{}
}

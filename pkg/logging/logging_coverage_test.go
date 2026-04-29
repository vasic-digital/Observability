package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNoOpLogger_Info_Coverage exercises the NoOpLogger.Info method body
// to achieve statement coverage on line 206.
func TestNoOpLogger_Info_Coverage(t *testing.T) {
	n := &NoOpLogger{}
	// Calling Info should not panic and should be a no-op.
	assert.NotPanics(t, func() {
		n.Info("coverage message")
	})
}

// TestNoOpLogger_Warn_Coverage exercises the NoOpLogger.Warn method body
// to achieve statement coverage on line 209.
func TestNoOpLogger_Warn_Coverage(t *testing.T) {
	n := &NoOpLogger{}
	assert.NotPanics(t, func() {
		n.Warn("coverage message")
	})
}

// TestNoOpLogger_Error_Coverage exercises the NoOpLogger.Error method body
// to achieve statement coverage on line 212.
func TestNoOpLogger_Error_Coverage(t *testing.T) {
	n := &NoOpLogger{}
	assert.NotPanics(t, func() {
		n.Error("coverage message")
	})
}

// TestNoOpLogger_Debug_Coverage exercises the NoOpLogger.Debug method body
// to achieve statement coverage on line 215.
func TestNoOpLogger_Debug_Coverage(t *testing.T) {
	n := &NoOpLogger{}
	assert.NotPanics(t, func() {
		n.Debug("coverage message")
	})
}

// TestNoOpLogger_AllMethods_ViaInterface ensures all NoOpLogger methods are
// exercised when accessed through the Logger interface.
func TestNoOpLogger_AllMethods_ViaInterface(t *testing.T) {
	// bluff-scan: no-assert-ok (null-implementation smoke — no-op type must accept all interface calls without panic)
	var l Logger = &NoOpLogger{}

	// Exercise every method through the interface to cover all no-op bodies.
	l.Info("msg")
	l.Warn("msg")
	l.Error("msg")
	l.Debug("msg")

	// Chain methods and call log methods on the returned logger.
	l.WithField("k", "v").Info("chained info")
	l.WithField("k", "v").Warn("chained warn")
	l.WithField("k", "v").Error("chained error")
	l.WithField("k", "v").Debug("chained debug")

	l.WithFields(map[string]interface{}{"a": 1}).Info("fields info")
	l.WithCorrelationID("cid").Info("corr info")
	l.WithError(nil).Info("nil err info")
}

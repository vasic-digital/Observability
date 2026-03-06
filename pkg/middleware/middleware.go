// Package middleware provides generic HTTP metrics middleware that works with
// any net/http compatible server. It uses the MetricsReporter interface to
// decouple metric recording from specific implementations (e.g., Prometheus).
package middleware

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// MetricsReporter defines the interface for recording HTTP metrics.
// Implementations bridge to concrete systems like Prometheus, StatsD, etc.
type MetricsReporter interface {
	// ObserveHTTPDuration records the duration of an HTTP request in seconds.
	ObserveHTTPDuration(method, path, status string, seconds float64)
	// IncrHTTPTotal increments the total HTTP request counter.
	IncrHTTPTotal(method, path, status string)
	// SetActiveConnections sets the current number of in-flight connections.
	SetActiveConnections(count float64)
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code before delegating to the wrapped writer.
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures an implicit 200 status and delegates to the wrapped writer.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// Middleware returns an http.Handler middleware that records HTTP metrics
// using the provided MetricsReporter. It tracks request duration, increments
// request counts, and monitors active (in-flight) connections.
func Middleware(reporter MetricsReporter) func(http.Handler) http.Handler {
	var activeConns int64

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			current := atomic.AddInt64(&activeConns, 1)
			reporter.SetActiveConnections(float64(current))

			start := time.Now()
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start).Seconds()
			status := fmt.Sprintf("%d", rw.statusCode)
			path := r.URL.Path

			reporter.ObserveHTTPDuration(r.Method, path, status, duration)
			reporter.IncrHTTPTotal(r.Method, path, status)

			current = atomic.AddInt64(&activeConns, -1)
			reporter.SetActiveConnections(float64(current))
		})
	}
}

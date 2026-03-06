package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockReporter records all MetricsReporter calls for verification.
type mockReporter struct {
	mu               sync.Mutex
	durations        []durationCall
	totals           []totalCall
	activeConnValues []float64
}

type durationCall struct {
	Method  string
	Path    string
	Status  string
	Seconds float64
}

type totalCall struct {
	Method string
	Path   string
	Status string
}

func (m *mockReporter) ObserveHTTPDuration(
	method, path, status string,
	seconds float64,
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.durations = append(m.durations, durationCall{
		Method:  method,
		Path:    path,
		Status:  status,
		Seconds: seconds,
	})
}

func (m *mockReporter) IncrHTTPTotal(method, path, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totals = append(m.totals, totalCall{
		Method: method,
		Path:   path,
		Status: status,
	})
}

func (m *mockReporter) SetActiveConnections(count float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeConnValues = append(m.activeConnValues, count)
}

func TestMiddleware_CallsReporterMethods(t *testing.T) {
	mock := &mockReporter{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	wrapped := Middleware(mock)(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify duration was observed
	require.Len(t, mock.durations, 1)
	assert.Equal(t, "GET", mock.durations[0].Method)
	assert.Equal(t, "/api/v1/test", mock.durations[0].Path)
	assert.Equal(t, "200", mock.durations[0].Status)
	assert.Greater(t, mock.durations[0].Seconds, 0.0)

	// Verify total was incremented
	require.Len(t, mock.totals, 1)
	assert.Equal(t, "GET", mock.totals[0].Method)
	assert.Equal(t, "/api/v1/test", mock.totals[0].Path)
	assert.Equal(t, "200", mock.totals[0].Status)
}

func TestMiddleware_CapturesStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   string
	}{
		{name: "200 OK", statusCode: http.StatusOK, expected: "200"},
		{name: "201 Created", statusCode: http.StatusCreated, expected: "201"},
		{name: "400 Bad Request", statusCode: http.StatusBadRequest, expected: "400"},
		{name: "404 Not Found", statusCode: http.StatusNotFound, expected: "404"},
		{name: "500 Internal Server Error",
			statusCode: http.StatusInternalServerError, expected: "500"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockReporter{}
			handler := http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.statusCode)
				},
			)

			wrapped := Middleware(mock)(handler)
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			require.Len(t, mock.durations, 1)
			assert.Equal(t, tc.expected, mock.durations[0].Status)
			require.Len(t, mock.totals, 1)
			assert.Equal(t, tc.expected, mock.totals[0].Status)
		})
	}
}

func TestMiddleware_DefaultsTo200WhenNoWriteHeader(t *testing.T) {
	mock := &mockReporter{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write body without explicit WriteHeader call
		_, _ = w.Write([]byte("implicit 200"))
	})

	wrapped := Middleware(mock)(handler)
	req := httptest.NewRequest(http.MethodGet, "/implicit", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	require.Len(t, mock.durations, 1)
	assert.Equal(t, "200", mock.durations[0].Status)
}

func TestMiddleware_DefaultsTo200WhenNoWrite(t *testing.T) {
	mock := &mockReporter{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do nothing — handler returns without writing
	})

	wrapped := Middleware(mock)(handler)
	req := httptest.NewRequest(http.MethodGet, "/empty", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	require.Len(t, mock.durations, 1)
	assert.Equal(t, "200", mock.durations[0].Status)
}

func TestMiddleware_HandlerChain(t *testing.T) {
	mock := &mockReporter{}
	var handlerCalled bool

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusAccepted)
	})

	wrapped := Middleware(mock)(handler)
	req := httptest.NewRequest(http.MethodPut, "/resource", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.True(t, handlerCalled, "inner handler must be called")
	assert.Equal(t, http.StatusAccepted, rec.Code)

	require.Len(t, mock.durations, 1)
	assert.Equal(t, "PUT", mock.durations[0].Method)
	assert.Equal(t, "/resource", mock.durations[0].Path)
	assert.Equal(t, "202", mock.durations[0].Status)
}

func TestMiddleware_ActiveConnectionTracking(t *testing.T) {
	mock := &mockReporter{}
	connInHandler := make(chan float64, 1)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the active connections value that was set before handler ran.
		mock.mu.Lock()
		if len(mock.activeConnValues) > 0 {
			connInHandler <- mock.activeConnValues[len(mock.activeConnValues)-1]
		}
		mock.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Middleware(mock)(handler)
	req := httptest.NewRequest(http.MethodGet, "/conn", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Active connections should have been 1 during handler execution
	activeInHandler := <-connInHandler
	assert.Equal(t, 1.0, activeInHandler)

	// After request completes, active connections should be back to 0
	mock.mu.Lock()
	lastValue := mock.activeConnValues[len(mock.activeConnValues)-1]
	mock.mu.Unlock()
	assert.Equal(t, 0.0, lastValue)
}

func TestMiddleware_ConcurrentRequests(t *testing.T) {
	mock := &mockReporter{}
	var maxSeen int64
	var maxMu sync.Mutex
	ready := make(chan struct{})
	proceed := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ready <- struct{}{}
		<-proceed
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Middleware(mock)(handler)

	const numRequests = 3
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/concurrent", nil)
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
		}()
	}

	// Wait for all goroutines to be inside the handler
	for i := 0; i < numRequests; i++ {
		<-ready
	}

	// Check max active connections
	mock.mu.Lock()
	for _, v := range mock.activeConnValues {
		if int64(v) > maxSeen {
			maxSeen = int64(v)
		}
	}
	mock.mu.Unlock()

	maxMu.Lock()
	assert.GreaterOrEqual(t, maxSeen, int64(numRequests),
		"active connections should reach at least %d concurrently", numRequests)
	maxMu.Unlock()

	// Release all handlers
	close(proceed)
	wg.Wait()

	// After all requests, active connections should be 0
	mock.mu.Lock()
	lastValue := mock.activeConnValues[len(mock.activeConnValues)-1]
	mock.mu.Unlock()
	assert.Equal(t, 0.0, lastValue)

	// Should have recorded metrics for all requests
	mock.mu.Lock()
	assert.Len(t, mock.durations, numRequests)
	assert.Len(t, mock.totals, numRequests)
	mock.mu.Unlock()
}

func TestMiddleware_RecordsHTTPMethod(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			mock := &mockReporter{}
			handler := http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			)

			wrapped := Middleware(mock)(handler)
			req := httptest.NewRequest(method, "/methods", nil)
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			require.Len(t, mock.durations, 1)
			assert.Equal(t, method, mock.durations[0].Method)
			require.Len(t, mock.totals, 1)
			assert.Equal(t, method, mock.totals[0].Method)
		})
	}
}

func TestMiddleware_ResponseWriterDelegation(t *testing.T) {
	mock := &mockReporter{}
	body := "hello world"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "test-value")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})

	wrapped := Middleware(mock)(handler)
	req := httptest.NewRequest(http.MethodGet, "/delegate", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, body, rec.Body.String())
	assert.Equal(t, "test-value", rec.Header().Get("X-Custom"))
}

func TestResponseWriter_WriteHeaderOnlyOnce(t *testing.T) {
	mock := &mockReporter{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		// Second call should not change the captured status
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Middleware(mock)(handler)
	req := httptest.NewRequest(http.MethodGet, "/double-write", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	require.Len(t, mock.durations, 1)
	assert.Equal(t, "404", mock.durations[0].Status,
		"first WriteHeader call should be captured")
}

# Lesson 4: Gin Integration and Middleware Adapters

## Learning Objectives

- Adapt observability components for use with the Gin web framework
- Compose tracing, metrics, and logging middleware into a unified request pipeline
- Understand the Gin handler function adapter pattern

## Key Concepts

- **Gin Adapter**: The `gin` package converts standard observability middleware into `gin.HandlerFunc` for seamless integration with Gin routers.
- **Request Pipeline**: Tracing middleware creates a span per request, metrics middleware records latency and status codes, and logging middleware logs request details with correlation IDs.
- **Middleware Composition**: Multiple observability middleware functions are composed in order, with each adding its layer of instrumentation before passing to the next handler.

## Code Walkthrough

### Source: `pkg/gin/gin.go`

The Gin adapter wraps observability middleware for Gin's handler signature. It bridges between the standard `net/http` middleware pattern and Gin's `HandlerFunc` interface.

### Source: `pkg/middleware/middleware.go`

The middleware package provides HTTP middleware that combines tracing, metrics, and logging into a single middleware function. It:

1. Extracts or generates a correlation ID
2. Starts a trace span for the request
3. Wraps the response writer to capture the status code
4. Records request duration and status in Prometheus metrics
5. Logs the request with all collected metadata

### Source: `pkg/gin/gin_test.go` and `pkg/middleware/middleware_test.go`

Tests verify that the middleware correctly records metrics, creates spans, and propagates correlation IDs through the Gin context.

## Practice Exercise

1. Set up a Gin router with the observability middleware. Make 10 requests and verify that Prometheus metrics show the correct request count and latency distribution.
2. Add trace context propagation: send a request with a `traceparent` header and verify the middleware creates a child span linked to the incoming trace.
3. Create a custom Gin middleware that adds application-specific labels (e.g., user role, API version) to the observability context. Verify these labels appear in logs and trace attributes.

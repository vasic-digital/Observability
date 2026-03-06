# Course: Observability in Go Services

## Module Overview

This course covers the `digital.vasic.observability` module, which provides distributed tracing, Prometheus metrics, structured logging, health check aggregation, analytics collection, and framework adapters. Each package is independent with no inter-package dependencies, and every package provides a no-op implementation for zero-infrastructure usage.

## Prerequisites

- Intermediate Go knowledge
- Familiarity with observability concepts (traces, metrics, logs)
- Basic understanding of OpenTelemetry and Prometheus
- Go 1.24+ installed

## Lessons

| # | Title | Duration |
|---|-------|----------|
| 1 | Distributed Tracing with OpenTelemetry | 50 min |
| 2 | Prometheus Metrics and Structured Logging | 45 min |
| 3 | Health Check Aggregation and Analytics | 45 min |
| 4 | Gin Integration and Middleware Adapters | 35 min |

## Source Files

- `pkg/trace/` -- OpenTelemetry distributed tracing with exporter selection
- `pkg/metrics/` -- Prometheus metrics collection with double-checked locking
- `pkg/logging/` -- Structured logging with logrus adapter and correlation IDs
- `pkg/health/` -- Health check aggregation with required/optional components
- `pkg/analytics/` -- ClickHouse analytics with NoOp fallback
- `pkg/gin/` -- Gin framework integration adapter
- `pkg/middleware/` -- HTTP middleware for observability

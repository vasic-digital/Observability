# Lesson 3: Health Check Aggregation and Analytics

## Learning Objectives

- Build a health check aggregator that runs checks in parallel with per-check timeouts
- Implement graceful degradation for analytics collection using the Null Object pattern
- Understand the distinction between required and optional health components

## Key Concepts

- **Aggregator Pattern**: The `Aggregator` collects multiple `CheckFunc` registrations and executes them in parallel during `Check()`. Each check runs in its own goroutine with `context.WithTimeout`.
- **Status Aggregation**: If any required component is unhealthy, overall status is `unhealthy`. If any optional component is unhealthy (no required failures), status is `degraded`. Otherwise `healthy`.
- **Graceful Degradation (Analytics)**: `NewCollector` attempts to connect to ClickHouse. On failure, it logs a warning and returns a `NoOpCollector`. Analytics should never block application startup.
- **SQL Safety**: The analytics `Query` method validates identifiers with `isValidIdentifier` (alphanumeric and underscores only). `ExecuteReadQuery` enforces read-only by accepting only SELECT statements.

## Code Walkthrough

### Source: `pkg/health/health.go`

The Aggregator uses `sync.RWMutex` for the components slice:

- `Register(name, check)` -- adds a required component
- `RegisterOptional(name, check)` -- adds an optional component
- `Check(ctx)` -- copies component slice under RLock, launches goroutines, waits with `sync.WaitGroup`

Each `CheckFunc` returns `nil` for healthy, an error for unhealthy. The result is a `Report` with per-component results and an overall status.

### Source: `pkg/analytics/analytics.go`

The ClickHouse collector provides:
- `Track(ctx, event)` -- inserts a single event
- `TrackBatch(ctx, events)` -- batch insert using transactions
- `Query(ctx, table, groupBy, filters)` -- validated aggregation queries
- `ExecuteReadQuery(ctx, sql, args)` -- arbitrary read-only queries

The factory function `NewCollector` returns `NoOpCollector` on connection failure, ensuring the application always has a usable collector.

### Source: `pkg/health/health_test.go` and `pkg/analytics/analytics_test.go`

Tests verify parallel health check execution, timeout handling, status aggregation logic, and NoOp behavior.

## Practice Exercise

1. Create an `Aggregator` with three health checks: a fast passing check, a slow passing check (300ms), and a failing check. Set the overall timeout to 500ms. Verify the report shows the correct per-component statuses.
2. Register a required check (database) and an optional check (cache). Test the three scenarios: both healthy, required unhealthy, optional unhealthy. Verify the overall status for each.
3. Write a mock `Collector` that records all tracked events in memory. Use it to verify that `TrackBatch` correctly inserts all events and that `Query` returns expected aggregations.

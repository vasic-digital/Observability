# CLAUDE.md - Observability Module


## Definition of Done

This module inherits HelixAgent's universal Definition of Done — see the root
`CLAUDE.md` and `docs/development/definition-of-done.md`. In one line: **no
task is done without pasted output from a real run of the real system in the
same session as the change.** Coverage and green suites are not evidence.

### Acceptance demo for this module

```bash
# Tracing, metrics, logging, and health-check aggregation (NoOp backends by default)
cd Observability && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v ./tests/integration/...
```
Expect: all integration tests PASS; `trace.InitTracer`, `metrics.NewPrometheusCollector`, `health.NewAggregator` all documented in `Observability/README.md` Quick Start. For a real OTLP backend set the exporter env vars per the README and repoint the trace init.


## Overview

`digital.vasic.observability` is a generic, reusable Go module for application observability. It provides distributed tracing (OpenTelemetry), Prometheus metrics collection, structured logging with correlation IDs, health check aggregation, and ClickHouse-backed analytics.

**Module**: `digital.vasic.observability` (Go 1.24+)

## Build & Test

```bash
go build ./...
go test ./... -count=1 -race
go test ./... -short              # Unit tests only
go test -tags=integration ./...   # Integration tests
go test -bench=. ./tests/benchmark/
```

## Code Style

- Standard Go conventions, `gofmt` formatting
- Imports grouped: stdlib, third-party, internal (blank line separated)
- Line length <= 100 chars
- Naming: `camelCase` private, `PascalCase` exported, acronyms all-caps
- Errors: always check, wrap with `fmt.Errorf("...: %w", err)`
- Tests: table-driven, `testify`, naming `Test<Struct>_<Method>_<Scenario>`

## Package Structure

| Package | Purpose |
|---------|---------|
| `pkg/trace` | OpenTelemetry tracing with OTLP/Jaeger/Zipkin/stdout exporters |
| `pkg/metrics` | Prometheus metrics collection (counters, histograms, gauges) |
| `pkg/logging` | Structured logging with correlation ID support (logrus) |
| `pkg/health` | Health check aggregation with required/optional components |
| `pkg/analytics` | ClickHouse analytics adapter with NoOp fallback |

## Key Interfaces

- `metrics.Collector` -- IncrementCounter, AddCounter, RecordLatency, RecordValue, SetGauge
- `logging.Logger` -- Info, Warn, Error, Debug, WithField, WithFields, WithCorrelationID, WithError
- `health.Checker` -- Check(ctx) returning aggregated Report
- `analytics.Collector` -- Track, TrackBatch, Query, Close

## Design Patterns

- **Strategy**: Pluggable exporters (OTLP, Jaeger, Zipkin, Stdout, None)
- **Adapter**: LogrusAdapter for Logger interface, PrometheusCollector for Collector
- **Null Object**: NoOpCollector, NoOpLogger for disabled subsystems
- **Aggregator**: Health check combining required/optional component results
- **Graceful Degradation**: Analytics falls back to NoOp when ClickHouse unavailable

## Commit Style

Conventional Commits: `feat(trace): add OTLP exporter support`


---

## ⚠️ MANDATORY: NO SUDO OR ROOT EXECUTION

**ALL operations MUST run at local user level ONLY.**

This is a PERMANENT and NON-NEGOTIABLE security constraint:

- **NEVER** use `sudo` in ANY command
- **NEVER** use `su` in ANY command
- **NEVER** execute operations as `root` user
- **NEVER** elevate privileges for file operations
- **ALL** infrastructure commands MUST use user-level container runtimes (rootless podman/docker)
- **ALL** file operations MUST be within user-accessible directories
- **ALL** service management MUST be done via user systemd or local process management
- **ALL** builds, tests, and deployments MUST run as the current user

### Container-Based Solutions
When a build or runtime environment requires system-level dependencies, use containers instead of elevation:

- **Use the `Containers` submodule** (`https://github.com/vasic-digital/Containers`) for containerized build and runtime environments
- **Add the `Containers` submodule as a Git dependency** and configure it for local use within the project
- **Build and run inside containers** to avoid any need for privilege escalation
- **Rootless Podman/Docker** is the preferred container runtime

### Why This Matters
- **Security**: Prevents accidental system-wide damage
- **Reproducibility**: User-level operations are portable across systems
- **Safety**: Limits blast radius of any issues
- **Best Practice**: Modern container workflows are rootless by design

### When You See SUDO
If any script or command suggests using `sudo` or `su`:
1. STOP immediately
2. Find a user-level alternative
3. Use rootless container runtimes
4. Use the `Containers` submodule for containerized builds
5. Modify commands to work within user permissions

**VIOLATION OF THIS CONSTRAINT IS STRICTLY PROHIBITED.**

## Integration Seams

| Direction | Sibling modules |
|-----------|-----------------|
| Upstream (this module imports) | none |
| Downstream (these import this module) | HelixLLM |

*Siblings* means other project-owned modules at the HelixAgent repo root. The root HelixAgent app and external systems are not listed here — the list above is intentionally scoped to module-to-module seams, because drift *between* sibling modules is where the "tests pass, product broken" class of bug most often lives. See root `CLAUDE.md` for the rules that keep these seams contract-tested.

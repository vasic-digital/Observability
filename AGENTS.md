# AGENTS.md - Multi-Agent Coordination Guide

## Overview

This document provides guidance for AI agents working on the `digital.vasic.observability` module. The module is a generic, reusable Go library for application observability with five packages: `trace`, `metrics`, `logging`, `health`, and `analytics`.

## Module Boundaries

- **Module path**: `digital.vasic.observability`
- **Go version**: 1.24+
- **Package root**: `pkg/`
- **No main package** -- this is a library module, not an executable.

## Package Responsibilities

| Package | Owner Concern | Dependencies |
|---------|--------------|--------------|
| `pkg/trace` | Distributed tracing via OpenTelemetry | `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/sdk` |
| `pkg/metrics` | Prometheus metric collection | `github.com/prometheus/client_golang` |
| `pkg/logging` | Structured logging with correlation IDs | `github.com/sirupsen/logrus` |
| `pkg/health` | Health check aggregation | stdlib only (`context`, `sync`, `time`, `fmt`) |
| `pkg/analytics` | ClickHouse event tracking | `github.com/ClickHouse/clickhouse-go/v2`, `github.com/sirupsen/logrus` |

## Inter-Package Rules

1. **No circular imports.** Packages must NOT import each other. Each package is fully independent.
2. **Interface-driven.** Each package defines its own interface (`Collector`, `Logger`, `Checker`, `Collector`). Consumers depend on interfaces, not concrete types.
3. **No shared state.** Packages do not share global variables. The `trace` package sets the global OTel provider but other packages do not read it.

## Agent Coordination Patterns

### When Adding a New Exporter to `trace`

1. Add the new `ExporterType` constant.
2. Add a case in `InitTracer` switch.
3. Add the setup function (follow `setupOTLPExporter` pattern).
4. Add unit tests with table-driven cases.
5. Update `CLAUDE.md` package table and `README.md` feature list.

### When Adding a New Metric Type to `metrics`

1. Add the method to the `Collector` interface.
2. Implement it on `PrometheusCollector`.
3. Add a no-op implementation on `NoOpCollector`.
4. Add `RegisterX` and `getOrCreateX` methods following the existing pattern.
5. Add table-driven tests.

### When Adding a New Log Level or Method to `logging`

1. Add the method to the `Logger` interface.
2. Implement on `LogrusAdapter`.
3. Implement on `NoOpLogger`.
4. Add tests.

### When Adding a New Health Check Pattern to `health`

1. Add the `CheckFunc` factory (follow `StaticCheck`, `TCPCheck` patterns).
2. If adding new status types, add to the `Status` constants.
3. Update `buildReport` aggregation logic if status priority changes.
4. Add tests.

### When Modifying `analytics`

1. All query construction must validate identifiers via `isValidIdentifier` to prevent SQL injection.
2. The `NewCollector` factory must always fall back to `NoOpCollector` on connection failure.
3. Batch operations must use transactions.
4. Add tests (unit tests can use `NoOpCollector`; integration tests require ClickHouse).

## Testing Conventions

- **File naming**: `<package>_test.go` in the same directory as the source.
- **Test naming**: `Test<Struct>_<Method>_<Scenario>` (e.g., `TestAggregator_Check_AllHealthy`).
- **Table-driven tests** with `testify/assert` and `testify/require`.
- **Unit tests**: Run with `go test ./... -short`. Must not require external services.
- **Integration tests**: Use `//go:build integration` tag. Require running ClickHouse, OTLP collector, etc.
- **Benchmarks**: Use `Benchmark<Function>` naming. Place in `tests/benchmark/`.

## Commit and PR Conventions

- Conventional Commits: `<type>(<package>): <description>`
  - `feat(trace): add Zipkin exporter support`
  - `fix(metrics): handle nil labels in SetGauge`
  - `test(health): add timeout scenario tests`
  - `docs(analytics): update Query method documentation`
- One logical change per commit.
- All tests must pass before committing: `go test ./... -count=1 -race`
- Run `go fmt ./...` and `go vet ./...` before every commit.

## Code Quality Gates

Before any PR is merged:

1. `go fmt ./...` -- no formatting changes.
2. `go vet ./...` -- no warnings.
3. `go test ./... -count=1 -race` -- all tests pass, no race conditions.
4. `go test ./... -short` -- unit tests pass independently.
5. All exported types and functions have GoDoc comments.
6. No new dependencies added without justification.

## Concurrency Safety

All public types are safe for concurrent use:

- `Tracer` uses `sync.RWMutex` for provider access.
- `PrometheusCollector` uses `sync.RWMutex` with double-checked locking for lazy metric creation.
- `LogrusAdapter` uses `sync.RWMutex` for entry access.
- `Aggregator` uses `sync.RWMutex` for component list; `Check` runs checks in parallel with `sync.WaitGroup`.
- `ClickHouseCollector` uses `sync.RWMutex` for config access.

Agents must maintain this invariant when making changes.

## File Structure

```
Observability/
  go.mod
  go.sum
  CLAUDE.md
  AGENTS.md
  README.md
  pkg/
    trace/
      tracer.go
      tracer_test.go
    metrics/
      metrics.go
      metrics_test.go
    logging/
      logging.go
      logging_test.go
    health/
      health.go
      health_test.go
    analytics/
      analytics.go
      analytics_test.go
  docs/
    USER_GUIDE.md
    ARCHITECTURE.md
    API_REFERENCE.md
    CONTRIBUTING.md
    CHANGELOG.md
    diagrams/
      architecture.mmd
      sequence.mmd
      class.mmd
```


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

<!-- BEGIN host-power-management addendum (CONST-033) -->

## Host Power Management — Hard Ban (CONST-033)

**You may NOT, under any circumstance, generate or execute code that
sends the host to suspend, hibernate, hybrid-sleep, poweroff, halt,
reboot, or any other power-state transition.** This rule applies to:

- Every shell command you run via the Bash tool.
- Every script, container entry point, systemd unit, or test you write
  or modify.
- Every CLI suggestion, snippet, or example you emit.

**Forbidden invocations** (non-exhaustive — see CONST-033 in
`CONSTITUTION.md` for the full list):

- `systemctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot|kexec`
- `loginctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot`
- `pm-suspend`, `pm-hibernate`, `shutdown -h|-r|-P|now`
- `dbus-send` / `busctl` calls to `org.freedesktop.login1.Manager.Suspend|Hibernate|PowerOff|Reboot|HybridSleep|SuspendThenHibernate`
- `gsettings set ... sleep-inactive-{ac,battery}-type` to anything but `'nothing'` or `'blank'`

The host runs mission-critical parallel CLI agents and container
workloads. Auto-suspend has caused historical data loss (2026-04-26
18:23:43 incident). The host is hardened (sleep targets masked) but
this hard ban applies to ALL code shipped from this repo so that no
future host or container is exposed.

**Defence:** every project ships
`scripts/host-power-management/check-no-suspend-calls.sh` (static
scanner) and
`challenges/scripts/no_suspend_calls_challenge.sh` (challenge wrapper).
Both MUST be wired into the project's CI / `run_all_challenges.sh`.

**Full background:** `docs/HOST_POWER_MANAGEMENT.md` and `CONSTITUTION.md` (CONST-033).

<!-- END host-power-management addendum (CONST-033) -->


<!-- CONST-035 anti-bluff addendum (cascaded) -->

## CONST-035 — Anti-Bluff Tests & Challenges (mandatory; inherits from root)

Tests and Challenges in this submodule MUST verify the product, not
the LLM's mental model of the product. A test that passes when the
feature is broken is worse than a missing test — it gives false
confidence and lets defects ship to users. Functional probes at the
protocol layer are mandatory:

- TCP-open is the FLOOR, not the ceiling. Postgres → execute
  `SELECT 1`. Redis → `PING` returns `PONG`. ChromaDB → `GET
  /api/v1/heartbeat` returns 200. MCP server → TCP connect + valid
  JSON-RPC handshake. HTTP gateway → real request, real response,
  non-empty body.
- Container `Up` is NOT application healthy. A `docker/podman ps`
  `Up` status only means PID 1 is running; the application may be
  crash-looping internally.
- No mocks/fakes outside unit tests (already CONST-030; CONST-035
  raises the cost of a mock-driven false pass to the same severity
  as a regression).
- Re-verify after every change. Don't assume a previously-passing
  test still verifies the same scope after a refactor.
- Verification of CONST-035 itself: deliberately break the feature
  (e.g. `kill <service>`, swap a password). The test MUST fail. If
  it still passes, the test is non-conformant and MUST be tightened.

## CONST-033 clarification — distinguishing host events from sluggishness

Heavy container builds (BuildKit pulling many GB of layers, parallel
podman/docker compose-up across many services) can make the host
**appear** unresponsive — high load average, slow SSH, watchers
timing out. **This is NOT a CONST-033 violation.** Suspend / hibernate
/ logout are categorically different events. Distinguish via:

- `uptime` — recent boot? if so, the host actually rebooted.
- `loginctl list-sessions` — session(s) still active? if yes, no logout.
- `journalctl ... | grep -i 'will suspend\|hibernate'` — zero broadcasts
  since the CONST-033 fix means no suspend ever happened.
- `dmesg | grep -i 'killed process\|out of memory'` — OOM kills are
  also NOT host-power events; they're memory-pressure-induced and
  require their own separate fix (lower per-container memory limits,
  reduce parallelism).

A sluggish host under build pressure recovers when the build finishes;
a suspended host requires explicit unsuspend (and CONST-033 should
make that impossible by hardening `IdleAction=ignore` +
`HandleSuspendKey=ignore` + masked `sleep.target`,
`suspend.target`, `hibernate.target`, `hybrid-sleep.target`).

If you observe what looks like a suspend during heavy builds, the
correct first action is **not** "edit CONST-033" but `bash
challenges/scripts/host_no_auto_suspend_challenge.sh` to confirm the
hardening is intact. If hardening is intact AND no suspend
broadcast appears in journal, the perceived event was build-pressure
sluggishness, not a power transition.

<!-- BEGIN no-session-termination addendum (CONST-036) -->

## User-Session Termination — Hard Ban (CONST-036)

**You may NOT, under any circumstance, generate or execute code that
ends the currently-logged-in user's desktop session, kills their
`user@<UID>.service` user manager, or indirectly forces them to
manually log out / power off.** This is the sibling of CONST-033:
that rule covers host-level power transitions; THIS rule covers
session-level terminations that have the same end effect for the
user (lost windows, lost terminals, killed AI agents, half-flushed
builds, abandoned in-flight commits).

**Why this rule exists.** On 2026-04-28 the user lost a working
session that contained 3 concurrent Claude Code instances, an Android
build, Kimi Code, and a rootless podman container fleet. The
`user.slice` consumed 60.6 GiB peak / 5.2 GiB swap, the GUI became
unresponsive, the user was forced to log out and then power off via
the GNOME shell. The host could not auto-suspend (CONST-033 was in
place and verified) and the kernel OOM killer never fired — but the
user had to manually end the session anyway, because nothing
prevented overlapping heavy workloads from saturating the slice.
CONST-036 closes that loophole at both the source-code layer and the
operational layer. See
`docs/issues/fixed/SESSION_LOSS_2026-04-28.md` in the HelixAgent
project.

**Forbidden direct invocations** (non-exhaustive):

- `loginctl terminate-user|terminate-session|kill-user|kill-session`
- `systemctl stop user@<UID>` / `systemctl kill user@<UID>`
- `gnome-session-quit`
- `pkill -KILL -u $USER` / `killall -u $USER`
- `dbus-send` / `busctl` calls to `org.gnome.SessionManager.Logout|Shutdown|Reboot`
- `echo X > /sys/power/state`
- `/usr/bin/poweroff`, `/usr/bin/reboot`, `/usr/bin/halt`

**Indirect-pressure clauses:**

1. Do not spawn parallel heavy workloads casually; check `free -h`
   first; keep `user.slice` under 70% of physical RAM.
2. Long-lived background subagents go in `system.slice`. Rootless
   podman containers die with the user manager.
3. Document AI-agent concurrency caps in CLAUDE.md.
4. Never script "log out and back in" recovery flows.

**Defence:** every project ships
`scripts/host-power-management/check-no-session-termination-calls.sh`
(static scanner) and
`challenges/scripts/no_session_termination_calls_challenge.sh`
(challenge wrapper). Both MUST be wired into the project's CI /
`run_all_challenges.sh`.

<!-- END no-session-termination addendum (CONST-036) -->

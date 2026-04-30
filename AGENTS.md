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



## Sixth Law — Real User Verification (Anti-Pseudo-Test Rule)

> Inherits from the root project's Anti-Bluff Testing Pact and the cross-project
> universal mandate (CONST-035). Submodule rules below are additive, never
> relaxing.

A test that passes while the feature it covers is broken for end users is the
most expensive kind of test in this codebase — it converts unknown breakage into
believed safety. This has happened in consuming projects before: tests and
Integration Challenge Tests executed green while large parts of the product
were unusable on a real device. That outcome is a constitutional failure, not a
coverage failure, and it MUST NOT recur in any module that depends on or is
depended on by this one.

Every test added MUST satisfy ALL of the following. Violation of any of them is
a release blocker, irrespective of coverage metrics, CI status, reviewer
sign-off, or schedule pressure.

1. **Same surfaces the user touches.** The test must traverse the production
   code path the user's action triggers, end to end, with no shortcut that
   bypasses real wiring.

2. **Provably falsifiable on real defects.** Before merging, the author MUST
   run the test once with the underlying feature deliberately broken (throw
   inside the function, return the wrong row, return the wrong status) and
   confirm the test fails with a clear assertion message. The PR description
   MUST state which deliberate break was used and what failure the test
   produced. A test that cannot be made to fail by breaking the thing it claims
   to verify is a bluff test by definition.

3. **Primary assertion on user-visible state.** The chief failure signal MUST
   be on something a real consumer could see or measure: rendered output,
   persisted database row, HTTP response body / status / header, file written
   to disk, packet on the wire. "Mock was invoked N times" is a permitted
   secondary assertion, never the primary one.

4. **Integration / Challenge tests are the load-bearing acceptance gate.** A
   green Challenge Test means a real consumer can complete the flow against
   real services — not "the wiring compiles". A feature for which a Challenge
   Test cannot be written is, by definition, not shippable.

5. **CI green is necessary, not sufficient.** Before any release tag is cut, a
   human (or a scripted black-box runner) MUST have exercised the feature
   end-to-end and observed the user-visible outcome.

6. **Inheritance.** This rule applies recursively to every consumer of this
   submodule. Consumer constitutions MAY add stricter rules but MUST NOT relax
   this one.

<!-- BEGIN anti-bluff-testing addendum (Article XI) -->

## Article XI — Anti-Bluff Testing (MANDATORY)

**Inherited from the umbrella project's Constitution Article XI.
Tests and Challenges that pass without exercising real end-user
behaviour are forbidden in this submodule too.**

Every test, every Challenge, every HelixQA bank entry MUST:

1. **Assert on a concrete end-user-visible outcome** — rendered DOM,
   DB rows that a real query would return, files on disk, media that
   actually plays, search results that actually contain expected
   items. Not "no error" or "200 OK".
2. **Run against the real system below the assertion.** Mocks/stubs
   are permitted ONLY in unit tests (`*_test.go` under `go test
   -short` or language equivalent). Integration / E2E / Challenge /
   HelixQA tests use real containers, real databases, real
   renderers. Unreachable real-system → skip with `SKIP-OK:
   #<ticket>`, never silently pass.
3. **Include a matching negative.** Every positive assertion is
   paired with an assertion that fails when the feature is broken.
4. **Emit copy-pasteable evidence** — body, screenshot, frame, DB
   row, log excerpt. Boolean pass/fail is insufficient.
5. **Verify "fails when feature is removed."** Author runs locally
   with the feature commented out; the test MUST FAIL. If it still
   passes, it's a bluff — delete and rewrite.
6. **No blind shells.** No `&& echo PASS`, `|| true`, `tee` exit
   laundering, `if [ -f file ]` without content assertion.

**Challenges in this submodule** must replay the user journey
end-to-end through the umbrella project's deliverables — never via
raw `curl` or third-party scripts. Sub-1-second Challenges almost
always indicate a bluff.

**HelixQA banks** declare executable actions
(`adb_shell:`, `playwright:`, `http:`, `assertVisible:`,
`assertNotVisible:`), never prose. Stagnation guard from Article I
§1.3 applies — frame N+1 identical to frame N for >10 s = FAIL.

**PR requirement:** every PR adding/modifying a test or Challenge in
this submodule MUST include a fenced `## Anti-Bluff Verification`
block with: (a) command run, (b) pasted output, (c) proof the test
fails when the feature is broken (second run with feature
commented-out showing FAIL).

**Cross-reference:** umbrella `CONSTITUTION.md` Article XI
(§§ 11.1 — 11.8).

<!-- END anti-bluff-testing addendum (Article XI) -->

<!-- BEGIN const035-strengthening-2026-04-29 -->

## CONST-035 — End-User Usability Mandate (2026-04-29 strengthening)

A test or Challenge that PASSES is a CLAIM that the tested behavior
**works for the end user of the product**. The HelixAgent project
has repeatedly hit the failure mode where every test ran green AND
every Challenge reported PASS, yet most product features did not
actually work — buggy challenge wrappers masked failed assertions,
scripts checked file existence without executing the file,
"reachability" tests tolerated timeouts, contracts were honest in
advertising but broken in dispatch. **This MUST NOT recur.**

Every PASS result MUST guarantee:

a. **Quality** — the feature behaves correctly under inputs an end
   user will send, including malformed input, edge cases, and
   concurrency that real workloads produce.
b. **Completion** — the feature is wired end-to-end from public
   API surface down to backing infrastructure, with no stub /
   placeholder / "wired lazily later" gaps that silently 503.
c. **Full usability** — a CLI agent / SDK consumer / direct curl
   client following the documented model IDs, request shapes, and
   endpoints SUCCEEDS without having to know which of N internal
   aliases the dispatcher actually accepts.

A passing test that doesn't certify all three is a **bluff** and
MUST be tightened, or marked `t.Skip("...SKIP-OK: #<ticket>")`
so absence of coverage is loud rather than silent.

### Bluff taxonomy (each pattern observed in HelixAgent and now forbidden)

- **Wrapper bluff** — assertions PASS but the wrapper's exit-code
  logic is buggy, marking the run FAILED (or the inverse: assertions
  FAIL but the wrapper swallows them). Every aggregating wrapper MUST
  use a robust counter (`! grep -qs "|FAILED|" "$LOG"` style) —
  never inline arithmetic on a command that prints AND exits
  non-zero.
- **Contract bluff** — the system advertises a capability but
  rejects it in dispatch. Every advertised capability MUST be
  exercised by a test or Challenge that actually invokes it.
- **Structural bluff** — `check_file_exists "foo_test.go"` passes
  if the file is present but doesn't run the test or assert anything
  about its content. File-existence checks MUST be paired with at
  least one functional assertion.
- **Comment bluff** — a code comment promises a behavior the code
  doesn't actually have. Documentation written before / about code
  MUST be re-verified against the code on every change touching the
  documented function.
- **Skip bluff** — `t.Skip("not running yet")` without a
  `SKIP-OK: #<ticket>` marker silently passes. Every skip needs the
  marker; CI fails on bare skips.

The taxonomy is illustrative, not exhaustive. Every Challenge or
test added going forward MUST pass an honest self-review against
this taxonomy before being committed.

<!-- END const035-strengthening-2026-04-29 -->

## ⚠️ Anti-Bluff Covenant — End-User Quality Guarantee (User mandate, 2026-04-28)

**Forensic anchor — direct user mandate (verbatim):**

> "We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used! This MUST NOT be the case and execution of tests and Challenges MUST guarantee the quality, the completion and full usability by end users of the product!"

**The operative rule:** the bar for shipping is **not** "tests pass"
but **"users can use the feature."**

Every PASS in this codebase MUST carry positive evidence captured
during execution that the feature works for the end user.
Metadata-only PASS, configuration-only PASS, "absence-of-error"
PASS, and grep-based PASS without runtime evidence are all
**critical defects** regardless of how green the summary line
looks.

**Tests and Challenges (HelixQA) are bound equally.** A Challenge
that scores PASS on a non-functional feature is the same class of
defect as a unit test that does. Both must produce positive
end-user evidence; both are subject to the parent
[`docs/guides/ATMOSPHERE_CONSTITUTION.md`](../docs/guides/ATMOSPHERE_CONSTITUTION.md)
§8.1 (positive-evidence-only validation) and §11 (anti-bluff)
quality bar.

**No false-success results are tolerable.** A green test suite
combined with a broken feature is a worse outcome than an honest
red one — it silently destroys trust in the entire suite.

**Cascade requirement:** this anchor (verbatim quote + operative
rule) MUST appear in every submodule's `CONSTITUTION.md`,
`CLAUDE.md`, and `AGENTS.md`. Non-compliance is a release blocker.

**Full text:** parent project's `CONSTITUTION.md` Article XI §11.9.

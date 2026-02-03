# Contributing

Thank you for your interest in contributing to `digital.vasic.observability`.

## Prerequisites

- Go 1.24 or later
- `gofmt` and `goimports` installed
- `golangci-lint` (optional, for linting)
- Docker or Podman (for integration tests requiring ClickHouse)

## Getting Started

1. Clone the repository.
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Verify the build:
   ```bash
   go build ./...
   ```
4. Run tests:
   ```bash
   go test ./... -count=1 -race
   ```

## Development Workflow

### Branch Naming

Use conventional branch prefixes:
- `feat/<description>` -- new features
- `fix/<description>` -- bug fixes
- `refactor/<description>` -- code restructuring
- `test/<description>` -- test additions or fixes
- `docs/<description>` -- documentation changes
- `chore/<description>` -- maintenance tasks

### Making Changes

1. Create a branch from `main`.
2. Make your changes following the code style guidelines below.
3. Add or update tests for your changes.
4. Run the quality checks:
   ```bash
   go fmt ./...
   go vet ./...
   go test ./... -count=1 -race
   ```
5. Commit with a conventional commit message.
6. Open a pull request.

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>
```

**Types**: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`

**Scopes**: `trace`, `metrics`, `logging`, `health`, `analytics`

**Examples**:
```
feat(trace): add Jaeger exporter support
fix(metrics): prevent panic on nil labels
test(health): add concurrent registration tests
docs(analytics): document ExecuteReadQuery method
refactor(logging): extract level mapping to function
```

## Code Style

### General Rules

- Format with `gofmt`. No exceptions.
- Group imports: stdlib, third-party, internal (blank line separated). Use `goimports`.
- Line length: 100 characters maximum (readability first).
- All exported types and functions must have GoDoc comments.

### Naming Conventions

| Category | Convention | Example |
|----------|-----------|---------|
| Private fields/functions | `camelCase` | `buildSampler`, `getOrCreateCounter` |
| Exported types/functions | `PascalCase` | `TracerConfig`, `NewAggregator` |
| Constants | `PascalCase` or `UPPER_SNAKE_CASE` | `ExporterOTLP`, `StatusHealthy` |
| Acronyms | All-caps | `HTTP`, `URL`, `ID`, `SQL`, `TLS` |
| Receivers | 1-2 letters | `t` for Tracer, `c` for Collector, `a` for Aggregator |

### Error Handling

- Always check errors.
- Wrap errors with context: `fmt.Errorf("failed to create exporter: %w", err)`.
- Use `defer` for cleanup (close connections, end spans).
- No-op implementations must not return errors from methods that the interface does not require to return errors.

### Concurrency

- All public types must be safe for concurrent use.
- Use `sync.RWMutex` for read-heavy, write-light patterns.
- Use `sync.WaitGroup` for parallel execution.
- Pass `context.Context` as the first parameter.

### Testing

- Use table-driven tests with `testify/assert` and `testify/require`.
- Test naming: `Test<Struct>_<Method>_<Scenario>`.
- Unit tests must not require external services. Use `-short` flag.
- Integration tests use `//go:build integration` tag.
- Benchmark tests use `Benchmark<Function>` naming.

Example:

```go
func TestAggregator_Check_RequiredFailure(t *testing.T) {
    tests := []struct {
        name           string
        requiredErr    error
        optionalErr    error
        expectedStatus health.Status
    }{
        {
            name:           "required fails",
            requiredErr:    fmt.Errorf("connection refused"),
            optionalErr:    nil,
            expectedStatus: health.StatusUnhealthy,
        },
        // more cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            agg := health.NewAggregator(nil)
            agg.Register("db", health.StaticCheck(tt.requiredErr))
            agg.RegisterOptional("cache", health.StaticCheck(tt.optionalErr))
            report := agg.Check(context.Background())
            assert.Equal(t, tt.expectedStatus, report.Status)
        })
    }
}
```

## Package Guidelines

### Adding to an Existing Package

1. Implement the change on the concrete type.
2. If the change affects the interface, update the interface and all implementations (including NoOp).
3. Add tests covering the new behavior.
4. Update `docs/API_REFERENCE.md`.

### Adding a New Package

This module has a fixed set of five packages. New packages require architectural discussion before implementation. If you believe a new package is needed, open an issue describing the use case.

## Quality Checklist

Before submitting a pull request, verify:

- [ ] `go fmt ./...` produces no changes
- [ ] `go vet ./...` reports no issues
- [ ] `go test ./... -count=1 -race` passes
- [ ] `go test ./... -short` passes (unit tests only)
- [ ] All new exported types/functions have GoDoc comments
- [ ] No new third-party dependencies without justification
- [ ] Test coverage maintained or improved
- [ ] API_REFERENCE.md updated for new public API
- [ ] CHANGELOG.md updated

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

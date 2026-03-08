#!/usr/bin/env bash
# observability_functionality_challenge.sh - Validates Observability module core functionality
# Checks metrics collectors, tracing, health checks, logging, and analytics
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MODULE_NAME="Observability"

PASS=0
FAIL=0
TOTAL=0

pass() { PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); echo "  FAIL: $1"; }

echo "=== ${MODULE_NAME} Functionality Challenge ==="
echo ""

# --- Section 1: Required packages ---
echo "Section 1: Required packages (5)"

for pkg in analytics health logging metrics trace; do
    echo "Test: Package pkg/${pkg} exists"
    if [ -d "${MODULE_DIR}/pkg/${pkg}" ]; then
        pass "Package pkg/${pkg} exists"
    else
        fail "Package pkg/${pkg} missing"
    fi
done

# --- Section 2: Metrics ---
echo ""
echo "Section 2: Metrics (Prometheus)"

echo "Test: Collector interface exists in metrics"
if grep -q "type Collector interface" "${MODULE_DIR}/pkg/metrics/"*.go 2>/dev/null; then
    pass "Collector interface exists in metrics"
else
    fail "Collector interface missing in metrics"
fi

echo "Test: PrometheusCollector struct exists"
if grep -q "type PrometheusCollector struct" "${MODULE_DIR}/pkg/metrics/"*.go 2>/dev/null; then
    pass "PrometheusCollector struct exists"
else
    fail "PrometheusCollector struct missing"
fi

echo "Test: NoOpCollector exists for testing"
if grep -q "type NoOpCollector struct" "${MODULE_DIR}/pkg/metrics/"*.go 2>/dev/null; then
    pass "NoOpCollector exists"
else
    fail "NoOpCollector missing"
fi

# --- Section 3: Tracing ---
echo ""
echo "Section 3: Tracing (OpenTelemetry)"

echo "Test: Tracer struct exists"
if grep -q "type Tracer struct" "${MODULE_DIR}/pkg/trace/"*.go 2>/dev/null; then
    pass "Tracer struct exists"
else
    fail "Tracer struct missing"
fi

echo "Test: TracerConfig struct exists"
if grep -q "type TracerConfig struct" "${MODULE_DIR}/pkg/trace/"*.go 2>/dev/null; then
    pass "TracerConfig struct exists"
else
    fail "TracerConfig struct missing"
fi

# --- Section 4: Health checks ---
echo ""
echo "Section 4: Health checks"

echo "Test: Checker interface exists"
if grep -q "type Checker interface" "${MODULE_DIR}/pkg/health/"*.go 2>/dev/null; then
    pass "Checker interface exists"
else
    fail "Checker interface missing"
fi

echo "Test: Aggregator struct exists"
if grep -q "type Aggregator struct" "${MODULE_DIR}/pkg/health/"*.go 2>/dev/null; then
    pass "Aggregator struct exists"
else
    fail "Aggregator struct missing"
fi

echo "Test: Report struct exists"
if grep -q "type Report struct" "${MODULE_DIR}/pkg/health/"*.go 2>/dev/null; then
    pass "Report struct exists"
else
    fail "Report struct missing"
fi

echo "Test: ComponentResult struct exists"
if grep -q "type ComponentResult struct" "${MODULE_DIR}/pkg/health/"*.go 2>/dev/null; then
    pass "ComponentResult struct exists"
else
    fail "ComponentResult struct missing"
fi

# --- Section 5: Logging ---
echo ""
echo "Section 5: Logging"

echo "Test: Logger interface exists"
if grep -q "type Logger interface" "${MODULE_DIR}/pkg/logging/"*.go 2>/dev/null; then
    pass "Logger interface exists"
else
    fail "Logger interface missing"
fi

echo "Test: NoOpLogger exists for testing"
if grep -q "type NoOpLogger struct" "${MODULE_DIR}/pkg/logging/"*.go 2>/dev/null; then
    pass "NoOpLogger exists"
else
    fail "NoOpLogger missing"
fi

# --- Section 6: Analytics ---
echo ""
echo "Section 6: Analytics (ClickHouse)"

echo "Test: ClickHouseCollector struct exists"
if grep -q "type ClickHouseCollector struct" "${MODULE_DIR}/pkg/analytics/"*.go 2>/dev/null; then
    pass "ClickHouseCollector struct exists"
else
    fail "ClickHouseCollector struct missing"
fi

echo "Test: Event struct exists in analytics"
if grep -q "type Event struct" "${MODULE_DIR}/pkg/analytics/"*.go 2>/dev/null; then
    pass "Event struct exists in analytics"
else
    fail "Event struct missing in analytics"
fi

# --- Section 7: Source structure completeness ---
echo ""
echo "Section 7: Source structure"

echo "Test: Each package has non-test Go source files"
all_have_source=true
for pkg in analytics health logging metrics trace; do
    non_test=$(find "${MODULE_DIR}/pkg/${pkg}" -name "*.go" ! -name "*_test.go" -type f 2>/dev/null | wc -l)
    if [ "$non_test" -eq 0 ]; then
        fail "Package pkg/${pkg} has no non-test Go files"
        all_have_source=false
    fi
done
if [ "$all_have_source" = true ]; then
    pass "All packages have non-test Go source files"
fi

echo ""
echo "=== Results: ${PASS}/${TOTAL} passed, ${FAIL} failed ==="
[ "${FAIL}" -eq 0 ] && exit 0 || exit 1

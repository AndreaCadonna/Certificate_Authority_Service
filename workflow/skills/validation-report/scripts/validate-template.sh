#!/usr/bin/env bash
# =============================================================================
# Validation Script Template
# =============================================================================
# This template provides the skeleton for an automated behavioral validation
# script. Replace placeholders with project-specific scenarios.
#
# Design principles:
# - Runs from clean state with zero human intervention
# - Tests every scenario from SPEC.md §6
# - Tests contract enforcement (attempts violations, confirms rejection)
# - Outputs clear PASS/FAIL per scenario
# - Returns non-zero exit code if any scenario fails
# =============================================================================

set -euo pipefail

# --- Configuration ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
FAILURES=0
PASSES=0
TOTAL=0

# --- Helpers ---

pass() {
    local msg="$1"
    echo "  PASS: $msg"
    PASSES=$((PASSES + 1))
    TOTAL=$((TOTAL + 1))
}

fail() {
    local msg="$1"
    echo "  FAIL: $msg"
    FAILURES=$((FAILURES + 1))
    TOTAL=$((TOTAL + 1))
}

section() {
    echo ""
    echo "=== $1 ==="
    echo ""
}

# --- Setup ---

section "Setup"
echo "Setting up clean validation environment..."
# TODO: Add project-specific setup here
# - Create temp directories
# - Clean previous state
# - Build/compile if needed

# --- Scenarios ---

# TODO: Replace with actual scenarios from SPEC.md §6

section "Scenario 1: [Name] (REQ-XX-NNN)"
# Given: [initial state]
# When: [action]
# Then: [expected outcome]
#
# Example:
# OUTPUT=$(command arg1 arg2 2>&1)
# if [ $? -eq 0 ]; then
#     pass "Command exited successfully"
# else
#     fail "Command returned non-zero exit code"
# fi
#
# echo "$OUTPUT" | grep -q "expected string" && \
#     pass "Output contains expected string" || \
#     fail "Output missing expected string"

# --- Contract Verification ---

# TODO: Replace with actual contracts from CONTRACTS.md

section "Contract Verification: CON-XX — [Name]"
# Test that the system enforces the contract by attempting a violation
#
# Example:
# OUTPUT=$(command --invalid-input 2>&1) || true
# if [ $? -ne 0 ] || echo "$OUTPUT" | grep -q "error"; then
#     pass "CON-XX: System correctly rejects invalid input"
# else
#     fail "CON-XX: System accepted invalid input"
# fi

# --- Teardown ---

section "Teardown"
echo "Cleaning up validation environment..."
# TODO: Add project-specific teardown here

# --- Summary ---

echo ""
echo "==========================================="
echo " Validation Summary"
echo "==========================================="
echo " Total:  $TOTAL"
echo " Passed: $PASSES"
echo " Failed: $FAILURES"
echo "==========================================="

if [ "$FAILURES" -gt 0 ]; then
    echo " VERDICT: FAILURES FOUND"
    exit 1
else
    echo " VERDICT: ALL PASS"
    exit 0
fi

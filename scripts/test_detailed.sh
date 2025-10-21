#!/bin/bash

# This script runs tests with detailed output to identify issues

echo "=== Testing from wemixvisor directory ==="
echo "Working directory: $(pwd)"

# Change to wemixvisor directory
cd /Users/wm-it-22-00661/workspace/cosmovisor/wemixvisor || exit 1

echo ""
echo "=== Checking go.mod ==="
if [ -f go.mod ]; then
    echo "go.mod found"
    head -5 go.mod
else
    echo "ERROR: go.mod not found!"
    exit 1
fi

echo ""
echo "=== Testing internal/config package ==="
go test -v ./internal/config/ 2>&1 | head -30

echo ""
echo "=== Testing pkg/types package ==="
go test -v ./pkg/types/ 2>&1 | head -30

echo ""
echo "=== Testing pkg/logger package ==="
go test -v ./pkg/logger/ 2>&1 | head -30

echo ""
echo "=== Testing internal/upgrade package ==="
go test -v ./internal/upgrade/ 2>&1 | head -30

echo ""
echo "=== Running all tests with verbose output ==="
go test -v ./... 2>&1 | grep -E "(FAIL|ERROR|panic)" || echo "No errors found in basic test run"

echo ""
echo "=== Test Summary ==="
go test ./... 2>&1
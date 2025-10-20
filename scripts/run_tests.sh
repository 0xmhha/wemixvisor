#!/bin/bash

# Run tests from wemixvisor directory
echo "Running unit tests for wemixvisor..."

# Get the script directory and navigate to project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
cd "$PROJECT_ROOT"

echo "Testing config package..."
go test -v ./internal/config/

echo "Testing types package..."
go test -v ./pkg/types/

echo "Testing logger package..."
go test -v ./pkg/logger/

echo "Testing upgrade package..."
go test -v ./internal/upgrade/

echo "All tests completed!"
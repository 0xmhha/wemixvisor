#!/bin/bash

# Run tests from wemixvisor directory
echo "Running unit tests for wemixvisor..."

cd /Users/wm-it-22-00661/workspace/cosmovisor/wemixvisor

echo "Testing config package..."
go test -v ./internal/config/

echo "Testing types package..."
go test -v ./pkg/types/

echo "Testing logger package..."
go test -v ./pkg/logger/

echo "Testing upgrade package..."
go test -v ./internal/upgrade/

echo "All tests completed!"
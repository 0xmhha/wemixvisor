#!/bin/bash

# Setup test environment for wemixvisor
TEST_HOME="${1:-$HOME/.wemixd_test}"

echo "Setting up test environment at: $TEST_HOME"

# Create directory structure
mkdir -p "$TEST_HOME/wemixvisor/genesis/bin"
mkdir -p "$TEST_HOME/wemixvisor/upgrades"
mkdir -p "$TEST_HOME/data"
mkdir -p "$TEST_HOME/logs"

# Copy mock node as wemixd
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cp "$SCRIPT_DIR/mock_node.sh" "$TEST_HOME/wemixvisor/genesis/bin/wemixd"
chmod +x "$TEST_HOME/wemixvisor/genesis/bin/wemixd"

# Create symlink for current
ln -sfn genesis "$TEST_HOME/wemixvisor/current"

echo "Test environment setup complete!"
echo ""
echo "Directory structure:"
tree -L 4 "$TEST_HOME" 2>/dev/null || find "$TEST_HOME" -type d | head -20
echo ""
echo "To use this environment, set:"
echo "  export DAEMON_HOME=$TEST_HOME"
echo "  export DAEMON_NAME=wemixd"
#!/bin/bash
# Mock node for testing wemixvisor

# Log all arguments received
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Mock node started with args: $@" >> /tmp/mock_node.log

# Handle version command
if [[ "$1" == "version" ]] || [[ "$1" == "--version" ]]; then
    echo "Mock Geth v1.10.0-stable"
    exit 0
fi

# Trap signals for graceful shutdown
trap 'echo "[$(date)] Received SIGTERM, shutting down gracefully..." >> /tmp/mock_node.log; exit 0' TERM
trap 'echo "[$(date)] Received SIGINT, shutting down..." >> /tmp/mock_node.log; exit 0' INT

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Mock node running with PID $$" >> /tmp/mock_node.log
echo "Mock node started with PID $$"
echo "Arguments: $@"
echo "Press Ctrl+C to stop..."

# Keep running until signal received
while true; do
    sleep 1
done
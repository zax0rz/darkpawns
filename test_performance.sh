#!/bin/bash

echo "=== Dark Pawns Performance Optimization Test ==="
echo ""

# Build the performance demo
echo "Building performance demo..."
cd /home/zach/.openclaw/workspace/darkpawns_repo
go build -o perf_demo examples/performance_integration.go

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build successful!"
echo ""

# Run the demo
echo "Running performance demo..."
./perf_demo

echo ""
echo "=== Testing Complete ==="

# Clean up
rm -f perf_demo
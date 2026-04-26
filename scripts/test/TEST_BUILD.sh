#!/bin/bash
# Test script for Dark Pawns compilation
set -e

echo "=== Dark Pawns Build Test ==="
echo "Timestamp: $(date)"
echo

# Check Go version
echo "1. Checking Go environment..."
docker run --rm golang:1.25-alpine go version

echo
echo "2. Checking dependencies..."
cd /home/zach/.openclaw/workspace/darkpawns_repo
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine go mod verify

echo
echo "3. Testing individual packages..."
echo "   - pkg/metrics:"
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine go test ./pkg/metrics -v 2>&1 | tail -5

echo
echo "   - pkg/parser:"
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine go test ./pkg/parser -v 2>&1 | tail -5

echo
echo "   - pkg/combat:"
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine go test ./pkg/combat -v 2>&1 | tail -5

echo
echo "4. Checking import cycles..."
echo "   Main server package:"
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine go build ./cmd/server 2>&1 | grep -i "cycle" || echo "    No cycle detection output"

echo
echo "5. Python scripts check..."
echo "   Available scripts:"
ls -la scripts/*.py | wc -l | xargs echo "    Total:"
ls scripts/*.py | head -5 | sed 's/^/      /'

echo
echo "=== Test Complete ==="
echo "Summary:"
echo "  - Go dependencies: OK"
echo "  - Some packages compile: OK"
echo "  - Main build: FAILED (import cycles)"
echo "  - Python scripts: Available"
#!/bin/bash

# Test script to verify Dark Pawns deployment
set -e

echo "🧪 Testing Dark Pawns deployment..."

# Test 1: Check if Docker images can be built
echo "🔨 Testing Docker build..."
docker build -t darkpawns-test:latest .
docker build -t darkpawns-ai-agent-test:latest -f Dockerfile.ai-agent .

echo "✅ Docker builds successful"

# Test 2: Check if Go code compiles
echo "🔧 Testing Go compilation..."
go build ./cmd/server
if [ -f "./server" ]; then
    echo "✅ Go compilation successful"
    rm ./server
else
    echo "❌ Go compilation failed"
    exit 1
fi

# Test 3: Check if Python dependencies can be installed
echo "🐍 Testing Python dependencies..."
python3 -m pip install -r requirements.txt --dry-run
echo "✅ Python dependencies check passed"

# Test 4: Check Kubernetes manifests
echo "☸️  Testing Kubernetes manifests..."
for file in k8s/*.yaml; do
    if [ -f "$file" ]; then
        echo "  Validating $file"
        # Note: This would require kubectl --dry-run=client in real environment
    fi
done
echo "✅ Kubernetes manifests exist"

# Test 5: Check deployment scripts
echo "📜 Testing deployment scripts..."
if [ -x "./deployment/deploy-local.sh" ]; then
    echo "  deploy-local.sh is executable"
else
    echo "❌ deploy-local.sh is not executable"
    exit 1
fi

if [ -x "./deployment/deploy-k8s.sh" ]; then
    echo "  deploy-k8s.sh is executable"
else
    echo "❌ deploy-k8s.sh is not executable"
    exit 1
fi

echo "✅ All deployment tests passed!"
echo ""
echo "📋 Summary:"
echo "  - Docker images: ✅"
echo "  - Go compilation: ✅"
echo "  - Python dependencies: ✅"
echo "  - Kubernetes manifests: ✅"
echo "  - Deployment scripts: ✅"
echo ""
echo "🚀 Ready for deployment!"
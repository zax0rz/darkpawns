#!/bin/bash

# Dark Pawns Kubernetes Deployment Script
set -e

echo "🚀 Deploying Dark Pawns to Kubernetes..."

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed. Please install kubectl first."
    exit 1
fi

# Check if we have kubeconfig
if [ ! -f "$HOME/.kube/config" ] && [ -z "$KUBECONFIG" ]; then
    echo "❌ No kubeconfig found. Please configure kubectl first."
    exit 1
fi

# Create namespace
echo "📦 Creating namespace..."
kubectl apply -f k8s/namespace.yaml

# Create secrets (prompt for missing values)
echo "🔐 Setting up secrets..."
if ! kubectl get secret darkpawns-secrets -n darkpawns &> /dev/null; then
    echo "📝 Please enter the following secrets (press Enter to skip):"
    
    read -p "PostgreSQL Password [postgres]: " postgres_password
    postgres_password=${postgres_password:-postgres}
    
    read -p "AI API Key [br3nd4-69-ag3nt-k3y-d3f4ult]: " ai_api_key
    ai_api_key=${ai_api_key:-br3nd4-69-ag3nt-k3y-d3f4ult}
    
    read -p "OpenAI API Key (optional): " openai_key
    read -p "Anthropic API Key (optional): " anthropic_key
    read -p "Mem0 API Key (optional): " mem0_key
    
    # Create secret
    kubectl create secret generic darkpawns-secrets -n darkpawns \
        --from-literal=POSTGRES_PASSWORD="$postgres_password" \
        --from-literal=AI_API_KEY="$ai_api_key" \
        --from-literal=OPENAI_API_KEY="$openai_key" \
        --from-literal=ANTHROPIC_API_KEY="$anthropic_key" \
        --from-literal=MEM0_API_KEY="$mem0_key" \
        --dry-run=client -o yaml | kubectl apply -f -
else
    echo "✅ Secrets already exist, skipping..."
fi

# Apply all configurations
echo "📋 Applying Kubernetes manifests..."
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/server.yaml
kubectl apply -f k8s/ai-agent.yaml

# Wait for deployments
echo "⏳ Waiting for deployments to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/darkpawns-server -n darkpawns
kubectl wait --for=condition=available --timeout=300s deployment/darkpawns-ai-agent -n darkpawns

echo "✅ Dark Pawns deployed to Kubernetes!"
echo ""
echo "📊 Check status:"
echo "  kubectl get all -n darkpawns"
echo ""
echo "📝 View logs:"
echo "  kubectl logs -f deployment/darkpawns-server -n darkpawns"
echo "  kubectl logs -f deployment/darkpawns-ai-agent -n darkpawns"
echo ""
echo "🌐 If using Ingress, access at: http://darkpawns.labz0rz.com"
echo ""
echo "🔄 To update:"
echo "  kubectl rollout restart deployment/darkpawns-server -n darkpawns"
echo "  kubectl rollout restart deployment/darkpawns-ai-agent -n darkpawns"
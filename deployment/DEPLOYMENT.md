# Dark Pawns Deployment Guide

This document covers how to deploy Dark Pawns locally and to production.

## Prerequisites

### For Local Deployment
- Docker 20.10+
- Docker Compose 2.0+
- Git

### For Kubernetes Deployment
- kubectl configured with cluster access
- Kubernetes cluster (minikube, k3s, EKS, GKE, AKS, etc.)
- Container registry access (Docker Hub, GHCR, ECR, etc.)

## Local Deployment

### Quick Start
```bash
# Clone the repository
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns

# Run the deployment script
./deployment/deploy-local.sh
```

### Manual Steps
1. **Set environment variables** (optional):
   ```bash
   cp .env.example .env
   # Edit .env to add your API keys
   ```

2. **Build and start services**:
   ```bash
   docker-compose build
   docker-compose up -d
   ```

3. **Verify services are running**:
   ```bash
   docker-compose ps
   ```

### Accessing Services
- **Game Server**: http://localhost:8080
- **WebSocket**: ws://localhost:8080/ws
- **PostgreSQL**: localhost:5432 (database: `darkpawns`)
- **Redis**: localhost:6379

### Useful Commands
```bash
# View logs
docker-compose logs -f server
docker-compose logs -f ai-agent

# Stop services
docker-compose down

# Rebuild and restart
docker-compose up -d --build

# Access database
docker-compose exec postgres psql -U postgres -d darkpawns
```

## Kubernetes Deployment

### Quick Start
```bash
# Ensure kubectl is configured
kubectl cluster-info

# Run the deployment script
./deployment/deploy-k8s.sh
```

### Manual Deployment Steps

1. **Create namespace**:
   ```bash
   kubectl apply -f k8s/namespace.yaml
   ```

2. **Create secrets**:
   ```bash
   kubectl create secret generic darkpawns-secrets -n darkpawns \
     --from-literal=POSTGRES_PASSWORD=your_password \
     --from-literal=AI_API_KEY=your_ai_key \
     --from-literal=OPENAI_API_KEY=your_openai_key \
     --from-literal=ANTHROPIC_API_KEY=your_anthropic_key \
     --from-literal=MEM0_API_KEY=your_mem0_key
   ```

3. **Apply all manifests**:
   ```bash
   kubectl apply -f k8s/
   ```

4. **Wait for deployment**:
   ```bash
   kubectl wait --for=condition=available --timeout=300s deployment/darkpawns-server -n darkpawns
   kubectl wait --for=condition=available --timeout=300s deployment/darkpawns-ai-agent -n darkpawns
   ```

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Postgres  │  │    Redis    │  │   Server    │        │
│  │  StatefulSet│  │  Deployment │  │ Deployment  │        │
│  │   (1 pod)   │  │   (1 pod)   │  │   (2 pods)  │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
│         │                 │                 │               │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐        │
│  │   Service   │  │   Service   │  │   Service   │        │
│  │   :5432     │  │   :6379     │  │   :8080     │        │
│  └─────────────┘  └─────────────┘  └──────┬──────┘        │
│                                            │               │
│                                    ┌──────▼──────┐        │
│                                    │   Ingress   │        │
│                                    │  (NGINX)    │        │
│                                    └──────┬──────┘        │
│                                           │               │
│                                    ┌──────▼──────┐        │
│                                    │   AI Agent  │        │
│                                    │ Deployment  │        │
│                                    │   (1 pod)   │        │
│                                    └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

### Monitoring and Maintenance

**Check status**:
```bash
kubectl get all -n darkpawns
kubectl get pods -n darkpawns
kubectl get svc -n darkpawns
```

**View logs**:
```bash
kubectl logs -f deployment/darkpawns-server -n darkpawns
kubectl logs -f deployment/darkpawns-ai-agent -n darkpawns
```

**Update deployment**:
```bash
# Update image
kubectl set image deployment/darkpawns-server server=ghcr.io/zax0rz/darkpawns:latest -n darkpawns

# Restart deployments
kubectl rollout restart deployment/darkpawns-server -n darkpawns
kubectl rollout restart deployment/darkpawns-ai-agent -n darkpawns
```

**Troubleshooting**:
```bash
# Describe pods for details
kubectl describe pod -n darkpawns -l app=darkpawns-server

# Check events
kubectl get events -n darkpawns --sort-by='.lastTimestamp'

# Access containers
kubectl exec -it deployment/darkpawns-server -n darkpawns -- /bin/sh
```

## CI/CD Pipeline

The GitHub Actions workflow (`.github/workflows/ci.yml`) provides:

1. **Testing**: Go tests, Python tests, server health check
2. **Build**: Docker images for server and AI agent
3. **Push**: Images to GitHub Container Registry
4. **Deploy**: Automatic deployment to Kubernetes on main branch push

### Manual Trigger
```bash
# Push to trigger CI/CD
git push origin main

# Or manually trigger from GitHub UI
```

## Environment Variables

### Required
- `POSTGRES_PASSWORD`: PostgreSQL password
- `AI_API_KEY`: Agent API key for authentication

### Optional
- `OPENAI_API_KEY`: OpenAI API key for AI features
- `ANTHROPIC_API_KEY`: Anthropic API key for AI features
- `MEM0_API_KEY`: Mem0 API key for memory features
- `WORLD_DIR`: Path to world files (default: `./lib`)

## Security Considerations

1. **Never commit secrets** to version control
2. **Use Kubernetes Secrets** or environment-specific `.env` files
3. **Enable TLS/SSL** for production deployments
4. **Set resource limits** to prevent resource exhaustion
5. **Regularly update** base images for security patches

## Backup and Recovery

### Database Backup
```bash
# Local
docker-compose exec postgres pg_dump -U postgres darkpawns > backup.sql

# Kubernetes
kubectl exec -n darkpawns deployment/darkpawns-postgres -- pg_dump -U postgres darkpawns > backup.sql
```

### Restore
```bash
# Local
docker-compose exec -T postgres psql -U postgres darkpawns < backup.sql

# Kubernetes
kubectl exec -i -n darkpawns deployment/darkpawns-postgres -- psql -U postgres darkpawns < backup.sql
```

## Scaling

### Horizontal Scaling
```bash
# Scale server pods
kubectl scale deployment darkpawns-server --replicas=3 -n darkpawns

# Scale AI agent pods
kubectl scale deployment darkpawns-ai-agent --replicas=2 -n darkpawns
```

### Vertical Scaling
Edit resource requests/limits in Kubernetes manifests:
- `k8s/server.yaml`
- `k8s/ai-agent.yaml`
- `k8s/postgres.yaml`
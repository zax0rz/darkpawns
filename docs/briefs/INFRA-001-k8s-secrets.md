# Brief: INFRA-001 — k8s Deployment Missing JWT_SECRET and ENCRYPTION_KEY

**Issues:** DP-550 (MEDIUM)
**Priority:** MEDIUM — infrastructure
**Files:** `k8s/server.yaml`

## Problem

`k8s/server.yaml` — The deployment manifest references `DATABASE_URL`, `REDIS_URL`, `AI_ENABLED` from a ConfigMap and `POSTGRES_PASSWORD` from a Secret, but does NOT reference `JWT_SECRET` or `ENCRYPTION_KEY`.

Both are required at runtime:
- `pkg/auth/jwt.go:69` — JWT signing requires `JWT_SECRET`
- `pkg/secrets/manager.go:32` — Encryption key for secrets management

If these aren't injected, the server either:
1. Fails to start (if env vars are required and missing)
2. Uses hardcoded/insecure defaults (worse)
3. Has them baked into the container image (visible in `kubectl describe pod` and git history)

## Required Fix

Add both to the k8s Secret and reference them in the deployment:

### Step 1: Add to k8s Secret (manual step for The Architect)

```bash
kubectl create secret generic darkpawns-secrets \
  --from-literal=JWT_SECRET="<generate-random-32-byte-hex>" \
  --from-literal=ENCRYPTION_KEY="<generate-random-32-byte-hex>" \
  --namespace=darkpawns \
  --dry-run=client -o yaml | kubectl apply -f -
```

Or add to an existing secret:
```bash
kubectl patch secret darkpawns-secrets \
  --namespace=darkpawns \
  -p '{"data":{"JWT_SECRET":"'$(echo -n '<value>' | base64)'","ENCRYPTION_KEY":"'$(echo -n '<value>' | base64)'"}}'
```

### Step 2: Add env vars to k8s/server.yaml

Add to the `env` section of the container spec:
```yaml
- name: JWT_SECRET
  valueFrom:
    secretKeyRef:
      name: darkpawns-secrets
      key: JWT_SECRET
- name: ENCRYPTION_KEY
  valueFrom:
    secretKeyRef:
      name: darkpawns-secrets
      key: ENCRYPTION_KEY
```

## Verification

- Confirm the secret exists: `kubectl get secret darkpawns-secrets -n darkpawns`
- After deploying, check pod logs for JWT initialization (no errors)
- Verify admin login still works (JWT generation succeeds)
- `kubectl describe pod` should NOT show plaintext JWT_SECRET or ENCRYPTION_KEY

## Context

This is a deployment configuration issue, not a code issue. The Go code already reads these from environment variables — the k8s manifest just doesn't provide them. **This should be done by The Architect or someone with k8s cluster access.**

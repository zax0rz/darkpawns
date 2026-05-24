---
title: "Installation"
description: "Install and set up the Dark Pawns server and client"
date: 2026-04-22
draft: false
section: "docs"
---

# Installation & Deployment Guide

This guide details the exact system requirements, compilation steps, command-line flags, and containerization structures required to build, run, and deploy the Dark Pawns MUD server.

---

## System Requirements

To compile and execute the Go port of the Dark Pawns server, your environment must satisfy the following dependencies:

*   **Go Compiler:** Go 1.26.3+ is required (as specified in `go.mod`).
*   **Database (Optional for sandbox, required for production):** PostgreSQL 15+ is used for character persistence, agent decision capture logging, and audit storage.
    *   *Graceful Fallback:* If no database is available, the server boots in sandbox mode. Characters can connect and explore, but their stats and items will not persist across disconnects.
*   **Operating System:** Fully compatible with macOS, Linux, and FreeBSD (raw POSIX bindings are utilized).

---

## Local Compilation

To clone, compile, and run the server locally:

1.  **Clone the Repository:**
    ```bash
    git clone https://github.com/zax0rz/darkpawns.git
    cd darkpawns
    ```
2.  **Compile the Binary:**
    ```bash
    go build -o server ./cmd/server
    ```
3.  **Boot the Server:**
    To run the server with default options, you must supply the path to the original area world files using the `-world` flag:
    ```bash
    ./server -world ./lib/world -port 4350
    ```
4.  **Connect:**
    Open another shell and connect via telnet:
    ```bash
    telnet localhost 4350
    ```

---

## Command-Line Flags

The server binary supports the following startup arguments:

| Flag | Parameter | Purpose | Default |
| --- | --- | --- | --- |
| `-world` | `<path>` | **Required.** Path to world files (`lib/` directory containing `wld/`, `mob/`, `obj/`, `zon/`, and `shp/` folders). | *None (Fails if omitted)* |
| `-scripts` | `<path>` | Path to Lua scripts directory. | `world/lib/scripts` |
| `-port` | `<port>` | Port on which the TCP telnet and HTTP/WS server will listen. | `4350` |
| `-db` | `<conn_string>` | PostgreSQL connection string. | `postgres://postgres:postgres@localhost/darkpawns?sslmode=disable` |
| `-web` | `<path>` | Path to legacy human web client static assets. | *None* |
| `-hugo` | `<path>` | Path to compiled Hugo static site (serves as root `/` path). | *None* |

---

## Containerization (Docker)

The repository provides five multi-stage Dockerfiles optimized for distinct production and observation tasks:

| Dockerfile | Purpose | Build Command |
| --- | --- | --- |
| `Dockerfile` | Standard production build including Lua VM sandboxes. | `docker build -f Dockerfile -t darkpawns .` |
| `Dockerfile.local` | Development workspace container with hot-reloading hooks. | `docker build -f Dockerfile.local -t darkpawns-dev .` |
| `Dockerfile.ai-agent` | AI agent sidecar image (Python). | `docker build -f Dockerfile.ai-agent -t dp-agent .` |
| `Dockerfile.privacy-filter` | Fail-closed PII and audit scrubbing sidecar. | `docker build -f Dockerfile.privacy-filter -t dp-privacy .` |
| `Dockerfile.prebuilt` | Lean runtime image utilizing a pre-built static Go binary. | `docker build -f Dockerfile.prebuilt -t darkpawns-static .` |

### Running via Docker
To boot the production server container locally, mounting the world files:
```bash
docker run -d \
  -p 4350:4350 \
  -v $(pwd)/lib/world:/app/lib/world \
  --name darkpawns-server \
  darkpawns
```

---

## Production Deployment (Kubernetes)

Full orchestration manifests reside inside the **`k8s/`** directory. To deploy a high-availability, persisted Dark Pawns cluster with Postgres backing and AI sidecars, execute the following sequence:

1.  **Initialize Namespace:**
    ```bash
    kubectl apply -f k8s/namespace.yaml
    ```
2.  **Mount Configs and Secrets:**
    ```bash
    kubectl apply -f k8s/configmap.yaml
    kubectl apply -f k8s/secrets.yaml
    ```
3.  **Deploy Postgres Persistence:**
    ```bash
    kubectl apply -f k8s/postgres.yaml
    ```
4.  **Deploy Redis (caching layer):**
    ```bash
    kubectl apply -f k8s/redis.yaml
    ```
5.  **Deploy Game Server & Services:**
    ```bash
    kubectl apply -f k8s/server.yaml
    ```
6.  **Deploy Autonomous Agent Sidecar:**
    ```bash
    kubectl apply -f k8s/ai-agent.yaml
    ```

Check the status of the cluster deployment:
```bash
kubectl get pods -n darkpawns
```

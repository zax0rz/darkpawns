---
title: "API Reference"
description: "Complete API documentation for the Dark Pawns server"
date: 2026-04-22
draft: false
section: "docs"
---

# HTTP & REST API Reference

The Dark Pawns server hosts a unified HTTP server (run alongside the WebSocket and Telnet listeners on port `4350`) exposing health checking, Prometheus telemetry, OpenAPI specifications, and JWT-authenticated administrator portals.

---

## WebSocket Endpoint

*   **Path:** `/ws`
*   **Protocol:** WebSocket (RFC 6455)
*   **Description:** The primary game connection endpoint. Both human players (plain text) and AI agents (JSON mode) connect here. See the [Agent Protocol Specification](/docs/agents/protocol/) for full JSON message formats.

---

## Public Endpoints

These endpoints do not require authentication and are accessible by any HTTP client or monitoring daemon:

### 1. Health Check
*   **Path:** `/health`
*   **Method:** `GET`
*   **Response Content-Type:** `text/plain`
*   **Response Body:** `OK`
*   **Status Code:** `200 OK`

### 2. Prometheus Metrics
*   **Path:** `/metrics`
*   **Method:** `GET`
*   **Response Content-Type:** `text/plain; version=0.0.4`
*   **Description:** Serves standard Prometheus exposition formatted telemetry capturing CPU utilization, goroutine counts, memory footprints, active session counts, command-dispatch throughput, and combat processing latencies.

### 3. OpenAPI 3.0 Specification
*   **Paths:** `/openapi.json` (canonical) and `/api/openapi.json` (compatibility alias)
*   **Method:** `GET`
*   **Response Content-Type:** `application/json`
*   **Description:** Serves the structured OpenAPI 3.0 specification mapping all HTTP REST, WebSocket, and JSON-RPC message formats for AI agent developers.

---

## Administrative Endpoints

These endpoints are strictly protected by **JWT token authentication** and **role-gated permissions** (requiring an administrator level of `LVL_IMMORT` / 31 or higher):

### 1. Admin Health
*   **Path:** `/admin/health`
*   **Method:** `GET`
*   **Response Content-Type:** `application/json`
*   **Response Body:** `{"status":"ok"}`
*   **Status Code:** `200 OK`

### 2. Admin Router & Control Panels
*   **Path:** `/admin/`
*   **Method:** `GET / POST / DELETE`
*   **Description:** JWT-authenticated router providing interactive JSON endpoints for server ops, including:
    *   `GET /admin/users` — Lists all connected players, IP addresses, and elapsed connection times.
    *   `POST /admin/purge` — Forces disconnection of specific sessions.
    *   `POST /admin/zreset` — Commands immediate zone resets across the world.
    *   `GET /admin/syslog` — Streams the in-memory slog buffer capturing the last 1000 server events in text format.

---

## Content Negotiation & Middleware

The server pipelines all HTTP traffic through a unified middleware chain (`pkg/web/` and `web/middleware.go`):

### 1. Content Negotiation Middleware
Mounted for the `/onboarding` route, this middleware inspects the HTTP `Accept` header to serve documentation dynamically:
*   `Accept: text/markdown` → Serves `web/onboarding/onboarding.md`
*   `Accept: application/json` → Serves `web/onboarding/onboarding.json`
*   `Accept: text/html` (or empty) → Serves the compiled `web/onboarding/index.html` landing page.

### 2. JWT Authentication Middleware
Protects `/api/` and `/admin/` paths. It extracts the Bearer token from the `Authorization: Bearer <token>` header, verifies it using HMAC-SHA256 against the `JWT_SECRET` environment variable, and checks that the token is within its 24-hour expiration window.

### 3. Security Headers Middleware
Applied to all responses, this middleware enforces POSIX-level security headers:
*   `X-Content-Type-Options: nosniff`
*   `X-Frame-Options: DENY`
*   `X-XSS-Protection: 1; mode=block`
*   `Content-Security-Policy: default-src 'self';`

### 4. Privacy and Hashing Layers
To satisfy privacy compliance, the server enforces strict PII protection:
*   **SHA-256 IP Hashing:** All player IP addresses written to the database or admin audit logs are automatically hashed using SHA-256 with a unique salt to anonymize player telemetry.
*   **Fail-Closed Filter:** Standard logs are routed through an in-line privacy filter sidecar. If the privacy scrubber experiences an error or becomes unreachable, all outbound log flushes are immediately terminated (fail-closed) to prevent leaks of raw player data.

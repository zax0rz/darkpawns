# Security Hardening Guide

## Overview
This guide documents the security hardening implemented in Dark Pawns and provides instructions for maintaining and extending security measures.

## Table of Contents
1. [CORS Configuration](#cors-configuration)
2. [JWT Authentication](#jwt-authentication)
3. [Input Validation](#input-validation)
4. [Secrets Management](#secrets-management)
5. [Security Headers](#security-headers)
6. [Rate Limiting](#rate-limiting)
7. [Audit Logging](#audit-logging)
8. [Deployment Security](#deployment-security)
9. [Monitoring & Maintenance](#monitoring--maintenance)

## CORS Configuration

### Configuration
CORS (Cross-Origin Resource Sharing) is configured via the `CORS_ALLOWED_ORIGINS` environment variable:

```bash
# Development (default)
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# Production
CORS_ALLOWED_ORIGINS=https://darkpawns.example.com,https://game.darkpawns.example.com
```

### Features
- **Environment-based configuration**: Different settings for development/production
- **Wildcard support**: Supports `*.example.com` patterns
- **Preflight handling**: Automatic OPTIONS request handling
- **WebSocket validation**: Origin validation for WebSocket connections

### Implementation Files
- `web/cors.go` - CORS middleware
- `pkg/session/manager.go` - WebSocket origin validation

## JWT Authentication

### Configuration
JWT authentication requires a secure secret:

```bash
# Generate a secure JWT secret (min 32 characters)
JWT_SECRET=$(openssl rand -base64 32)
```

### Token Structure
```json
{
  "player_name": "PlayerName",
  "is_agent": false,
  "agent_key_id": 0,
  "exp": 1672531200,
  "iat": 1672444800,
  "iss": "darkpawns",
  "sub": "PlayerName"
}
```

### Usage
1. **Login**: Client receives JWT token in login response
2. **API Requests**: Include token in `Authorization: Bearer <token>` header
3. **Validation**: Server validates token on each protected request

### Implementation Files
- `pkg/auth/jwt.go` - JWT generation and validation
- `pkg/session/manager.go` - Token generation in login flow
- `pkg/session/protocol.go` - Token field in StateData

## Input Validation

### Validation Layers
1. **Player Names**: Length (2-32 chars), character set, reserved names
2. **Command Input**: SQL injection, XSS, path traversal detection
3. **API Input**: All user-provided data validated

### Patterns Blocked
- **SQL Injection**: `UNION SELECT`, `DROP TABLE`, `--`, `;`
- **XSS**: `<script>`, `javascript:`, `onload=`, `data:`
- **Path Traversal**: `../`, `..\`, `/etc/passwd`

### Implementation Files
- `pkg/validation/validation.go` - Player name validation
- `pkg/validation/input.go` - Comprehensive input validation
- `pkg/session/manager.go` - Validation in message handling

## Secrets Management

### Encryption
Secrets are encrypted using AES-256-GCM:

```go
// Encryption key (32 bytes)
ENCRYPTION_KEY=$(openssl rand -base64 32)
```

### Storage Options
1. **Environment Variables**: Primary method for development
2. **Encrypted Files**: `/run/secrets/<name>.enc` for production
3. **Secret Managers**: Extensible for cloud providers (AWS Secrets Manager, etc.)

### Implementation Files
- `pkg/secrets/manager.go` - Secret encryption/decryption
- `.env.example` - Template with secure defaults

## Security Headers

### Headers Implemented
```http
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws: wss:; font-src 'self'; object-src 'none'; media-src 'self'; frame-src 'none'; frame-ancestors 'none'
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), microphone=(), camera=()
Strict-Transport-Security: max-age=31536000; includeSubDomains  # Production only
```

### Implementation Files
- `web/security.go` - Security headers middleware
- `cmd/server/main_web.go` - Middleware integration

## Rate Limiting

### Protection Areas
1. **Login Attempts**: 5 requests/second per IP
2. **Commands**: 10 commands/second per session
3. **API Endpoints**: Configurable per endpoint

### Implementation Files
- `pkg/auth/ratelimit.go` - IP-based rate limiting
- `pkg/session/manager.go` - Command rate limiting

## Audit Logging

### Events Logged
1. **Authentication**: Login attempts, successes, failures
2. **Security**: Rate limit hits, validation failures
3. **Administration**: Admin actions, configuration changes

### Log Format
```json
{
  "timestamp": "2026-04-22T21:34:00Z",
  "event_type": "login_attempt",
  "player_name": "PlayerName",
  "ip_address": "192.168.1.100",
  "success": true,
  "details": "Login successful"
}
```

### Implementation Files
- `pkg/audit/logger.go` - Audit logging system
- `pkg/session/manager.go` - Audit logging integration

## Deployment Security

### Docker Security
```yaml
# docker-compose.yml security settings
services:
  server:
    read_only: true  # Read-only filesystem
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
```

### Kubernetes Security
```yaml
# k8s/deployment.yaml security settings
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop:
      - ALL
```

### TLS/HTTPS
```bash
# Enable TLS in production
USE_TLS=true
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem
```

## Monitoring & Maintenance

### Regular Tasks
1. **Log Review**: Daily review of security audit logs
2. **Secret Rotation**: Quarterly rotation of JWT and encryption keys
3. **Dependency Updates**: Weekly security updates for dependencies
4. **Security Scanning**: Monthly vulnerability scans

### Monitoring Metrics
- Failed login attempts per hour
- Rate limit hits
- Invalid token attempts
- Security event frequency

### Incident Response
1. **Detection**: Monitor audit logs for anomalies
2. **Containment**: Block malicious IPs, revoke compromised tokens
3. **Investigation**: Analyze logs, identify attack vectors
4. **Remediation**: Patch vulnerabilities, update security controls

## Security Testing

### Automated Tests
```bash
# Run security tests
go test ./pkg/auth/... -v
go test ./pkg/validation/... -v

# Security scanning
gosec ./...
govulncheck ./...
```

### Manual Testing
1. **Penetration Testing**: Quarterly external penetration tests
2. **Code Review**: Security-focused code reviews for all changes
3. **Configuration Review**: Monthly security configuration audits

## Compliance

### Standards Implemented
- **OWASP Top 10**: Protection against common web vulnerabilities
- **CIS Benchmarks**: Security configuration benchmarks
- **GDPR**: Data protection and privacy

### Documentation
- Security policies and procedures
- Incident response plan
- Data protection impact assessments

## Emergency Procedures

### Security Breach
1. **Immediate Actions**:
   - Isolate affected systems
   - Preserve logs and evidence
   - Notify security team
2. **Containment**:
   - Block malicious traffic
   - Revoke compromised credentials
   - Apply emergency patches
3. **Recovery**:
   - Restore from clean backups
   - Deploy security updates
   - Monitor for recurrence

### Contact Information
- **Security Team**: security@darkpawns.example.com
- **Emergency Contact**: +1-555-123-4567
- **Security Mailing List**: security-alerts@darkpawns.example.com

## Updates & Maintenance

This guide should be reviewed and updated quarterly to reflect:
- New security threats and vulnerabilities
- Changes in compliance requirements
- Updates to security tools and practices
- Lessons learned from security incidents

---

*Last Updated: 2026-04-22*  
*Version: 2.0*  
*Author: Agent 87 (Security Hardening Subagent)*
# Privacy Filter Integration for Dark Pawns

**Last updated:** 2026-09-18

## Overview

The Privacy Filter integration uses OpenAI's Privacy Filter to detect and redact Personally Identifiable Information (PII) from game logs before storage or processing. This helps protect player privacy and comply with data protection regulations.

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌────────────────────┐
│  Dark Pawns     │    │  Privacy Filter  │    │  OpenAI Privacy    │
│    Server       │────│     Client       │────│     Filter API     │
│                 │    │  (Go package)    │    │  (Python service)  │
└─────────────────┘    └──────────────────┘    └────────────────────┘
         │                        │                        │
         │                        │                        │
         ▼                        ▼                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌────────────────────┐
│   Game Logs     │    │  Configuration   │    │  1.5B Parameter    │
│  (WebSocket,    │    │   (.env files)   │    │      Model         │
│    HTTP, etc.)  │    │                  │    │                    │
└─────────────────┘    └──────────────────┘    └────────────────────┘
```

## PII Categories Detected

The OpenAI Privacy Filter detects 8 categories of PII:

1. **Account Numbers** - Bank accounts, credit cards, etc.
2. **Addresses** - Physical addresses
3. **Emails** - Email addresses
4. **Persons** - Names of people
5. **Phones** - Phone numbers
6. **URLs** - Web addresses
7. **Dates** - Dates that could identify individuals
8. **Secrets** - Passwords, API keys, tokens

## Installation

The service is native Python, run on the same host as the game server:

```bash
python3 -m venv ~/.venvs/darkpawns-privacy
~/.venvs/darkpawns-privacy/bin/pip install -r deployment/requirements.txt
```

The `opf` package downloads the model weights (~3 GB BF16, Apache 2.0,
`openai/privacy-filter`) to `~/.opf/privacy_filter` the first time the model
is constructed — at service startup, so the download is visible in the logs.
Pre-seed that directory or set `OPF_CHECKPOINT` to use an existing
checkpoint. The game server never downloads the model; only this service
touches it, and text never leaves the host.

Run the service:

```bash
cd deployment
~/.venvs/darkpawns-privacy/bin/uvicorn privacy_filter_api:app --host 127.0.0.1 --port 8001
```

Bind and device are environment-controlled: `PRIVACY_FILTER_API_HOST`
(default `127.0.0.1`), `PRIVACY_FILTER_API_PORT` (default `8001`),
`PRIVACY_FILTER_DEVICE` (default `cpu`; the model is CPU-feasible). Then
point the game server at it with `PRIVACY_FILTER_URL=http://127.0.0.1:8001`.

The wire contract between this service and the Go client is pinned by tests
on both sides: `deployment/test_privacy_filter_api.py` (no model needed,
`python -m pytest`) and the stub-server tests in `pkg/privacy`.

**Wiring status:** the PII slog handler in `pkg/privacy` is still not
installed in `cmd/server`, on purpose — but not for the reason it is tempting
to assume. `Client.FilterText` does return `[FILTERED]` when the service is
unreachable, yet `PIIHandler` discards that fallback and restores the original
message (see Fallback Strategies below). So an unwired or unreachable filter
does not destroy logs: it lets them through **unfiltered**, carrying only a
`pii_filter_error` attr. The service must exist first because otherwise there
is no protection while appearing to have some, which is the worse failure.

Two things must be settled before that wiring lands, and neither belongs here:
`PIIHandler.Handle` is synchronous and calls the filter once per message plus
once per string attribute, so a record with three string attrs costs four
sequential round-trips to CPU inference against a 10-pulse-per-second game
loop; and `fallbackFilter` returns a sentinel its only caller throws away,
where a real local regex scrubber would give operators who never run this
service actual protection.

## Configuration

### Environment Variables

Create a `.env.privacy` file or set environment variables:

```bash
# Privacy Filter Service
PRIVACY_FILTER_URL=http://localhost:8001
PRIVACY_FILTER_ENABLED=true

# What to filter (comma-separated)
PRIVACY_FILTER_CATEGORIES=account_number,address,email,person,phone,url,secret

# Replacement text
PRIVACY_FILTER_REPLACEMENT=[REDACTED]

# Game-specific settings
FILTER_PLAYER_NAMES=true
FILTER_LOCATION_NAMES=false
FILTER_COMMANDS=false
FILTER_COMBAT_DETAILS=false
```

### Programmatic Configuration

```go
import "github.com/zax0rz/darkpawns/pkg/privacy"

// Load from environment
config := privacy.LoadConfig()

// Or create manually
config := privacy.Config{
    URL:        "http://localhost:8001",
    Enabled:    true,
    Categories: []string{"person", "email", "phone"},
}

// Create client
client := privacy.NewClient(config.URL, config.ToFilterConfig())
```

## Usage

### Basic Logging

```go
import "github.com/zax0rz/darkpawns/pkg/privacy"

// Use global logger
privacy.Println("Player John Doe (john@example.com) logged in from 192.168.1.100")
// Output: Player [REDACTED] ([REDACTED]) logged in from 192.168.1.100

// Create custom logger
client := privacy.NewClient("http://localhost:8001", privacy.DefaultFilterConfig())
logger := privacy.NewPrivacyLogger(client, "[GAME] ", log.LstdFlags)
logger.Printf("Player %s purchased item %d", playerName, itemID)
```

### HTTP Middleware

```go
import (
    "net/http"
    "github.com/zax0rz/darkpawns/pkg/privacy"
)

client := privacy.NewClient("http://localhost:8001", privacy.DefaultFilterConfig())
handler := privacy.HTTPMiddleware(yourHandler, client)

http.ListenAndServe(":4350", handler)
```

### WebSocket Logging

```go
import "github.com/zax0rz/darkpawns/pkg/privacy"

wsLogger := privacy.NewWebSocketLogger(client, "[WS] ")

// Log incoming messages
wsLogger.LogIncoming(sessionID, message)

// Log outgoing messages  
wsLogger.LogOutgoing(sessionID, message)

// Log events
wsLogger.LogEvent(sessionID, "connect", "Player connected from "+remoteAddr)
```

## Performance Considerations

### GPU vs CPU
- **GPU Recommended**: The 1.5B parameter model runs significantly faster on GPU
- **CPU Fallback**: Works on CPU but slower (~2-3 seconds per request)
- **Batch Processing**: Use `BatchFilter` for multiple texts to reduce overhead

### Caching
Consider implementing caching for frequently logged patterns:
```go
type CachingFilter struct {
    client *privacy.Client
    cache  *lru.Cache
}

func (cf *CachingFilter) FilterText(text string) (string, []string, error) {
    if cached, ok := cf.cache.Get(text); ok {
        return cached.(string), []string{"cached"}, nil
    }
    filtered, detected, err := cf.client.FilterText(text)
    if err == nil {
        cf.cache.Add(text, filtered)
    }
    return filtered, detected, err
}
```

### Fallback Strategies
When the privacy filter service is unavailable:

1. **Fail closed** (default): `Client.FilterText` replaces the filtered
   output with `[FILTERED]` and returns an error, so callers can tell
   degradation from health (DP-1241).
2. **PII slog handler**: on a filter error it keeps the original message and
   annotates it with `pii_filter_error`, rather than destroying it — the
   regression tests in `slog_handler_test.go` pin this.

## Testing

```bash
# Go unit tests (stub servers, no service needed)
go test -v ./pkg/privacy/...

# Service contract tests (no model download)
cd deployment && python -m pytest test_privacy_filter_api.py

# Live check against a running service — the only test that touches the
# real model; skips when PRIVACY_FILTER_URL is unset
make privacy-test
```

Test examples:
```go
func TestPlayerLogin(t *testing.T) {
    client := privacy.NewClient(testURL, privacy.DefaultFilterConfig())
    
    // Test with PII
    filtered, _, _ := client.FilterText(
        "Player John Doe (john@example.com, 555-1234) logged in",
    )
    
    assert.NotContains(t, filtered, "John Doe")
    assert.NotContains(t, filtered, "john@example.com")
    assert.NotContains(t, filtered, "555-1234")
}
```

## Monitoring

### Health Checks
```bash
# Check privacy filter service
curl http://localhost:8001/health

# Get available categories
curl http://localhost:8001/categories
```

### Metrics
The integration exposes Prometheus metrics:
- `privacy_filter_requests_total`
- `privacy_filter_duration_seconds`
- `privacy_filter_errors_total`
- `privacy_filter_categories_detected`

### Logging
Set log level via `PRIVACY_FILTER_LOG_LEVEL`:
- `debug`: Detailed processing information
- `info`: Normal operation logs
- `warn`: Warnings only
- `error`: Errors only

## Security Considerations

### Data Flow
1. Log text is sent to privacy filter service
2. PII is detected and replaced locally
3. Only filtered text is stored/processed
4. Original text with PII is never persisted

### Network Security
- Use HTTPS for production deployments
- Consider running privacy filter on same host/network
- Implement authentication if service is exposed

### Model Security
- The 1.5B parameter model runs locally
- No data sent to external OpenAI servers
- Apache 2.0 licensed - can be audited

## Troubleshooting

### Common Issues

1. **Service Unavailable**
   ```
   Error: connection refused
   Fix: Check the configured privacy-service URL and its service-manager logs.
   ```

2. **Slow Performance**
   ```
   Fix: Enable GPU support or increase batch size
   ```

3. **Incorrect Filtering**
   ```
   Fix: Check configured categories and test with /filter endpoint
   ```

4. **Memory Issues**
   ```
   Fix: The model weights are ~3 GB BF16; CPU inference needs headroom
   above that. Reduce batch size or use GPU.
   ```

### Debug Mode
Enable debug logging:
```bash
PRIVACY_FILTER_LOG_LEVEL=debug ./server
```

## References

- [OpenAI Privacy Filter GitHub](https://github.com/openai/privacy-filter)
- [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0)
- [Dark Pawns Documentation](../README.md)

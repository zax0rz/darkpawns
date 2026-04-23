# Privacy Filter Integration for Dark Pawns

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

### Option 1: Docker Compose (Recommended)

```bash
# Start with privacy filter
docker-compose -f docker-compose.yml -f docker-compose.privacy.yml up -d

# Or use the combined command
make up-with-privacy
```

### Option 2: Manual Setup

1. **Start Privacy Filter Service:**
```bash
cd deployment
docker build -f ../Dockerfile.privacy-filter -t privacy-filter .
docker run -p 8001:8000 --gpus all privacy-filter
```

2. **Configure Dark Pawns Server:**
```bash
cp .env.privacy.example .env.privacy
# Edit .env.privacy with your settings
```

3. **Start Server with Privacy Filter:**
```bash
PRIVACY_FILTER_URL=http://localhost:8001 ./server
```

## Configuration

### Environment Variables

Create a `.env.privacy` file or set environment variables:

```bash
# Privacy Filter Service
PRIVACY_FILTER_URL=http://privacy-filter:8000
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

http.ListenAndServe(":8080", handler)
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

1. **Basic Masking** (default): Simple regex patterns for common PII
2. **Pass Through**: Log everything (not recommended for production)
3. **Error**: Fail the log operation

## Testing

Run the integration tests:

```bash
# Unit tests
cd pkg/privacy
go test -v

# Integration test (requires privacy filter service)
PRIVACY_FILTER_URL=http://localhost:8001 go test -v -tags=integration
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
   Fix: Check if privacy filter service is running: docker ps | grep privacy-filter
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
   Fix: The model requires ~6GB RAM. Reduce batch size or use GPU.
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
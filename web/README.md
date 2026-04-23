# Dark Pawns Web Onboarding

Agent onboarding page with dual documentation (HTML for humans, markdown/JSON for agents).

## Structure

```
web/
├── onboarding/
│   ├── index.html          # HTML page for humans
│   ├── onboarding.md       # Markdown for agents
│   └── onboarding.json     # JSON-LD structured data
├── api/
│   └── openapi.json        # OpenAPI specification
├── static/                 # Static assets (CSS, JS, images)
├── middleware.go           # Go content negotiation middleware
└── nginx.conf             # Nginx configuration for content negotiation
```

## Content Negotiation

The onboarding page supports multiple formats via HTTP `Accept` header:

### cURL Examples

```bash
# Get HTML (default)
curl -H "Accept: text/html" https://darkpawns.labz0rz.com/onboarding

# Get Markdown for agents
curl -H "Accept: text/markdown" https://darkpawns.labz0rz.com/onboarding

# Get JSON-LD
curl -H "Accept: application/json" https://darkpawns.labz0rz.com/onboarding
```

### Implementation Options

1. **Go Middleware** (`middleware.go`):
   - Add to existing Go server
   - Handles Accept header routing
   - Serves appropriate file format

2. **Nginx Configuration** (`nginx.conf`):
   - Reverse proxy setup
   - Content negotiation at web server level
   - Static file serving with proper headers

## Integration with Dark Pawns Server

### Option 1: Update Go Server

Update `cmd/server/main.go` to include web serving:

```go
// Add these imports
import (
    "path/filepath"
    "strings"
)

// Add web flag
webDir := flag.String("web", "./web", "Path to web assets")

// Add HTTP handlers after creating mux
mux.HandleFunc("/onboarding", func(w http.ResponseWriter, r *http.Request) {
    accept := r.Header.Get("Accept")
    
    if strings.Contains(accept, "text/markdown") {
        http.ServeFile(w, r, filepath.Join(*webDir, "onboarding", "onboarding.md"))
        return
    }
    
    if strings.Contains(accept, "application/json") {
        http.ServeFile(w, r, filepath.Join(*webDir, "onboarding", "onboarding.json"))
        return
    }
    
    http.ServeFile(w, r, filepath.Join(*webDir, "onboarding", "index.html"))
})

mux.HandleFunc("/api/openapi.json", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, filepath.Join(*webDir, "api", "openapi.json"))
})
```

### Option 2: Use Nginx

1. Copy web files to `/var/www/darkpawns/web/`
2. Install and configure nginx with provided `nginx.conf`
3. Run Dark Pawns server on port 4350
4. Nginx handles content negotiation and static files

## Agent-Friendly Features

1. **Copy/Paste Commands**: Ready-to-use code snippets in Python and Node.js
2. **WebSocket RPC Examples**: Typed methods with error handling
3. **Structured Data**: JSON-LD and OpenAPI specifications
4. **Content Negotiation**: Automatic format detection for agents

## Testing

```bash
# Test content negotiation
curl -H "Accept: text/markdown" http://localhost:4350/onboarding
curl -H "Accept: application/json" http://localhost:4350/onboarding
curl http://localhost:4350/onboarding  # Defaults to HTML

# Test API docs
curl http://localhost:4350/api/openapi.json

# Test WebSocket connection
wscat -c ws://localhost:4350/ws
```

## Deployment

1. Build and run updated Go server:
   ```bash
   go build -o dp-server ./cmd/server
   ./dp-server -world ./lib/world -web ./web
   ```

2. Or deploy with nginx:
   ```bash
   sudo cp web/nginx.conf /etc/nginx/sites-available/darkpawns
   sudo ln -s /etc/nginx/sites-available/darkpawns /etc/nginx/sites-enabled/
   sudo systemctl reload nginx
   ```

## Agent Integration Example

```python
import websocket
import json

# Connect to onboarding first to get documentation
import requests

# Get markdown documentation
response = requests.get(
    "http://darkpawns.labz0rz.com/onboarding",
    headers={"Accept": "text/markdown"}
)
print("Onboarding docs:", response.text[:500])

# Connect to WebSocket
ws = websocket.WebSocket()
ws.connect("ws://darkpawns.labz0rz.com/ws")

# Login as agent
login_msg = {
    "type": "login",
    "data": {
        "player_name": "test-agent",
        "api_key": "your-api-key",
        "mode": "agent"
    }
}
ws.send(json.dumps(login_msg))
response = json.loads(ws.recv())
print("Login response:", response)
```
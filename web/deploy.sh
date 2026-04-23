#!/bin/bash
# Deployment script for Dark Pawns web onboarding

set -e

echo "🚀 Deploying Dark Pawns web onboarding..."

# Check if we're in the right directory
if [ ! -f "web/README.md" ]; then
    echo "❌ Error: Run this script from the darkpawns_repo root directory"
    exit 1
fi

# Parse arguments
DEPLOY_MODE=${1:-"local"}
SERVER_URL=${2:-"http://localhost:4350"}

echo "📦 Deployment mode: $DEPLOY_MODE"
echo "🌐 Server URL: $SERVER_URL"

case $DEPLOY_MODE in
    "local")
        echo "🔧 Setting up local development..."
        
        # Test if server is running
        if curl -s "$SERVER_URL/health" > /dev/null; then
            echo "✅ Server is running at $SERVER_URL"
        else
            echo "⚠️  Server not running at $SERVER_URL"
            echo "   Start server with: go run ./cmd/server -world ./lib/world"
        fi
        
        # Test content negotiation
        echo "🧪 Testing content negotiation..."
        python3 web/test_onboarding.py test "$SERVER_URL"
        
        echo ""
        echo "✅ Local setup complete!"
        echo "   Onboarding: $SERVER_URL/onboarding"
        echo "   API docs: $SERVER_URL/api/openapi.json"
        echo "   Health: $SERVER_URL/health"
        ;;
        
    "nginx")
        echo "🔧 Setting up nginx deployment..."
        
        # Check for nginx
        if ! command -v nginx &> /dev/null; then
            echo "❌ nginx not found. Install with: sudo apt install nginx"
            exit 1
        fi
        
        # Create web directory
        sudo mkdir -p /var/www/darkpawns
        sudo cp -r web/* /var/www/darkpawns/web/
        sudo chown -R www-data:www-data /var/www/darkpawns
        
        # Copy nginx config
        sudo cp web/nginx.conf /etc/nginx/sites-available/darkpawns
        sudo ln -sf /etc/nginx/sites-available/darkpawns /etc/nginx/sites-enabled/
        
        # Test nginx config
        sudo nginx -t
        
        echo ""
        echo "📋 Nginx configuration complete. Next steps:"
        echo "1. Update server_name in web/nginx.conf if needed"
        echo "2. Set up SSL certificates for HTTPS"
        echo "3. Reload nginx: sudo systemctl reload nginx"
        echo "4. Start Dark Pawns server: ./dp-server -world ./lib/world"
        ;;
        
    "docker")
        echo "🐳 Setting up Docker deployment..."
        
        # Check for docker
        if ! command -v docker &> /dev/null; then
            echo "❌ Docker not found. Install Docker first."
            exit 1
        fi
        
        # Create Dockerfile for web server
        cat > web/Dockerfile << 'EOF'
FROM nginx:alpine

COPY . /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
EOF
        
        # Build and run
        cd web
        docker build -t darkpawns-web .
        
        echo ""
        echo "📋 Docker setup complete. Run with:"
        echo "  docker run -p 8080:80 darkpawns-web"
        echo ""
        echo "⚠️  Note: This only serves the web UI. You still need to run"
        echo "   the Dark Pawns game server separately."
        ;;
        
    "test")
        echo "🧪 Running tests..."
        
        # Test all content types
        echo "Testing HTML response..."
        curl -s -H "Accept: text/html" "$SERVER_URL/onboarding" | grep -q "<html>" && echo "✅ HTML OK" || echo "❌ HTML failed"
        
        echo "Testing Markdown response..."
        curl -s -H "Accept: text/markdown" "$SERVER_URL/onboarding" | grep -q "^#" && echo "✅ Markdown OK" || echo "❌ Markdown failed"
        
        echo "Testing JSON response..."
        curl -s -H "Accept: application/json" "$SERVER_URL/onboarding" | python3 -m json.tool > /dev/null 2>&1 && echo "✅ JSON OK" || echo "❌ JSON failed"
        
        echo "Testing OpenAPI spec..."
        curl -s "$SERVER_URL/api/openapi.json" | python3 -m json.tool > /dev/null 2>&1 && echo "✅ OpenAPI OK" || echo "❌ OpenAPI failed"
        
        echo "Testing health endpoint..."
        curl -s "$SERVER_URL/health" | grep -q "OK" && echo "✅ Health OK" || echo "❌ Health failed"
        
        echo ""
        echo "✅ All tests completed!"
        ;;
        
    "update-server")
        echo "🔄 Updating Go server with web support..."
        
        # Check if main_web.go exists
        if [ ! -f "cmd/server/main_web.go" ]; then
            echo "❌ main_web.go not found. Creating from template..."
            
            # Create backup of original main.go
            cp cmd/server/main.go cmd/server/main.go.backup
            
            # Create new main.go with web support
            cat > cmd/server/main.go << 'EOF'
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
	"github.com/zax0rz/darkpawns/pkg/session"
)

func main() {
	var (
		worldDir   = flag.String("world", "", "Path to world files (lib directory)")
		scriptsDir = flag.String("scripts", "", "Path to Lua scripts (defaults to world/lib/scripts)")
		port       = flag.String("port", "4350", "Server port")
		dbURL      = flag.String("db", "postgres://postgres:postgres@localhost/darkpawns?sslmode=disable", "Database URL")
		webDir     = flag.String("web", "./web", "Path to web assets")
	)
	flag.Parse()

	if *worldDir == "" {
		log.Fatal("Usage: server -world <path-to-lib>")
	}

	log.Println("Dark Pawns Phase 1 Server Starting...")

	// Parse world files
	log.Printf("Loading world from %s...", *worldDir)
	parsedWorld, err := parser.ParseWorld(*worldDir)
	if err != nil {
		log.Fatalf("Failed to parse world: %v", err)
	}
	log.Println(parsedWorld.Stats())

	// Create game world
	gameWorld, err := game.NewWorld(parsedWorld)
	if err != nil {
		log.Fatalf("Failed to create game world: %v", err)
	}

	// Initialize scripting engine
	if *scriptsDir == "" {
		*scriptsDir = *worldDir + "/scripts"
	}
	log.Printf("Loading scripts from %s...", *scriptsDir)
	worldAdapter := game.NewWorldScriptableAdapter(gameWorld)
	scriptEngine := scripting.NewEngine(*scriptsDir, worldAdapter)
	game.ScriptEngine = scriptEngine

	// Connect to database
	log.Println("Connecting to database...")
	database, err := db.New(*dbURL)
	if err != nil {
		log.Printf("Warning: Database connection failed: %v", err)
		log.Println("Continuing without persistence...")
		database = nil
	} else {
		defer database.Close()
		log.Println("Database connected.")
	}

	// Create session manager
	manager := session.NewManager(gameWorld, database)
	manager.SetCombatBroadcastFunc()                  // Enable combat messages to rooms
	manager.SetDeathFunc()                            // Enable death/respawn handling
	manager.RegisterMemoryHooks()                     // Enable narrative memory writes on kill/death
	manager.SetDamageFunc()                           // Enable HEALTH dirty-tracking for agents
	manager.SetScriptFightFunc()                      // Enable mob fight scripts after each combat round
	game.SetAICombatEngine(manager.GetCombatEngine()) // Enable AI to use combat

	// Create HTTP mux
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", manager.HandleWebSocket)

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK\n"))
	})

	// API endpoints
	mux.HandleFunc("/api/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(*webDir, "api", "openapi.json"))
	})

	// Onboarding with content negotiation
	mux.HandleFunc("/onboarding", func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		
		// Check for markdown request
		if strings.Contains(accept, "text/markdown") {
			http.ServeFile(w, r, filepath.Join(*webDir, "onboarding", "onboarding.md"))
			return
		}
		
		// Check for JSON request
		if strings.Contains(accept, "application/json") {
			http.ServeFile(w, r, filepath.Join(*webDir, "onboarding", "onboarding.json"))
			return
		}
		
		// Default to HTML
		http.ServeFile(w, r, filepath.Join(*webDir, "onboarding", "index.html"))
	})

	// Serve static files
	fs := http.FileServer(http.Dir(filepath.Join(*webDir, "static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Default handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Redirect to onboarding for root
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/onboarding", http.StatusFound)
			return
		}
		
		w.Write([]byte(`Dark Pawns Phase 1 Server

Endpoints:
  /ws           - WebSocket game connection
  /onboarding   - Agent onboarding (HTML/Markdown/JSON)
  /api/openapi.json - OpenAPI specification
  /health       - Health check

Connect via WebSocket: ws://` + r.Host + `/ws

Protocol:
  {"type":"login","data":{"player_name":"YourName"}}
  {"type":"command","data":{"command":"look"}}
  {"type":"command","data":{"command":"north"}}
  {"type":"command","data":{"command":"say","args":["hello"]}}
`))
	})

	// Start zone resets in background (initial + periodic every 60s)
	go func() {
		log.Printf("Starting zone resets...")
		if err := gameWorld.StartZoneResets(); err != nil {
			log.Printf("Zone reset error: %v", err)
		} else {
			log.Printf("Zone resets complete")
		}
		gameWorld.StartPeriodicResets(60 * time.Second)
	}()

	// Start server
	addr := ":" + *port
	log.Printf("Server listening on %s", addr)
	log.Printf("WebSocket endpoint: ws://localhost%s/ws", addr)
	log.Printf("Onboarding: http://localhost%s/onboarding", addr)
	log.Printf("API docs: http://localhost%s/api/openapi.json", addr)

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down...")
}
EOF
            echo "✅ Created new main.go with web support"
        else
            echo "✅ main_web.go already exists. Copying to main.go..."
            cp cmd/server/main_web.go cmd/server/main.go
        fi
        
        # Build the server
        echo "🔨 Building server..."
        go build -o dp-server ./cmd/server
        
        echo ""
        echo "✅ Server updated with web support!"
        echo "   Run with: ./dp-server -world ./lib/world -web ./web"
        ;;
        
    *)
        echo "❌ Unknown deployment mode: $DEPLOY_MODE"
        echo ""
        echo "Available modes:"
        echo "  local       - Local development setup"
        echo "  nginx       - Nginx reverse proxy setup"
        echo "  docker      - Docker container setup"
        echo "  test        - Run tests"
        echo "  update-server - Update Go server with web support"
        exit 1
        ;;
esac

echo ""
echo "🎉 Deployment script complete!"
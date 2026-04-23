package privacy

// IntegrationExample shows how to integrate privacy filter with Dark Pawns server
// This is an example - actual integration would be in the server main.go

/*
Example integration with Dark Pawns server:

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	
	"github.com/zax0rz/darkpawns/pkg/privacy"
	"github.com/zax0rz/darkpawns/pkg/session"
)

func main() {
	// Load privacy filter configuration
	privacyConfig := privacy.LoadConfig()
	
	// Create privacy filter client
	var privacyClient *privacy.Client
	if privacyConfig.Enabled {
		filterConfig := privacyConfig.ToFilterConfig()
		privacyClient = privacy.NewClient(privacyConfig.URL, filterConfig)
		
		// Test connection
		if _, _, err := privacyClient.FilterText("test"); err != nil {
			log.Printf("Warning: Privacy filter unavailable: %v", err)
			log.Println("Continuing with fallback filtering...")
		} else {
			log.Println("Privacy filter connected successfully")
		}
	} else {
		log.Println("Privacy filter disabled")
		privacyClient = privacy.NewClient("disabled", privacy.DefaultFilterConfig())
	}
	
	// Set up global logger with privacy filter
	privacyLogger := privacy.NewPrivacyLogger(privacyClient, "[DARKPAWNS] ", log.LstdFlags)
	
	// Replace standard log with privacy-aware logger
	log.SetOutput(privacyLogger)
	
	// Create session manager with privacy-aware logging
	manager := session.NewManager(gameWorld, database)
	
	// Wrap HTTP handler with privacy middleware
	handler := privacy.HTTPMiddleware(
		http.HandlerFunc(manager.HandleWebSocket),
		privacyClient,
	)
	
	http.HandleFunc("/ws", handler)
	
	// Use privacy-aware logging throughout
	privacy.Printf("Server starting on port %s", *port)
	
	// Example: Log player actions with PII filtering
	logPlayerAction := func(playerName, action, details string) {
		message := fmt.Sprintf("Player %s %s: %s", playerName, action, details)
		privacy.Println(message)
	}
	
	// In your game logic:
	logPlayerAction("John Doe", "logged in", "from IP 192.168.1.100")
	// Output: Player [REDACTED] logged in: from IP 192.168.1.100
}

// WebSocket handler integration example
type PrivacyAwareSession struct {
	wsLogger *privacy.WebSocketLogger
	sessionID string
}

func NewPrivacyAwareSession(sessionID string, client *privacy.Client) *PrivacyAwareSession {
	return &PrivacyAwareSession{
		wsLogger: privacy.NewWebSocketLogger(client, "[SESSION] "),
		sessionID: sessionID,
	}
}

func (s *PrivacyAwareSession) HandleMessage(message string) {
	// Log incoming message with PII filtering
	s.wsLogger.LogIncoming(s.sessionID, message)
	
	// Process message...
	
	// Log outgoing response
	response := "Welcome to Dark Pawns!"
	s.wsLogger.LogOutgoing(s.sessionID, response)
}

// Database logging integration
type PrivacyAwareDatabase struct {
	db      *sql.DB
	client  *privacy.Client
}

func (padb *PrivacyAwareDatabase) LogQuery(query string, args ...interface{}) {
	// Filter any PII in query arguments
	filteredQuery := query
	for _, arg := range args {
		if str, ok := arg.(string); ok {
			filtered, _, _ := padb.client.FilterText(str)
			// Replace in query log (simplified example)
			filteredQuery = strings.ReplaceAll(filteredQuery, str, filtered)
		}
	}
	
	log.Printf("DB Query: %s", filteredQuery)
}

// Combat logging example
func LogCombatWithPrivacy(client *privacy.Client, attacker, defender string, damage int) {
	// Only filter player names if configured
	config := privacy.LoadConfig()
	
	message := fmt.Sprintf("%s attacks %s for %d damage", attacker, defender, damage)
	
	if config.FilterPlayerNames {
		filtered, _, _ := client.FilterText(message)
		privacy.Println(filtered)
	} else {
		privacy.Println(message)
	}
}
*/
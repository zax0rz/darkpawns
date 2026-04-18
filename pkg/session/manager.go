// Package session manages WebSocket connections and player sessions.
package session

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Manager handles all active sessions.
type Manager struct {
	mu            sync.RWMutex
	sessions      map[string]*Session // keyed by player name
	world         *game.World
	combatEngine  *combat.CombatEngine
}

// NewManager creates a new session manager.
func NewManager(world *game.World) *Manager {
	ce := combat.NewCombatEngine()
	
	// Set up broadcast function for combat messages
	// This will be set properly after manager is created via a setter
	
	ce.Start()
	
	return &Manager{
		sessions:     make(map[string]*Session),
		world:        world,
		combatEngine: ce,
	}
}

// SetCombatBroadcastFunc sets the broadcast function for combat messages.
// Must be called after the manager is created and before combat starts.
func (m *Manager) SetCombatBroadcastFunc() {
	m.combatEngine.SetBroadcastFunc(func(roomVNum int, message string, exclude string) {
		msg, _ := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "combat",
				Text: message,
			},
		})
		m.BroadcastToRoom(roomVNum, msg, exclude)
	})
}

// GetCombatEngine returns the combat engine for AI integration.
func (m *Manager) GetCombatEngine() *combat.CombatEngine {
	return m.combatEngine
}

// HandleWebSocket upgrades HTTP to WebSocket and manages the session.
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	session := &Session{
		conn:    conn,
		manager: m,
		send:    make(chan []byte, 256),
	}

	// Start goroutines for reading and writing
	go session.writePump()
	go session.readPump()
}

// Register adds a session for a player.
func (m *Manager) Register(playerName string, s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[playerName]; exists {
		return ErrPlayerAlreadyOnline
	}

	m.sessions[playerName] = s
	s.playerName = playerName
	return nil
}

// Unregister removes a session.
func (m *Manager) Unregister(playerName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[playerName]; ok {
		delete(m.sessions, playerName)
		close(s.send)
	}
}

// GetSession returns a session by player name.
func (m *Manager) GetSession(playerName string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[playerName]
	return s, ok
}

// BroadcastToRoom sends a message to all players in a room.
func (m *Manager) BroadcastToRoom(roomVNum int, message []byte, excludePlayer string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, s := range m.sessions {
		if name == excludePlayer {
			continue
		}
		if s.player != nil && s.player.GetRoom() == roomVNum {
			select {
			case s.send <- message:
			default:
				// Channel full, drop message
			}
		}
	}
}

// Session represents a single WebSocket connection.
type Session struct {
	conn       *websocket.Conn
	manager    *Manager
	send       chan []byte
	player     *game.Player
	playerName string
	authenticated bool
}

// readPump reads messages from the WebSocket.
func (s *Session) readPump() {
	defer func() {
		s.manager.Unregister(s.playerName)
		s.conn.Close()
	}()

	s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.conn.SetPongHandler(func(string) error {
		s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if err := s.handleMessage(message); err != nil {
			log.Printf("Handle message error: %v", err)
			s.sendError(err.Error())
		}
	}
}

// writePump writes messages to the WebSocket.
func (s *Session) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		s.conn.Close()
	}()

	for {
		select {
		case message, ok := <-s.send:
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			s.conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages.
func (s *Session) handleMessage(data []byte) error {
	var msg ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	switch msg.Type {
	case MsgLogin:
		return s.handleLogin(msg.Data)
	case MsgCommand:
		if !s.authenticated {
			return ErrNotAuthenticated
		}
		return s.handleCommand(msg.Data)
	default:
		return ErrUnknownMessageType
	}
}

// handleLogin authenticates a player.
func (s *Session) handleLogin(data json.RawMessage) error {
	var login LoginData
	if err := json.Unmarshal(data, &login); err != nil {
		return err
	}

	if login.PlayerName == "" {
		return ErrInvalidPlayerName
	}

	// TODO: Check password against database
	// For Phase 1: auto-create/login player

	s.player = game.NewPlayer(0, login.PlayerName, 3001) // Start in room 3001
	s.authenticated = true

	if err := s.manager.Register(login.PlayerName, s); err != nil {
		return err
	}

	if err := s.manager.world.AddPlayer(s.player); err != nil {
		s.manager.Unregister(login.PlayerName)
		return err
	}

	// Send welcome
	s.sendWelcome()

	// Broadcast to room
	enterMsg, _ := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "enter",
			Text: s.player.Name + " has arrived.",
		},
	})
	s.manager.BroadcastToRoom(s.player.GetRoom(), enterMsg, s.player.Name)

	return nil
}

// handleCommand processes game commands.
func (s *Session) handleCommand(data json.RawMessage) error {
	var cmd CommandData
	if err := json.Unmarshal(data, &cmd); err != nil {
		return err
	}

	return ExecuteCommand(s, cmd.Command, cmd.Args)
}

// sendWelcome sends the initial game state to the player.
func (s *Session) sendWelcome() {
	room, _ := s.manager.world.GetRoom(s.player.GetRoom())

	state := StateData{
		Player: PlayerState{
			Name:      s.player.Name,
			Health:    s.player.Health,
			MaxHealth: s.player.MaxHealth,
			Level:     s.player.Level,
		},
		Room: RoomState{
			VNum:        room.VNum,
			Name:        room.Name,
			Description: room.Description,
			Exits:       getExitNames(room.Exits),
		},
	}

	msg, _ := json.Marshal(ServerMessage{
		Type: MsgState,
		Data: state,
	})
	s.send <- msg
}

// sendError sends an error message to the player.
func (s *Session) sendError(text string) {
	msg, _ := json.Marshal(ServerMessage{
		Type: MsgError,
		Data: ErrorData{Message: text},
	})
	select {
	case s.send <- msg:
	default:
	}
}

func getExitNames(exits map[string]parser.Exit) []string {
	var names []string
	for dir := range exits {
		names = append(names, dir)
	}
	return names
}

// Errors
var (
	ErrPlayerAlreadyOnline = fmt.Errorf("player already online")
	ErrNotAuthenticated    = fmt.Errorf("not authenticated")
	ErrUnknownMessageType  = fmt.Errorf("unknown message type")
	ErrInvalidPlayerName   = fmt.Errorf("invalid player name")
)
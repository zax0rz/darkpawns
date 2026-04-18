package session

import "encoding/json"

// Client to Server message types
const (
	MsgLogin    = "login"
	MsgCommand  = "command"
)

// Server to Client message types
const (
	MsgState    = "state"
	MsgEvent    = "event"
	MsgError    = "error"
	MsgText     = "text"
)

// ClientMessage is a message from client to server.
type ClientMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// ServerMessage is a message from server to client.
type ServerMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// LoginData is sent to authenticate.
type LoginData struct {
	PlayerName string `json:"player_name"`
	Password   string `json:"password,omitempty"` // For Phase 1, optional
}

// CommandData is a player command.
type CommandData struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

// StateData represents the game state sent to client.
type StateData struct {
	Player PlayerState `json:"player"`
	Room   RoomState   `json:"room"`
}

// PlayerState represents player info in state.
type PlayerState struct {
	Name      string `json:"name"`
	Health    int    `json:"health"`
	MaxHealth int    `json:"max_health"`
	Level     int    `json:"level"`
}

// RoomState represents room info in state.
type RoomState struct {
	VNum        int      `json:"vnum"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Exits       []string `json:"exits"`
	Players     []string `json:"players,omitempty"`
}

// EventData represents a game event.
type EventData struct {
	Type string `json:"type"`   // "enter", "leave", "say", "combat"
	From string `json:"from,omitempty"`
	Text string `json:"text"`
}

// ErrorData represents an error message.
type ErrorData struct {
	Message string `json:"message"`
}

// TextData is a simple text message.
type TextData struct {
	Text string `json:"text"`
}
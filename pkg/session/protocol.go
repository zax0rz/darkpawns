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
// For new characters, include class and race.
// Class: 0=Mage 1=Cleric 2=Thief 3=Warrior 4=Magus 5=Avatar 6=Assassin 7=Paladin 8=Ninja 9=Psionic 10=Ranger 11=Mystic
// Race:  0=Human 1=Elf 2=Dwarf 3=Kender 4=Minotaur
type LoginData struct {
	PlayerName string `json:"player_name"`
	Password   string `json:"password,omitempty"`
	Class      int    `json:"class,omitempty"` // 0-11, defaults to Warrior if omitted
	Race       int    `json:"race,omitempty"`  // 0-4, defaults to Human if omitted
	NewChar    bool   `json:"new_char,omitempty"` // true = create new character
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
	Class     string `json:"class,omitempty"`
	Race      string `json:"race,omitempty"`
	Str       int    `json:"str,omitempty"`
	Int       int    `json:"int,omitempty"`
	Wis       int    `json:"wis,omitempty"`
	Dex       int    `json:"dex,omitempty"`
	Con       int    `json:"con,omitempty"`
	Cha       int    `json:"cha,omitempty"`
}

// RoomState represents room info in state.
type RoomState struct {
	VNum        int      `json:"vnum"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Exits       []string `json:"exits"`
	Players     []string `json:"players,omitempty"`
	Items       []string `json:"items,omitempty"`
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
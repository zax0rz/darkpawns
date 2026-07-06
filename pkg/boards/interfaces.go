package boards

// BoardWorld is the minimal world surface required by the board system.
type BoardWorld interface {
	RoomEcho(roomVNum int, message string, excludeName string)
}

// BoardPlayer is the minimal player surface required by the board system.
type BoardPlayer interface {
	GetLevel() int
	GetName() string
	SendMessage(msg string)
	GetRoomVNum() int
}

package game

import "errors"

// Sentinel errors for game operations.
var (
	ErrInventoryFull     = errors.New("inventory full")
	ErrObjectNotFound    = errors.New("object not found")
	ErrInvalidLocation   = errors.New("invalid object location")
	ErrEquipSlotOccupied = errors.New("equipment slot occupied")
)

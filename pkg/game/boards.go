// boards.go — Bulletin board system (ported from boards.c)
//
// Boards are object special procedures (gen_board) attached to board objects.
// When a player in a room with a board types READ/WRITE/REMOVE/LOOK at the
// board, the gen_board spec fires and handles the command.
//
// NOTE: Currently no command dispatcher calls GetObjSpec or GetRoomSpec.
// Boards will work once spec procs are wired into the command pipeline.

package game

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Board constants matching boards.h
const (
	NumBoards        = 12
	MaxBoardMessages = 60
	MaxMessageLength = 4096
	BoardMagic       = 1048575
)

// BoardMsgInfo describes one message on a board.
type BoardMsgInfo struct {
	SlotNum    int    `json:"slot"`    // index into msgStorage
	Heading    string `json:"heading"` // header line
	Level      int    `json:"level"`   // level of poster
	HeadingLen int    `json:"-"`       // serialization helper
	MessageLen int    `json:"-"`       // serialization helper
}

// BoardInfo describes one board definition.
type BoardInfo struct {
	VNum      int    `json:"vnum"`       // vnum of the board object
	ReadLvl   int    `json:"read_lvl"`   // min level to read
	WriteLvl  int    `json:"write_lvl"`  // min level to write
	RemoveLvl int    `json:"remove_lvl"` // min level to remove
	Filename  string `json:"filename"`   // save file path
	RNUM      int    `json:"-"`          // runtime number (resolved later)
}

// BoardSystem holds the global state for all bulletin boards.
type BoardSystem struct {
	mu              sync.RWMutex
	boards          []BoardInfo
	msgIndex        [NumBoards][MaxBoardMessages]BoardMsgInfo
	numOfMsgs       [NumBoards]int
	msgStorage      [NumBoards*MaxBoardMessages + 5]string
	msgStorageTaken [NumBoards*MaxBoardMessages + 5]bool
	loaded          bool
	BasePath        string // directory for board save files

	// World reference for room-level echoes (actToRoom, broadcast).
	// Set via SetWorld() after BoardSystem construction.
	world *World
}

// SetWorld attaches a World reference to the BoardSystem, enabling room
// echoes and broadcasts for board operations (read, write, remove).
func (bs *BoardSystem) SetWorld(w *World) {
	bs.world = w
}

var defaultBoardInfo = []BoardInfo{
	{VNum: 8099, ReadLvl: 0, WriteLvl: 0, RemoveLvl: 50, Filename: "etc/board.mort"},
	{VNum: 8064, ReadLvl: 0, WriteLvl: 0, RemoveLvl: 50, Filename: "etc/board.customs"},
	{VNum: 8065, ReadLvl: 0, WriteLvl: 0, RemoveLvl: 50, Filename: "etc/board/chosen"},
	{VNum: 8098, ReadLvl: 50, WriteLvl: 50, RemoveLvl: 61, Filename: "etc/board.immort"},
	{VNum: 8096, ReadLvl: 50, WriteLvl: 50, RemoveLvl: 61, Filename: "etc/board.social"},
	{VNum: 8097, ReadLvl: 50, WriteLvl: 50, RemoveLvl: 60, Filename: "etc/board.freeze"},
	{VNum: 19652, ReadLvl: 0, WriteLvl: 0, RemoveLvl: 0, Filename: "etc/board.trinity"},
	{VNum: 19601, ReadLvl: 0, WriteLvl: 0, RemoveLvl: 0, Filename: "etc/board.neosunz"},
	{VNum: 19627, ReadLvl: 0, WriteLvl: 0, RemoveLvl: 0, Filename: "etc/board.arithrix"},
	{VNum: 19666, ReadLvl: 0, WriteLvl: 0, RemoveLvl: 0, Filename: "etc/board.silent_shadows"},
	{VNum: 19677, ReadLvl: 0, WriteLvl: 0, RemoveLvl: 0, Filename: "etc/board.domination"},
	{VNum: 19640, ReadLvl: 0, WriteLvl: 0, RemoveLvl: 0, Filename: "etc/board.tripnosis"},
}

// InitBoards creates and initializes the board system.
func InitBoards(basePath string) *BoardSystem {
	bs := &BoardSystem{
		boards:   make([]BoardInfo, NumBoards),
		BasePath: basePath,
	}
	copy(bs.boards, defaultBoardInfo)
	bs.load()
	bs.loaded = true
	return bs
}

// load reads all board data files.
func (bs *BoardSystem) load() {
	for i := 0; i < NumBoards; i++ {
		bs.loadBoard(i)
	}
}

// loadBoard reads one board's data from its save file.
func (bs *BoardSystem) loadBoard(boardType int) {
	if boardType < 0 || boardType >= NumBoards {
		return
	}
	path := filepath.Join(bs.BasePath, bs.boards[boardType].Filename)
// #nosec G304
	f, err := os.Open(path)
	if err != nil {
		return // file doesn't exist yet = empty board
	}
	defer f.Close()

	var num int32
	if err := binary.Read(f, binary.LittleEndian, &num); err != nil {
		bs.resetBoard(boardType)
		return
	}
	if num < 1 || num > MaxBoardMessages {
		bs.resetBoard(boardType)
		return
	}
	bs.numOfMsgs[boardType] = int(num)

	for i := 0; i < int(num); i++ {
		var info struct {
			SlotNum    int32
			_          [4]byte // padding for pointer (heading) — not serialized
			Level      int32
			HeadingLen int32
			MessageLen int32
		}
		if err := binary.Read(f, binary.LittleEndian, &info); err != nil {
			bs.resetBoard(boardType)
			return
		}
		if info.HeadingLen <= 0 {
			bs.resetBoard(boardType)
			return
		}

		heading := make([]byte, info.HeadingLen)
		if _, err := f.Read(heading); err != nil {
			bs.resetBoard(boardType)
			return
		}

		bs.msgIndex[boardType][i] = BoardMsgInfo{
			SlotNum:    int(info.SlotNum),
			Heading:    string(heading[:info.HeadingLen-1]), // strip null
			Level:      int(info.Level),
			HeadingLen: int(info.HeadingLen),
			MessageLen: int(info.MessageLen),
		}

		if info.MessageLen > 0 {
			slot := int(info.SlotNum)
			if slot >= 0 && slot < len(bs.msgStorage) {
				msgBytes := make([]byte, info.MessageLen)
				if _, err := f.Read(msgBytes); err != nil {
					continue
				}
				bs.msgStorage[slot] = string(msgBytes[:info.MessageLen-1])
				bs.msgStorageTaken[slot] = true
			}
		}

		if info.HeadingLen > 0 && int(info.SlotNum) >= 0 && int(info.SlotNum) < len(bs.msgStorage) {
			bs.msgStorageTaken[info.SlotNum] = true
		}
	}
}

// saveBoard writes one board's data to its save file.
func (bs *BoardSystem) saveBoard(boardType int) {
	if boardType < 0 || boardType >= NumBoards {
		return
	}
	bs.mu.RLock()
	num := bs.numOfMsgs[boardType]
	if num == 0 {
		bs.mu.RUnlock()
		path := filepath.Join(bs.BasePath, bs.boards[boardType].Filename)
// #nosec G104
		os.Remove(path)
		return
	}
	bs.mu.RUnlock()

	path := filepath.Join(bs.BasePath, bs.boards[boardType].Filename)
// #nosec G104
	os.MkdirAll(filepath.Dir(path), 0750)

// #nosec G304
	f, err := os.Create(path)
	if err != nil {
		BasicMudLogf("SYSERR: Board save failed: %v", err)
		return
	}
	defer f.Close()

	bs.mu.RLock()
	defer bs.mu.RUnlock()

// #nosec G115
	if err := binary.Write(f, binary.LittleEndian, int32(num)); err != nil {
		return
	}

	for i := 0; i < num; i++ {
		mi := bs.msgIndex[boardType][i]
// #nosec G115
		headingLen := int32(len(mi.Heading) + 1)
		var messageLen int32
		var msgStr string
		if mi.SlotNum >= 0 && mi.SlotNum < len(bs.msgStorage) {
			msgStr = bs.msgStorage[mi.SlotNum]
// #nosec G115
			messageLen = int32(len(msgStr) + 1)
		}

		// Write binary header matching C struct layout
// #nosec G115
		if err := binary.Write(f, binary.LittleEndian, int32(mi.SlotNum)); err != nil {
			return
		}
		// Skip the heading pointer (4 bytes padding)
		if err := binary.Write(f, binary.LittleEndian, int32(0)); err != nil {
			return
		}
// #nosec G115
		if err := binary.Write(f, binary.LittleEndian, int32(mi.Level)); err != nil {
			return
		}
		if err := binary.Write(f, binary.LittleEndian, headingLen); err != nil {
			return
		}
		if err := binary.Write(f, binary.LittleEndian, messageLen); err != nil {
			return
		}

		// Write heading
		if _, err := f.Write([]byte(mi.Heading + "\x00")); err != nil {
			return
		}
		// Write message text
		if messageLen > 0 {
			if _, err := f.Write([]byte(msgStr + "\x00")); err != nil {
				return
			}
		}
	}
}

// resetBoard clears all messages on a board and deletes its save file.
func (bs *BoardSystem) resetBoard(boardType int) {
	if boardType < 0 || boardType >= NumBoards {
		return
	}
	bs.mu.Lock()
	defer bs.mu.Unlock()

	for i := 0; i < MaxBoardMessages; i++ {
		mi := bs.msgIndex[boardType][i]
		if mi.SlotNum >= 0 && mi.SlotNum < len(bs.msgStorage) {
			bs.msgStorage[mi.SlotNum] = ""
			bs.msgStorageTaken[mi.SlotNum] = false
		}
		bs.msgIndex[boardType][i] = BoardMsgInfo{SlotNum: -1}
	}
	bs.numOfMsgs[boardType] = 0
	path := filepath.Join(bs.BasePath, bs.boards[boardType].Filename)
// #nosec G104
	os.Remove(path)
}

// findSlot returns the first unused storage slot index.
func (bs *BoardSystem) findSlot() int {
	for i := range bs.msgStorageTaken {
		if !bs.msgStorageTaken[i] {
			bs.msgStorageTaken[i] = true
			return i
		}
	}
	return -1
}

// WriteMessage allocates a slot for a new board message and returns
// the slot index so the session layer can set up the editor.
// Returns -1 on failure.
func (bs *BoardSystem) WriteMessage(boardType int, ch *Player, arg string) int {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if boardType < 0 || boardType >= NumBoards {
		return -1
	}
	if ch.Level < bs.boards[boardType].WriteLvl {
		ch.SendMessage("You are not holy enough to write on this board.\r\n")
		return -1
	}
	if bs.numOfMsgs[boardType] >= MaxBoardMessages {
		ch.SendMessage("The board is full.\r\n")
		return -1
	}
	slot := bs.findSlot()
	if slot == -1 {
		ch.SendMessage("The board is malfunctioning - sorry.\r\n")
		BasicMudLogf("SYSERR: Board: failed to find empty slot on write.")
		return -1
	}

	arg = strings.TrimSpace(arg)
	if len(arg) > 81 {
		arg = arg[:81]
	}
	if arg == "" {
		ch.SendMessage("We must have a headline!\r\n")
		return -1
	}

	now := time.Now()
	tmStr := now.Format("Mon Jan 2 15:04:05 2006")
	heading := fmt.Sprintf("%6.10s %-12s :: %s", tmStr, "("+ch.Name+")", arg)

	idx := bs.numOfMsgs[boardType]
	bs.msgIndex[boardType][idx] = BoardMsgInfo{
		SlotNum: slot,
		Heading: heading,
		Level:   ch.Level,
	}
	bs.msgStorage[slot] = ""
	bs.numOfMsgs[boardType]++

	return boardType + BoardMagic
}

// ShowBoard displays the board's message list.
func (bs *BoardSystem) ShowBoard(boardType int, ch *Player) bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if boardType < 0 || boardType >= NumBoards {
		return false
	}
	if ch.Level < bs.boards[boardType].ReadLvl {
		ch.SendMessage("You try but fail to understand the holy words.\r\n")
		return true
	}

	// Room echo deferred: BoardSystem has no World reference.
	// Will be addressed when BoardSystem gains world access or an event bus is added.

	var buf strings.Builder
	buf.WriteString("This is a bulletin board.  Usage: READ/REMOVE <messg #>, WRITE <header>.\r\n" +
		"You will need to look at the board to save your message.\r\n")

	if bs.numOfMsgs[boardType] == 0 {
		buf.WriteString("The board is empty.\r\n")
	} else {
		fmt.Fprintf(&buf, "There are %d messages on the board.\r\n", bs.numOfMsgs[boardType])
		for i := 0; i < bs.numOfMsgs[boardType]; i++ {
			if bs.msgIndex[boardType][i].Heading != "" {
				fmt.Fprintf(&buf, "%-2d : %s\r\n", i+1, bs.msgIndex[boardType][i].Heading)
			} else {
				BasicMudLogf("SYSERR: The board is fubar'd.")
				ch.SendMessage("Sorry, the board isn't working.\r\n")
				return true
			}
		}
	}

	ch.SendMessage(buf.String())
	return true
}

// DisplayMsg shows a specific message on the board.
func (bs *BoardSystem) DisplayMsg(boardType int, ch *Player, arg string) bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if boardType < 0 || boardType >= NumBoards {
		return false
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		return false
	}

	// Check if arg is just "board" / "bulletin" — redirect to ShowBoard
	if isAbbrev(arg, "board") || isAbbrev(arg, "bulletin") || strings.EqualFold(arg, "board") || strings.EqualFold(arg, "bulletin") {
		bs.mu.RUnlock()
		defer bs.mu.RLock()
		bs.mu.RLock()
		// This is awkward with read locks. Let me just handle it differently.
		// Return false so caller catches "read board" as "show board"
		return false
	}

	// Parse message number
	msg := atoi(arg)
	if msg <= 0 {
		return false
	}

	if ch.Level < bs.boards[boardType].ReadLvl {
		ch.SendMessage("You try but fail to understand the holy words.\r\n")
		return true
	}
	if bs.numOfMsgs[boardType] == 0 {
		ch.SendMessage("The board is empty!\r\n")
		return true
	}
	if msg < 1 || msg > bs.numOfMsgs[boardType] {
		ch.SendMessage("That message exists only in your imagination.\r\n")
		return true
	}

	ind := msg - 1
	mi := bs.msgIndex[boardType][ind]
	if mi.SlotNum < 0 || mi.SlotNum >= len(bs.msgStorage) {
		ch.SendMessage("Sorry, the board is not working.\r\n")
		BasicMudLogf("SYSERR: Board is screwed up.")
		return true
	}
	if mi.Heading == "" {
		ch.SendMessage("That message appears to be screwed up.\r\n")
		return true
	}
	if bs.msgStorage[mi.SlotNum] == "" {
		ch.SendMessage("That message seems to be empty.\r\n")
		return true
	}

	reply := fmt.Sprintf("Message %d : %s\r\n\r\n%s\r\n", msg, mi.Heading, bs.msgStorage[mi.SlotNum])
	ch.SendMessage(reply)
	return true
}

// RemoveMsg removes a message from the board.
func (bs *BoardSystem) RemoveMsg(boardType int, ch *Player, arg string) bool {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if boardType < 0 || boardType >= NumBoards {
		return false
	}
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return false
	}

	msg := atoi(arg)
	if msg <= 0 {
		return false
	}

	if bs.numOfMsgs[boardType] == 0 {
		ch.SendMessage("The board is empty!\r\n")
		return true
	}
	if msg < 1 || msg > bs.numOfMsgs[boardType] {
		ch.SendMessage("That message exists only in your imagination.\r\n")
		return true
	}

	ind := msg - 1
	mi := bs.msgIndex[boardType][ind]
	if mi.Heading == "" {
		ch.SendMessage("That message appears to be screwed up.\r\n")
		return true
	}

	// Check if player can remove this message
	playerTag := fmt.Sprintf("(%s)", ch.Name)
	if ch.Level < bs.boards[boardType].RemoveLvl && !strings.Contains(mi.Heading, playerTag) {
		ch.SendMessage("You are not holy enough to remove other people's messages.\r\n")
		return true
	}
	if ch.Level < mi.Level && ch.Level < 59 { // LVL_IMPL-1
		ch.SendMessage("You can't remove a message holier than yourself.\r\n")
		return true
	}

	slot := mi.SlotNum
	if slot < 0 || slot >= len(bs.msgStorage) {
		BasicMudLogf("SYSERR: The board is seriously screwed up.")
		ch.SendMessage("That message is majorly screwed up.\r\n")
		return true
	}

	// Free storage
	bs.msgStorage[slot] = ""
	bs.msgStorageTaken[slot] = false

	// Compact the message list
	for j := ind; j < bs.numOfMsgs[boardType]-1; j++ {
		bs.msgIndex[boardType][j] = bs.msgIndex[boardType][j+1]
	}
	bs.msgIndex[boardType][bs.numOfMsgs[boardType]-1] = BoardMsgInfo{SlotNum: -1}
	bs.numOfMsgs[boardType]--

	ch.SendMessage("Message removed.\r\n")
	// Room echo when a message is removed
	if bs.world != nil {
		actToRoom(bs.world, ch.GetRoomVNum(),
			fmt.Sprintf("%s removed a message from the board.\r\n", ch.Name),
			ch.Name)
	}

	// Save after removal (release lock first)
	bs.mu.Unlock()
	bs.saveBoard(boardType)
	bs.mu.Lock()

	return true
}

// FindBoard searches the player's current room for a board object.
func (w *World) FindBoard(ch *Player) int {
	if w.Boards == nil {
		return -1
	}
	roomVNum := ch.GetRoomVNum()
	items := w.GetItemsInRoom(roomVNum)
	for _, obj := range items {
		for i := 0; i < NumBoards; i++ {
			if obj.VNum == w.Boards.boards[i].VNum {
				return i
			}
		}
	}
	return -1
}

// genBoard is the spec procedure for bulletin board objects.
// It intercepts read/write/remove/look/examine commands when a board is present.
func genBoard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	_ = me // board objects don't have mob instances, but spec signature requires it
	if w.Boards == nil {
		return false
	}
	if ch == nil {
		return false
	}

	boardType := w.FindBoard(ch)
	if boardType == -1 {
		BasicMudLogf("SYSERR: degenerate board!")
		return false
	}

	switch cmd {
	case "write":
		magic := w.Boards.WriteMessage(boardType, ch, arg)
		if magic > 0 {
			// Return magic value so session layer can pick up editor setup
			ch.WriteMagic = magic
		}
		return true
	case "look", "examine":
		return w.Boards.ShowBoard(boardType, ch)
	case "read":
		return w.Boards.DisplayMsg(boardType, ch, arg)
	case "remove":
		return w.Boards.RemoveMsg(boardType, ch, arg)
	}

	return false
}

// GetOrInitBoards ensures the board system is initialized.
func (w *World) GetOrInitBoards(basePath string) *BoardSystem {
	if w.Boards == nil {
		w.Boards = InitBoards(basePath)
	}
	return w.Boards
}

func init() {
	// Register gen_board as an object spec procedure
	RegisterSpec("gen_board", genBoard)
}

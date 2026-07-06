package boards

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// mockBoardPlayer is a test double for BoardPlayer.
type mockBoardPlayer struct {
	mu       sync.Mutex
	name     string
	level    int
	roomVNum int
	messages []string
}

func newMockBoardPlayer(name string, level, roomVNum int) *mockBoardPlayer {
	return &mockBoardPlayer{
		name:     name,
		level:    level,
		roomVNum: roomVNum,
	}
}

func (m *mockBoardPlayer) GetLevel() int    { return m.level }
func (m *mockBoardPlayer) GetName() string  { return m.name }
func (m *mockBoardPlayer) GetRoomVNum() int { return m.roomVNum }
func (m *mockBoardPlayer) SendMessage(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *mockBoardPlayer) lastMessage() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return ""
	}
	return m.messages[len(m.messages)-1]
}

func (m *mockBoardPlayer) allMessages() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return strings.Join(m.messages, "")
}

// mockBoardWorld is a test double for BoardWorld.
type mockBoardWorld struct {
	mu     sync.Mutex
	echoes []string
}

func (m *mockBoardWorld) RoomEcho(roomVNum int, message string, excludeName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.echoes = append(m.echoes, message)
}

func TestDisplayMsgBoardArgumentDoesNotLeakReadLock(t *testing.T) {
	bs := InitBoards(t.TempDir())
	ch := newMockBoardPlayer("Alice", 1, 1001)

	if bs.DisplayMsg(0, ch, "board") {
		t.Fatal("DisplayMsg(board) = true, want false redirect")
	}

	done := make(chan int, 1)
	go func() {
		done <- bs.WriteMessage(0, ch, "test heading")
	}()

	select {
	case magic := <-done:
		if magic != BoardMagic {
			t.Fatalf("WriteMessage magic = %d, want %d", magic, BoardMagic)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("WriteMessage blocked after DisplayMsg(board), read lock likely leaked")
	}
}

func TestBoardSystem_InitAndWrite(t *testing.T) {
	bs := InitBoards(t.TempDir())
	ch := newMockBoardPlayer("Bob", 10, 2001)

	magic := bs.WriteMessage(0, ch, "hello world")
	if magic != BoardMagic {
		t.Fatalf("WriteMessage magic = %d, want %d", magic, BoardMagic)
	}

	bs.AppendBoardLine(magic, "first line")
	bs.AppendBoardLine(magic, "second line")

	bs.FinalizeBoardWrite(magic, ch)

	if !strings.Contains(ch.lastMessage(), "Message written.") {
		t.Fatalf("expected 'Message written.' message, got %q", ch.lastMessage())
	}

	// Verify the message is readable.
	out := newMockBoardPlayer("Bob", 10, 2001)
	if !bs.DisplayMsg(0, out, "1") {
		t.Fatal("DisplayMsg(1) = false, want true")
	}
	if !strings.Contains(out.allMessages(), "hello world") {
		t.Fatalf("expected heading in output, got %q", out.allMessages())
	}
	if !strings.Contains(out.allMessages(), "first line") || !strings.Contains(out.allMessages(), "second line") {
		t.Fatalf("expected body lines in output, got %q", out.allMessages())
	}
}

func TestBoardSystem_ShowBoard_Empty(t *testing.T) {
	bs := InitBoards(t.TempDir())
	ch := newMockBoardPlayer("Carol", 1, 3001)

	if !bs.ShowBoard(0, ch) {
		t.Fatal("ShowBoard = false, want true")
	}
	if !strings.Contains(ch.allMessages(), "The board is empty.") {
		t.Fatalf("expected empty board message, got %q", ch.allMessages())
	}
}

func TestBoardSystem_RemoveMsg_LevelCheck(t *testing.T) {
	bs := InitBoards(t.TempDir())
	poster := newMockBoardPlayer("Dave", 50, 4001)
	remover := newMockBoardPlayer("Eve", 1, 4001)

	magic := bs.WriteMessage(0, poster, "important news")
	bs.AppendBoardLine(magic, "body text")
	bs.FinalizeBoardWrite(magic, poster)

	// Low-level remover cannot remove a high-level poster's message.
	if !bs.RemoveMsg(0, remover, "1") {
		t.Fatal("RemoveMsg = false, want true")
	}
	if !strings.Contains(remover.lastMessage(), "not holy enough") {
		t.Fatalf("expected level check rejection, got %q", remover.lastMessage())
	}

	// Message should still be present.
	out := newMockBoardPlayer("Out", 1, 4001)
	if !bs.DisplayMsg(0, out, "1") {
		t.Fatal("DisplayMsg after failed remove = false, want true")
	}
}

func TestBoardSystem_RemoveMsg_RoomEcho(t *testing.T) {
	bs := InitBoards(t.TempDir())
	world := &mockBoardWorld{}
	bs.SetWorld(world)

	poster := newMockBoardPlayer("Frank", 60, 7001)
	remover := newMockBoardPlayer("Frank", 60, 7001)

	magic := bs.WriteMessage(0, poster, "removable")
	bs.AppendBoardLine(magic, "body")
	bs.FinalizeBoardWrite(magic, poster)

	if !bs.RemoveMsg(0, remover, "1") {
		t.Fatal("RemoveMsg = false, want true")
	}

	world.mu.Lock()
	defer world.mu.Unlock()
	if len(world.echoes) != 1 {
		t.Fatalf("expected 1 room echo, got %d", len(world.echoes))
	}
	if !strings.Contains(world.echoes[0], "removed a message") {
		t.Fatalf("expected room echo about removed message, got %q", world.echoes[0])
	}
}

func TestBoardSystem_FullBoard_RejectsOverflow(t *testing.T) {
	bs := InitBoards(t.TempDir())
	ch := newMockBoardPlayer("Flooder", 60, 5001)

	for i := 0; i < MaxBoardMessages; i++ {
		magic := bs.WriteMessage(0, ch, "flood")
		if magic <= 0 {
			t.Fatalf("WriteMessage #%d failed unexpectedly", i+1)
		}
		bs.FinalizeBoardWrite(magic, ch)
	}

	// 61st message should be rejected.
	magic := bs.WriteMessage(0, ch, "one too many")
	if magic != -1 {
		t.Fatalf("WriteMessage on full board = %d, want -1", magic)
	}
	if !strings.Contains(ch.lastMessage(), "full") {
		t.Fatalf("expected 'full' rejection, got %q", ch.lastMessage())
	}
}

func TestBoardSystem_DisplayMsg_InvalidNumber(t *testing.T) {
	bs := InitBoards(t.TempDir())
	ch := newMockBoardPlayer("Grace", 1, 6001)

	// Non-numeric argument is rejected.
	if bs.DisplayMsg(0, ch, "abc") {
		t.Fatal("DisplayMsg(abc) = true, want false")
	}
	// Zero / negative numbers are rejected.
	if bs.DisplayMsg(0, ch, "0") {
		t.Fatal("DisplayMsg(0) = true, want false")
	}

	// Add a real message so out-of-range can be tested.
	magic := bs.WriteMessage(0, ch, "real message")
	bs.FinalizeBoardWrite(magic, ch)

	// Out-of-range number is reported to the player (returns true).
	out := newMockBoardPlayer("Out", 1, 6001)
	if !bs.DisplayMsg(0, out, "99") {
		t.Fatal("DisplayMsg(99) on populated board = false, want true")
	}
	if !strings.Contains(out.lastMessage(), "imagination") {
		t.Fatalf("expected out-of-range message, got %q", out.lastMessage())
	}
}

package telnet

import (
	"bufio"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/session"
)

type tempNetError struct {
	error
}

func (e tempNetError) Temporary() bool {
	return true
}

func (e tempNetError) Timeout() bool {
	return false
}

type mockListener struct {
	acceptCount int
	errs        []error
	conns       chan net.Conn
	closed      chan struct{}
	mu          sync.Mutex
}

func (m *mockListener) Accept() (net.Conn, error) {
	m.mu.Lock()
	idx := m.acceptCount
	m.acceptCount++
	m.mu.Unlock()

	if idx < len(m.errs) {
		if m.errs[idx] != nil {
			return nil, m.errs[idx]
		}
	}

	select {
	case conn := <-m.conns:
		return conn, nil
	case <-m.closed:
		return nil, errors.New("listener closed")
	}
}

func (m *mockListener) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	select {
	case <-m.closed:
	default:
		close(m.closed)
	}
	return nil
}

func (m *mockListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 7777}
}

type pipeConn struct {
	net.Conn
}

func (p pipeConn) SetDeadline(t time.Time) error      { return nil }
func (p pipeConn) SetReadDeadline(t time.Time) error  { return nil }
func (p pipeConn) SetWriteDeadline(t time.Time) error { return nil }
func (p pipeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}
func (p pipeConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 7777}
}

func TestListen_RetriesOnTemporaryError(t *testing.T) {
	origListenTCP := listenTCP
	defer func() { listenTCP = origListenTCP }()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	wrappedServerConn := pipeConn{serverConn}

	conns := make(chan net.Conn, 1)
	conns <- wrappedServerConn

	ml := &mockListener{
		errs: []error{
			tempNetError{errors.New("temporary error 1")},
			tempNetError{errors.New("temporary error 2")},
		},
		conns:  conns,
		closed: make(chan struct{}),
	}

	listenTCP = func(network, address string) (net.Listener, error) {
		return ml, nil
	}

	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum:        game.MortalStartRoom,
				Name:        "The Adventurers Guild",
				Description: "A grand hall where adventurers gather to seek glory.",
				Zone:        80,
			},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer w.StopAITicker()
	manager := session.NewManager(w, nil)

	err = Listen(0, manager)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer Stop()

	// Wait and read greeting to verify connection was accepted
	buf := make([]byte, 1024)
	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := clientConn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from connection: %v", err)
	}
	if n == 0 {
		t.Fatal("read 0 bytes from connection")
	}

	ml.mu.Lock()
	count := ml.acceptCount
	ml.mu.Unlock()

	if count < 3 {
		t.Errorf("expected at least 3 accept attempts, got %d", count)
	}
}

func TestReadLine_BufferLimit(t *testing.T) {
	oversized := strings.Repeat("A", maxInputLen*2)
	input := oversized + "\r\nlook\r\n"

	tc := &telnetConn{
		br: bufio.NewReader(strings.NewReader(input)),
	}

	// First read: should return truncated string of maxInputLen 'A's
	line, ok := tc.readLine()
	if !ok {
		t.Fatal("expected readLine to succeed")
	}
	expectedLen := maxInputLen
	if len(line) != expectedLen {
		t.Errorf("expected line length %d, got %d", expectedLen, len(line))
	}
	expectedLine := strings.Repeat("A", maxInputLen)
	if line != expectedLine {
		t.Errorf("expected truncated line")
	}

	// Second read: should return "look"
	line2, ok2 := tc.readLine()
	if !ok2 {
		t.Fatal("expected second readLine to succeed")
	}
	if line2 != "look" {
		t.Errorf("expected line to be %q, got %q", "look", line2)
	}
}

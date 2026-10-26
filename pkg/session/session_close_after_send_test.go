package session

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func TestCloseAfterSendDrainsBeforeClosingTransport(t *testing.T) {
	s := &Session{send: make(chan []byte, 2)}
	s.send <- []byte("This body has been usurped!\r\n")
	s.send <- []byte("Multiple login detected -- disconnecting.\r\n")

	var drained strings.Builder
	var transportClosed bool
	s.SetDrainBeforeCloseFunc(func() {
		for message := range s.SendChannel() {
			drained.Write(message)
		}
	})
	s.SetCloseFunc(func() { transportClosed = true })

	s.CloseAfterSend()

	want := "This body has been usurped!\r\nMultiple login detected -- disconnecting.\r\n"
	if got := drained.String(); got != want {
		t.Fatalf("drained output = %q, want %q", got, want)
	}
	if !transportClosed {
		t.Fatal("transport was not closed after the writer drained")
	}
}

func TestDupeCheckDrainsNonReadingClientWithoutManagerOrWorldLock(t *testing.T) {
	m, old, _, fresh := dupeCheckFixture(t)
	writerConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = writerConn.Close()
		_ = clientConn.Close()
	})
	writeStarted := make(chan struct{})
	writerDone := make(chan struct{})
	writerErr := make(chan error, 1)
	old.SetDrainBeforeCloseFunc(func() { <-writerDone })
	old.SetCloseFunc(func() { _ = writerConn.Close() })
	go func() {
		defer close(writerDone)
		for message := range old.SendChannel() {
			select {
			case <-writeStarted:
			default:
				close(writeStarted)
			}
			if _, err := writerConn.Write(message); err != nil {
				writerErr <- err
				return
			}
		}
		writerErr <- nil
	}()

	dupeDone := make(chan bool, 1)
	go func() { dupeDone <- fresh.performDupeCheck() }()
	select {
	case <-writeStarted:
	case <-time.After(time.Second):
		t.Fatal("old session writer did not reach the non-reading client")
	}

	managerLockFree := make(chan struct{})
	managerSessionCount := 0
	go func() {
		m.mu.Lock()
		managerSessionCount = len(m.sessions)
		m.mu.Unlock()
		close(managerLockFree)
	}()
	select {
	case <-managerLockFree:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("CloseAfterSend held Manager.mu while the client was not reading")
	}
	if managerSessionCount == 0 {
		t.Fatal("Manager.mu access completed without observing the registered session")
	}

	worldLockFree := make(chan struct{})
	go func() {
		_ = m.world.GetPlayerCount()
		close(worldLockFree)
	}()
	select {
	case <-worldLockFree:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("CloseAfterSend held a world lock while the client was not reading")
	}
	if err := writerConn.SetWriteDeadline(time.Now().Add(300 * time.Millisecond)); err != nil {
		t.Fatalf("set blocked-writer deadline: %v", err)
	}

	select {
	case ok := <-dupeDone:
		if !ok {
			t.Fatal("duplicate login did not take over the body")
		}
	case <-time.After(time.Second):
		t.Fatal("CloseAfterSend did not finish after the blocked write deadline")
	}
	var netErr net.Error
	if err := <-writerErr; !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("non-reading client write error = %v, want deadline timeout", err)
	}
	if !old.SendClosed() || fresh.player != old.player {
		t.Fatal("dupe cleanup did not finish normally after the transport deadline")
	}
}

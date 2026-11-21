package oraclediff

import (
	"bufio"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunSetupWaitsForEnterGamePromptPastFirstByteWait(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	var enteredWorld atomic.Bool
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = conn.Write([]byte("By what name do you wish to be known? "))
		r := bufio.NewReader(conn)
		replies := []string{
			"Password: ",
			"*** PRESS RETURN: ",
			"   Make your choice: ",
		}
		for _, reply := range replies {
			if _, readErr := r.ReadString('\n'); readErr != nil {
				return
			}
			_, _ = conn.Write([]byte(reply))
		}
		// Enter-game step ("1"): stall longer than firstByteWait to simulate
		// a slow SQLite saveCreationRecord fsync under heavy census load.
		if _, readErr := r.ReadString('\n'); readErr != nil {
			return
		}
		time.Sleep(150 * time.Millisecond)
		enteredWorld.Store(true)
		_, _ = conn.Write([]byte("\r\nWelcome to Dark Pawns! May your visit here be... Interesting.\r\n\r\n10H 100M 82V > "))
	}()

	rawConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = rawConn.Close() }()

	conn := NewTCPConn(rawConn)
	conn.SetFirstByteWait(50 * time.Millisecond)

	transcript, err := RunSetup(conn, []string{"Peer", "pass", "", "1"}, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("RunSetup: %v", err)
	}
	if !enteredWorld.Load() {
		t.Fatalf("RunSetup returned before server completed world entry (DP-1623 warmup peer race); transcript=%q", transcript)
	}
	if !strings.Contains(transcript, "Welcome to Dark Pawns!") {
		t.Fatalf("RunSetup transcript missing game-entry welcome: %q", transcript)
	}
	<-serverDone
}

func TestRunSetupWaitsForSlowGreetingUpToFirstByteWait(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	var sentBeforeGreeting atomic.Bool
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		// Stall greeting longer than quiescence (20ms) but within firstByteWait (250ms).
		time.Sleep(80 * time.Millisecond)
		// Check whether the client already sent its first setup line before the greeting.
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Millisecond))
		buf := make([]byte, 1)
		if n, _ := conn.Read(buf); n > 0 {
			sentBeforeGreeting.Store(true)
			return
		}
		_ = conn.SetReadDeadline(time.Time{})
		_, _ = conn.Write([]byte("By what name do you wish to be known? "))
		r := bufio.NewReader(conn)
		if _, readErr := r.ReadString('\n'); readErr != nil {
			return
		}
		_, _ = conn.Write([]byte("Password: "))
	}()

	rawConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = rawConn.Close() }()

	conn := NewTCPConn(rawConn)
	conn.SetFirstByteWait(250 * time.Millisecond)

	transcript, err := RunSetup(conn, []string{"Peer"}, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("RunSetup: %v", err)
	}
	<-serverDone
	if sentBeforeGreeting.Load() {
		t.Fatalf("RunSetup sent step 1 before receiving slow initial greeting (waited quiescence instead of firstByteWait)")
	}
	if !strings.Contains(transcript, "By what name do you wish to be known? ") {
		t.Fatalf("RunSetup missed slow initial greeting: %q", transcript)
	}
	if !strings.Contains(transcript, "Password: ") {
		t.Fatalf("RunSetup missed password prompt after slow greeting: %q", transcript)
	}
}

//go:build linux

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

// The real helper owns no listener. A foreign live endpoint must not receive
// even the harness's initial connection, much less fixture login/setup (R5h).
func TestForeignListenerRejectedBeforeDial(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p, err := startProcess(ctx, "foreign fixture", t.TempDir(), os.Environ(), "sleep", "10")
	if err != nil {
		t.Fatal(err)
	}
	defer p.stop()
	if conn, err := dialWhenReady(p, ln.Addr().String(), time.Second); err == nil {
		_ = conn.Close()
		t.Fatal("foreign listener was accepted")
	}
	if err := ln.(*net.TCPListener).SetDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if conn, err := ln.Accept(); err == nil {
		_ = conn.Close()
		t.Fatal("foreign endpoint received a connection")
	}
}

func TestOwnedListenerRecognizesSocketInode(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	own, foreign, err := ownedListener(os.Getpid(), ln.Addr().(*net.TCPAddr).Port)
	if err != nil || !own || foreign {
		t.Fatalf("owned=%v foreign=%v err=%v", own, foreign, err)
	}
}

func TestGoWHODReserved(t *testing.T) {
	c, cw, g, gw, h, err := allocatePorts()
	if err != nil {
		t.Fatal(err)
	}
	for _, ln := range []net.Listener{c, cw, g, gw, h} {
		defer func() { _ = ln.Close() }()
	}
	if got, want := gw.Addr().(*net.TCPAddr).Port, g.Addr().(*net.TCPAddr).Port+1; got != want {
		t.Fatalf("Go WHOD=%d want %d", got, want)
	}
	if ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", gw.Addr().(*net.TCPAddr).Port)); err == nil {
		_ = ln.Close()
		t.Fatal("Go WHOD was unreserved")
	}
}

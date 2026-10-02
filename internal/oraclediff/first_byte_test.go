package oraclediff

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

func TestTCPFirstByteWaitUnderCPULoad(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()
	go func() {
		if _, err := bufio.NewReader(server).ReadString('\n'); err != nil {
			return
		}
		// Artificial CPU work, rather than a sleep: the response cannot be
		// produced during the old 300ms window. This is the missing-first-byte
		// class demonstrated by the retained zreset * census failure.
		until := time.Now().Add(650 * time.Millisecond)
		for time.Now().Before(until) {
		}
		_, _ = server.Write([]byte("Reset world.\r\n"))
	}()
	conn := NewTCPConn(client)
	conn.SetFirstByteWait(2 * time.Second)
	if err := conn.Send("zreset *"); err != nil {
		t.Fatal(err)
	}
	got, err := conn.ReadUntilQuiescent(300 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Reset world.\r\n" {
		t.Fatalf("CPU-delayed response lost: got %q", got)
	}
}

func TestTCPFirstByteWaitKeepsTrailingSilence(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()
	go func() {
		if _, err := bufio.NewReader(server).ReadString('\n'); err != nil {
			return
		}
		_, _ = server.Write([]byte("complete\r\n"))
	}()
	conn := NewTCPConn(client)
	conn.SetFirstByteWait(2 * time.Second)
	if err := conn.Send("look"); err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	got, err := conn.ReadUntilQuiescent(300 * time.Millisecond)
	if err != nil || got != "complete\r\n" {
		t.Fatalf("got %q, %v", got, err)
	}
	if elapsed := time.Since(before); elapsed >= time.Second {
		t.Fatalf("first-byte allowance extended trailing silence: %s", elapsed)
	}
}

func TestTCPFirstByteWaitEmptyResponseIsBounded(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()
	conn := NewTCPConn(client)
	conn.SetFirstByteWait(650 * time.Millisecond)
	go func() { _, _ = bufio.NewReader(server).ReadString('\n') }()
	if err := conn.Send("silent command"); err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	got, err := conn.ReadUntilQuiescent(50 * time.Millisecond)
	elapsed := time.Since(before)
	if err != nil || got != "" {
		t.Fatalf("empty response: %q, %v", got, err)
	}
	if elapsed < 500*time.Millisecond || elapsed > 2*time.Second {
		t.Fatalf("empty response wait not bounded by first-byte allowance: %s", elapsed)
	}
}

func TestWSFirstByteCPUHelper(t *testing.T) {
	if os.Getenv("DP_WS_CPU_HELPER") != "1" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Printf("%s\r\n", scanner.Text()) // browser-local echo, not a response
		until := time.Now().Add(650 * time.Millisecond)
		for time.Now().Before(until) {
		}
		fmt.Print("Reset world.\r\n")
	}
}

func TestWSFirstByteWaitUnderCPULoadIgnoresLocalEcho(t *testing.T) {
	t.Setenv("DP_WS_CPU_HELPER", "1")
	conn, err := NewWSConn(os.Args[0], "-test.run=^TestWSFirstByteCPUHelper$", "client", "ws://unused")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	conn.SetFirstByteWait(2 * time.Second)
	if err := conn.Send("zreset *"); err != nil {
		t.Fatal(err)
	}
	got, err := conn.ReadUntilQuiescent(300 * time.Millisecond)
	if err != nil || got != "Reset world.\r\n" {
		t.Fatalf("CPU-delayed browser response lost after local echo: %q, %v", got, err)
	}
}

func TestSlowIssuingConnectionCapturesPeerAfterActor(t *testing.T) {
	for _, step := range []string{"slow", "send:observer slow"} {
		t.Run(step, func(t *testing.T) {
			actorClient, actorServer := net.Pipe()
			peerClient, peerServer := net.Pipe()
			for _, c := range []net.Conn{actorClient, actorServer, peerClient, peerServer} {
				defer func(c net.Conn) { _ = c.Close() }(c)
			}
			actor, peer := NewTCPConn(actorClient), NewTCPConn(peerClient)
			actor.SetFirstByteWait(2 * time.Second)
			peer.SetFirstByteWait(2 * time.Second)
			issuer, observer := actorServer, peerServer
			if step != "slow" {
				issuer, observer = peerServer, actorServer
			}
			go func() {
				if _, err := bufio.NewReader(issuer).ReadString('\n'); err != nil {
					return
				}
				until := time.Now().Add(650 * time.Millisecond)
				for time.Now().Before(until) {
				}
				go func() { _, _ = observer.Write([]byte("observer sees slow command\r\n")) }()
				_, _ = issuer.Write([]byte("actor finishes slow command\r\n"))
			}()
			blocks, err := RunAudienceProbe(actor, map[string]Conn{"observer": peer}, []string{step}, 300*time.Millisecond)
			if err != nil {
				t.Fatal(err)
			}
			if len(blocks) != 2 || blocks[0].Output != "actor finishes slow command\r\n" || blocks[1].Output != "observer sees slow command\r\n" {
				t.Fatalf("slow issuing connection or following peer capture lost output: %+v", blocks)
			}
		})
	}
}

func TestPassivePeerUsesShortQuiescence(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()
	conn := NewTCPConn(client)
	conn.SetFirstByteWait(2 * time.Second)
	before := time.Now()
	got, err := conn.ReadUntilQuiescent(300 * time.Millisecond)
	if err != nil || got != "" {
		t.Fatalf("passive peer: %q %v", got, err)
	}
	if elapsed := time.Since(before); elapsed >= time.Second {
		t.Fatalf("passive peer used actor allowance: %s", elapsed)
	}
}

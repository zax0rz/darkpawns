package session

import (
	"strings"
	"testing"
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

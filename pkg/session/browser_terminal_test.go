package session

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestBrowserTerminalForwardsGMCPWithoutChangingText(t *testing.T) {
	gmcp, err := json.Marshal(ServerMessage{Type: MsgGMCP, Data: GMCPData{
		Package: "Comm.Channel.Text",
		JSON:    `{"channel":"say","talker":"Walker","text":"Walker says hi"}`,
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := renderForBrowserTerminal(gmcp)
	if !ok || !bytes.Equal(got, gmcp) {
		t.Fatalf("GMCP frame changed: ok=%t got=%s", ok, got)
	}
	text := []byte(`{"type":"event","data":{"text":"Walker says hi\r\n"}}`)
	got, ok = renderForBrowserTerminal(text)
	if !ok {
		t.Fatal("canonical text was dropped")
	}
	var frame ServerMessage
	if err := json.Unmarshal(got, &frame); err != nil {
		t.Fatal(err)
	}
	if frame.Type != MsgOut {
		t.Fatalf("text frame type = %q", frame.Type)
	}
}

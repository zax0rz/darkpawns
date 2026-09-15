package db

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestMailReceivedObjectSerializesWithoutFixedBlockPadding proves that the
// native received mail object reaches PlayerToRecord with semantic text and
// therefore no JSONB-invalid NUL escape.
func TestMailReceivedObjectSerializesWithoutFixedBlockPadding(t *testing.T) {
	const (
		body     = "proof-mail-body"
		mailText = "mail header\r\nFrom: Sender\r\n\r\n" + body
	)

	player := game.NewPlayer(901, "MailSavePayload", 1001)
	obj := (&game.World{}).CreateMailObject(player, mailText)
	player.Inventory.RestoreItem(obj)

	record, err := PlayerToRecord(player, nil)
	if err != nil {
		t.Fatalf("PlayerToRecord: %v", err)
	}
	if !json.Valid(record.Inventory) {
		t.Fatalf("inventory payload is not valid JSON: %q", record.Inventory)
	}
	if bytes.IndexByte(record.Inventory, 0) >= 0 {
		t.Fatal("serialized inventory contains a raw NUL byte")
	}

	payload := string(record.Inventory)
	if strings.Contains(payload, `\u0000`) {
		t.Fatalf("received mail inventory contains JSON NUL escape: %q", payload)
	}
	if !strings.Contains(payload, body) {
		t.Fatalf("received mail body missing from inventory payload: %q", payload)
	}
	t.Logf("received_mail_serialization nul_escape=false body=%q inventory=%s", body, payload)
}

// TestSerializerControlRetainsEmbeddedNUL is a separately labeled control for
// the existing serializer. This repair does not strip NULs globally from
// player JSON; only fixed mail text is converted at the mail read boundary.
func TestSerializerControlRetainsEmbeddedNUL(t *testing.T) {
	player := game.NewPlayer(902, "MailSerializerControl", 1001)
	obj := (&game.World{}).CreateMailObject(player, "body")
	obj.Runtime.ShortDesc = "control\x00tail"
	player.Inventory.RestoreItem(obj)

	record, err := PlayerToRecord(player, nil)
	if err != nil {
		t.Fatalf("PlayerToRecord: %v", err)
	}
	if !strings.Contains(string(record.Inventory), `\u0000`) {
		t.Fatalf("serializer control lost embedded NUL escape: %q", record.Inventory)
	}
	t.Logf("serializer_control embedded_nul_preserved=true inventory=%s", record.Inventory)
}

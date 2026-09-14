package db

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestMailSaveSerializationCharacterizesFixedBlockPadding proves the exact
// serializer boundary independently of PostgreSQL. The control differs only
// by C-style termination of the fixed-block text; both payloads remain valid
// JSON text, but only the padded one contains the JSONB-invalid \u0000 escape.
func TestMailSaveSerializationCharacterizesFixedBlockPadding(t *testing.T) {
	const body = "proof-mail-body"
	mailText := "mail header\r\n" + body + strings.Repeat("\x00", game.MailHeaderDataSize-len(body))

	cases := []struct {
		name       string
		mailText   string
		wantEscape bool
	}{
		{name: "fixed_block_text", mailText: mailText, wantEscape: true},
		{name: "terminated_control", mailText: strings.TrimRight(mailText, "\x00"), wantEscape: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			player := game.NewPlayer(901, "MailSavePayload", 1001)
			obj := (&game.World{}).CreateMailObject(player, tc.mailText)
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
			gotEscape := strings.Contains(payload, `\u0000`)
			if gotEscape != tc.wantEscape {
				t.Fatalf("JSON NUL escape = %t, want %t; payload=%q", gotEscape, tc.wantEscape, payload)
			}
			sanitized := strings.ReplaceAll(payload, `\u0000`, "<NUL>")
			payloadHash := sha256.Sum256(record.Inventory)
			t.Logf("serialized_inventory_sha256=%s serialized_inventory_sanitized=%s runtime_nul_count=%d", hex.EncodeToString(payloadHash[:]), sanitized, strings.Count(tc.mailText, "\x00"))
		})
	}
}

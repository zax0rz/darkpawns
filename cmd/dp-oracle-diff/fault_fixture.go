package main

// fault_fixture.go — opt-in fault-injection verbs
// (docs/design/fault-injection-fixture.md). Every verb mutates BOTH engines'
// disposable harness trees so the C oracle and the Go port observe the same
// unreadable or corrupt state; the reference oracle binary is never touched.
//
// The two engines keep different internal file layouts (Go's board rows are
// 20 bytes where C's struct is 32 on this platform, and Go's mail blocks are
// 512 bytes where C's are 100), so corrupt-file verbs write one
// format-correct file per side carrying the same malformation. The compared
// surface is each engine's player-facing behaviour, not the file bytes.

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

// faultTrees are the two disposable roots the harness mutates.
type faultTrees struct {
	oracleLib string // C oracle's -d data root (its CWD after chdir)
	goRoot    string // Go server's CWD (game root = parent of the world dir)
	goWorld   string // Go server's -world directory
}

// faultBoardFiles maps the board OBJECT vnum a scenario spawns to the board
// save filename both engines use (src/boards.c:92-102). Only the boards the
// fault scenarios exercise are listed; adding one is a one-line change here
// plus the symmetry test.
var faultBoardFiles = map[int]string{
	19652: "etc/board.trinity",
}

// faultPaths resolves a logical fault target to the two physical paths.
func faultPaths(t faultTrees, ff oraclediff.FaultFixture) (cPath, goPath string, err error) {
	switch ff.Verb {
	case "fault-dir":
		switch ff.Target {
		case "mail":
			return filepath.Join(t.oracleLib, "etc", "plrmail"), filepath.Join(t.goRoot, "data", "mail"), nil
		case "object_idlist":
			return filepath.Join(t.oracleLib, "object_idlist"), filepath.Join(t.goRoot, "object_idlist"), nil
		}
	case "fault-corrupt-mail":
		return filepath.Join(t.oracleLib, "etc", "plrmail"), filepath.Join(t.goRoot, "data", "mail"), nil
	case "fault-board":
		vnum, convErr := strconv.Atoi(ff.Target)
		if convErr == nil {
			if name, ok := faultBoardFiles[vnum]; ok {
				return filepath.Join(t.oracleLib, filepath.FromSlash(name)), filepath.Join(t.goWorld, filepath.FromSlash(name)), nil
			}
		}
	case "fault-crash-file":
		// C's crash file is plrobjs/<bucket>/<lower(name)>.objs, bucket by
		// first letter (src/utils.c:555-578). Go's twin is the store-read
		// fault driven by DP_FAULT_OBJECT_SAVE in the session layer; there is
		// no Go path to mutate.
		return filepath.Join(t.oracleLib, "plrobjs", crashFileBucket(ff.Target), strings.ToLower(ff.Target)+".objs"), "", nil
	}
	return "", "", fmt.Errorf("fault fixture %v: unknown target", ff)
}

// faultDirAt replaces one path with a directory: fopen("r"/"r+b"/"w") and
// os.Open/os.Create all fail with EISDIR, deterministically, regardless of
// the runner's uid.
func faultDirAt(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("fault-dir %s: %w", path, err)
	}
	return os.MkdirAll(path, 0o750)
}

// applyFaultFixtures runs every boot-time [fixture] fault verb.
func applyFaultFixtures(t faultTrees, fixtures []oraclediff.FaultFixture) error {
	for _, ff := range fixtures {
		if err := applyFaultVerb(t, ff); err != nil {
			return err
		}
	}
	return nil
}

// applyFaultStep parses and applies one "~fault <verb> ..." probe step.
func applyFaultStep(t faultTrees, step string) error {
	fields := strings.Fields(strings.TrimPrefix(step, oraclediff.FaultControlPrefix))
	ff, ok := parseFaultStep(fields)
	if !ok {
		return fmt.Errorf("unknown fault step %q", step)
	}
	return applyFaultVerb(t, ff)
}

func parseFaultStep(fields []string) (oraclediff.FaultFixture, bool) {
	if len(fields) == 0 {
		return oraclediff.FaultFixture{}, false
	}
	switch strings.ToLower(fields[0]) {
	case "fault-dir":
		if len(fields) == 2 && (fields[1] == "object_idlist" || fields[1] == "mail") {
			return oraclediff.FaultFixture{Verb: "fault-dir", Target: fields[1]}, true
		}
	case "fault-corrupt-mail":
		if len(fields) == 2 && (fields[1] == "unaligned" || fields[1] == "bad-header") {
			return oraclediff.FaultFixture{Verb: "fault-corrupt-mail", Target: fields[1]}, true
		}
	}
	return oraclediff.FaultFixture{}, false
}

func applyFaultVerb(t faultTrees, ff oraclediff.FaultFixture) error {
	switch ff.Verb {
	case "fault-dir":
		cPath, goPath, err := faultPaths(t, ff)
		if err != nil {
			return err
		}
		if err := faultDirAt(cPath); err != nil {
			return err
		}
		if goPath != "" {
			return faultDirAt(goPath)
		}
		return nil
	case "fault-corrupt-mail":
		return corruptMail(t, ff.Target)
	case "fault-board":
		return corruptBoard(t, ff)
	case "fault-crash-file":
		// The Go side of this fault is DP_FAULT_OBJECT_SAVE, set once in the
		// engine environment before boot (main.go); here only C's file is
		// replaced.
		cPath, _, err := faultPaths(t, ff)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(cPath), 0o750); err != nil {
			return err
		}
		return faultDirAt(cPath)
	}
	return fmt.Errorf("fault fixture %v: unknown verb", ff)
}

// corruptMail plants the mail store with the requested fault.
//
// unaligned: one stray byte. C's scan_file fails the BLOCK_SIZE alignment
// check (src/mail.c:241-244) and db.c:365-370 disables mail at boot; Go's
// scanFile fails its own 512-byte alignment the same way (pkg/game/mail.go).
//
// bad-header: overwrite the first block's block_type with known garbage so
// the next read_delete deterministically fails the header check
// (src/mail.c:423-427). block_type is the first field on both engines.
func corruptMail(t faultTrees, mode string) error {
	cPath := filepath.Join(t.oracleLib, "etc", "plrmail")
	goPath := filepath.Join(t.goRoot, "data", "mail")
	if err := os.MkdirAll(filepath.Dir(cPath), 0o750); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(goPath), 0o750); err != nil {
		return err
	}
	switch mode {
	case "unaligned":
		if err := os.WriteFile(cPath, []byte{0x7f}, 0o600); err != nil {
			return err
		}
		return os.WriteFile(goPath, []byte{0x7f}, 0o600)
	case "bad-header":
		for _, p := range []string{cPath, goPath} {
			data, err := os.ReadFile(p)
			if err != nil || len(data) == 0 {
				st, _ := os.Stat(p)
				size := int64(-1)
				if st != nil {
					size = st.Size()
				}
				return fmt.Errorf("fault-corrupt-mail bad-header: %s has no block to corrupt (send mail first): readErr=%v size=%d", p, err, size)
			}
			garbage := []byte{0x64, 0x00, 0x00, 0x00} // 100: neither HEADER(-1) nor DELETED(-3)
			copy(data[:len(garbage)], garbage)
			if err := os.WriteFile(p, data, 0o600); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("fault-corrupt-mail: unknown mode %q", mode)
}

// corruptBoard writes a crafted one-message board file per engine with the
// requested malformation. The on-disk row header differs per engine (C's
// struct board_msginfo is 32 bytes with an 8-byte heading pointer; the port's
// is 20), so each side gets a format-correct file carrying the same logical
// fault (src/boards.c:495-530; pkg/boards/boards.go loadBoard).
func corruptBoard(t faultTrees, ff oraclediff.FaultFixture) error {
	cPath, goPath, err := faultPaths(t, ff)
	if err != nil {
		return err
	}
	slot := 0
	switch ff.Detail {
	case "empty-message-valid-slot", "nul-heading":
		slot = 0
	case "empty-message-huge-slot":
		slot = 99999
	default:
		return fmt.Errorf("fault-board: unknown malformation %q", ff.Detail)
	}
	heading := append([]byte("planted headline"), 0x00)
	messageLen := int32(0)
	if ff.Detail == "nul-heading" {
		heading = []byte{0x00}
		messageLen = int32(len("planted body") + 1)
	}

	// writeRow lays out one row header. Go's 20-byte row: slot 0, level 8,
	// heading_len 12, message_len 16. C's struct board_msginfo is 32 bytes
	// with an 8-byte heading pointer and 4-byte tail padding
	// (src/boards.h:40-46): slot 0, level 16, heading_len 20, message_len 24.
	writeRow := func(rowLen int, levelOff int, slot int32) []byte {
		row := make([]byte, rowLen)
		binary.LittleEndian.PutUint32(row[0:4], uint32(int32u(int(slot)))) // slot_num
		binary.LittleEndian.PutUint32(row[levelOff:levelOff+4], 1)         // level
		binary.LittleEndian.PutUint32(row[levelOff+4:levelOff+8], int32u(len(heading)))
		binary.LittleEndian.PutUint32(row[levelOff+8:levelOff+12], uint32(messageLen))
		return row
	}

	fileBody := func(row []byte, msg string) []byte {
		out := make([]byte, 4, 4+len(row)+len(heading)+len(msg)+8)
		binary.LittleEndian.PutUint32(out[0:4], 1) // num_of_msgs = 1
		out = append(out, row...)
		out = append(out, heading...)
		if msg != "" {
			out = append(out, append([]byte(msg), 0x00)...)
		}
		return out
	}

	for _, side := range []struct {
		path string
		row  []byte
		msg  string
	}{
		// C: slot(4) pad(4) pointer(8) level(4) heading_len(4) message_len(4) tail pad(4) = 32.
		{cPath, writeRow(32, 16, int32(slot)), pickMsg(ff.Detail, "planted body")},
		// Go: slot(4) pad(4) level(4) heading_len(4) message_len(4) = 20.
		{goPath, writeRow(20, 8, int32(slot)), pickMsg(ff.Detail, "planted body")},
	} {
		if err := os.MkdirAll(filepath.Dir(side.path), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(side.path, fileBody(side.row, side.msg), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// crashFileBucket is C's first-letter player-file bucket (src/utils.c:555-578).
func crashFileBucket(name string) string {
	if name == "" {
		return "ZZZ"
	}
	switch strings.ToLower(name[:1]) {
	case "a", "b", "c", "d", "e":
		return "A-E"
	case "f", "g", "h", "i", "j":
		return "F-J"
	case "k", "l", "m", "n", "o":
		return "K-O"
	case "p", "q", "r", "s", "t":
		return "P-T"
	case "u", "v", "w", "x", "y", "z":
		return "U-Z"
	}
	return "ZZZ"
}

func pickMsg(detail, body string) string {
	if detail == "nul-heading" {
		return body
	}
	return ""
}

func int32u(v int) uint32 {
	if v < 0 {
		return 0
	}
	return uint32(v)
}

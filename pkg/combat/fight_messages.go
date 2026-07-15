package combat

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	miscdata "github.com/zax0rz/darkpawns/lib/misc"
)

// FightMessageAction contains the three audience-specific forms of one fight
// message. An empty field represents the C messages file's "#" sentinel and
// must not be sent.
type FightMessageAction struct {
	Attacker string
	Victim   string
	Room     string
}

// LoadEmbeddedFightMessages parses the canonical corpus embedded in the
// server binary, so runtime working-directory changes cannot disable combat
// flavor or alter RNG draw order.
func LoadEmbeddedFightMessages() (FightMessages, error) {
	messages, err := ParseFightMessages(bytes.NewReader(miscdata.FightMessages))
	if err != nil {
		return nil, fmt.Errorf("parse embedded fight messages: %w", err)
	}
	return messages, nil
}

// FightMessageVariant contains all four outcomes in one M record from
// lib/misc/messages.
type FightMessageVariant struct {
	Die  FightMessageAction
	Miss FightMessageAction
	Hit  FightMessageAction
	God  FightMessageAction
}

// FightMessages groups fight-message variants by attack type. Variants are in
// C selection order, which is the reverse of their order in the data file:
// load_messages() prepends every record to its attack type's linked list.
type FightMessages map[int][]FightMessageVariant

// Variants returns the C-ordered message variants for attackType.
func (messages FightMessages) Variants(attackType int) ([]FightMessageVariant, bool) {
	variants, ok := messages[attackType]
	return variants, ok
}

// LoadFightMessages opens and parses a CircleMUD misc/messages file.
func LoadFightMessages(path string) (FightMessages, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("open fight messages: %w", err)
	}
	defer func() { _ = file.Close() }()

	messages, err := ParseFightMessages(file)
	if err != nil {
		return nil, fmt.Errorf("parse fight messages %q: %w", path, err)
	}
	return messages, nil
}

// ParseFightMessages parses the format consumed by C's load_messages(). Each
// M record consists of an attack type followed by twelve action lines: the
// attacker, victim, and room forms for die, miss, hit, and god outcomes.
func ParseFightMessages(reader io.Reader) (FightMessages, error) {
	scanner := bufio.NewScanner(reader)
	lineNumber := 0
	nextLine := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		lineNumber++
		return strings.TrimSuffix(scanner.Text(), "\r"), true
	}

	messages := make(FightMessages)
	for {
		line, ok := nextLine()
		if !ok {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNumber+1, err)
			}
			return nil, fmt.Errorf("line %d: unexpected EOF before $ terminator", lineNumber+1)
		}

		if line == "" || line[0] == '*' {
			continue
		}
		if line[0] == '$' {
			return messages, nil
		}
		if line[0] != 'M' {
			return nil, fmt.Errorf("line %d: expected M record or $ terminator, got %q", lineNumber, line)
		}

		typeLine, ok := nextLine()
		if !ok {
			return nil, fmt.Errorf("line %d: unexpected EOF reading attack type", lineNumber+1)
		}
		attackType, err := strconv.Atoi(strings.TrimSpace(typeLine))
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid attack type %q: %w", lineNumber, typeLine, err)
		}

		actions := make([]string, 12)
		for i := range actions {
			action, actionOK := nextLine()
			if !actionOK {
				return nil, fmt.Errorf("line %d: unexpected EOF reading action %d for attack type %d", lineNumber+1, i+1, attackType)
			}
			if action != "#" {
				actions[i] = action
			}
		}

		variant := FightMessageVariant{
			Die:  FightMessageAction{Attacker: actions[0], Victim: actions[1], Room: actions[2]},
			Miss: FightMessageAction{Attacker: actions[3], Victim: actions[4], Room: actions[5]},
			Hit:  FightMessageAction{Attacker: actions[6], Victim: actions[7], Room: actions[8]},
			God:  FightMessageAction{Attacker: actions[9], Victim: actions[10], Room: actions[11]},
		}

		// C's load_messages() links each new record at the head, making dice(1,
		// number_of_attacks) traverse variants in reverse file order.
		variants := messages[attackType]
		variants = append(variants, FightMessageVariant{})
		copy(variants[1:], variants[:len(variants)-1])
		variants[0] = variant
		messages[attackType] = variants
	}
}

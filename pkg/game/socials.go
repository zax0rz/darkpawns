package game

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	miscdata "github.com/zax0rz/darkpawns/lib/misc"
)

// Social represents an emote/social command.
type Social struct {
	Name              string
	MinLevel          int // legacy generated name: C social hide flag
	HideFlag          int // legacy generated name: C minimum victim position
	MinVictimPosition int // explicit override for programmatically defined socials
	Messages          []string
}

func (s *Social) hidesInvisibleActor() bool {
	return s.MinLevel != 0
}

func (s *Social) minimumVictimPosition() int {
	if s.MinVictimPosition != 0 {
		return s.MinVictimPosition
	}
	return s.HideFlag
}

// Socials contains all the social emotes from the original Dark Pawns.
// It is parsed from the embedded canonical lib/misc/socials records.
var Socials = parseSocials(miscdata.Socials)

func parseSocials(data []byte) map[string]*Social {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	socials := make(map[string]*Social)
	lineNumber := 0
	terminated := false

	for scanner.Scan() {
		lineNumber++
		header := strings.TrimSpace(scanner.Text())
		if header == "" {
			continue
		}
		if header == "$" {
			terminated = true
			break
		}

		fields := strings.Fields(header)
		if len(fields) != 3 {
			panic(fmt.Sprintf("invalid social header at line %d: %q", lineNumber, header))
		}
		minLevel, err := strconv.Atoi(fields[1])
		if err != nil {
			panic(fmt.Sprintf("invalid social hide flag at line %d: %q: %v", lineNumber, header, err))
		}
		minVictimPosition, err := strconv.Atoi(fields[2])
		if err != nil {
			panic(fmt.Sprintf("invalid social victim position at line %d: %q: %v", lineNumber, header, err))
		}

		messages := make([]string, 0, 8)
		for len(messages) < 3 || (messages[2] != "#" && len(messages) < 8) {
			if !scanner.Scan() {
				panic(fmt.Sprintf("social %q ended before its messages", fields[0]))
			}
			lineNumber++
			messages = append(messages, scanner.Text())
		}
		if _, exists := socials[fields[0]]; exists {
			panic(fmt.Sprintf("duplicate social %q", fields[0]))
		}
		socials[fields[0]] = &Social{
			Name:     fields[0],
			MinLevel: minLevel,
			HideFlag: minVictimPosition,
			Messages: messages,
		}
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Sprintf("reading embedded socials: %v", err))
	}
	if !terminated {
		panic("embedded socials are missing the terminal $ record")
	}
	return socials
}

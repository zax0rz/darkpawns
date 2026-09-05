package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

var fixtureDirectionNumber = map[string]int{
	"north": 0,
	"east":  1,
	"south": 2,
	"west":  3,
	"up":    4,
	"down":  5,
}

func applyRoomFixtures(worldDir string, exits []oraclediff.RoomExitFixture, flags []oraclediff.RoomFlagFixture, sectors []oraclediff.RoomSectorFixture) error {
	for _, fixture := range exits {
		if err := rewriteRoomRecord(worldDir, fixture.RoomVNum, func(record string) (string, error) {
			return replaceRoomExits(record, fixture)
		}); err != nil {
			return err
		}
	}
	for _, fixture := range flags {
		if err := rewriteRoomRecord(worldDir, fixture.RoomVNum, func(record string) (string, error) {
			return setRoomFlag(record, fixture)
		}); err != nil {
			return err
		}
	}
	for _, fixture := range sectors {
		if err := rewriteRoomRecord(worldDir, fixture.RoomVNum, func(record string) (string, error) {
			return setRoomSector(record, fixture)
		}); err != nil {
			return err
		}
	}
	return nil
}

func rewriteRoomRecord(worldDir string, roomVNum int, rewrite func(string) (string, error)) error {
	paths, err := filepath.Glob(filepath.Join(worldDir, "wld", "*.wld"))
	if err != nil {
		return fmt.Errorf("list room files: %w", err)
	}
	sort.Strings(paths)
	marker := fmt.Sprintf("#%d\n", roomVNum)
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read room file %s: %w", path, readErr)
		}
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		start := strings.Index(text, marker)
		if start < 0 || (start > 0 && text[start-1] != '\n') {
			continue
		}
		rest := text[start+len(marker):]
		endOffset := strings.Index(rest, "\n#")
		end := len(text)
		if endOffset >= 0 {
			end = start + len(marker) + endOffset + 1
		}
		updatedRecord, rewriteErr := rewrite(text[start:end])
		if rewriteErr != nil {
			return fmt.Errorf("rewrite room %d: %w", roomVNum, rewriteErr)
		}
		updated := text[:start] + updatedRecord + text[end:]
		info, statErr := os.Stat(path)
		if statErr != nil {
			return fmt.Errorf("stat room file %s: %w", path, statErr)
		}
		if writeErr := os.WriteFile(path, []byte(updated), info.Mode().Perm()); writeErr != nil { // #nosec G703 -- disposable oracle fixture path resolved from the trusted world tree
			return fmt.Errorf("write room file %s: %w", path, writeErr)
		}
		return nil
	}
	return fmt.Errorf("room %d not found under %s", roomVNum, filepath.Join(worldDir, "wld"))
}

func replaceRoomExits(record string, fixture oraclediff.RoomExitFixture) (string, error) {
	lines := strings.Split(strings.TrimSuffix(record, "\n"), "\n")
	header, err := roomHeaderLine(lines)
	if err != nil {
		return "", err
	}
	body := make([]string, 0, len(lines)-header)
	for i := header + 1; i < len(lines); {
		if len(lines[i]) == 2 && lines[i][0] == 'D' && lines[i][1] >= '0' && lines[i][1] <= '5' {
			next, consumeErr := consumeExitBlock(lines, i)
			if consumeErr != nil {
				return "", consumeErr
			}
			i = next
			continue
		}
		body = append(body, lines[i])
		i++
	}

	result := append([]string(nil), lines[:header+1]...)
	fixtureDirections := []string{fixture.Direction}
	if fixture.Direction == "all" {
		fixtureDirections = []string{"north", "east", "south", "west", "up", "down"}
	}
	for _, fixtureDirection := range fixtureDirections {
		if fixtureDirection == "" {
			continue
		}
		direction, ok := fixtureDirectionNumber[fixtureDirection]
		if !ok {
			return "", fmt.Errorf("invalid direction %q", fixtureDirection)
		}
		result = append(
			result,
			fmt.Sprintf("D%d", direction),
			"~",
			fixture.Keyword+"~",
			fmt.Sprintf("%d -1 %d", fixture.DoorState, fixture.ToRoom),
		)
	}
	result = append(result, body...)
	return strings.Join(result, "\n") + "\n", nil
}

func setRoomSector(record string, fixture oraclediff.RoomSectorFixture) (string, error) {
	lines := strings.Split(strings.TrimSuffix(record, "\n"), "\n")
	header, err := roomHeaderLine(lines)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(lines[header])
	if len(fields) < 6 {
		return "", fmt.Errorf("room header has %d fields, want at least 6", len(fields))
	}
	fields[5] = strconv.Itoa(fixture.Sector)
	lines[header] = strings.Join(fields, " ")
	return strings.Join(lines, "\n") + "\n", nil
}

func setRoomFlag(record string, fixture oraclediff.RoomFlagFixture) (string, error) {
	lines := strings.Split(strings.TrimSuffix(record, "\n"), "\n")
	header, err := roomHeaderLine(lines)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(lines[header])
	if len(fields) < 6 {
		return "", fmt.Errorf("room header has %d fields, want at least 6", len(fields))
	}
	word := fixture.Bit / 16
	bit := uint(fixture.Bit % 16)
	value, parseErr := strconv.ParseUint(fields[word+1], 10, 64)
	if parseErr != nil {
		return "", fmt.Errorf("parse flag word %q: %w", fields[word+1], parseErr)
	}
	if fixture.Enabled {
		value |= 1 << bit
	} else {
		value &^= 1 << bit
	}
	fields[word+1] = strconv.FormatUint(value, 10)
	lines[header] = strings.Join(fields, " ")
	return strings.Join(lines, "\n") + "\n", nil
}

func roomHeaderLine(lines []string) (int, error) {
	tildeCount := 0
	for i := 1; i < len(lines); i++ {
		if strings.HasSuffix(lines[i], "~") {
			tildeCount++
			if tildeCount == 2 {
				if i+1 >= len(lines) {
					break
				}
				return i + 1, nil
			}
		}
	}
	return 0, fmt.Errorf("room name/description terminators not found")
}

func consumeExitBlock(lines []string, start int) (int, error) {
	i := start + 1
	for range 2 {
		found := false
		for i < len(lines) {
			line := lines[i]
			i++
			if strings.HasSuffix(line, "~") {
				found = true
				break
			}
		}
		if !found {
			return 0, fmt.Errorf("unterminated exit string after %q", lines[start])
		}
	}
	if i >= len(lines) {
		return 0, fmt.Errorf("missing exit values after %q", lines[start])
	}
	return i + 1, nil
}

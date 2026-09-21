package olc

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// NewZoneMax is the upper bound in zedit_new_zone (src/zedit.c:116).
const NewZoneMax = 326

type NewZoneFile struct {
	Extension string
	Contents  string
}

// NewZoneFiles is the C zedit_new_zone file set. Keep the strings here so the
// telnet and web creation paths share the same byte-level defaults.
func NewZoneFiles(number int) []NewZoneFile {
	start := number * 100
	return []NewZoneFile{
		{Extension: "zon", Contents: fmt.Sprintf("#%d\nNew Zone~\n%d 30 2\nS\n$\n", number, start+99)},
		{Extension: "wld", Contents: fmt.Sprintf("#%d\nThe Begining~\nNot much here.\n~\n%d 0 0\nS\n$\n", start, number)},
		{Extension: "mob", Contents: "$\n"},
		{Extension: "obj", Contents: "$\n"},
		{Extension: "shp", Contents: "$~\n"},
	}
}

func WriteNewZoneFiles(worldPath string, number int) error {
	for _, file := range NewZoneFiles(number) {
		path := filepath.Join(worldPath, file.Extension, fmt.Sprintf("%d.%s", number, file.Extension))
		if err := os.WriteFile(filepath.Clean(path), []byte(file.Contents), 0o666); err != nil {
			return fmt.Errorf("write new zone %s: %w", path, err)
		}
	}
	return nil
}

// UpdateNewZoneIndex inserts the new file before the first existing entry
// with a greater number, matching zedit_create_index's sorted insertion.
func UpdateNewZoneIndex(worldPath string, number int, extension string) error {
	directory := filepath.Join(worldPath, extension)
	oldPath := filepath.Clean(filepath.Join(directory, "index"))
	newPath := filepath.Clean(filepath.Join(directory, "newindex"))
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("SYSERR: OLC: Failed to open %s", oldPath)
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	var out strings.Builder
	found := false
	for _, line := range lines {
		if line == "" {
			continue
		}
		if line == "$" {
			if !found {
				fmt.Fprintf(&out, "%d.%s\n", number, extension)
			}
			out.WriteString("$\n")
			break
		}
		if !found {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				if existing, parseErr := strconv.Atoi(fields[0]); parseErr == nil && existing >= number {
					found = true
					if existing > number {
						fmt.Fprintf(&out, "%d.%s\n", number, extension)
					}
				}
			}
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	if err := os.WriteFile(newPath, []byte(out.String()), 0o666); err != nil {
		return err
	}
	if err := os.Remove(oldPath); err != nil {
		_ = os.Remove(newPath)
		return err
	}
	return os.Rename(newPath, oldPath)
}

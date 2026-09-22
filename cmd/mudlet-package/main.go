// Command mudlet-package assembles mudlet/darkpawns.xml, the importable
// Mudlet package, from the Lua sources in mudlet/src.
//
//	go run ./cmd/mudlet-package          # rewrite mudlet/darkpawns.xml
//	go run ./cmd/mudlet-package -check   # fail if it is out of date
package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	root := flag.String("root", "mudlet", "package directory (holds VERSION and src/)")
	check := flag.Bool("check", false, "report whether darkpawns.xml is current instead of writing it")
	flag.Parse()

	built, err := Build(*root)
	if err != nil {
		slog.Error("build Mudlet package", "error", err)
		os.Exit(1)
	}
	out := filepath.Join(*root, "darkpawns.xml")
	if *check {
		current, err := os.ReadFile(filepath.Clean(out))
		if err != nil || !bytes.Equal(current, built) {
			slog.Error("Mudlet package is out of date; run go run ./cmd/mudlet-package", "file", out)
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(filepath.Clean(out), built, 0o644); err != nil { // #nosec G306 -- a published, world-readable artifact
		slog.Error("write Mudlet package", "file", out, "error", err)
		os.Exit(1)
	}
	fmt.Println("wrote", out)
}

// Script is one Lua source file, named for the Mudlet script list.
type Script struct {
	Name   string
	Source string
}

// Scripts reads root/src/*.lua in name order, with {{VERSION}} substituted.
func Scripts(root string) (version string, scripts []Script, err error) {
	raw, err := os.ReadFile(filepath.Clean(filepath.Join(root, "VERSION")))
	if err != nil {
		return "", nil, err
	}
	version = strings.TrimSpace(string(raw))
	paths, err := filepath.Glob(filepath.Join(root, "src", "*.lua"))
	if err != nil {
		return "", nil, err
	}
	if len(paths) == 0 {
		return "", nil, fmt.Errorf("no Lua sources in %s", filepath.Join(root, "src"))
	}
	sort.Strings(paths)
	for _, path := range paths {
		source, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return "", nil, err
		}
		name := strings.TrimSuffix(filepath.Base(path), ".lua")
		if _, rest, ok := strings.Cut(name, "-"); ok {
			name = rest
		}
		scripts = append(scripts, Script{
			Name:   "Dark Pawns " + name,
			Source: strings.ReplaceAll(string(source), "{{VERSION}}", version),
		})
	}
	return version, scripts, nil
}

// Build renders the package XML in the layout Mudlet exports.
func Build(root string) ([]byte, error) {
	version, scripts, err := Scripts(root)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE MudletPackage>\n<MudletPackage version=\"1.001\">\n")
	b.WriteString("\t<TriggerPackage />\n\t<TimerPackage />\n")
	b.WriteString("\t<AliasPackage>\n")
	b.WriteString("\t\t<Alias isActive=\"yes\" isFolder=\"no\">\n")
	writeElement(&b, 3, "name", "dp")
	writeElement(&b, 3, "script", "DarkPawns.command(matches[2])")
	writeElement(&b, 3, "command", "")
	writeElement(&b, 3, "packageName", "")
	writeElement(&b, 3, "regex", `^dp(?:\s+(\w+))?$`)
	b.WriteString("\t\t</Alias>\n\t</AliasPackage>\n")
	b.WriteString("\t<ActionPackage />\n\t<ScriptPackage>\n")
	b.WriteString("\t\t<ScriptGroup isActive=\"yes\" isFolder=\"yes\">\n")
	writeElement(&b, 3, "name", "Dark Pawns "+version)
	writeElement(&b, 3, "packageName", "")
	writeElement(&b, 3, "script", "")
	b.WriteString("\t\t\t<eventHandlerList />\n")
	for _, script := range scripts {
		b.WriteString("\t\t\t<Script isActive=\"yes\" isFolder=\"no\">\n")
		writeElement(&b, 4, "name", script.Name)
		writeElement(&b, 4, "packageName", "")
		writeElement(&b, 4, "script", script.Source)
		b.WriteString("\t\t\t\t<eventHandlerList />\n\t\t\t</Script>\n")
	}
	b.WriteString("\t\t</ScriptGroup>\n\t</ScriptPackage>\n")
	b.WriteString("\t<KeyPackage />\n\t<HelpPackage>\n")
	writeElement(&b, 2, "helpURL", "https://github.com/zax0rz/darkpawns/tree/main/mudlet")
	b.WriteString("\t</HelpPackage>\n</MudletPackage>\n")
	return b.Bytes(), nil
}

func writeElement(b *bytes.Buffer, depth int, name, text string) {
	b.WriteString(strings.Repeat("\t", depth))
	b.WriteString("<" + name + ">")
	var escaped bytes.Buffer
	if err := xml.EscapeText(&escaped, []byte(text)); err != nil {
		panic(err) // EscapeText only fails on a failing writer; bytes.Buffer never does
	}
	// EscapeText encodes newlines as &#xA;; Mudlet exports them literally, and
	// literal newlines keep the package readable in a diff.
	b.WriteString(strings.ReplaceAll(escaped.String(), "&#xA;", "\n"))
	b.WriteString("</" + name + ">\n")
}

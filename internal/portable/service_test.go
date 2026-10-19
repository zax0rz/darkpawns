package portable

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestNativeServiceDefinitionsPreserveArguments(t *testing.T) {
	launcher := `/release with space/$HOME%/darkpawns`
	instance := `/world with <tags> & "quotes"/world`
	for _, platform := range []string{"linux", "darwin", "windows"} {
		data, err := renderService(platform, "darkpawns-test", launcher, instance, "operator")
		if err != nil {
			t.Fatal(err)
		}
		if platform == "linux" {
			text := string(data)
			if !strings.Contains(text, "$$HOME%%") || !strings.Contains(text, `\"quotes\"`) || !strings.Contains(text, "KillMode=mixed") {
				t.Fatal(text)
			}
			continue
		}
		decoder := xml.NewDecoder(bytes.NewReader(data))
		var text []string
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			if chars, ok := token.(xml.CharData); ok {
				text = append(text, string(chars))
			}
		}
		joined := strings.Join(text, "\n")
		if platform == "windows" {
			var task struct {
				Actions struct {
					Exec struct {
						Command   string
						Arguments string
					}
				}
			}
			if err := xml.Unmarshal(data, &task); err != nil {
				t.Fatal(err)
			}
			encoded := strings.TrimPrefix(task.Actions.Exec.Arguments, "-NoProfile -NonInteractive -WindowStyle Hidden -EncodedCommand ")
			data, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Fatal(err)
			}
			words := make([]uint16, len(data)/2)
			for i := range words {
				words[i] = binary.LittleEndian.Uint16(data[2*i:])
			}
			script := string(utf16.Decode(words))
			if !strings.Contains(script, launcher) || !strings.Contains(script, instance) {
				t.Fatal(script)
			}
			if strings.Contains(task.Actions.Exec.Arguments, "%") || !strings.HasSuffix(task.Actions.Exec.Command, `\powershell.exe`) {
				t.Fatal(task)
			}
		} else if !strings.Contains(joined, launcher) || !strings.Contains(joined, instance) {
			t.Fatal(joined)
		}
	}
}

func TestServiceOwnershipRejectsChangedDefinition(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux file-backed registration unit")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	instance, err := canonicalInstance(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	name := serviceName(instance)
	path, err := servicePath(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeExclusive(path, []byte("external edit")); err != nil {
		t.Fatal(err)
	}
	record := serviceRecord{Platform: runtime.GOOS, Name: name, Path: path, Launcher: filepath.Join(instance, "launcher"), SHA256: strings.Repeat("0", 64)}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instance, ".service.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ServiceInstalled(instance); err == nil {
		t.Fatal("modified definition treated as owned")
	}
}

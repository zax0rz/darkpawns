package portable

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
		if !strings.Contains(joined, launcher) {
			t.Fatal(joined)
		}
		want := instance
		if platform == "windows" {
			want = windowsArg(instance)
		}
		if !strings.Contains(joined, want) {
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

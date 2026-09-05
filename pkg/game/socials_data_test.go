package game

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type oracleSocial struct {
	hideFlag int
	minPos   int
	messages []string
}

func TestSocialTableMatchesCData(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	oracle, err := readOracleSocials(filepath.Join(filepath.Dir(filename), "../../lib/misc/socials"))
	if err != nil {
		t.Fatal(err)
	}
	if len(Socials) != len(oracle) {
		t.Fatalf("social count mismatch: Go=%d C=%d", len(Socials), len(oracle))
	}

	for name, expected := range oracle {
		actual, ok := Socials[name]
		if !ok {
			t.Errorf("Go table is missing C social %q", name)
			continue
		}
		if actual.MinLevel != expected.hideFlag {
			t.Errorf("%s hide flag: Go=%d C=%d", name, actual.MinLevel, expected.hideFlag)
		}
		if actual.HideFlag != expected.minPos {
			t.Errorf("%s minimum victim position: Go=%d C=%d", name, actual.HideFlag, expected.minPos)
		}

		if len(actual.Messages) != len(expected.messages) {
			t.Errorf("%s message count: Go=%d C=%d", name, len(actual.Messages), len(expected.messages))
			continue
		}
		for index, want := range expected.messages {
			if actual.Messages[index] != want {
				t.Errorf("%s message %d: Go=%q C=%q", name, index, actual.Messages[index], want)
			}
		}
	}
}

func TestSocialTargetLookupHonorsVisibility(t *testing.T) {
	w, actor, target, _, _ := newChannelWorld(t)
	target.SetAffect(affInvisible, true)

	if got := w.findSocialTarget(actor, target.Name); got != nil {
		t.Fatalf("invisible target resolved as %q", got.GetName())
	}
	actor.SetAffect(affDetectInvisible, true)
	if got := w.findSocialTarget(actor, target.Name); got != target {
		t.Fatalf("detect-invisible actor did not resolve target: %v", got)
	}
}

func readOracleSocials(path string) (map[string]oracleSocial, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	socials := make(map[string]oracleSocial)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		header := scanner.Text()
		if strings.TrimSpace(header) == "" {
			continue
		}
		if strings.TrimSpace(header) == "$" {
			break
		}
		fields := strings.Fields(header)
		if len(fields) != 3 {
			return nil, &socialParseError{line: header}
		}

		var hideFlag, minPos int
		if _, err := fmt.Sscanf(fields[1], "%d", &hideFlag); err != nil {
			return nil, err
		}
		if _, err := fmt.Sscanf(fields[2], "%d", &minPos); err != nil {
			return nil, err
		}

		messages := make([]string, 0, 8)
		for len(messages) < 3 || (messages[2] != "#" && len(messages) < 8) {
			if !scanner.Scan() {
				return nil, scanner.Err()
			}
			messages = append(messages, scanner.Text())
		}
		socials[fields[0]] = oracleSocial{hideFlag: hideFlag, minPos: minPos, messages: messages}
	}
	return socials, scanner.Err()
}

type socialParseError struct {
	line string
}

func (e *socialParseError) Error() string {
	return "invalid social header: " + e.line
}

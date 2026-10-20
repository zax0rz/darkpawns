package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpDoesNotPrintDispatchParserUsage(t *testing.T) {
	for _, arg := range []string{"--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			root := command()
			var out, errOut bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&errOut)
			forwarded, err := forwardActive(root, []string{arg})
			if err != nil || forwarded {
				t.Fatalf("dispatch: forwarded=%v err=%v", forwarded, err)
			}
			root.SetArgs([]string{arg})
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if errOut.Len() != 0 {
				t.Fatalf("stray dispatch output: %s", errOut.String())
			}
			if !strings.Contains(out.String(), "darkpawns [command]") {
				t.Fatalf("missing actual help: %s", out.String())
			}
		})
	}
}

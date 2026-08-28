package game

import (
	"reflect"
	"testing"
)

func TestExpandAliasDepthMatchesCBranches(t *testing.T) {
	tests := []struct {
		name    string
		alias   Alias
		command string
		want    []string
	}{
		{
			name:    "simple discards original arguments",
			alias:   Alias{Alias: "l", Replacement: " look", Type: AliasSimple},
			command: "l nowhere",
			want:    []string{" look"},
		},
		{
			name:    "semicolon splits commands",
			alias:   Alias{Alias: "c", Replacement: " alias;alias", Type: AliasComplex},
			command: "c",
			want:    []string{" alias", "alias"},
		},
		{
			name:    "positional token",
			alias:   Alias{Alias: "p", Replacement: " say $1", Type: AliasComplex},
			command: "p hello world",
			want:    []string{" say hello"},
		},
		{
			name:    "glob token",
			alias:   Alias{Alias: "g", Replacement: " say $*", Type: AliasComplex},
			command: "g hello world",
			want:    []string{" say  hello world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExpandAlias([]Alias{tt.alias}, tt.command)
			if !ok {
				t.Fatal("ExpandAlias did not find the configured alias")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ExpandAlias(%q) = %#v, want %#v", tt.command, got, tt.want)
			}
		})
	}
}

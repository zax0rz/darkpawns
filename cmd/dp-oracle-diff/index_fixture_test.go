package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

func TestApplyIndexFixturesInsertsEachFileInNumericOrder(t *testing.T) {
	tests := []struct {
		name      string
		directory string
		index     string
		fixtures  []string
		want      string
		apply     func(string, []string) error
	}{
		{
			name:      "object",
			directory: "obj",
			index:     "10.obj\n50.obj\n100.obj\n$\n",
			fixtures:  []string{"75.obj", "25.obj"},
			want:      "10.obj\n25.obj\n50.obj\n75.obj\n100.obj\n$\n",
			apply: func(worldDir string, names []string) error {
				fixtures := make([]oraclediff.ObjIndexFixture, 0, len(names))
				for _, name := range names {
					fixtures = append(fixtures, oraclediff.ObjIndexFixture{FileName: name})
				}
				return applyObjIndexFixtures(worldDir, fixtures)
			},
		},
		{
			name:      "world",
			directory: "wld",
			index:     "100.wld\n200.wld\n300.wld\n$\n",
			fixtures:  []string{"250.wld", "150.wld"},
			want:      "100.wld\n150.wld\n200.wld\n250.wld\n300.wld\n$\n",
			apply: func(worldDir string, names []string) error {
				fixtures := make([]oraclediff.WldIndexFixture, 0, len(names))
				for _, name := range names {
					fixtures = append(fixtures, oraclediff.WldIndexFixture{FileName: name})
				}
				return applyWldIndexFixtures(worldDir, fixtures)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worldDir := t.TempDir()
			indexDir := filepath.Join(worldDir, tt.directory)
			if err := os.Mkdir(indexDir, 0o750); err != nil {
				t.Fatal(err)
			}
			indexPath := filepath.Join(indexDir, "index")
			if err := os.WriteFile(indexPath, []byte(tt.index), 0o600); err != nil {
				t.Fatal(err)
			}

			if err := tt.apply(worldDir, tt.fixtures); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("index contents:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

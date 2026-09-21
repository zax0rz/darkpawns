package admin

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/olc"
)

func TestOLCSchemaGoldens(t *testing.T) {
	for _, kind := range []string{"room", "mob", "obj", "shop", "zone"} {
		t.Run(kind, func(t *testing.T) {
			schema, ok := olc.SchemaForKind(kind)
			if !ok {
				t.Fatalf("schema kind %q is not registered", kind)
			}
			got, err := json.MarshalIndent(schema, "", "  ")
			if err != nil {
				t.Fatalf("marshal schema: %v", err)
			}
			got = append(got, '\n')

			path := filepath.Join("testdata", "olc-schema-"+kind+".json")
			if os.Getenv("UPDATE_OLC_SCHEMA_GOLDENS") == "1" {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatalf("write schema golden: %v", err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read schema golden %s: %v (regenerate with UPDATE_OLC_SCHEMA_GOLDENS=1)", path, err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("schema changed; update %s deliberately with UPDATE_OLC_SCHEMA_GOLDENS=1", path)
			}
		})
	}
}

func TestOLCSchemaOmitsFaithfulNoOps(t *testing.T) {
	schema, ok := olc.SchemaForKind("obj")
	if !ok {
		t.Fatal("object schema is not registered")
	}
	for _, field := range schema.Fields {
		if field.Key == "level" || field.Key == "timer" || field.Key == "set_level" || field.Key == "set_timer" {
			t.Errorf("object schema exposes dead field %q", field.Key)
		}
	}
}

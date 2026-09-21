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

func TestOLCSchemaP9Descriptors(t *testing.T) {
	mob, ok := olc.SchemaForKind("mob")
	if !ok {
		t.Fatal("mob schema is not registered")
	}
	var actionFlagStorage string
	hasNoise := false
	for _, field := range mob.Fields {
		if field.Key == "noise" {
			hasNoise = true
		}
		if field.Key == "action_flags" && len(field.Options) > 5 {
			actionFlagStorage = field.Options[5].Storage
		}
	}
	if !hasNoise {
		t.Fatal("mob schema omits noise")
	}
	if actionFlagStorage != "AGGRESSIVE" {
		t.Fatalf("action flag storage = %q, want AGGRESSIVE", actionFlagStorage)
	}

	object, ok := olc.SchemaForKind("obj")
	if !ok || object.Applies == nil {
		t.Fatal("object schema omits applies descriptor")
	}
	if object.Applies.Max != 6 || object.Applies.AddOperation != "add_affect" || object.Applies.RemoveOperation != "remove_affect" {
		t.Fatalf("applies descriptor = %+v", object.Applies)
	}
	if len(object.Applies.Options) != len(olc.ApplyTypeNames) {
		t.Fatalf("apply options = %d, want %d", len(object.Applies.Options), len(olc.ApplyTypeNames))
	}

	room, ok := olc.SchemaForKind("room")
	if !ok || room.Exits == nil {
		t.Fatal("room schema omits exit descriptor")
	}
	if got, want := len(room.Exits.DoorOptions), 3; got != want {
		t.Fatalf("door options = %d, want %d", got, want)
	}
	if got, want := room.Exits.DoorOptions[1].Label, "Closeable door"; got != want {
		t.Fatalf("closeable door label = %q, want %q", got, want)
	}
}

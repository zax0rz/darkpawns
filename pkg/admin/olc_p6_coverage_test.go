package admin

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestOLCP6WireCoverage is the mechanical census for the write contract. The
// expected side is the telnet menu census (field names are deliberately
// boring, because the labels remain pkg/session's concern); the source side
// is extracted from the actual admin switch statements. A new wire case or a
// new census entry therefore fails until both sides are reviewed together.
func TestOLCP6WireCoverage(t *testing.T) {
	room := []string{
		"set_room_name", "set_room_description", "set_room_flag", "set_room_sector",
		"set_script_name", "set_script_flag", "ensure_exit", "set_exit_target",
		"set_exit_description", "set_exit_keywords", "set_exit_key", "set_exit_door_flags",
		"purge_exit", "add_extra_description", "remove_extra_description", "set_extra_keyword",
		"set_extra_description", "copy_room",
	}
	mob := []string{
		"set_keywords", "set_short_description", "set_long_description", "set_detailed_description",
		"set_sex", "set_hitroll", "set_damroll", "set_damage_dice", "set_damage_sides",
		"set_hp_dice", "set_hp_sides", "set_hp_plus", "set_ac", "set_exp", "set_gold",
		"set_position", "set_default_position", "set_attack", "set_level", "set_alignment",
		"set_race", "set_action_flag", "set_affect_flag", "set_noise", "set_script_name",
		"set_script_flag",
	}
	object := []string{
		"set_keywords", "set_short_description", "set_long_description", "set_action_description",
		"set_value1", "set_value2", "set_value3", "set_value4", "toggle_container_flag",
		"set_type", "set_extra_flag", "set_wear_flag", "set_weight", "set_cost", "set_cost_per_day",
		"set_timer", "set_level", "set_script_name", "set_script_flag", "add_affect", "remove_affect",
		"add_extra_description", "remove_extra_description", "set_extra_keywords", "set_extra_description",
	}
	shop := []string{
		"add_product", "remove_product", "set_buy_profit", "set_sell_profit", "set_keeper",
		"set_flags", "set_with_who", "add_room", "remove_room", "set_open_hour1", "set_open_hour2",
		"set_close_hour1", "set_close_hour2", "set_no_item1", "set_no_item2", "set_no_buy",
		"set_no_cash1", "set_no_cash2", "set_buy_message", "set_sell_message", "add_namelist",
		"remove_namelist", "set_no_trade",
	}
	zone := []string{
		"set_name", "set_lifespan", "set_reset_mode", "set_top_room", "add_command",
		"modify_command", "remove_command", "reorder_command",
	}
	want := map[string][]string{"room": room, "mob": mob, "obj": object, "shop": shop, "zone": zone}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(thisFile)
	got := map[string][]string{
		"room": sourceSwitchCases(t, filepath.Join(root, "olc_write.go"), "roomOperation"),
		"mob":  sourceEntityCases(t, filepath.Join(root, "olc_entity_write.go"), "mob"),
		"obj":  sourceEntityCases(t, filepath.Join(root, "olc_entity_write.go"), "obj"),
		"shop": sourceEntityCases(t, filepath.Join(root, "olc_entity_write.go"), "shop"),
		"zone": sourceEntityCases(t, filepath.Join(root, "olc_entity_write.go"), "zone"),
	}
	for editor, expected := range want {
		t.Run(editor, func(t *testing.T) {
			assertSameStrings(t, expected, got[editor])
		})
	}
}

func TestOLCP6NewWireNamesMapToSemanticOperations(t *testing.T) {
	entityCases := map[string]map[string]olc.OperationKind{
		"mob": {
			"set_action_flag": olc.OpSetMobActionFlag, "set_affect_flag": olc.OpSetMobAffectFlag,
			"set_noise": olc.OpSetMobNoise, "set_script_name": olc.OpSetMobScriptName,
			"set_script_flag": olc.OpSetMobScriptFlag,
		},
		"obj": {
			"set_type": olc.OpSetObjType, "set_extra_flag": olc.OpSetObjExtraFlag,
			"set_wear_flag": olc.OpSetObjWearFlag, "set_weight": olc.OpSetObjWeight,
			"set_cost": olc.OpSetObjCost, "set_cost_per_day": olc.OpSetObjCostPerDay,
			"set_timer": olc.OpSetObjTimer, "set_level": olc.OpSetObjLevel,
			"set_script_name": olc.OpSetObjScriptName, "set_script_flag": olc.OpSetObjScriptFlag,
		},
		"shop": {
			"set_no_item1": olc.OpSetShopMessage, "set_no_item2": olc.OpSetShopMessage,
			"set_no_buy": olc.OpSetShopMessage, "set_no_cash1": olc.OpSetShopMessage,
			"set_no_cash2": olc.OpSetShopMessage, "set_buy_message": olc.OpSetShopMessage,
			"set_sell_message": olc.OpSetShopMessage, "add_namelist": olc.OpAddShopNamelist,
			"remove_namelist": olc.OpRemoveShopNamelist, "set_no_trade": olc.OpSetShopNoTrade,
		},
	}
	for entity, cases := range entityCases {
		for wire, want := range cases {
			t.Run(entity+"/"+wire, func(t *testing.T) {
				got, err := entityOperation(map[string]olc.Kind{"mob": olc.KindMob, "obj": olc.KindObject, "shop": olc.KindShop}[entity], entityPatchOperation{Kind: wire})
				if err != nil {
					t.Fatal(err)
				}
				if got.Kind != want {
					t.Fatalf("wire kind = %d, want %d", got.Kind, want)
				}
			})
		}
	}

	roomCases := map[string]olc.OperationKind{
		"set_script_name": olc.OpSetRoomScriptName,
		"set_script_flag": olc.OpSetRoomScriptFlag,
	}
	for wire, want := range roomCases {
		t.Run("room/"+wire, func(t *testing.T) {
			got, err := roomOperation((*game.World)(nil), 1001, roomPatchOperation{Kind: wire})
			if err != nil {
				t.Fatal(err)
			}
			if got.Kind != want {
				t.Fatalf("wire kind = %d, want %d", got.Kind, want)
			}
		})
	}
}

func TestOLCP6ScriptOperationsWriteLiveWorld(t *testing.T) {
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 1101, ScriptName: "mob-old"}},
		Objs:  []parser.Obj{{VNum: 1201, ScriptName: "obj-old"}},
		Zones: []parser.Zone{{Number: 1, TopRoom: 1999}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(world.StopAITicker)

	mobName, err := entityOperation(olc.KindMob, entityPatchOperation{Kind: "set_script_name"})
	if err != nil {
		t.Fatal(err)
	}
	bindEntityLiveScriptOperation(world, olc.KindMob, 1101, &mobName)
	mobName.Text = "mob-live"
	if err := olc.Apply(mobName); err != nil {
		t.Fatal(err)
	}
	mobFlag, err := entityOperation(olc.KindMob, entityPatchOperation{Kind: "set_script_flag", Bit: 1, Enabled: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	bindEntityLiveScriptOperation(world, olc.KindMob, 1101, &mobFlag)
	if err := olc.Apply(mobFlag); err != nil {
		t.Fatal(err)
	}
	if mob, ok := world.SnapshotMob(1101); !ok || mob.ScriptName != "mob-live" || mob.LuaFunctions != 2 {
		t.Fatalf("live mob script = %+v, exists=%v", mob, ok)
	}

	objName, err := entityOperation(olc.KindObject, entityPatchOperation{Kind: "set_script_name"})
	if err != nil {
		t.Fatal(err)
	}
	bindEntityLiveScriptOperation(world, olc.KindObject, 1201, &objName)
	objName.Text = "obj-live"
	if err := olc.Apply(objName); err != nil {
		t.Fatal(err)
	}
	objFlag, err := entityOperation(olc.KindObject, entityPatchOperation{Kind: "set_script_flag", Bit: 2, Enabled: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	bindEntityLiveScriptOperation(world, olc.KindObject, 1201, &objFlag)
	if err := olc.Apply(objFlag); err != nil {
		t.Fatal(err)
	}
	if object, ok := world.SnapshotObj(1201); !ok || object.ScriptName != "obj-live" || object.LuaFunctions != 4 {
		t.Fatalf("live object script = %+v, exists=%v", object, ok)
	}

	roomName, err := roomOperation(world, 1001, roomPatchOperation{Kind: "set_script_name", Text: "room-live"})
	if err != nil {
		t.Fatal(err)
	}
	if err := olc.Apply(roomName); err != nil {
		t.Fatal(err)
	}
	roomFlag, err := roomOperation(world, 1001, roomPatchOperation{Kind: "set_script_flag", Bit: 4, Enabled: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	if err := olc.Apply(roomFlag); err != nil {
		t.Fatal(err)
	}
	if room, ok := world.SnapshotRoom(1001); !ok || room.ScriptName != "room-live" || room.ScriptFunctions != 16 {
		t.Fatalf("live room script = %+v, exists=%v", room, ok)
	}
}

func boolPtr(value bool) *bool { return &value }

func sourceSwitchCases(t *testing.T, path, function string) []string {
	t.Helper()
	file := parseSourceFile(t, path)
	var got []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != function {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			clause, ok := node.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, expr := range clause.List {
				if literal, ok := expr.(*ast.BasicLit); ok && literal.Kind == token.STRING {
					got = append(got, strings.Trim(literal.Value, `"`))
				}
			}
			return true
		})
	}
	return uniqueSorted(got)
}

func sourceEntityCases(t *testing.T, path, entity string) []string {
	t.Helper()
	file := parseSourceFile(t, path)
	entityKind := map[string]string{"mob": "KindMob", "obj": "KindObject", "shop": "KindShop", "zone": "KindZone"}[entity]
	var got []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "entityOperation" {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			clause, ok := node.(*ast.CaseClause)
			if !ok || len(clause.List) != 1 {
				return true
			}
			kind, ok := clause.List[0].(*ast.SelectorExpr)
			if !ok || kind.Sel.Name != entityKind {
				return true
			}
			for _, statement := range clause.Body {
				ast.Inspect(statement, func(child ast.Node) bool {
					nested, ok := child.(*ast.CaseClause)
					if !ok {
						return true
					}
					for _, expr := range nested.List {
						if literal, ok := expr.(*ast.BasicLit); ok && literal.Kind == token.STRING {
							got = append(got, strings.Trim(literal.Value, `"`))
						}
					}
					return true
				})
			}
			return false
		})
	}
	return uniqueSorted(got)
}

func parseSourceFile(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := goparser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return file
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	values = values[:0]
	for value := range seen {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func assertSameStrings(t *testing.T, want, got []string) {
	t.Helper()
	want = uniqueSorted(append([]string(nil), want...))
	got = uniqueSorted(append([]string(nil), got...))
	if strings.Join(want, "\x00") != strings.Join(got, "\x00") {
		t.Fatalf("telnet fields and wire ops differ:\nwant=%v\ngot=%v", want, got)
	}
}

// zedit re-prompts on a prototype real_mobile/real_object cannot find; the web
// refuses the same commands. vnum 0 is not special: it fails here only
// because the fixture has no prototype 0.
func TestZoneCommandPrototypesMatchZeditChecks(t *testing.T) {
	world := newOLCTestWorld(t)
	cases := []struct {
		name string
		cmd  parser.ZoneCommand
		want string
	}{
		{"mob exists", parser.ZoneCommand{Command: "M", Arg1: 2001, Arg3: 1001}, ""},
		{"mob vnum 0", parser.ZoneCommand{Command: "M", Arg1: 0, Arg3: 1001}, "That mobile does not exist (vnum 0)."},
		{"give missing obj", parser.ZoneCommand{Command: "G", Arg1: 9999}, "That object does not exist (vnum 9999)."},
		{"put into missing container", parser.ZoneCommand{Command: "P", Arg1: 3001, Arg3: 4444}, "That object does not exist (vnum 4444)."},
		{"remove missing obj", parser.ZoneCommand{Command: "R", Arg1: 1001, Arg2: 1, Arg3: 5555}, "That object does not exist (vnum 5555)."},
		{"remove mob", parser.ZoneCommand{Command: "R", Arg1: 1001, Arg2: 0, Arg3: 2001}, ""},
		{"door has no prototype", parser.ZoneCommand{Command: "D", Arg1: 1001, Arg2: 0, Arg3: 1}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range []olc.OperationKind{olc.OpAddZoneCommand, olc.OpModifyZoneCommand} {
				cmd := tc.cmd
				err := validateZoneCommandPrototypes(world, olc.Operation{Kind: kind, Command: &cmd})
				got := ""
				if err != nil {
					got = err.Error()
				}
				if got != tc.want {
					t.Fatalf("op %d: error = %q, want %q", kind, got, tc.want)
				}
			}
		})
	}
}

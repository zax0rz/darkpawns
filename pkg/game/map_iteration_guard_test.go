package game

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mapRangeAllowlist lists every permitted direct `range` over World's internal
// maps (w.players, w.activeMobs, w.objectInstances) outside world_order.go and
// world.go's C-ordered snapshot helpers. Each entry corresponds to an audited
// order-free, go-only, or immediately-sorted site in
// docs/fidelity/iteration-order-sites.tsv.
var mapRangeAllowlist = map[string]string{
	// world_order.go canonical C-order builders
	"world_order.go:cOrderedCharacters": "Canonical C character_list builder; sorts by characterSequence descending.",
	"world_order.go:COrderedObjects":    "Canonical C object_list builder; sorts by obj.ID descending.",

	// world.go C-ordered snapshot helpers & audited order-free / go-only lookups
	"world.go:GetPlayersInRoom":      "Canonical C room-player helper; sorts by RoomEntrySequence descending.",
	"world.go:GetMobsInRoom":         "Canonical C room-mob helper; sorts by RoomEntrySequence descending.",
	"world.go:GetPlayer":             "Case-insensitive lookup by unique online player name.",
	"world.go:GetPlayerByID":         "Lookup by unique runtime player ID.",
	"world.go:SetObjectExtraDesc":    "Max-ID reduction under w.mu.Lock to pick C object_list head.",
	"world.go:SetObjectExtraFlag":    "Max-ID reduction under w.mu.Lock to pick C object_list head.",
	"world.go:AddPlayer":             "Online duplicate-name check and Go-only runtime ID allocator.",
	"world.go:ExtractMob":            "Pointer-identity lookup m == mob to enqueue extraction.",
	"world.go:newestMobByNameLocked": "Max characterSequence/ID reduction under w.mu to pick C character_list head.",

	// Audited order-free / immediately-sorted sites in pkg/game
	"animate_dead.go:CanRaiseUndeadI":           "Integer count of mob followers; order-independent.",
	"char_mgmt.go:HasPendingExtractions":        "Boolean existence check for PLR_EXTRACT/MOB_EXTRACT.",
	"char_mgmt.go:ExtractPendingPlayers":        "Collects flagged bodies then sorts by characterSequence descending.",
	"clan_admin.go:doClanDestroy":               "Clears ClanID/ClanRank and saves each member; no output or draws.",
	"clan_admin.go:playingPlayerNames":          "Populates map[string]bool set; order-independent.",
	"clan_membership.go:doClanEnroll":           "Collects online applicants then sorts by ID/Name ascending.",
	"clan_membership.go:doClanMembers":          "Collects online members then sorts by ID/Name ascending.",
	"death.go:handleMobDeath":                   "Pointer-identity lookup mob == victim to retire map key.",
	"follow.go:dieFollowerBody":                 "Collects followers into set then sorts by FollowingSequence descending.",
	"graph.go:findPlayerByName":                 "Case-insensitive lookup by unique online player name.",
	"inventory.go:GetObjNum":                    "Max-ID reduction matching vnum (C object_list head).",
	"other_mount.go:mountedFollowerFor":         "Collects mounted followers then sorts by FollowingSequence descending.",
	"party.go:GetFollowers":                     "Collects followers then sorts by FollowingSequence descending.",
	"party.go:GetFollowersInRoom":               "Collects room followers then sorts by FollowingSequence descending.",
	"party.go:OrderedGroupFollowers":            "Canonical C ch->followers helper; sorts by FollowingSequence descending.",
	"shoot_target.go:ResolveShootTarget":        "Collects room occupants then sorts by RoomEntrySequence descending.",
	"world_medit.go:RefreshLiveMobStrings":      "Independent prototype clone refresh on every matching mob; no output/draws.",
	"world_movement.go:NumFollowers":            "Integer count of followers; order-independent.",
	"world_oedit.go:mutateLiveObjExtraDesc":     "Independent extra-desc update on every matching object; no output/draws.",
	"world_oedit.go:RefreshLiveObjInstances":    "Independent prototype refresh on every matching object; no output/draws.",
	"world_player.go:countMobInstances":         "Integer count of active mobs matching vnum; order-independent.",
	"world_player.go:countObjectInstances":      "Integer count of active objects matching vnum; order-independent.",
	"world_scriptable.go:RemoveObjByInstanceID": "Unique objInstanceID lookup across inventories; at most one match.",
}

var guardedWorldMaps = map[string]bool{
	"players":         true,
	"activeMobs":      true,
	"objectInstances": true,
}

func checkFileForUnallowlistedMapRanges(fset *token.FileSet, fileName string, f *ast.File, allowlist map[string]string) []string {
	var violations []string
	base := filepath.Base(fileName)
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		key := base + ":" + fn.Name.Name
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			rs, ok := n.(*ast.RangeStmt)
			if !ok {
				return true
			}
			sel, ok := rs.X.(*ast.SelectorExpr)
			if !ok || !guardedWorldMaps[sel.Sel.Name] {
				return true
			}
			if _, allowed := allowlist[key]; !allowed {
				pos := fset.Position(rs.Pos())
				violations = append(violations, fmt.Sprintf("%s:%d (%s): unallowlisted range over .%s — use C-ordered helper (GetMobsInRoom/GetPlayersInRoom/GetAllMobs/GetAllPlayers/GetAllObjects/actChar) or document in mapRangeAllowlist",
					base, pos.Line, fn.Name.Name, sel.Sel.Name))
			}
			return true
		})
	}
	return violations
}

// TestMapIterationGuard enforces R3 by rejecting any new unallowlisted direct
// range over World.players, World.activeMobs, or World.objectInstances.
func TestMapIterationGuard(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	fset := token.NewFileSet()
	var violations []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", name, err)
		}
		violations = append(violations, checkFileForUnallowlistedMapRanges(fset, name, f, mapRangeAllowlist)...)
	}
	if len(violations) > 0 {
		t.Fatalf("unallowlisted direct world-map range(s) found:\n%s", strings.Join(violations, "\n"))
	}
}

// TestMapIterationGuardRejectsPlantedLoop proves R5h: a planted direct range
// over w.activeMobs outside the allowlist fails the guard.
func TestMapIterationGuardRejectsPlantedLoop(t *testing.T) {
	const planted = `package game
func plantedBadCaller(w *World) {
	for _, m := range w.activeMobs {
		_ = m
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "planted.go", planted, 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	violations := checkFileForUnallowlistedMapRanges(fset, "planted.go", f, mapRangeAllowlist)
	if len(violations) != 1 || !strings.Contains(violations[0], "plantedBadCaller") || !strings.Contains(violations[0], ".activeMobs") {
		t.Fatalf("expected 1 planted-loop violation for plantedBadCaller/.activeMobs, got: %v", violations)
	}
}

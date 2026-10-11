package scripting

import "testing"

// luaProofCase is one DP-1371 D7 row from
// docs/fidelity/depth/handoff/2026-10-09-dp-1371-lua-needs-proof-survey.tsv.
// The payloads come from the generated 2026-10-11-dp-1371-lua-proofs.tsv, which
// the extractor re-derives from src/scripts.c, so every expectation stays
// citable (R5g) instead of transcribed.
//
// Scope of the claim (R5h/R4): these subtests prove each binding's
// bad-argument arm through a real Lua call, with the bridge as the observation
// point. They pin the exact payload and that the file-TRUE sink was not used.
// They do *not* pin the type and level — Bridge.Log carries neither — and they
// are not independent mutation controls: the eight D7 families share one
// representative control each (2026-10-11-dp-1371-pr4-lua-proofs-controls.py).
type luaProofCase struct {
	site          string // the survey's site column, kept for failure messages
	call          string // a Lua call that passes the bad argument
	payload       string // C's exact diagnostic
	arm           string // which guard the input reaches, when the payload is shared
	unknownSocial bool   // lua_social's "unknown command" arm needs Social to fail
}

// proofBridge is recordingBridge with lua_social's failure arm switched on: the
// fake's Social always succeeds, and C logs when do_social finds no command.
type proofBridge struct {
	*recordingBridge
	unknownSocial bool
}

func (p *proofBridge) Social(CharRef, CharRef, string) bool { return !p.unknownSocial }

// luaProofCases is the 57-row table, one row per C site.
var luaProofCases = []luaProofCase{
	{"src/scripts.c:137", `action(42, "north")`, "[Lua] Invalid argument passed to lua_action.", "", false},
	{"src/scripts.c:160", `aff_flagged(42)`, "[Lua] Invalid argument passed to lua_aff_flagged.", "", false},
	{"src/scripts.c:184", `aff_flags({}, "toggle", 1)`, "[Lua] Invalid set/remove to lua_aff_flags.", "", false},
	{"src/scripts.c:188", `aff_flags(42)`, "[Lua] Invalid argument passed to lua_aff_flags.", "", false},
	{"src/scripts.c:241", `cansee(42)`, "[Lua] Invalid argument to lua_cansee.", "", false},
	{"src/scripts.c:380", `echo(42, "room", "x")`, "[Lua] Invalid arguments passed to lua_echo.", "", false},
	{"src/scripts.c:422", `equip_char(42, {})`, "[Lua] Invalid arguments passed to lua_equip_char.", "", false},
	{"src/scripts.c:447", `exit_flags({}, 0, "toggle", 1)`, "[Lua] Invalid set/remove to lua_exit_flags.", "", false},
	{"src/scripts.c:451", `exit_flags(42)`, "[Lua] Invalid argument passed to lua_exit_flags.", "", false},
	{"src/scripts.c:475", `exit_flagged(42)`, "[Lua] Invalid argument passed to lua_exit_flagged.", "", false},
	{"src/scripts.c:491", `extchar(42)`, "[Lua] Invalid char passed to lua_extchar", "", false},
	{"src/scripts.c:507", `extobj(42)`, "[Lua] Invalid object passed to lua_extobj", "", false},
	{"src/scripts.c:537", `extra(42, "x")`, "[Lua] Invalid argument passed to lua_extra.", "", false},
	{"src/scripts.c:566", `follow(42)`, "[Lua] Invalid argument passed to lua_follow.", "", false},
	{"src/scripts.c:608", `inworld({})`, "[Lua] Invalid argument passed to lua_inworld.", "inworld kind is not a string", false},
	{"src/scripts.c:617", `inworld("mob", {})`, "[Lua] Invalid argument passed to lua_inworld.", "inworld mob vnum is not a number", false},
	{"src/scripts.c:621", `inworld("char", {})`, "[Lua] Invalid argument passed to lua_inworld.", "inworld char name is not a string", false},
	{"src/scripts.c:631", `inworld("bogus", 1)`, "[Lua] Invalid argument passed to lua_inworld.", "inworld kind is unknown, so the default arm", false},
	{"src/scripts.c:692", `ishunt(42)`, "[Lua] Invalid argument passed to lua_ishunt.", "", false},
	{"src/scripts.c:712", `isnpc(42)`, "[Lua] Invalid argument to lua_isnpc.", "", false},
	{"src/scripts.c:775", `load_room({})`, "[Lua] Invalid argument passed to lua_load_room.", "", false},
	{"src/scripts.c:787", `log({})`, "[Lua] Invalid argument passed to lua_log.", "", false},
	{"src/scripts.c:806", `mload(1, 1)`, "[Lua] Invalid mobile vnum passed to lua_mload.", "", false},
	{"src/scripts.c:815", `mload({})`, "[Lua] Invalid arguments passed to lua_mload.", "", false},
	{"src/scripts.c:838", `mob_flagged(42)`, "[Lua] Invalid argument passed to lua_mob_flagged.", "", false},
	{"src/scripts.c:862", `mob_flags({}, "toggle", 1)`, "[Lua] Invalid set/remove to lua_mob_flags.", "", false},
	{"src/scripts.c:866", `mob_flags(42)`, "[Lua] Invalid argument passed to lua_mob_flags.", "", false},
	{"src/scripts.c:901", `mount({}, nil, "jump")`, "[Lua] Invalid position to lua_mount.", "", false},
	{"src/scripts.c:903", `mount(42)`, "[Lua] Invalid argument passed to lua_mount.", "", false},
	{"src/scripts.c:942", `obj_extra({}, "toggle", 1)`, "[Lua] Invalid set/remove to lua_obj_extra.", "", false},
	{"src/scripts.c:946", `obj_extra(42)`, "[Lua] Invalid argument passed to lua_obj_extra.", "", false},
	{"src/scripts.c:969", `obj_flagged(42)`, "[Lua] Invalid argument passed to lua_obj_flagged.", "", false},
	{"src/scripts.c:990", `oload(me, 1, "char")`, "[Lua] lua_oload returned an unknown object.", "", false},
	{"src/scripts.c:1001", `oload(me, 1, "bogus")`, "[Lua] Invalid location specified in lua_oload.", "", false},
	{"src/scripts.c:1008", `oload(42, 1, "room")`, "[Lua] Invalid argument passed to lua_oload.", "", false},
	{"src/scripts.c:1035", `objfrom(42, "room")`, "[Lua] Invalid argument passed to lua_objfrom.", "", false},
	{"src/scripts.c:1106", `obj_list("x", "bogus")`, "[Lua] Invalid location to search in lua_obj_list.", "", false},
	{"src/scripts.c:1123", `obj_list({})`, "[Lua] Invalid argument passed to lua_obj_list.", "", false},
	{"src/scripts.c:1169", `objto(42, "room")`, "[Lua] Invalid argument passed to lua_objto.", "", false},
	{"src/scripts.c:1192", `plr_flagged(42)`, "[Lua] Invalid argument passed to lua_plr_flagged.", "", false},
	{"src/scripts.c:1216", `plr_flags({}, "toggle", 1)`, "[Lua] Invalid set/remove to lua_plr_flags.", "", false},
	{"src/scripts.c:1220", `plr_flags(42)`, "[Lua] Invalid argument passed to lua_plr_flags.", "", false},
	{"src/scripts.c:1255", `raw_kill(42, nil, 1)`, "[Lua] Invalid arguments passed to lua_raw_kill.", "", false},
	{"src/scripts.c:1287", `say({})`, "[Lua] Invalid argument passed to lua_say.", "", false},
	{"src/scripts.c:1304", `save_char(42)`, "[Lua] Invalid argument passed to lua_save_char.", "", false},
	{"src/scripts.c:1320", `save_obj(42)`, "[Lua] Invalid argument passed to lua_save_obj.", "", false},
	{"src/scripts.c:1336", `save_room(42)`, "[Lua] Invalid argument passed to lua_save_room.", "", false},
	{"src/scripts.c:1360", `set_hunt(42)`, "[Lua] Invalid arguments passed to lua_set_hunt.", "", false},
	{"src/scripts.c:1419", `social(me, "nosuchcommand")`, "[Lua] Unknown command passed to lua_social.", "", true},
	{"src/scripts.c:1423", `social(42, "wave")`, "[Lua] Invalid argument passed to lua_social.", "", false},
	{"src/scripts.c:1466", `spell(nil, nil, 0, 1)`, "[Lua] No spell passed to lua_spell.", "", false},
	{"src/scripts.c:1486", `spell(42)`, "[Lua] Invalid argument passed to lua_spell.", "", false},
	{"src/scripts.c:1513", `steal(42, 42)`, "[Lua] Invalid argument passed to lua_steal.", "", false},
	{"src/scripts.c:1532", `tport(me, 9999)`, "[Lua] Invalid room passed to lua_tport.", "", false},
	{"src/scripts.c:1559", `tport(42, 1)`, "[Lua] Invalid argument passed to lua_tport.", "", false},
	{"src/scripts.c:1586", `tell({}, "x")`, "[Lua] Invalid argument passed to lua_tell.", "", false},
	{"src/scripts.c:1604", `unaffect(42)`, "[Lua] Invalid argument passed to lua_unaffect.", "", false},
}

// TestLuaProofBadArgumentProducers proves every one of the 57
// implemented-needs-proof rows through a real Lua call: the binding's
// bad-argument arm emits C's exact payload exactly once, and never through the
// file-TRUE sink (file FALSE).
func TestLuaProofBadArgumentProducers(t *testing.T) {
	if len(luaProofCases) != 57 {
		t.Fatalf("table holds %d rows, want the survey's 57", len(luaProofCases))
	}
	for _, tc := range luaProofCases {
		t.Run(tc.site, func(t *testing.T) {
			dir := t.TempDir()
			writeScript(t, dir, "proof.lua",
				"function oncmd()\n  local _ = "+tc.call+"\n  return TRUE\nend\n")
			bridge := &proofBridge{recordingBridge: newRecordingBridge(), unknownSocial: tc.unknownSocial}
			engine := NewEngine(dir, worldWith(100))
			t.Cleanup(func() { engine.l.Close() })
			me := player
			handled, err := engine.RunScript(&ScriptContext{World: bridge, MeRef: &me, OwnerType: "mob"}, "proof.lua", "oncmd")
			if err != nil {
				t.Fatalf("run: %v (logs=%q)", err, bridge.logs)
			}
			if !handled {
				t.Fatalf("script must run to completion; logs=%q", bridge.logs)
			}
			if len(bridge.logs) != 1 || bridge.logs[0] != tc.payload {
				t.Fatalf("%s: log sequence = %q, want exactly [%q] (arm: %s)",
					tc.site, bridge.logs, tc.payload, tc.arm)
			}
			if len(bridge.logFiles) != 0 {
				t.Fatalf("%s: file-TRUE sink used: %+v", tc.site, bridge.logFiles)
			}
		})
	}
}

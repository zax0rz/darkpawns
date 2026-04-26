# C → Go Port Plan — Dark Pawns

> **Goal:** 100% faithful C-to-Go port of all ~69K lines of Dark Pawns MUD source.
> **Strategy:** Ordered by gameplay impact. Each wave = build → QA → fix → push.
> **Audit date:** 2026-04-25 — full function-level audit in COMPLETENESS-AUDIT.md
> **Model note:** DeepSeek V4 Flash is the daily driver. Documented here so any model can pick up without loss.

---

## Current State — HARD DATA (2026-04-25)

```
C source (src/):        68,823 lines across ~60 .c files
Go codebase (pkg/):     68,681 lines across 186 .go files
Build:                  go build ./... clean
go vet:                 vet passes clean
go test:                all packages pass (cached/ok)
Git status:             clean (main)

Truly unported:         ~15,000 C lines (67% ported)
Skipped (OLC/SPA):      ~7,908 C lines (editors — proper admin)
```

### What's fully ported

| Area | C source | Go | Status |
|------|----------|-----|--------|
| Item cmds | act.item.c (1,789) | ✅ act_item.go + session | 3 trivial stubs only |
| Offensive | act.offensive.c (1,510) | ✅ act_offensive + session + combat | Fully ported |
| Movement | act.movement.c (951) | ✅ act_movement + session movement_cmds | Fully ported |
| Communication | act.comm.c (1,566) | ✅ act_comm + session comm_cmds | Fully ported |
| Player cmds | act.other.c (1,947) | ✅ act_other + act_other_bridge | Fully ported |
| Socials | act.social.c (305) | ✅ act_social + session | Fully ported |
| Spells | magic/spells/spell_parser (4,843) | ✅ pkg/spells/ (8 files, 1,846 Go) | Fully ported |
| Combat | fight.c (2,033) | ✅ pkg/combat/ (1,995 Go) | Fully ported |
| Logging/Utils | utils.c (980) | ✅ game logging.go | Fully ported |
| Spec assign | spec_assign.c (642) | ✅ game spec_assign.go | Fully ported |
| Shops | shop.c (1,445) | ✅ game + command + session | Fully ported |
| Clan | clan.c (1,574) | ✅ game clans.go | Fully ported |
| Boards | boards.c (551) | ✅ game boards.go | Fully ported |
| Mobact | mobact.c (408) | ✅ game mobact.go | Fully ported |
| Mobprogs | mobprog.c (646) | ✅ game mobprogs.go | Fully ported |
| Gate/Graph/Mail | gate/graph/mail | ✅ game gate.go/graph.go/mail.go | Fully ported |
| Constants | constants.c (1,450) | ✅ game constants.go | Fully ported |
| Alias/Ban/Weather/dream | alias/ban/weather | ✅ session/engine | Fully ported |
| Tattoo/Mapcode | tattoo/mapcode | ✅ session tattoo.go/map_cmds.go | Fully ported |

### Partially ported

| Area | C source | Go | Status | What's missing |
|------|----------|-----|--------|----------------|
| Informative cmds | act.informative.c (2,803) | 🔸 ~95% | **do_gen_ps only** — credits, news, motd, version, clear, whoami, wizlist, immlist, handbook, policies, future, player_list. ~150 C lines, low complexity. |
| Wizard cmds | act.wizard.c (3,863) | 🔸 ~40% (1,574 Go) | **20+ commands:** heal, restore, set, clone, damage, morph, poofin/out, socials, slist, zlist, string, audit, syscheck, cstat/mstat/ostat, write_help, find, rumor, purge. |
| Display prefs | act.display.c (717) | 🟡 Mostly | Some toggle preferences may still be C-only. |
| Spec procs | spec_procs.c/2/3 (6,021) | 🔸 ~48% (2,924 Go) | Lua scripts fill gaps. GetMobSpec/GetObjSpec/GetRoomSpec wiring needed. |
| Infra (handler) | handler.c (1,616) | 🔸 Partial | obj_to_char/from_char/obj_to_obj — scripting stubs. char_from_room/char_to_room — via Player.SetRoom() but spec_procs2.go has TODO comments. equip_char/unequip_char — scripting stubs. apply_ac — unconfirmed. |
| Objsave (logic) | objsave.c (1,250) | 🔸 Types done | Crash_load/save/crashsave/rentsave ~900 lines pending player/descriptor wiring. |
| House | house.c (744) | 🔸 Stubs | Crashsave, delete_file, listrent, house_boot, hcontrol ~600 lines pending objsave. |
| Game infra | comm.c (2,637) interpreter.c (2,365) db.c (3,219) | 🟡 Mostly | Command dispatch replaced by Go. Game loop/descriptors in session/manager + telnet/listener. World loading in parser/. Player saving/loading, zone resets still C. create_entry/Valid_Name/isbanned — ❓ in auth or moderation? |

### Not ported (intentionally skipped)

| File | Lines | Reason |
|------|-------|--------|
| OLC editors (medit/oedit/redit/sedit/zedit/olc/improved-edit/luaedit/tedit/poof/file-edit) | ~7,908 | Replaced by Web Admin SPA / proper admin commands |
| new_cmds.c / new_cmds2.c | ~3,800 | Dark Pawns custom commands — decision needed |

### Unported C files with small footprint (check if needed)

- **dream.c** (223) — ❓ ported in Lua or Go?
- **poof.c** (102) — ⏭️ skipped (SPA)
- **ident.c** (277) — ❓ ported in moderation?
- **random.c** (73) — ❓ likely in utils
- **ocr (oc.c)** (180) — ⏭️ OLC
- **luaedit.c** (58) — ⏭️ OLC
- **svn_version.c** (1) — trivial

---

## Priority Queue (ordered by gameplay impact)

### Tier 1 — Player-facing, low effort
1. **`do_gen_ps`** — credits, news, info, motd, version, clear, whoami, wizlist, immlist, handbook, policies, future, player_list. ~150 C lines, pure text-file reads. Estimated: 1 wave.
2. **class.c data tables** — class names, prac_params, thaco, spell_levels, titles, guild info. Static const data that Go needs native access to. `obj_to_obj` function also stub. Estimated: 1 wave.

### Tier 2 — Admin experience
3. **Wizard commands** — heal, restore, set, clone, damage, morph, poofin/out, slist, zlist, string, audit, syscheck, cstat/mstat/ostat. ~20 functions. Estimated: 2-3 waves.
4. **Remaining act.display.c toggles** — estimate 3-5 functions.

### Tier 3 — Systems that work but need completion
5. **handler.c stubs in scripting** — obj_to_char/from_char, equip_char/unequip_char. spec_procs2.go has ~10 TODO comments referencing these. Estimated: 1 wave.
6. **Spec proc wiring** — GetMobSpec/GetObjSpec/GetRoomSpec called in main loop but no-op. Prevents all object/mob/room specs from firing (shops, etc.). Estimated: 1 wave.

### Tier 4 — Persistence (gameplay-critical but complex)
7. **Objsave logic** — Crash_load/save/crashsave/rentsave ~900 lines. Pending player/descriptor interface solidification. Blocked on Tier 3.
8. **House logic** — ~600 lines. Pending objsave full logic. Blocked on Tier 3+7.

### Tier 5 — Unknown/decide
9. **new_cmds.c + new_cmds2.c** — ~3,800 lines of Dark Pawns custom commands. Some are already integrated into skill_commands.go. Need audit of what's actually remaining.
10. **remnant: dream.c, ident.c, random.c** — verify if ported or still needed.

---

## OLC files replaced by Web Admin SPA (NOT porting)

| File | Lines | Replacement |
|------|-------|-------------|
| oedit.c | 1,564 | SPA object editor |
| redit.c | 1,078 | SPA room editor |
| medit.c | 1,126 | SPA mob editor |
| sedit.c | 1,178 | SPA shop editor |
| zedit.c | 1,276 | SPA zone editor |
| olc.c | 524 | SPA OLC framework |
| improved-edit.c | 627 | SPA text editor |
| luaedit.c | 58 | Monaco editor |
| tedit.c | 98 | SPA trigger editor |
| poof.c | 102 | SPA poof messages |
| file-edit.c | 199 | SPA file upload |
| **Total** | **7,830** | |

---

## Session Startup Quick-Reference

Every session: read COMPLETENESS-AUDIT.md and this file. The top 5-10 things to port are in the Priority Queue above. Don't re-audit — the data's here.

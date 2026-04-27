# Dark Pawns — Port Scope & Remaining Work

**Date:** 2026-04-26 (updated)  
**Method:** Function-by-function comparison of every C source file against Go counterparts.

---

## Summary

**Port is effectively complete.** The ~2,400 C line estimate from the initial audit was inflated — most functions either exist in Go under different names or are handled by Go's runtime (GC, goroutines, stdlib). Actual remaining work is ~250 C lines of optional polish features.

| Status | Systems |
|--------|---------|
| ✅ Done | mobact, clans, shops, mail, dreams, tattoos, whod, spec_assign, limits (full), follow chain, affect system, handler helpers, houses (full persistence), boards (full save/load), weather, bans, socials, wizard stat, graph/BFS, events/queue |
| 🟡 Optional polish | mapcode (ASCII map), text editor pagination helpers, remove_follower (specific target), get_number/get_obj_num/get_char_num |

---

## Fully Ported (verified)

### Core Gameplay
- **limits.c** — Full XP table + regen formulas (`ExpNeededForLevel`, `FindExp`, `ManaGain`, `HitGain`, `MoveGain` + NPC variants)
- **handler.c** — `obj_to_obj` → `AddToContainer`, `obj_from_obj` → `RemoveFromContainer`, `stop_follower` → `StopFollower`, `add_follower` → `AddFollower`, `die_follower` → `DieFollower`, `circle_follow` → `CircleFollow`, `can_speak` → `CanSpeak`, `set_hunting` → `SetHunting`
- **follow.go** — Complete follow chain management with `GetFollowers`, `GetFollowersInRoom`, `NumFollowers`
- **affect system** — Full port in `pkg/engine/affect_helpers.go`: `AffectTotal`, `AffectModify`, `AffectToChar`, `AffectToChar2`, `AffectRemove`, `AffectFromChar`, `AffectedBySpell`, `AffectJoin`

### Persistence
- **objsave.c** — JSON-based player save/load in `pkg/game/save.go` + `CrashLoad` at line 392
- **house.c** — Full JSON persistence: `ObjToStore`/`ObjFromStore` (JSON), `houseLoad` (reads objects from save file), `houseCrashsave` (writes JSON), `HouseListrent` (displays item names)
- **boards.c** — Full save/load/reset in `pkg/game/boards.go`
- **bans.c** — `LoadBanned`, `WriteBanList`, `IsBanned` in `pkg/game/bans.go`
- **weather.c** — `AnotherHour`, `WeatherChange` in `pkg/game/weather.go`

### AI & Pathfinding
- **mobact.c** — Full mob AI (wander, aggro, memory, scavenging, spec_proc dispatch) via goroutines + `aiticker`
- **graph.c** — BFS pathfinding: `findFirstStep`, `doTrack`, `huntVictim`, `mobIsIntelligent`
- **events.c + queue.c** — Replaced by Go goroutines + `time.Ticker` (ai ticker, point update ticker)

### Commands & Features
- **act.wizard.c** — `doStat` in `pkg/game/modify.go`
- **socials** — Hardcoded Go map in `pkg/game/socials.go` (no file loading needed)
- **clan.c, shop.c, mail.c, dream.c, tattoo.c, whod.c** — All fully ported

---

## Remaining Optional Work (~250 C lines)

### mapcode.c — ASCII map rendering (~226 C lines) [P4]
`map()`, `do_map()` — renders ASCII map of surrounding rooms. Nice-to-have, not blocking anything.
**Status:** Not started. Would go in `pkg/game/mapcode.go`.

### modify.c — Text editor pagination (~93 C lines) [P4]
`next_page()`, `count_pages()`, `quad_arg()` — used by the improved-edit text editor system.
**Status:** Not started. Only needed when/if the text editor (`improved-edit.c`) is ported.

### handler.c — Low-priority helpers (~30 C lines) [P4]
- `get_number()` — Parse leading number from string. Call sites in C: ~12. **Zero call sites in Go** — existing Go functions handle parsing differently. Add on-demand.
- `get_obj_num()` — Find object instance by rnum. Call sites in C: ~3. **Zero call sites in Go.** Add on-demand.
- `get_char_num()` — Find char by rnum. Call sites in C: ~3. **Zero call sites in Go.** Add on-demand.
- `remove_follower()` — Remove specific follower from someone's list. `StopFollower` covers the common case. Add on-demand.

---

## NOT needed (Go handles differently)
- **`free_char`** — Go GC handles memory cleanup
- **OLC editors** (oedit, redit, zedit, medit, sedit) — separate tooling
- **String utilities** (str_dup, str_cmp, etc.) — Go stdlib
- **Random numbers** (prng_*) — Go math/rand
- **Networking primitives** (init_socket, new_descriptor, etc.) — Go net package
- **Logging** (mudlog, basic_mud_log) — Go slog
- **File editors** (improved-edit, file-edit) — different UX pattern
- **comm.c functions** — Go networking/session layer
- **Social file loading** — Hardcoded Go map replaces C .soc file parser

---

## Dead Code to Clean
- `pkg/game/act_item_stubs.go` — Stubs for EatFood, DrinkLiquid, MakeCorpse. No references from any other file.
- `pkg/game/damage_stubs.go` — Stubs for perform_act/do_forced. No references from any other file.
- Stale comments in `pkg/game/objsave.go:668` (CrashLoad comment stub)

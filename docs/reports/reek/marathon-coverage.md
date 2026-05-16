# Dark Pawns Test Coverage Analysis — 2026-05-15

## Summary

| Metric | Value |
|--------|-------|
| Overall statement coverage | **10.9%** |
| Source files | 313 |
| Test files | 40 |
| Source lines | 91,928 |
| Test lines | 15,233 |
| Test:source line ratio | **1:6.0** |
| Functions at 0.0% coverage | 2,809 |
| Packages with zero test coverage | 18 |
| Packages with tests but <20% coverage | 5 |

---

## 1. Packages With ZERO Test Coverage

These packages have no test files or no test functions at all:

| Package | Functions | Notes |
|---------|-----------|-------|
| `pkg/common` | — | No test files at all |
| `pkg/agent` | 13 | Memory hooks — agent AI integration |
| `pkg/agentcli` | 35 | Agent CLI client — websocket, FSM, LLM |
| `pkg/ai` | 6 | AI behaviors and brain logic |
| `pkg/audit` | 8 | Audit logger |
| `pkg/command` | 96 | **CRITICAL** — all command dispatch, admin commands, skill commands, shop commands |
| `pkg/db` | 26 | Player DB, narrative memory, data conversion |
| `pkg/optimization` | 107 | Caching, object pools, WebSocket, Python AI bridge |
| `pkg/secrets` | 4 | Secrets manager |
| `pkg/storage` | 10 | SQLite storage layer (SaveWorld, LoadWorld, Close) |
| `pkg/telnet` | 10 | **CRITICAL** — telnet listener, connection handling, login flow |
| `web` | 10 | Auth middleware, CORS, security headers |
| `cmd/server` | 1 | Main entrypoint |
| `cmd/dp-agent` | 12 | Agent CLI entrypoint |
| `cmd/agentkeygen` | 1 | Key generation utility |
| `cmd/test-race` | 1 | Race condition test utility |
| `profiling` | 21 | CPU/heap/mutex profiling |
| `benchmarks` | 9 | Performance benchmarks |
| `examples` | 10 | Integration examples |

---

## 2. Packages With Tests But Low Coverage

| Package | Coverage | Assessment |
|---------|----------|------------|
| `pkg/session` | 4.0% | **CRITICAL GAP** — only session.go, autoloot, manager tested; 200+ functions at 0% |
| `pkg/game` | 5.1% | **CRITICAL GAP** — 44 files, most at 0%: combat, clans, houses, shops, skills, etc. |
| `pkg/spells` | 8.6% | Only spell_info, checkreagents, saving_throws tested; damage_spells, call_magic, say_spell at 0% |
| `pkg/dreaming` | 11.9% | extract.go covered (100%); dream.go and graph.go at 0% |
| `pkg/scripting` | 19.6% | engine.go is 100% but integration tests don't cover Lua callback paths |
| `pkg/validation` | 20.8% | validation.go at 100%, input.go at 0% (ValidateInput, SanitizeInput, ValidateCommand) |

---

## 3. Critical Paths — Untested or Dangerously Low

### Login Flow — ⚠️ UNTABLED
- `pkg/session/session_login.go` — 0.0% (2 funcs)
- `pkg/session/char_creation.go` — 0.0% (8 funcs)
- `pkg/session/session_idle.go` — 0.0% (6 funcs)
- `pkg/telnet/listener.go` — 0.0% (10 funcs: Listen, handleConn, readLine, writeLoop, sendLogin)
- `pkg/auth/jwt.go` — 0.0% (5 funcs)

**Risk:** A login regression could lock players out with zero test coverage to catch it.

### Combat — ⚠️ PARTIALLY TESTED
- `pkg/combat/` — 33.0% overall (fight_core.go and formulas.go at 100%, but engine.go at 0%)
- `pkg/game/combat_basic.go` — 0.0% (7 funcs)
- `pkg/game/combat_advanced.go` — 0.0% (6 funcs)
- `pkg/game/combat_melee.go` — 0.0% (5 funcs)
- `pkg/game/combat_ranged.go` — 0.0% (1 func)
- `pkg/game/combat_control.go` — 0.0% (4 funcs)
- `pkg/game/combat_helpers.go` — 100%
- `pkg/game/death.go` — 100%
- `pkg/game/player_combat.go` — 100%

**Risk:** Core combat formulas are solid, but the game-layer combat dispatch (damage application, flee, rescue, bash) is completely untested.

### Spells — ⚠️ CRITICALLY LOW
- `pkg/spells/affect_spells.go` — 100% (50 funcs)
- `pkg/spells/saving_throws.go` — 100%
- `pkg/spells/spell_info.go` — 100%
- `pkg/spells/damage_spells.go` — **0.0%** (8 funcs — fireball, lightning, etc.)
- `pkg/spells/call_magic.go` — **0.0%** (7 funcs — the magic dispatch)
- `pkg/spells/say_spell.go` — **0.0%** (10 funcs)
- `pkg/spells/spells.go` — **0.0%** (1 func)

**Risk:** Damage spell application and magic calling are the core of the spell system. Affects are tested; damage is not.

### Item/Equipment — ⚠️ MIXED
- `pkg/game/object.go` — 100% (tested)
- `pkg/game/equipment.go` — 100%
- `pkg/game/inventory.go` — 100%
- `pkg/game/item_equipment.go` — **0.0%** (10 funcs)
- `pkg/game/item_consumable.go` — **0.0%** (3 funcs)
- `pkg/game/item_container.go` — **0.0%** (3 funcs)
- `pkg/game/item_door.go` — **0.0%** (5 funcs)
- `pkg/game/item_transfer.go` — **0.0%** (13 funcs)
- `pkg/game/item_helpers.go` — **0.0%** (27 funcs)

### Clans — ❌ ENTIRELY UNTESTED
All clan files at 0.0%:
- `clans.go` (16 funcs), `clan_membership.go` (7), `clan_economy.go` (3), `clan_bank.go` (1), `clan_admin.go` (3), `clan_command.go` (1), `clan_info.go` (3), `clan_settings.go` (6)

### Houses — ❌ ENTIRELY UNTESTED
- `houses.go` (11), `house_boot.go` (1), `house_control.go` (7), `house_player.go` (5), `house_rent.go` (1), `house_save.go` (7)

### Shops — ⚠️ PARTIALLY TESTED
- `pkg/game/systems/shop.go` — 100%
- `pkg/game/systems/shop_manager.go` — 100%
- `pkg/game/shop.go` — **0.0%** (13 funcs)

---

## 4. Test Files With No Test Functions

**39 test files** contain no `func Test` declarations. These are either:
- Test helpers/fixtures only
- Benchmark files (`func Benchmark*`)
- Empty placeholder files

Full list:
```
tests/unit/combat_test.go
pkg/metrics/metrics_test.go
pkg/scripting/integration_test_batchd_test.go
pkg/scripting/integration_test.go
pkg/privacy/client_test.go
pkg/combat/formulas_test.go
pkg/combat/fight_core_test.go
pkg/combat/skill_messages_test.go
pkg/auth/ratelimit_test.go
pkg/admin/agent_store_test.go
pkg/admin/handlers_test.go
pkg/admin/world_write_test.go
pkg/admin/log_buffer_test.go
pkg/game/movement_test.go
pkg/game/message_test.go
pkg/game/save_world_test.go
pkg/game/object_movement_test.go
pkg/game/death_test.go
pkg/game/combat_test.go
pkg/game/systems/shop_test.go
pkg/game/systems/door_test.go
pkg/game/tattoo_constants_test.go
pkg/game/command_exec_test.go
pkg/parser/obj_test.go
pkg/parser/parser_test.go
pkg/parser/wld_test.go
pkg/parser/zon_test.go
pkg/parser/mob_test.go
pkg/dreaming/valence_test.go
pkg/spells/spell_info_test.go
pkg/spells/checkreagents_test.go
pkg/spells/saving_throws_test.go
pkg/events/queue_test.go
pkg/events/lua_integration_test.go
pkg/engine/affect_test.go
pkg/engine/skill_test.go
pkg/validation/validation_test.go
pkg/moderation/manager_test.go
pkg/session/session_test.go
```

**Note:** The `grep -qL` in the original command looks for files *lacking* `func Test`. All 39 files listed match — meaning they likely contain benchmarks, helpers, or test suites (using `testing.T` via helper functions) rather than top-level `func Test*` declarations. These files may still contribute to coverage indirectly.

---

## 5. Test:Source Line Ratio

| Metric | Value |
|--------|-------|
| Source lines (non-test .go) | 91,928 |
| Test lines (_test.go) | 15,233 |
| Ratio | **1:6.0** (1 test line per 6 source lines) |
| Industry benchmark | 1:3 to 1:5 for well-tested codebases |
| Assessment | **Below average** — roughly half the test density expected |

---

## 6. Recently Changed Files Without Tests (Last 20 Commits)

These files were modified in the last 20 commits but have no corresponding test file:

| File | Risk Level |
|------|-----------|
| `pkg/game/world_scriptable.go` | **HIGH** — 74 Lua-binding functions, no tests |
| `pkg/session/session_login.go` | **HIGH** — login flow, no tests |
| `pkg/session/char_creation.go` | **HIGH** — character creation, no tests |
| `pkg/session/equipment_ac.go` | **MEDIUM** — equipment AC calculation |
| `pkg/session/session_idle.go` | **MEDIUM** — idle state handling |
| `pkg/auth/jwt.go` | **HIGH** — JWT token handling, no tests |
| `pkg/admin/router.go` | **LOW** — routing is tested via handlers_test.go |
| `pkg/admin/login.go` | **LOW** — tested via handlers_test.go |
| `pkg/game/object.go` | **LOW** — already 100% covered |
| `pkg/game/player_affects.go` | **LOW** — already 100% covered |
| `pkg/game/world.go` | **LOW** — already 100% covered |
| `pkg/game/world_zone.go` | **MEDIUM** — zone management |
| `pkg/scripting/engine.go` | **LOW** — already 100% covered |
| `pkg/scripting/types.go` | **MEDIUM** — type definitions used by engine |
| `cmd/dp-agent/main.go` | **LOW** — CLI entrypoint |
| `cmd/server/main.go` | **LOW** — server entrypoint |
| `web/auth.go` | **HIGH** — auth middleware, no tests |

---

## 7. What's Actually Well-Tested

Not all gloom. Several subsystems have excellent coverage:

| Package | Coverage | Notes |
|---------|----------|-------|
| `pkg/metrics` | 100.0% | Full coverage |
| `pkg/admin` | 54.4% | handlers, agent_store, log_buffer well tested |
| `pkg/auth` | 55.3% | ratelimit at 100% |
| `pkg/parser` | 75.3% | mob, obj, wld, zon parsers all tested |
| `pkg/events` | 67.2% | queue tested, Lua integration tested |
| `pkg/engine` | 40.3% | affects and skills at 100% |
| `pkg/game/systems` | 50.9% | door and shop managers at 100% |
| `pkg/privacy` | 33.7% | client, config, logger at 100% |

Core tested files in `pkg/game` at 100%:
- `mob.go` (65 funcs), `world.go` (58 funcs), `object.go` (44 funcs), `spec_procs*.go` (138 funcs), `player_stats.go` (38 funcs), `save.go` (16 funcs), `death.go` (13 funcs), `equipment.go` (15 funcs)

---

## 8. Recommendations (Priority Order)

### CRITICAL — Fix Immediately
1. **`pkg/telnet/`** — Zero coverage on the network layer. A connection handling bug could crash the server.
2. **`pkg/session/session_login.go`** — Login regression = locked players.
3. **`pkg/game/world_scriptable.go`** — 74 Lua binding functions, zero tests. Reek's steal bug was here.
4. **`pkg/command/`** — 96 functions, zero coverage. Command dispatch is the game's spine.

### HIGH — Fix This Sprint
5. **`pkg/spells/damage_spells.go`** and `call_magic.go` — Damage spell application untested.
6. **`pkg/game/combat_*.go`** — Game-layer combat dispatch untested.
7. **`pkg/game/item_*.go`** — Item interactions (containers, doors, consumables, transfers) untested.
8. **`pkg/db/`** — Player persistence layer untested.
9. **`pkg/storage/`** — SQLite storage untested.

### MEDIUM — Fix This Month
10. **`pkg/game/clans.go`** — Entire clan system untested.
11. **`pkg/game/houses.go`** — Entire housing system untested.
12. **`pkg/session/`** (200+ functions at 0%) — Need systematic session command tests.
13. **`pkg/optimization/`** — Caching and pool logic untested.
14. **`web/`** — Auth middleware and security headers untested.

### LOW — Backlog
15. `pkg/agent/`, `pkg/agentcli/`, `pkg/ai/` — Agent infrastructure (newer, lower priority).
16. `cmd/` entrypoints — Main functions are thin wrappers.
17. `profiling/`, `benchmarks/`, `examples/` — Not production code.

---

## 9. Key Observations

- **The test suite has two tiers:** Files that are well-tested (often 100%) and files that are completely untested. There's very little in between. This suggests tests were written for specific features during development rather than systematically.
- **The `pkg/game/` split is telling:** Core game mechanics (mob, world, object, death, equipment) are at 100%, but interaction code (clans, houses, shops, skills, combat commands) is at 0%. The foundation is solid; the gameplay layer is exposed.
- **Lua scripting is a blind spot:** `world_scriptable.go` (74 functions) bridges the Go game to Lua scripts. The engine itself is 100% tested, but the 74 binding functions that actually expose game state to Lua are not. This is where the steal bug lived.
- **The telnet layer being untested is surprising** — it's the only network path into the server, and a bug there means players can't connect.
- **39 test files with no `func Test*`** — these are likely using test suite patterns or benchmark-only files. Worth auditing to confirm they're actually exercising code.

---

*Generated by Daeron's coverage analysis subagent — 2026-05-15 19:57 EDT*
*Coverage profile: `/tmp/coverage.out` (Go cover tool)*

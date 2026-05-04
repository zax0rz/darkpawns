# Test Coverage Status — 2026-05-04

## Summary
**13 of 24 packages tested.** 9,696 lines of test code across 23 test files. All tests pass.

| Package | Tests | Status |
|---------|-------|--------|
| auth | ratelimit_test.go (344) | ✅ |
| combat | formulas_test.go (748) | ✅ |
| engine | skill_test.go (476), affect_test.go (415) | ✅ |
| events | queue_test.go (287), lua_integration_test.go (277) | ✅ |
| game | object_movement_test.go (1136), save_world_test.go (312) | ✅ |
| game/systems | door_test.go (573), shop_test.go (305) | ✅ |
| metrics | metrics_test.go (79) | ✅ |
| moderation | manager_test.go (206) | ✅ |
| parser | mob_test.go (598), obj_test.go (708), zon_test.go (464), parser_test.go (355), wld_test.go (113) | ✅ |
| privacy | client_test.go (190) | ✅ |
| scripting | integration_test.go (1262), integration_test_batchd_test.go (220) | ✅ |
| spells | spell_info_test.go (263), saving_throws_test.go (226) | ✅ |
| validation | validation_test.go (139) | ✅ |

## Untested Packages (11)
- agent/ — 1 file (memory_hooks.go)
- ai/ — 2 files (behaviors.go, brain.go) — mostly interfaces + game-layer wired
- audit/ — 1 file (logger.go) — file I/O, needs temp dir setup
- command/ — 13 files, 3.1K LOC — command routing, shop cmds, skill cmds — large
- common/ — 4 files, 185 LOC — all interfaces
- db/ — 3 files, 795 LOC — player persistence, narrative memory — needs SQLite
- optimization/ — 6 files, 2.3K LOC — object pools, caches — complex
- secrets/ — 1 file (manager.go) — crypto/file I/O, needs env setup
- session/ — 70+ files, 13K LOC — session management, command dispatch — very large
- storage/ — 2 files (interface.go, sqlite.go) — needs SQLite
- telnet/ — 1 file (listener.go) — network I/O

## Real Bugs Found
1. ExtraDesc trailing newline — parser joins `~`-terminated desc lines with `\n`, trailing `~` on its own line adds trailing newline. **Accepted as valid parser behavior** (not a bug per DikuMUD convention).
2. Dollar end marker — `$` in obj files is a block delimiter (E/A/S inner loop), not a file terminator. Outer loop continues. **Not a bug** — matches C parser behavior correctly.

**No actual code bugs found.** The code is solid.

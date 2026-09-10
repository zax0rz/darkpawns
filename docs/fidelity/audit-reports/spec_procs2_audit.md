# Port Fidelity Audit: Module 51 (`spec_procs2.c`)

This audit examines the port fidelity between the legacy C source file `src/spec_procs2.c` and its Go counterparts in `pkg/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/spec_procs2.c` (2,300 lines)
- **Functions & Features**:
  - **Advanced Regional Mechanics**: Implements unique gameplay systems including game rooms (`normal_checker` for the checkerboard), petrifying gazes (`medusa`), item stealers (`eq_thief`), remort facilitators (`remorter`), custom alchemist shops (`pissedalchemist`), active room blockers (`no_move_east`, `no_move_west`), and castle protection guards (`castle_guard_north`, `castle_guard_down`).

### Go Port Files
- **Go Implementation**:
  - [pkg/game/spec_procs2.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/spec_procs2.go): Full port of advanced special procedures, including all movement blockages, remorting, medusa stone gazes, and game checkerboards.

---

## 2. High-Fidelity Validation

The Go implementation in `spec_procs2.go` is exceptionally thorough and faithful to the original C mechanics:

1. **The Checkerboard System (`specNormalChecker`)**:
   - Flawlessly ports the complex logical coordinates and movement rules for checkerboard game room NPCs, matching the original matrix and movement constraints.
2. **Medusa Petrifying Gaze (`specMedusa`)**:
   - Accurately captures Medusa's gaze attack, calling `spells.Cast` for `SpellPetrify` on a random 1-in-4 probability during active combat.
3. **Multi-Directional Castle Guards**:
   - Flawlessly ports directional blockers (`specCastleGuardEast`, `specCastleGuardNorth`, `specCastleGuardDown`, etc.) that intercept exit requests (`CmdIS("north")` etc.) and verify castle permissions before permitting passage.
4. **Remorter Facilitator**:
   - Seamlessly ports the remort checking and character resetting sequence (`specRemorter`), ensuring proper character reincarnation attributes.

---

## 3. Go's Architectural Improvements Over C

- **Robust State Isolation**: In C, checking checkerboard pieces required iterating global arrays with raw memory offsets. Go leverages safe, locked maps and slices, preventing pointer corruption.
- **Clean Registry Coupling**: All procedures in `spec_procs2.go` are safely registered via `init()` to the central `SpecRegistry` at boot time, preventing compilation leaks.

---

## 4. Summary of Recommended Next Steps

1. **Verify Spell Integration**:
   Ensure `SpellPetrify` has comprehensive unit test coverage within `pkg/spells/` to prevent regression of Medusa's petrifying combat routines.

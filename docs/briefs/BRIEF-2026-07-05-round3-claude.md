# Brief: Round 3 Claude — Mob Dice Parser (CRITICAL) + Zone L Command (Fidelity)

**Issues:** DP-806, DP-895
**Date:** 2026-07-05
**Priority:** Medium (2) — DP-806 is functionally critical
**Effort:** M each

---

## DP-806: Mob dice-expression stats silently left at zero (CRITICAL — Fidelity Bust)

**Problem:** `pkg/parser/mob.go:260-279` — The stats line parser assumes 9 space-separated integer fields (`38 -18 -28 38 5 5078 25 4 25`), but actual `.mob` files in `lib/world/mob/` use `NdS+P` dice notation (`38 -18 -28 38d5+5078 25d4+25`). When `strings.Fields()` produces only 5 tokens, the `if len(stats) >= 9` guard fails and **all dice fields remain zero**. Every mob in the game gets 100 HP (default fallback in `pkg/game/mob.go:91-96`) and zero base damage (0d0+0).

**Impact:** A level 38 mob that should have average HP of ~5192 (38×3+5078) gets 100. All mobs deal zero base dice damage. This affects every mob loaded from `.mob` files.

**Cite:** C source — `src/db.c:1041-1047`. Uses `sscanf` with format `" %d %d %d %dd%d+%d %dd%d+%d "` which natively parses `NdS+P` dice notation into separate integer fields. On parse failure, C calls `exit(1)` — never silently continues. The Go port lost this dice parsing capability entirely.

**Fix:** Add a `parseDiceExpr(s string) (num, sides, plus int, err error)` helper that splits on 'd' and '+'/'-' (e.g. `"38d5+5078"` → 38, 5, 5078). Support variants: `NdS`, `NdS+P`, `NdS-P`. Then update mob.go stats parsing to handle both 5-field (dice notation) and 9-field (space-separated) formats for backward compatibility with test data. Log a warning on parse failure instead of silently zeroing.

**Regression Test:**
1. Parse a real `.mob` file with `38d5+5078 25d4+25` format — verify HP.Num=38, HP.Sides=5, HP.Plus=5078, Damage.Num=25, Damage.Sides=4, Damage.Plus=25.
2. Edge cases: `1d1+0`, `1d1`, `10d10+100`, `0d0+5`, negative plus values.
3. Backward compat: old 9-field format should still parse correctly.
4. Verify mob HP calculation in `pkg/game/mob.go:91-96` uses parsed dice correctly.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/parser/... && go test ./pkg/game/...`

---

## DP-895: Zone 'L' command parsing semantics disagree with C (Fidelity)

**Problem:** Three mutually contradictory interpretations of the zone 'L' command exist:

| Location | Comment says Arg3 is | Code treats Arg3 as |
|---|---|---|
| `pkg/parser/zon.go:122,172` | `lock_state` (integer 0/1/2) | Parsed as generic int |
| `pkg/parser/parser.go:172,177-179` | `key_vnum` (object vnum) | Validated against `objVnums` |
| `src/db.c:2097-2104` | Loop count | Used as loop iteration count |

The Go port misinterpreted 'L' as a door/lock command. In C, 'L' means **"Start/End Looping"** — a flow-control construct for zone resets. `arg2` = loop control (0 = start, non-zero = end/decrement), `arg3` = loop count. The parser.go validation checking arg3 as an object vnum is wrong.

**Cite:** C source — `src/db.c:2097-2104` (zone reset execution):
```c
case 'L':         /* Start/End Looping */
    if (!ZCMD.arg2) {
        tmp_cmd = cmd_no;
        loop = ZCMD.arg3;
        last_cmd = 1;
    } else
        if (--loop > 0)
            cmd_no = tmp_cmd;
    break;
```
Also `src/db.c:997` — vnum resolution: 'L' and 'D' share same case, only `arg1` (room vnum) resolved via `real_room()`.

**Fix:**
1. In `zon.go`: Update the 'L' case comment from "Lock door" to "Start/End Looping". arg2 = loop control, arg3 = loop count.
2. In `parser.go`: Remove the `objVnums[cmd.Arg3]` validation for 'L' commands — arg3 is a loop count, not an object vnum.
3. Verify that zone reset execution in the Go code properly handles 'L' as loop control (may already work if arg2/arg3 are parsed correctly as ints).
4. Check whether any `.zon` files actually use 'L' commands and verify behavior matches C.

**Regression Test:** Parse a zone file with 'L' commands, verify they are interpreted as loop control, not door locking.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/parser/...`

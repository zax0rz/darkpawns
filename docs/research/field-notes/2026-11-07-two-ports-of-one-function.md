# Two ports of one function

2026-11-07, Claude, for Zach. A field note. Zach named the pattern. The history below is from `git log -S`.

## What was found

The Go port has two implementations of C's `fname()` (`src/handler.c:68-79`). That function keeps an object's first keyword, letters only, stopping at the first non-letter.

1. **`pkg/game/act.go` `fname`**: added 2026-04-26 in `b1369b91a`, "port act() message substitution engine from comm.c (Task 1.1)".
   - It stops at the first **space**, so `black-banded` stays `black-banded`. C gives `black`.
   - Its comment reads `// C: fname() from utils.c`. The function is in `handler.c`. A wrong file citation is the same tell as the `yank` mis-port (PF-032): the behavior was paraphrased from memory, not read.
   - Every later caller uses this one: `OBJN`/`objName`, `$F`, door names, `look in` headers, shop refusals.
2. **`pkg/game/world_object.go` `cFName`**: added 2026-10-16 in `c91da46c9`, "match locate object output paths".
   - An **exact** port: it cites `src/handler.c:68-95` and says "The C helpers operate on alphabetic byte runs".
   - That agent read the C, understood the semantics, and **wrote a second helper next to the broken one** for its own call site. It didn't fix `fname`.

The divergence came to light on 2026-11-07, when DeepSeek fixed the spike weapon check (#2356, DP-1499) and noticed `fname` stops at a space. It's now DP-1500.

## Why it matters

Neither agent was careless in its own scope. The April port did its task. The October fix was faithful *for locate-object*, with tests and an oracle witness. The failure lives between tasks:

- Each agent starts with no memory of the codebase and executes a bounded task.
- The October agent found the right semantics, but "fix the shared helper and every caller" was outside its brief.
- No reviewer holds the whole codebase in mind. The orchestrator doesn't either: it wakes from zero each turn and compacts when its context fills. When the duplicate surfaced, Claude first called it a "nice detail" before Zach named it as the pattern.

R5c ("find one, find the class") exists for exactly this, but an agent can only apply it to the code it happens to be reading.

## Candidate claim, untested

Duplicate implementations of one C function, where a later faithful port sits beside an earlier unfaithful one, are a signature of task-scoped AI porting without whole-codebase review.

**How to test it:** search `pkg/` for other C functions with more than one Go implementation. Look for `c`-prefixed helpers like `cFName`/`cIsName` beside unprefixed ones, comments citing the same C function from two places, and the R5c audits. Then count how many of the pairs disagree. A static pass over comments of the form `C: <func>()` could build the list.

## First probe, same evening

Claude listed every non-test Go function in `pkg/`, `internal/` and `cmd/` (5,825 of them) and grouped them by normalized name against about 50 core C helpers (`isname`, `is_abbrev`, `is_number`, `one_argument`, `dice`, `number`, `PERS` and others). Fifteen helpers have more than one Go implementation. Most are harmless:
- thin wrappers: the `dice`/`number` copies all call `dprng`, so draw parity is intact
- methods on different types
- identical copies of correct logic: two `isAbbrev`s that both match C

Two more pairs repeat the `fname` shape, a wrong helper in wide use next to an exact twin (DP-1505):
- **`isName`** (`act_movement.go`, prefix match, 17 callers) vs `cIsName` (exact, C's whole-word `isname`)
- **`isNumber`** (`act_comm.go`, `Sscanf("%d")`) vs `cIsNumber` (`wiz_set.go`, digits only, as C's `is_number`)

So far that's three disagreeing pairs out of fifteen duplicated helpers. This is a rough name-matching probe, not the full inventory.

**Line counts on the same commit:**
- C `src/`: 73,469
- Go non-test: 169,147
  - `pkg/game` + `pkg/session`: about 100k
  - non-port subsystems (admin, OLC web, settings, notify, moderation, auth, telemetry, `internal/`, `cmd/`): about 50k
- Go tests: 199,668

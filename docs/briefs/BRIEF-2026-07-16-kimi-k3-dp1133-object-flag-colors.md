# BRIEF 2026-07-16 — Kimi K3: DP-1133 object-flag color parity

**Executor:** Kimi K3 in Kimi Code. **Reviewer/gate owner:** Codex.
**Branch:** `kimi/dp1133-object-flag-colors`, created fresh from current `main` in
your own checkout/worktree. **One focused PR; do not merge it.**
**Verified brief base:** `main` at `66f02adf` (PR #371 merged).

Read this entire brief before editing. This task is intentionally small but has
a fidelity trap: the Linear ticket overstates where C applies color. Match the C
source and the context matrix below, not a global interpretation of the ticket.

## Goal

Fix DP-1133 so Dark Pawns object annotations use C's ANSI colors in the views
that call C `show_obj_to_char`, while preserving plain text in views that call
C `oc_show_list`.

For a viewer at **complete color level** (`COLOR_LEV == C_CMP == 3`):

- blessed + detect alignment: `blue glow` is blue (`KBLU`, `\x1b[34m`);
- magic + detect magic: `yellow glow` is yellow (`KYEL`, `\x1b[33m`);
- glow: `glowing` is white (`KWHT`, `\x1b[37m`);
- each colored phrase resets with `KNRM` (`\x1b[0m`) before its closing `)`;
- invisible and humming annotations remain uncolored.

At color levels 0, 1, and 2, all annotations remain byte-for-byte plain.

## The fidelity trap: correct the ticket with source

DP-1133 says both `show_obj_to_char` and `oc_show_list` colorize the flags. That
is not what this repository's C source does.

### C path A: `show_obj_to_char` — color at exact level 3

Authoritative source: `src/act.informative.c:95-170`.

For bless, magic, and glow, C constructs the annotation in this exact order:

```c
strcat(buf, " (");
if (COLOR_LEV(ch)==C_CMP)
  strcat(buf, KBLU /* or KYEL / KWHT */);
strcat(buf, "blue glow" /* or yellow glow / glowing */);
if (COLOR_LEV(ch)==C_CMP)
  strcat(buf, KNRM);
strcat(buf, ")");
```

`src/screen.h` defines:

```c
#define KNRM "\x1B[0m"
#define KYEL "\x1B[33m"
#define KBLU "\x1B[34m"
#define KWHT "\x1B[37m"
#define C_CMP 3
```

This path is used for direct object look/examine, self equipment, equipment on a
looked-at character, and the C peek-inventory path.

### C path B: `oc_show_list` — deliberately plain

Authoritative source: `src/oc.c:82-170`. It prints `...it glows blue`,
`...it glows gold`, and `...it glows white` without ANSI at every color level.
`list_obj_to_char` routes room contents, container contents, and ordinary player
inventory through this path (`src/act.informative.c:241-285`, calls around
lines 832, 960, and 1465).

Therefore:

| Go view | C analogue | ANSI at level 3? |
|---|---|---:|
| direct `look <object>` / `examine <object>` flags | `show_obj_to_char`, modes 5/6 | yes |
| `equipment` | `show_obj_to_char`, mode 1 | yes |
| equipment shown by `look <character>` | `show_obj_to_char`, mode 1 | yes |
| room object list | `list_obj_to_char` → `oc_show_list` | no |
| container contents | `list_obj_to_char` → `oc_show_list` | no |
| ordinary `inventory` | `list_obj_to_char` → `oc_show_list` | no |

A patch that globally injects color into every `objectVisibleFlags` result is
wrong and must fail the negative tests below.

## Current Go anchors

### `pkg/game/look.go`

- `objectVisibleFlags` (~line 750) builds plain `(blue glow)`, `(yellow glow)`,
  `(glowing)`, etc.
- It is shared by both C path types:
  - **color:** `appendObjectLook`, `appendPlayerEquipment`,
    `appendMobEquipment`;
  - **plain:** `roomObjectLines`, `visibleObjectShortLines` (container contents).
- `observationColors` (~line 1013) emits color when computed level is **at
  least 2**. Do not call it blindly here: C requires exact complete level 3 for
  these object annotations.

### `pkg/game/item_views.go`

- `DoEquipment` calls the duplicate `equipmentVisibleFlags`, which is plain.
- `DoInventory` / `formatOCShowListLine` is the Go `oc_show_list` path and must
  stay plain.

## Implementation constraints

Use one context-aware flag renderer for the identical parenthesized annotation
vocabulary. A small API such as either of these is acceptable:

```go
func objectVisibleFlags(ch *Player, object *ObjectInstance, colorize bool) string
```

or:

```go
func objectVisibleFlags(ch *Player, object *ObjectInstance) string       // plain
func coloredObjectVisibleFlags(ch *Player, object *ObjectInstance) string
```

The implementation must satisfy all of the following:

1. Only the three words/phrases are wrapped in ANSI. Parentheses and the leading
   space are not colored. Exact complete-color output for an object carrying all
   three relevant flags is:

   ```text
    (\x1b[34mblue glow\x1b[0m) (\x1b[33myellow glow\x1b[0m) (\x1b[37mglowing\x1b[0m)
   ```

2. Require both `PrfColor1` and `PrfColor2`, i.e. computed level 3. Levels 0,
   1, and 2 remain plain. You may add a narrowly named helper such as
   `completeObservationColor`; do not change `observationColors` semantics for
   unrelated views unless you first prove every existing caller wants a
   different threshold.
3. Use the colored form only in:
   - `appendObjectLook` (including modes 5 and 6),
   - `appendPlayerEquipment`,
   - `appendMobEquipment`,
   - `DoEquipment`.
4. Keep the plain form in:
   - `roomObjectLines`,
   - `visibleObjectShortLines` / container listings,
   - `DoInventory` / `formatOCShowListLine`.
5. Remove or delegate `equipmentVisibleFlags` if practical so the two
   parenthesized renderers cannot drift again. Do not broaden this into a general
   color-system refactor.
6. Preserve annotation order and visibility gates: invisible, bless+detect
   align, magic+detect magic, glow, hum.

`(covered)` is not modeled in Go (there is no covered-slot implementation in
`pkg/game` at this base). Do not invent that model in this PR.

## Required tests — prove RED before implementation, GREEN after

Add focused tests in `pkg/game` (prefer `item_views_test.go` plus an existing
look/observation test file if integration coverage belongs there).

### 1. Exact helper/context matrix

Build an object with bless, magic, and glow flags. Give the viewer detect-align
and detect-magic. Table-test color levels 0, 1, 2, and 3 by setting
`PrfColor1`/`PrfColor2` with `SetPlrFlag`.

- Levels 0/1/2: exact plain string
  `" (blue glow) (yellow glow) (glowing)"`.
- Level 3: exact ANSI string
  `" (\x1b[34mblue glow\x1b[0m) (\x1b[33myellow glow\x1b[0m) (\x1b[37mglowing\x1b[0m)"`.

Include invisible and hum in at least one case and assert they contain no color
codes.

### 2. Positive integration paths

At complete color level, assert exact ANSI appears in:

- `DoEquipment` output;
- direct object look/examine output (exercise `DoLookTarget` or the narrowest
  public path that reaches `appendObjectLook`);
- equipment rendered while looking at another player or mob, if the existing
  observation test scaffolding makes this concise.

At least `DoEquipment` and direct object look are mandatory. A helper-only test
is not sufficient.

### 3. Negative integration controls (load-bearing)

At complete color level, assert **no `\x1b[` occurs** in:

- `DoInventory` output;
- the isolated room object line (`roomObjectLines` is directly callable from a
  same-package test; do not reject unrelated room-title ANSI from a full room
  view);
- a container-contents list if it can be covered concisely with existing test
  scaffolding.

Inventory and room are mandatory. These tests prevent the tempting but wrong
global-color patch.

### 4. Capability gates

Assert missing detect-align hides the bless annotation and missing detect-magic
hides the magic annotation, unchanged from current semantics.

Before editing production code, run the new positive tests against `main` and
capture the failing output in your handoff. After implementation, run the same
named tests and capture their passing output. Do not claim red→green without
showing the exact test command and result.

## Oracle methodology note

Do not add or modify a differential scenario for this task.
`internal/oraclediff/normalize.go` strips ANSI, so the oracle cannot distinguish
this bug from correct output. The unit tests asserting raw escape sequences are
the fidelity gate.

**Never edit** `internal/oraclediff/normalize.go` to make this test observable;
that would change the harness contract and contaminate unrelated scenarios.

## Scope lock

Expected production files:

- `pkg/game/look.go`
- `pkg/game/item_views.go`

Expected test files: existing `pkg/game/*_test.go` files only.

Also allowed: this brief need not be edited by the executor.

Do not touch:

- anything under `src/` (read-only oracle reference);
- `internal/oraclediff/normalize.go` or oracle scenarios;
- combat damage colors / `DamMessage` (a separate color-parity sibling);
- DP-1131 stacked-object vocabulary or spacing;
- covered equipment modeling;
- session color-command behavior;
- website files, maps, or generated reports.

If the correct fix requires production files outside the two expected files,
stop and explain why before expanding scope.

## Required repository gate

Run every command yourself after `make fmt`:

```bash
make fmt
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

All four verification commands must pass. Do not suppress errors with new
`#nosec` annotations. Do not rely on an agent/subagent's report; include the
terminal results in the handoff.

## Commit, PR, and handoff

Use a conventional commit, for example:

```text
fix: color object flags in C show-object views
```

Push `kimi/dp1133-object-flag-colors` and open one PR to `main`. The PR body
must link DP-1133 and summarize the crucial path distinction: show-object views
color at complete level; oc-list views remain plain.

Then stop. Do not merge, do not mark DP-1133 Done, and do not modify unrelated
Linear issues. Return all of:

1. PR URL, branch, and commit SHA;
2. changed-file list;
3. named RED test command + failure observed on pre-fix `main`;
4. named GREEN test command + passing result;
5. results for format, build, vet, full tests, and golangci-lint;
6. any uncertainty or deviation from this brief.

Codex will inspect the diff independently, re-run the gates, verify the C path
matrix, and decide whether DP-1133 can close.

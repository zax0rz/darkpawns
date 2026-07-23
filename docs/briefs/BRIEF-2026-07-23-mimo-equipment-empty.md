# BRIEF (mimo) — equipment display: C-faithful empty branch + header

**Owner:** mimo-v2.5-pro. **Gate:** Claude runs the differential oracle red→green.
**Git:** NONE — isolated worktree, no git commands; edit, test, report. Operator commits.
**Closes:** DP-1198. Sized: S/M.
**Cite:** `src/act.informative.c:1468-1495` (do_equipment); rules **R1**, **R4**
(`docs/fidelity/RULEBOOK.md`).

## The C truth (the whole function is 27 lines — read it)

```
"You are using:\r\n"                          ← ALWAYS, first, even when empty
per worn slot (in wear-position order):
  visible   → where[i] + the object's short display (show_obj_to_char mode 1)
  invisible → where[i] + "Something.\r\n"
nothing worn at all → " Nothing.\r\n"          ← leading space, after the header
```

The `where[]` slot-label strings live in act.informative.c (grep `where[] =` /
`<used as light>` style labels) — byte-copy them if the Go labels differ.

## The Go bug

The current equipment display (find it: the test asserting it is
`pkg/session/equipment_ac_test.go`, marked with a DP-1198 comment) invents
"You are not wearing anything." for the empty case, and the tests also assert an
"Armor Class: 10" line — verify whether the Go equipment command prints an AC
line at all; C's do_equipment does NOT (AC belongs to score). If Go's equipment
handler prints AC, that's an R4 invention to remove; if the AC assertion belongs
to a different (score-path) helper, leave score alone — score is oracle-green.

## Fix

1. Empty case: `You are using:\r\n Nothing.\r\n` — exact bytes, leading space
   included.
2. Non-empty case: header + slot label + item short-desc per C's order; the
   invisible-object branch prints `Something.\r\n` (only reachable with
   invisible gear — implement it faithfully even though tests can't easily
   reach it; note it).
3. Remove any AC line from the equipment display if present (with a note in
   your report). Do NOT touch score.
4. Update the DP-1198-marked test expectations to the C truth (the marker says
   exactly this) and remove the marker comments — they exist to be retired.

## Tests

- empty equipment → exact two-line output
- one worn item → header + correct slot label + item name, no AC line
- keep/extend anything the marked tests covered that's still true

## Verification

`go build ./... && go vet ./pkg/session/... && go test ./pkg/session/...` green,
plus `gofumpt -w` on every file you touch (CI enforces it). No git. End with the
list of files modified.

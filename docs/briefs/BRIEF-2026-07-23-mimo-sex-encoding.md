# BRIEF (mimo) — sex-encoding convergence: fix genderPronoun + truth in comments

**Owner:** mimo-v2.5-pro. **Gate:** Claude reviews; oracle probe only if the slap
path proves mortal-reachable.
**Git:** NONE — isolated worktree, no git commands. Operator commits.
**Closes:** DP-1196's sibling DP-1197. Sized: S.
**Cite:** DP-1197 audit comment (read it first — the map of who uses which
encoding); `src/structs.h:203-205`; rules **R4**, **R5c**.

## Ground truth (from the audit — do not re-derive, verify)

Go's actor encoding is 0=male, 1=female, 2=neutral — consistently used by
creation, storage, `pkg/game/act.go` helpers, and `objectivePronoun`. C-encoded
mob data translates at `MobInstance.GetSex()` (pkg/game/mob.go:195). The ONLY
code outlier is `genderPronoun` (pkg/command/skill_commands.go:1491): it returns
1→himself / 0→herself — inverted. Its single call site (skill_commands.go:1223)
makes male players render "herself".

## Fix

1. **Constants**: add to pkg/game (near the Actor/Sex definitions):
   `SexMale = 0`, `SexFemale = 1`, `SexNeutral = 2` with a comment stating
   plainly: this is GO's actor encoding, deliberately different from C's
   SEX_* (structs.h: 0=neutral/1=male/2=female); C-encoded data translates at
   the Actor boundary (see MobInstance.GetSex).
2. **genderPronoun**: fix to the Go encoding (male→himself, female→herself,
   neutral→itself) using the constants. Update the DP-1197-marked test in
   pkg/command/skill_commands_test.go to the corrected truth and remove the
   marker comment.
3. **Lying comments**: rewrite `pkg/game/player.go:72` and the
   `objectivePronoun` header (pkg/session/use_cmds.go:104-105) — both currently
   claim C-parity for the non-C convention. State the Go encoding + boundary
   translation instead.
4. **Magic numbers**: replace bare 0/1/2 sex literals with the constants at the
   obvious sites (act.go helpers, GetSex translation, player.go:348,
   session_login.go:95, char_creation.go, genderPronoun, objectivePronoun).
   Do NOT touch pkg/parser or pkg/admin (they carry C-encoded world data —
   leave their raw ints, add a one-line comment where they cross into game).
5. **Slap provenance check (report-only)**: identify which command reaches
   skill_commands.go:1223, and grep C source for "slaps" / the message shape.
   If C implements it via the socials data file rather than code, REPORT that
   (with the C cite) — do not rewrite the message in this brief.

## Tests

- genderPronoun table test: male→himself, female→herself, neutral→itself
- constants match the documented values (cheap tripwire against future drift)
- existing act.go / pronoun tests still green

## Verification

`go build ./... && go vet ./... && go test ./pkg/command/... ./pkg/game/... ./pkg/session/...`
green; `gofumpt -w` every touched file. No git. End with files modified + the
slap-provenance finding.

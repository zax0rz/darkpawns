# BRIEF (glm) — port the `skillset` immortal command (R2 debt; unblocks skill-layer gating)

**Owner:** glm-5.2. **Gate:** comprehensive unit tests matching C byte-for-byte
(now) + Claude adds an oracle scenario once the immortal-fixture harness support
lands (in parallel). CI green.
**Git:** branch off `main` as `glm/skillset`, commit, push, open a PR. Do NOT
merge. Sized to one PR (S/M).
**Closes:** reachability debt — `skillset` is flagged `missing / no implementation
found` on the ratchet (`docs/reports/reachability-2026-07-23.tsv:369`). After this
it becomes `registered` (the ratchet *improves*).
**Why now:** `skillset` is the faithful, in-game way to grant a fixture character
a skill so we can oracle-gate the combat skill layer (bash/kick/disarm/…) as
openers — instead of inventing a test seam. It's owed under **R2** regardless.
**Cite:** `src/modify.c:255-330` (do_skillset), `src/spell_parser.c:370`
(find_skill_num), `src/interpreter.c:704` (registration, `LVL_GRGOD`,
`POS_SLEEPING`); rules **R1**, **R2**, **R4** (`docs/fidelity/RULEBOOK.md`).

## The C truth — do_skillset, exactly

Registered: `{ "skillset", POS_SLEEPING, do_skillset, LVL_GRGOD, 0 }`. Syntax:
`skillset <name> '<skill>' <value>`. Steps, in order, each with its exact bytes:

1. **No argument** → print syntax + the full skill list:
   - `"Syntax: skillset <name> '<skill>' <value>\r\n"`
   - then `"Skill being one of the following:\n\r"` followed by every entry of the
     `spells[]` table, **4 per line**, each formatted `%18s` (18-wide,
     right-justified), **skipping entries whose name starts with `!`**, a
     `"\r\n"` after every 4th, and a trailing `"\n\r"`. Reproduce the column
     layout byte-for-byte (Go already has the `spells[]` names — see
     `practice.go` / `spells.GetSpellName`; iterate the same table in the same
     order, same `!`-skip, same widths).
2. **Target lookup** `get_char_vis(ch, name)`; not found → `NOPERSON`
   = `"No-one by that name here.\r\n"` (the C global — use this exact string, the
   one already used in look.go/movement; do NOT invent a variant — see DP-1200).
3. `skip_spaces`; empty rest → `"Skill name expected.\n\r"`.
4. First non-space must be `'` → else `"Skill must be enclosed in: ''\n\r"`.
5. Read to the closing `'` (lowercasing inside); no closing quote →
   `"Skill must be enclosed in: ''\n\r"`.
6. `find_skill_num(quoted)` → skill number; `<= 0` → `"Unrecognized skill.\n\r"`.
   **Go already has `FindSkillNum` (practice.go:134) — reuse it verbatim**
   (case-insensitive prefix match, first/lowest match).
7. Next arg = value; empty → `"Learned value expected.\n\r"`; `atoi`;
   `< 0` → `"Minimum value for learned is 0.\n\r"`; `> 100` →
   `"Max value for learned is 100.\n\r"`.
8. `IS_NPC(vict)` → `"You can't set NPC skills.\n\r"`.
9. `mudlog(...)` — **server-side only, NOT player-facing: skip it**, note the skip
   in the PR (same policy as do_help's usage-file write).
10. `SET_SKILL(vict, skill, value)`.
11. Confirmation to the actor: `"You change %s's %s to %d.\n\r"` with
    `GET_NAME(vict)`, the skill's `spells[]` display name, and the value.

### ⚠️ Newline conventions are byte-exact and MIXED
C uses `\n\r` for most of these (`"Skill name expected.\n\r"`, the confirmation,
etc.) but `\r\n` for the syntax line and `NOPERSON`. Copy each terminator exactly
as the C source has it — do not normalize to one form. A normalized newline is a
fidelity bug (R1).

## The Go plumbing

- **Register** like its siblings `set`/`advance`/`restore` (already in
  `commands.go:207-219`) — same `LVL_GRGOD` gate, `POS_SLEEPING` min-position.
  Follow how those wiz commands set their min level (mirror `cmdSet`/`cmdAdvance`
  registration + `commandGate`).
- **Skill number ↔ name bridge:** Go stores skills by name string
  (`Player.SetSkill(name, value)`, player_affects.go:154), C by number.
  `FindSkillNum(arg)` → number; map back to the canonical display name
  (`spells.GetSpellName(num)` / the same table used in step 1) and call
  `SetSkill(canonicalName, value)`. Confirm the round-trip name matches the
  `spells[]` display name used in the confirmation string.
- **Target resolution:** mirror C `get_char_vis` scope for a wiz command
  targeting a player by name; on miss emit the exact `NOPERSON` bytes.

## Tests (this carries verification until the oracle scenario lands)

Byte-for-byte unit tests, one per message path — assert the EXACT string incl.
the specific `\n\r` vs `\r\n` terminator:
- no-arg syntax + skill-list formatting (widths, 4/line, `!`-skip, terminators) —
  assert against a known slice of the `spells[]` table;
- NOPERSON on unknown target;
- each error: skill-name-expected, must-be-quoted (open and unterminated),
  unrecognized-skill, value-expected, min<0, max>100, NPC-target;
- **success:** `skillset <player> 'backstab' 75` → `GetSkill("backstab") == 75`
  and the confirmation line is byte-exact.
- level gate: a mortal (< LVL_GRGOD) gets the standard command-not-found gate,
  not skillset.

## Oracle gate (Claude, in parallel — informational)

`skillset`'s own gate needs an immortal actor, which comes from the
empty-player-store God-bootstrap harness support Claude is scoping now. Once it
lands, Claude authors a `skillset` scenario (a first-player God runs the no-arg
list, a successful set, and the error paths as probes). Until then, the unit
tests above are the gate. Design the command so nothing depends on test-only
state — same code path for interactive and oracle.

## Guardrails

- **Never** edit `src/` or `darkpawns-c-oracle/` — reference only.
- `make reachability` — `skillset` should flip `missing` → `registered` (do NOT
  hand-edit the report; the generator picks it up from the registry).
- `go test -race`; **run `golangci-lint`**; `gofumpt -w` every file you touch.
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.

## Deliverable

A faithful `do_skillset` port registered at `LVL_GRGOD`, reusing `FindSkillNum`,
with byte-exact messages (mind the mixed `\n\r`/`\r\n`), the mudlog skipped, and
full unit tests. Claude adds the oracle scenario once the immortal fixture lands.

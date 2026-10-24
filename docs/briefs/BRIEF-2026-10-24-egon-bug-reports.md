# Fix brief: Egon bug reports, 2026-10-24

- **Reporter:** Egon (player), via in-game mudmail to Stink, Sat Oct 24, 2026, 05:45 in-game time.
- **Triage:** Chungus (verifier lane), Sat Oct 24, 2026, morning. All C cites are the C oracle tree (`darkpawns-c-oracle`, branch `dp-oracle-seam`); all Go cites are `zax0rz/darkpawns` main as of Oct 24, 2026, checked via GitHub code search and file reads.
- **Implementer:** GLM-5.3 (handoff from Zach).
- **Reviewer/merger:** Zach. Nothing here is approved for production deploy; Zach merges and deploys.
- **Scope:** Two confirmed code fixes (Fix 1, Fix 2) plus two investigations that must report before any code changes (Investigation 3, Investigation 4). One explicit non-issue at the end — do not "fix" it.

## Summary

| # | Report | Verdict | Work |
|---|--------|---------|------|
| 1 | `track` returns "huh?!", command not recognized | Confirmed code gap | Implement + register the player `track` command |
| 2 | Mortal `visible` prints "You fade into view" but the player stays invisible; shopkeepers cannot see them | Confirmed code bug | Make the mortal `visible` path strip the invis spell affect, like C `appear()` |
| 3 | Fleeing passes locked doors and lands in death traps | Guards present on main; almost certainly production lagging main | Verify prod build version first; code change only if it reproduces on a build that has the guards |
| 4 | `house guest Egon` returns "Um.. this house seems to be screwed up" | Missing house control record for that room, or wrong room | Read-only diagnosis first; no data edits without Zach's explicit word |

## Fix 1: the `track` command does not exist as a player command

### Symptom

Egon: "track doesn't work at all... i just get a huh?!, as command not recognized."

### C behavior (the spec)

- Registration: `src/interpreter.c:782`: `{ "track", POS_STANDING, do_track, 0, 0 }`. No level gate; standing position.
- Handler: `ACMD(do_track)`, `src/graph.c:187-251`. Semantics, in order:
  1. Class gate: must be warrior, paladin, or ranger, else `You have no idea how.\r\n`.
  2. Skill gate: must have `SKILL_TRACK`, else the same message.
  3. No argument: `Whom are you trying to track?\r\n`.
  4. Target not visible/found (`get_char_vis`): `You can't sense a trail to them from here.\r\n`.
  5. Target has `AFF_NOTRACK` or is `MOB_SENTINEL`: `You sense no trail.\r\n`.
  6. Non-NPC target with `SKILL_EVASION`: roll `number(1, 151)`; if roll <= evasion skill, `You sense no trail.\r\n`.
  7. `dir = find_first_step(ch->in_room, vict->in_room)`:
     - `BFS_ERROR`: `Hmm.. something seems to be wrong.\r\n`.
     - `BFS_ALREADY_THERE`: `You're already in the same room!!\r\n`.
     - `BFS_NO_PATH`: `You can't sense a trail to <HMHR(vict)> from here.\r\n`.
     - Otherwise: `num = number(0, 101)`; if the tracker is outside and it is raining, `num += 35`. If `num >= SKILL_TRACK` (failure), pick a random direction, retrying up to 10 times while `!CAN_GO`. On success, `improve_skill(ch, SKILL_TRACK)`. Then if `CAN_GO(ch, dir)`: `You sense a trail <dirs[dir]> from here!\r\n`, else `There doesn't seem to be any way out of here!\r\n`.

### Go state (the gap)

- The BFS primitive is ported: `pkg/game/graph.go` has `FindFirstStep` / `findFirstStep`, plus `huntVictim` and related mob-hunting machinery.
- No player-facing `track` handler was found on main by code search. Two docs mention a `doTrack` (`docs/reports/wiring-plan.md` says, verbatim, "doTrack needs to be registered as a command"; `docs/fidelity/audit-reports/graph_audit.md` describes a player `doTrack()` in `graph.go`), but the current `graph.go` function list contains no such handler. Treat the docs as stale until proven otherwise during implementation: if a handler does exist somewhere, register it instead of writing a second one; if it does not, write it.

### Fix

1. Implement the player `track` command with the exact C gates, rolls, and messages above. Copy message strings byte-for-byte, including punctuation and `\r\n`. Do not invent new messages and do not "improve" the wording.
2. Reuse the existing `findFirstStep` BFS. Check its `BFS_ERROR` / `BFS_ALREADY_THERE` / `BFS_NO_PATH` sentinel values against the C definitions and use them, not new ones.
3. Register it in the session command table following the existing pattern, e.g. `pkg/session/commands.go`: `registerCommand("visible", wrapArgs(cmdVisible), ...)` is the model. Position gate standing, no level gate, matching C.
4. Do not add a `vis`-style abbreviation or alias (see Non-issue below).

### Proof required

- Unit tests covering, at minimum: class gate, skill gate, missing argument, target-not-found, NOTRACK/sentinel, evasion roll, all three BFS sentinel outcomes, and the success message naming the correct direction.
- Revert test: with the registration removed, typing `track` must return the parser's unrecognized-command response; with it restored, the command works. Demonstrate this, do not just assert it.
- Full `go test ./...` green, plus the repo's lint gate.

## Fix 2: mortal `visible` does not remove spell invisibility

### Symptom

Egon (mortal character): after becoming invisible, he could not go visible again. Shopkeepers could not see him, so he could not trade. He described it as annoying; in practice it is a soft lockout from every shop for the spell's whole duration.

### C behavior (the spec)

- Registration: `src/interpreter.c:802`: `{ "visible", POS_RESTING, do_visible, 1, 0 }`.
- Handler: `ACMD(do_visible)`, `src/act.other.c:557-590`:
  - If affected by `SKILL_KK_ZAI`: `You cannot become visible until your zai ends!\r\n`, return.
  - If immortal: `perform_immort_vis`, return.
  - If `AFF_INVISIBLE`: `appear(ch)` + `You fade into view.\r\n`.
  - If `AFF_SNEAK`: `You stop sneaking.\r\n`, remove sneak and stealth affects, clear the bit.
  - If nothing changed: `You are already visible.\r\n`.
- `appear()`, `src/fight.c:108-121`, is the critical piece: if affected by `SPELL_INVISIBLE`, remove that spell affect; then clear `AFF_INVISIBLE` **and** `AFF_HIDE`; then the room message (`$n slowly fades into existence.` for mortals, the "strange presence" line for immortals).

### Go state (the bug)

- Handler: `pkg/game/other_character.go:11`, `doVisible`. The immortal branch correctly calls `performImmortVis` (`pkg/game/invis.go`), which calls `appear()` and therefore strips the spell record. The mortal branch does not:
  ```go
  if ch.IsAffected(affInvisible) {
      ch.SetAffect(affInvisible, false)   // clears the raw bit only
      ch.SendMessage("You fade into view.\r\n")
      altered = true
  }
  ```
- Why clearing the bit is not enough: `Player.IsAffected` (`pkg/game/player_social.go:272-294`) reads **two** sources — the raw `Affects` bitmask **and** `ActiveAffects`, any active spell affect whose engine flag matches. The invis spell applies exactly such a record: `pkg/spells/affect_spells.go:188-199`, `NewAffectDirect(spellNum, ..., engine.AFFInvisible, ...)`. So after mortal `doVisible`, `IsAffected(affInvisible)` still returns true from the surviving spell record, while the player has been told "You fade into view." The message lies.
- The tell that this is an oversight: the sneak branch immediately below in the same function *does* remove the sneak and stealth spell records (`RemoveAffectBySpell`). The invis branch simply lacks its equivalent.
- Registration is fine and is not the problem: `pkg/session/commands.go` registers `"visible"` via `cmdVisible` → `ExecVisible` → `doVisible`.

### Fix

1. In the mortal branch of `doVisible`, mirror C `appear()`: remove the invis spell affect (`RemoveAffectBySpell(spellInvisible)`, spell number 29 — the constant already exists in `pkg/game/invis.go`), clear the `AFF_INVISIBLE` bit, and clear the `AFF_HIDE` bit. Prefer calling the existing `appear()` helper if its room-message behavior matches C for mortals; otherwise replicate its three removals inline and keep the existing player-facing messages untouched.
2. Do not change the immortal branch, the kk_zai gate, or any message string.
3. Do not change `IsAffected`'s dual-source semantics — other systems depend on it. The fix belongs in `doVisible`, not in the visibility check.

### Proof required

- Regression test, the exact reported scenario: mortal under the invis spell casts/uses nothing else, runs `visible`, and afterwards `IsAffected(affInvisible)` is false, the spell record is gone, and a shopkeeper-style visibility check passes.
- Companion tests: sneak + invis combined; already-visible message when nothing is set; kk_zai gate still blocks; immortal path unchanged.
- Revert test: with the fix reverted, the regression test must fail in the reported way (message printed, still invisible). Demonstrate this.

## Investigation 3: flee through locked doors and into death traps (do NOT code first)

### Symptom

Egon: "you can flee past locked doors, so in seilons can flee past the room where you are, or in ender throne room you can flee into torak, and have to summon/recall out." Also: "make sure you can't flee into death traps."

### C behavior (the spec)

`ACMD(do_flee)`, `src/act.offensive.c:360-408`: up to six random direction attempts; a direction is tried only if `CAN_GO(ch, attempt)` (exit exists, destination real, **not closed**) **and** the destination room is **not** `ROOM_DEATH`-flagged. Only then `do_simple_move`.

### Go state

Current main's `DoFlee` (`pkg/game/flee.go`) implements exactly those two guards in its attempt loop: it skips exits with `ExitClosed` set and skips destinations flagged `roomFlagDeath` before moving, for both player and mob bodies.

### Task (investigation, not implementation)

1. Determine which build production is actually running (live banner / version) and whether it predates the `flee.go` guards. The standing hypothesis, from triage, is production lagging main, not a new code bug.
2. If production predates the guards: report that finding and stop. The fix is deployment, which is Zach's call, not a code change in this brief.
3. Only if the bug reproduces on a build that already has the guards, treat it as a new bug and report a reproduction (exact rooms, exact exit states) before writing any fix. Candidate paths to check in that case, not assumptions: asymmetric exits (closed flag on one side only), the mob flee path (`doFleeMoveMob` → `doSimpleMoveMob`), and exits whose closed state is set by scripts rather than the world file.
4. Do not change `flee.go` on speculation. Do not "harden" flee beyond C semantics (no new restrictions C does not have).

## Investigation 4: `house guest Egon` returns "Um.. this house seems to be screwed up" (do NOT edit data)

### Symptom

Zach, standing in the old Trinity clan tower, ran `house guest Egon` and got exactly: `Um.. this house seems to be screwed up`.

### What that error means (verified)

Go `DoHouse` (`pkg/game/house_player.go`) gates in order: (1) current room must carry the `ROOM_HOUSE` flag, else "You must be in your house to set guests."; (2) `findHouse` must find a house control record whose vnum equals the current room's vnum, else "Um.. this house seems to be screwed up."; (3) caller must be the record's owner or an immortal, else "Only the primary owner can set guests."; (4) guest toggle via player lookup.

Zach passed gate 1 and failed gate 2: the room is flagged as a house room, but no control record matches that exact room.

### Hypotheses, in order of likelihood for a tower this old

1. **Wrong room.** `find_house` matches the record's vnum to the *exact* room the player stands in, in both C and Go. A multi-floor tower has several house-flagged rooms; the record points at one of them. The command may work from another floor (ground floor or atrium most likely).
2. **Dropped record.** At boot, both C (`src/house.c:290`, "owner no longer exists -- skip") and Go (`pkg/game/house_boot.go`, owner must resolve via player lookup) silently skip any house record whose registered owner's player file no longer resolves. Trinity's tower dates to an era whose characters may not exist in the current player base. Worse, Go's boot then rewrites the control file with only the surviving records (`saveHouseControl`), so a skipped house is erased from the file, not merely unloaded.
3. **Never migrated.** The old C binary control file may never have been converted to Go's JSON format (`house_control.json`, per-house `<vnum>.house` files). If so, *no* house works anywhere, and this is not Trinity-specific.

### Task (read-only diagnosis)

1. Check `house_control.json` on production (read-only): is there a Trinity tower record at all? If yes, which vnum, and does its owner resolve to an existing player?
2. Check whether `house` works in any other house on the game. If no house works anywhere, hypothesis 3 is the story.
3. Try `house guest` (list form, no argument) from each floor of the tower to test hypothesis 1. Note: the list form only lists guests; adding/removing is the same command with a name, and toggling removes an existing guest — do not run the add form until the record question is settled.
4. Verify whether the immortal `hcontrol` command (C's house build/destroy command) is ported to Go. Triage could not find it by code search; confirm or refute.
5. **Do not** edit `house_control.json`, add or remove guests, transfer ownership, or restart production. Report findings to Zach; recreating a dropped house is a separate decision for him, and it may currently require data editing plus a restart rather than an in-game command.

## Explicit non-issue: there is no `vis` command, and that is correct

- C has no `vis` command: only `visible` (`interpreter.c:802`), and C's `find_command` (`interpreter.c:1395-1404`) is exact-match (`strcmp`). Typing `vis` in the original game also fails.
- Go likewise registers only `visible`. If Egon typed `vis` and got the unrecognized-command response, that is faithful behavior, not a bug.
- Do not add a `vis` alias or abbreviation support unless Zach explicitly asks for it. Fidelity means matching C here, including its exact-match lookup.

## Global constraints for the implementer

- Work in a fresh branch off main. Do not touch the C oracle tree, the world/mob/obj/zone files, or anything on production. No deploys, no migrations, no data edits.
- Copy C message strings byte-for-byte. Do not invent, reword, or "improve" player-facing text.
- Do not change semantics beyond the two fixes: no new restrictions, no new commands, no new aliases, no behavior changes to adjacent systems (sneak, hide, flee, shops, hunting).
- Prove each fix by reverting it: the new regression test must fail without the fix and pass with it. Show this, per the standing implement lane.
- Gates before handoff: full `go test ./...` green, repo lint gate green, and (where the oracle diff harness applies) no new normalized divergence.
- Report back per item: fixed with test evidence, or investigation findings with the exact evidence (build version, file contents, command output). Do not mark Investigation 3 or 4 "fixed" on the basis of a hypothesis.
- Zach merges. Egon is the original reporter; in-game verification with him is Zach's call, not the implementer's.

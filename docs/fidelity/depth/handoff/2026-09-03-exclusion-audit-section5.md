# Section 5.2 exclusion spot-check — audit record

Date: 2026-09-03
Author: Claude (post-hoc review stream, per the `~/dp-audit-PICKUP.md` charter)
Scope: verify the `excluded` rows of `docs/fidelity/depth/*.tsv` against the
read-only C authority (`src/`), per audit Section 5.2. Any exclusion that does
not hold converts to `proven` or `blocked` with notes.

Method: pure `src/` reading + `spec_assign.c` cross-checks. No oracle runs, so
zero contention with the concurrent `glm/depth-whirlpool` full-corpus regression.

## Result summary

56 excluded rows at audit time. **~48 verified holding, 3 converted
`excluded -> blocked`, ~4 soft/tentative findings.**

Conversion committed on `main` as `07c3a687e`
("docs(fidelity): reclassify vampire/werewolf eat/drink rows excluded->blocked").
Net: blocked 48->51, excluded 56->53 (no cases created/destroyed; reclassified).

## Verified holding (~42)

- **18 unassigned spec procs** — every proc is absent from the *comment-filtered*
  `ASSIGN(MOB|ROOM|OBJ)` set in `spec_assign.c`: mayor, enter_circle, elevator,
  elemental_room, fearface, couch, tipster, eviltrade, evillead, little_boy, ira,
  hisc, portal_room, no_move_down, alien_elevator, turn_undead, itoh, mirror.
  NOTE: a naive `grep ASSIGN` is comment-blind — `hisc`'s `ASSIGNMOB(14412, hisc)`
  at `spec_assign.c:382` is commented out (`/* ... */`), so hisc is genuinely
  unassigned. Re-extract the wired set excluding comment lines.
- **2 no-spec-flag mobs** — rescuer (15808) and pissedalchemist (15814) ARE
  `ASSIGNMOB`'d (`spec_assign.c:421-422`), but their `158.mob` action bitvectors
  are `153608` / `26650` — both even, so bit 0 (`MOB_SPEC`, `structs.h:247`) is
  clear and the dispatcher never calls them.
- **2 fighting-branch spec rows** — cityguard-fighting-branch, take-to-jail-
  fighting-branch — rest on the verified `mobact.c:71` gate
  (`if (!IS_MOB(ch) || FIGHTING(ch) || !AWAKE(ch)) continue;` *before* the
  `MOB_SPEC` dispatch), same gate ZCode verified for `snake`.
- **10 NPC/descriptor early-returns** — abilities, help (`!ch->desc` at 1579),
  groinrip / headbutt (`if (!ch->desc && !subcmd) return`), save, qui, quit
  (`if (IS_NPC(ch) || !ch->desc) return`), cast (`IS_NPC` at `spell_parser.c:923`
  — cited line drifted), page (`if (IS_NPC(ch)) "Monsters can't page.."`,
  `act.comm.c:1114`), flee (`IS_NPC(ch) && GET_MOB_WAIT(ch)`).
- **3 DG mob-script subsystem** — give.mob-ongive (`MS_ONGIVE`), movement.mob-
  entry-prog (`MS_GREET`), mount.script-entry — consistent with the project-wide
  "DG scripts not ported" exclusion (see the whirlpool handoff).
- **2 UB rows** — dns, auto: both build a string with
  `sprintf(buf, "%s...", buf, ...)` (source == destination = C undefined
  behavior). No faithful target exists.
- **2 socials** — sniff, snore: defined with only the two no-arg message lines
  then `#` in `lib/misc/socials` (no target-message slots), so the target
  branches cannot render.
- **+ mount.cannot-mount** — subtle but holds: `CAN_MOUNT(ch)` is
  `(!IS_MOB(ch) && !IS_MOUNTED(ch))`, and the earlier `else if(IS_MOUNTED(ch))`
  already caught the mounted case, so `!CAN_MOUNT(ch)` reduces to `IS_MOB(ch)` —
  players never satisfy it. **get.palm-quiet-path** — the palm sub-branch, proven
  via its own `spec-proc-no-get-palm` vehicle (one of ZCode's 9).

## Finding — 3 rows converted `excluded -> blocked` (committed `07c3a687e`)

`drink.vampire`, `eat.vampire`, `eat.werewolf-corpse` were filed `excluded`
(implying unreachable) but gate on **reachable player state**:

- `PLR_VAMPIRE` is set by the **Dracula spec proc** (`spec_procs.c:1829`) and a
  spell (`spells.c:784`). The `drink`/`eat` vampirism branches fire for a vampire
  PC (`act.item.c:983`, `:1121`).
- `AFF_WEREWOLF` is set by `act.other.c:1441`. The `eat.werewolf-corpse` savage-
  eat branch (`act.item.c:1050`) fires for a werewolf PC — and its own note
  already said "Go TODO" (i.e. unported → a live vehicle is RED-on-main).

These are real, reachable behaviors that merely need the transformation staged
first, which is `blocked` (reachable, oracle-staging-hard), not `excluded`.
Filing them `excluded` dropped them from the honest-gap count — the
"manifest completion != port completion" trap. Prove via a Dracula-bite /
vampire-spell (+ night for drink) and a werewolf-transform + corpse vehicle.

## Additional verified holding (the tail)

- **spike.breed-killer-call** — `breed_killer` returns on `if (cmd || !AWAKE(ch))`
  (`spec_procs2.c:1685`), so a command-triggered call (spike) no-ops.
- **sacrifice.unreachable** — no `sacrifice` command in `cmd_info`; only `donate`
  (SCMD_DONATE) and `junk` (SCMD_JUNK) exist. Genuinely unregistered.
- **luaedit.superseded-by-web-admin** — `{ "luaedit", ..., do_luaedit, LVL_BUILDER,
  LVL_HIGOD }` — immortal/builder tool, superseded; out of player-fidelity scope.
- **pray-item-reward-world** — reward branch gated on sentinel names
  ("no entry here" / "neither here" / "this is not here") no real player can hold.
- **use.skill-extension** — a deliberate Go-only extension (`CmdUseSkill`) with no C
  `do_use` analog; documented intentional divergence, nothing in C to match.
- **fly-exit-up-npc-gate** — the `IS_NPC(ch)` sub-gate of the room spec
  (`spec_procs3.c:1293`); players never take the NPC path.

## Soft / tentative findings (left as-is, flagged for a ruling or confirmation)

- **shoot.handler-fighting-unreachable** — reachable in principle (a fighting PC
  types `shoot` -> `if (FIGHTING(ch)) "already engaged in close-range combat!"`,
  `act.offensive.c:770`) but un-stageable via the mid-combat-injection blind spot
  (DP-1202). Defensible as `excluded` *by convention* (same basis as the
  `*-fighting-branch` rows); needs a consistent excluded-vs-blocked ruling for
  "requires mid-combat command injection."
- **use.tattoo** — note says "its own surface," but no tattoo surface appears in
  the manifests, `use_tattoo` (`tattoo.c`) returns FALSE early, and the tattoo
  needle (obj 27) world-placement is unconfirmed. Confirm it is genuinely
  deferred-and-tracked vs. a quiet gap (softer than the vampire case).
- **mindlink.success-psionic-mob-unreachable** — the success branch needs a
  victim with >=100 mana (a psionic mob); confirm no such reachable mob target
  exists (`new_cmds2.c:280+`).
- **obj.field-object-special-unreachable** — `field_object` IS wired
  (`spec_procs3.c:456`); the specific unreachability gate/reason was not
  deep-verified this pass.

## Meta-note

Most excluded rows DO carry a prose reason in the `notes` column ($9) — an
earlier claim of "bare-citation reasons" was an artifact of reading $8 (`c_site`)
instead of $9. That said, several notes are thin (a phrase, not a
reachability argument); a one-line *why* per excluded row would let a reader
audit the manifest without re-deriving from `src/`.

## Not deep-verified this pass (1 row)

room.jail-commandless-body (`spec_procs2.c:1470-1493`) — the jail room's
commandless-body path; left unchecked. Likely legit (parallels the jail
mechanics), confirm on a follow-up.

## Next

1. Rule on the soft/tentative findings above (esp. `use.tattoo` and the `shoot`
   mid-combat excluded-vs-blocked convention).
2. Verify the 1 remaining row (jail-commandless-body).
3. When the box is free, an isolated re-run confirms `force-mob` is a
   pre-existing red (known-blocked `force.visible-npc-target` /
   `force.npc-command-interpreter`), not a whirlpool regression — its diff is
   pure NPC target resolution ("No-one by that name here"), unrelated to the
   whirlpool branch's changes.

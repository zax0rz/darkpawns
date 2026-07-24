# BRIEF (glm) — DP-1206: faithful skill-known entry gate (kill the invented level/class gate)

**Owner:** glm-5.2. **Gate:** byte-exact unit tests now (one per Wave-1 command's
unknown-skill message); Claude oracle-gates the openers after. CI green.
**Git:** branch off `main` as `glm/skill-known-gate`. Edit → commit → push → open a
PR. Do NOT merge. Sized to one PR (M).
**Finding:** DP-1206 (High, normal-play-reachable). **Cite:**
`src/act.offensive.c` (do_bash:427, do_kick:592, do_trip:742, do_headbutt:383,
do_disarm:199), `src/new_cmds.c:744` (do_backstab), do_rescue; Go
`pkg/game/skills.go:184` (`CanUseSkill`), `pkg/command/skill_commands.go` (the
callers); rules **R1** (player-facing bytes are law), **R4** (no invention).

---

## The bug — Go invented a level/class gate that does not exist in C

Go's `CanUseSkill` (`pkg/game/skills.go:184-212`) gates skill USE on:
1. class membership → `"You have no idea how."` (invented), and
2. learn-level → `fmt.Sprintf("You must be at least level %d to use that skill.", minLevel)` (invented).

**C does neither.** Every combat-skill command gates purely on whether the actor
*knows the skill* — `if (!GET_SKILL(ch, SKILL_X))` — and emits **that command's own
hardcoded message**, then returns. There is no class check and no generic
level-message anywhere in these commands. Concretely, `do_kick`
(`act.offensive.c:592`):

```c
ACMD(do_kick)
{
  ...
  if (!GET_SKILL(ch, SKILL_KICK))
    {
      send_to_char("You'd better leave all the martial arts to fighters.\r\n", ch);
      return;
    }
  ...
}
```

### Divergences this produces (all normal-play-reachable)
- **Learned-but-low-level:** an L1 warrior granted `bash` (skill 75, e.g. via
  `skillset`) — C proceeds and bashes; Go blocks with `"You must be at least
  level 3 to use that skill."`. *(This is exactly what `combat-bash-opener` hit —
  it's the gate blocking the whole L3+ opener sweep.)*
- **Right-level-but-unpracticed:** a fresh L10 warrior who never practiced bash
  (skill 0) types `bash` — C stops them with the martial-arts message; Go lets
  them whiff (wrong bytes + wrong flow).
- **Wrong-class message:** a mage types `kick` — C: `"You'd better leave all the
  martial arts to fighters.\r\n"`; Go: `"You have no idea how."`.
- **Cross-class grant:** a non-warrior granted `kick` via `skillset` (skill>0) —
  C lets them kick; Go blocks with `"You have no idea how."`.

## The C truth — exact per-command unknown-skill message + ordering

Each is the FIRST-ish gate in its `do_` function. Reproduce **byte-exact**,
including the `\r\n` terminator, and at the **same position** in the handler's
sequence as C (ordering vs the peaceful-room check varies — see the notes):

| skill (Go const) | exact C bytes (incl. terminator) | C cite | ordering note |
|---|---|---|---|
| `SkillKick` | `You'd better leave all the martial arts to fighters.\r\n` | act.offensive.c:594 | first check in do_kick; unconditional `return` |
| `SkillBash` | `You'd better leave all the martial arts to fighters.\r\n` | act.offensive.c:429 | first check; `if(!subcmd) return;` (see §subcmd) |
| `SkillDisarm` | `You'd better leave all the martial arts to fighters.\r\n` | act.offensive.c:199 | first check; `if(!subcmd) return;` |
| `SkillTrip` | `You'd better leave the sneaky stuff to the thieves.\r\n` | act.offensive.c:742 | first check; `if(!subcmd) return;` |
| `SkillBackstab` | `You'd better leave the sneaky stuff to the thieves.\r\n` | new_cmds.c:744 | (verify exact position in do_backstab) |
| `SkillHeadbutt` | `You aren't qualified to headbutt anyone!\r\n` | act.offensive.c:383 | **AFTER** the peaceful-room check `"The Gods prevent thy violent act.\r\n"` |
| `SkillRescue` | `But only true warriors can do this!\r\n` | do_rescue | first check |

**Read each C `do_` function yourself and confirm the byte string + terminator +
its position** before copying — do not trust this table blindly; it is the anchor,
the C source is law (R5e: the call path must be verified).

### §subcmd — the `if (!subcmd) return;` pattern
`bash`/`trip`/`disarm` send the message but only `return` when `!subcmd` (subcmd
is nonzero only for internal combo invocations). The **player-typed command path
is `subcmd == 0`**, so it always returns after the message. Go's skill commands
are the player path (subcmd 0) → send message, then `return`. If Go has no
internal subcmd-driven caller for these, that's simply "always return." Do not
invent a subcmd path.

## The fix — staged, fork `CanUseSkill` on an audited-skill table

Do **not** rip out `CanUseSkill` wholesale (it still backs ~20 un-audited skills —
circle, charge, shoot, subdue, …). Instead flip only the Wave-1 combat skills to
the faithful gate, leaving everything else byte-identical to today:

1. Add a table of the audited exact messages:
   ```go
   // SkillUnknownMsg is the exact bytes a command sends when the actor does not
   // know the skill (C: `if (!GET_SKILL(ch, SKILL_X))`). Byte-exact incl.
   // terminator; ported per-command from the C do_ function. DP-1206.
   var SkillUnknownMsg = map[string]string{
       SkillKick:     "You'd better leave all the martial arts to fighters.\r\n",
       SkillBash:     "You'd better leave all the martial arts to fighters.\r\n",
       SkillDisarm:   "You'd better leave all the martial arts to fighters.\r\n",
       SkillTrip:     "You'd better leave the sneaky stuff to the thieves.\r\n",
       SkillBackstab: "You'd better leave the sneaky stuff to the thieves.\r\n",
       SkillHeadbutt: "You aren't qualified to headbutt anyone!\r\n",
       SkillRescue:   "But only true warriors can do this!\r\n",
   }
   ```
2. In `CanUseSkill`, fork on membership in that table:
   ```go
   func CanUseSkill(p *Player, skillName string) (bool, string) {
       if msg, audited := SkillUnknownMsg[skillName]; audited {
           // FAITHFUL path (DP-1206): C gates on !GET_SKILL, no class/level.
           if p.GetSkill(skillName) == 0 {
               return false, msg
           }
           // position handling unchanged (see caveat below)
           if bad, pmsg := skillPositionGate(p, skillName); bad {
               return false, pmsg
           }
           return true, ""
       }
       // LEGACY path — un-audited skills keep today's class/level/position
       // behavior verbatim until their Wave-2 audit. (existing code)
       ...
   }
   ```
   (Factor the current position block into `skillPositionGate` or inline it — your
   call; keep its behavior identical.)

### Caveats — do NOT touch these here (they are separate findings)
- **Position messages/ordering:** keep the existing position block behavior
  exactly as today for the audited skills. C enforces min-position in the command
  interpreter with its own messages; auditing that is a *separate* finding — do not
  change position bytes in this PR.
- **Peaceful-room ordering:** for `headbutt` the skill-known gate must sit **after**
  the peaceful-room check (C order). Verify where each Go handler does its
  peaceful check and place the `CanUseSkill`/skill-known call to match C's order
  for that command. For kick/bash/trip/disarm the skill gate is effectively first
  (before peaceful), which is where the top-of-handler `CanUseSkill` call already
  sits — confirm per command.
- **The hit/miss message body** (kick emitting its own strings instead of routing
  through `skill_message`) is **DP-1207**, a *separate* brief. This PR only fixes
  the ENTRY gate. After this lands, the openers will pass the gate and then surface
  the DP-1207 message divergence — that's expected and correct.
- **Class/level data:** leave `SkillClassReq` in place (still used by the legacy
  path + practice/learn logic). You are not deleting it, only bypassing it for the
  audited skills' USE gate.

## Tests (byte-exact, one per audited command)
For each Wave-1 skill, assert the EXACT unknown-skill bytes (incl. `\r\n`):
- **skill == 0 → blocked with the exact per-skill message** (e.g. a warrior with
  `GetSkill("bash")==0` → `"You'd better leave all the martial arts to fighters.\r\n"`;
  a character with `GetSkill("trip")==0` → the sneaky-stuff line; headbutt →
  qualified line; rescue → true-warriors line).
- **skill > 0 → gate passes** regardless of class/level (e.g. `SetSkill("bash",75)`
  on an L1 warrior → `CanUseSkill` returns `true` — the old level-3 block is gone).
- **cross-class grant passes:** a non-warrior with `SetSkill("kick",50)` →
  `CanUseSkill` true (C has no class gate).
- **legacy path unchanged:** a still-un-audited skill (e.g. `SkillCharge`) returns
  today's class/level messages exactly — regression guard that the fork didn't
  perturb the un-audited set.

## Oracle gate (Claude, after merge — informational)
I re-run the opener suite: `combat-backstab-opener` **stays green** (granted
backstab, skill>0, unchanged), `combat-bash-opener` now **passes the gate** and
surfaces the next (DP-1207-class) divergence, and I add an **unlearned-skill**
opener (skill 0 → exact message, red→green on this PR). Until then the unit tests
are the gate.

## Guardrails
- **Never** edit `src/` or `darkpawns-c-oracle/` — read-only reference.
- `go build ./...`, `go vet ./...`, `go test ./... -race`.
- **run `golangci-lint`** and `gofumpt -w` on every file you touch.
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.

## Deliverable
A faithful skill-known entry gate for the seven Wave-1 combat skills
(backstab/bash/kick/trip/disarm/headbutt/rescue): `SkillUnknownMsg` table with
byte-exact per-command messages, `CanUseSkill` forked so audited skills gate on
`GetSkill()==0` (no class/level) while un-audited skills keep today's behavior,
correct peaceful-vs-skill ordering per command, position bytes untouched, and the
unit tests above. Claude oracle-gates the openers.

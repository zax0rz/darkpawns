---
tags: [active, reference, port, fidelity]
last_updated: 2026-06-18
author: Claude Code (Opus) with The Architect
---
# Port Reachability Map

The concrete punch list for "completing the game port." This answers one
question per C command: **can a player actually invoke it in the Go port today?**

Scope note: this is about *gameplay* fidelity. Client/agent/agentcli/protocol
work is explicitly deferred (see `remaining-implementation-brief.md` for that).

## Methodology

- **C side:** `src/interpreter.c` command table — 503 entries = 183 socials
  (`do_action`, data-driven) + 320 non-social commands (~203 distinct handlers).
- **Go side:** the real dispatch is a single `cmdRegistry` (`pkg/command/registry.go`),
  entered via `ExecuteCommand` (`pkg/session/commands.go:330`). A command is
  *reachable* if its name or an alias is registered, OR it is a social
  (`game.Socials` fallback), OR it is intercepted by a mob/room/object **spec
  procedure** before registry lookup (banking, shopkeepers, etc.).
- **Important:** the registry does **exact-match only — no CircleMUD-style
  abbreviation/prefix matching.** In C, `mur` matches `murder`, `inv` matches
  `inventory`. In Go only explicitly-registered short aliases (`n`, `k`, …) and
  player-defined aliases work. This is a fidelity gap in its own right.

## Headline numbers

| | count |
|---|---|
| C non-social commands | ~290 (after removing abbreviation stubs `qui`/`shutdow`/`whod`) |
| Reachable in Go (registry names+aliases) | 208 |
| Socials reachable (data-driven) | 183 |
| C non-social commands **not** directly reachable | 146 |
| └─ of those, an implementation exists in `pkg/` | 64 |
| └─ of those, no implementation found | 82 |

The 146 overcounts real gaps: many "coded" entries are reachable through spec
procs, and many "missing" entries are immortal/OLC tooling. The buckets below
separate signal from noise.

---

## Bucket A — Coded-but-unwired skills (QUICK WINS) ⭐ — DONE (2026-06-18)

**Status:** 22 skills wired + `dragon`/`tiger`/`flesh` aliases added in
`pkg/session/commands.go`, guarded by `TestBucketASkillsRegistered`. Positions/
levels mirror `src/interpreter.c`. Excluded from this batch:
- `dig` (LVL_BUILDER) and `mold` (LVL_IMMORT) — builder/immortal tools, covered
  by the web admin (see Bucket D); wire only if an in-game builder path is wanted.
- `pick` — already reachable via `cmdPick`/`door_cmds.go`; `CmdPickLock` is a
  dead alternate.
- `detect` — deferred; verify it doesn't shadow spell-based detection first.

Original list below for reference.



**~25 combat/class skill handlers are fully implemented in `pkg/command` but
never registered.** They are dead to players. Wiring each is a one-line
`cmdRegistry.Register(...)` like the working `bash` line:

```go
cmdRegistry.Register("bash", wrapSkill(command.CmdBash), "Bash a target…", 0, combat.PosFighting)
```

This is the single highest-value, lowest-risk port-completion work. Verify per
skill: (1) handler signature fits `wrapSkill`, (2) correct C command name (some
differ — see notes), (3) MinPosition/level gate.

| C command | Handler (`pkg/command`) | Notes |
|---|---|---|
| `bearhug` | `CmdBearhug` | |
| `behead` | `CmdBehead` | |
| `bite` | `CmdBite` | |
| `carve` | `CmdCarve` | |
| `compare` | `CmdCompare` | object compare |
| `cutthroat` | `CmdCutthroat` | |
| `dig` | `CmdDig` | |
| `disarm` | `CmdDisarm` | **core combat skill — currently unreachable** |
| `groinrip` | `CmdGroinrip` | |
| `mindlink` | `CmdMindlink` | |
| `mold` | `CmdMold` | |
| `palm` | `CmdPalm` | |
| `point` | `CmdPoint` | |
| `scrounge` | `CmdScrounge` | |
| `sharpen` | `CmdSharpen` | |
| `slug` | `CmdSlug` | |
| `smackheads` | `CmdSmackheads` | |
| `strike` | `CmdStrike` | |
| `tag` | `CmdTag` | |
| `turn` | `CmdTurn` | turn undead |
| `aid` | `CmdFirstAid` | name differs (handler = FirstAid) |
| `alter` / `flesh` | `CmdFleshAlter` | name differs |
| `serpent` | `CmdSerpentKick` | C name `serpent`, handler = SerpentKick |
| `scan` | `CmdScan` | not in C interpreter table but a useful info skill |
| `detect` | `CmdDetect` | |
| `pick` | `CmdPickLock` | verify C name (`pick`) |

Non-command handlers (not top-level, skip): `CmdConfirmForget` (confirmation
step), `CmdUseSkill` (generic dispatcher).

**Naming-fidelity gap:** `dragon`/`tiger` ARE registered, but under
`dragonkick`/`tigerpunch` — not their C names. Add aliases so `dragon` and
`tiger` work as in the original.

---

## Bucket B — Genuinely missing playable commands

### Foundational blocker — timed affect → stat pipeline (FIXED 2026-06-18, PR #27)

Before implementing buff skills, note: most Bucket B class skills (berserk,
kuji-kiri seals rin/kyo/toh/kai/zai) buff **hitroll/damroll/AC** via timed
affects. The live path `cast → CallMagic → MagAffects → AddAffect` stores the
affect in `ActiveAffects` with the correct Location/Magnitude, **but base stats
are never modified and the stat getters ignored `ActiveAffects`** — so every such
buff (and the buff *spells* bless/armor/metalskin/psyshield) was silently inert.

**Fixed (PR #22):** `GetHitroll`/`GetDamroll`/`GetAC` now fold in `ActiveAffects`
via `Player.sumAffectModsLocked(location)` (read-side summation, matching how
equipment affects already work; expiry via `RemoveAffectBySpell` auto-reverts).
Guarded by `TestActiveAffectsModifyStats` / `TestActiveAffectsStackAndExpire`.

**Fixed (PR #27):** summation extended to `GetStr/GetDex/GetInt/GetWis/GetCon/
GetCha`, `GetMaxHP/GetMaxMana/GetMaxMove`, and `GetSavingThrow` — full buff-spell
fidelity for strength, bless's saving-throw component, etc. Guarded by
`TestActiveAffectsModifyCoreStats`.

**Resolved (2026-06-18):** the "dual affect-path routing" note below was wrong
— re-verified directly against the code rather than trusting this doc.
`spells.Cast`'s `am *engine.AffectManager` param was already dead (its own
comment said so) and `ApplySpellAffects` (the function that would have used
it) had zero callers anywhere. Curse/poison/sleep/blindness/sanctuary always
routed through the live path (`MagAffects → applyAffect → AddAffect →
ActiveAffects`), same as every other spell — there was no second live system,
just dead code sitting next to it. Deleted `engine.AffectManager`,
`AffectTickSystem`, `TickManager`, and `spells.ApplySpellAffects`; dropped the
unused `am` param from `spells.Cast`.

**Separate, still-unaddressed dead system (out of scope for that cleanup):**
`engine.AffectFromChar`/`MasterAffect`/`StatModifiable` in
`pkg/engine/affect_helpers.go` — a third, more literal `handler.c` port. The
one live call site (`pkg/game/affect_update.go`'s expiry path) is a no-op
because nothing live ever populates `Player.MasterAffects`. Riskier to remove
(touches the `Player` struct and possibly save-file serialization) — flagged
for a future PR, not done here.

### Missing skills/commands

- **Martial-arts forms — DONE (PR #27):** `jin` `kai` `kyo` `retsu` `rin` `sha`
  `zai` `zhen` (kuji-kiri seals) wired via `game.DoKujiKiri` + `CmdKujiKiri`.
  `shadow` `spike` `stake` `kabuki` still missing — see Part 2 below.
- **Combat/movement skills:** `berserk` — DONE (PR #27), `game.DoBerserk` +
  `CmdBerserk`. `mount` — DONE (Part 1 below, trivial alias of `ride`).
  `charge` `track` still missing — track has pathfinding scaffolding in
  `pkg/game/graph.go`. See Part 2 below.

### Bucket B Part 1 — shared-handler aliases, simple flavor/info commands — DONE (2026-06-18)

Cross-referenced the "missing" list above against `src/interpreter.c`'s real
top-level command names. Most of these turned out to be either (a) aliases of
an already-correct, already-wired handler registered only under the wrong
name, (b) game logic that was fully implemented but never called from any
command path, or (c) simple static-text/flavor commands with no missing game
logic. None of this needed new game logic:

- **Shared-handler aliases** (same Go handler, real C name was missing):
  `abilities` (was wired as `abils` only — `cmdAbils` in `cmd_info.go`),
  `glance` (alias of `diagnose` — `do_diagnose`), `mount` (alias of `ride` —
  `do_ride`), `reallyquit` (alias of `quit`; this port's `cmdQuit` doesn't
  implement C's temple/equipment-loss gating, so the two names just behave
  identically here rather than C's distinct behavior).
- **Real behavioral variants of an existing handler** (parameterized via a
  bool flag): `sip` (drink without depleting the container or applying
  condition/poison effects — `do_drink` SCMD_SIP) and `taste` (eat without
  FULL gain, decrements the food's bite counter instead of consuming it
  outright — `do_eat` SCMD_TASTE).
- **Implemented but never called:** `game.DoInsult`/`game.DoDream` in
  `pkg/game/act_social.go` had zero callers — wired via new
  `World.ExecInsult`/`ExecDream` bridges plus `cmdInsult`/`cmdDream` in
  `pkg/session/act_social.go`.
- **New simple flavor command:** `think` (`cmdThink` in `comm_cmds.go`,
  ported from `do_think` incl. `PLR_NOSHOUT`/zero-INT gating and
  `PRF_NOREPEAT` echo suppression).
- **New static info-text commands** (`pkg/session/gen_ps_cmds.go`, ported
  from `do_gen_ps`): `credits` `news` `policy` `handbook` (LVL_IMMORT)
  `future` `whoami`. `version` shows the Go runtime version to immortals in
  place of C's SVN revision/compile timestamp — this port has no build-time
  tracking infrastructure, so that's an honest substitute, not a faithful
  port of that one detail.
- **Registry/level-gate fix, not new logic:** `reroll` (LVL_GRGOD) and
  `unaffect` (LVL_GOD) are real top-level C command names for two of
  `wizutil`'s sub-actions, gated stricter than the blanket LVL_IMMORT the
  `wizutil` meta-command applies. Extracted `wizutilDispatch` in
  `wiz_system.go` so the meta-command and the two standalone names share one
  implementation.

Guarded by `pkg/session/bucket_b_test.go` (registry wiring, sip/taste vs.
drink/eat behavior, think/insult/dream message+gating, reroll/unaffect
level-gate enforcement, gen_ps static-text output incl. the file-not-found
fallback).

### Bucket B Part 2 — deferred (needs new game logic or schema changes)

- `shadow` (quiet `follow`) and `kabuki` (quiet `hide`) are **not** trivial
  quiet-flag variants — both gate a real mechanic (`SKILL_SHADOW`-conferred
  dodge bonus for shadow; an intro flavor line for kabuki) behind a skill
  constant that doesn't exist yet in `pkg/game`/`pkg/command`.
- `spike` `stake` `charge` `track` `grab` `leave` `search` `enter` need new
  standalone game logic (`enter` has a `spec_procs_missing.go` stub; `track`
  has pathfinding scaffolding in `pkg/game/graph.go` but no `do_track` port).
- `fill` needs nontrivial two-argument fountain-fill parsing
  (`do_pour`/SCMD_FILL in `src/act.item.c`).
- `socials` (list known socials) needs a registry schema change — there's no
  `IsSocial`-style flag on `command.Registry.Entry` to filter by.
- `ident` — explicitly skipped, same low-value judgment call as Bucket E's
  joke/no-op toggles.
- `orgasm` (`do_otouch` in `src/new_cmds.c`) — explicitly skipped: an
  immortal-only NSFW joke command, not worth porting.
- `junk` was already done via Bucket C's `donate`/`junk` work (PR #30/#31) —
  removed from this list, it was a stale leftover.

⚠ Part 2's list is from a name/handler probe; confirm each against `src/` and
a `grep` for alternate names before deciding rewrite vs. wire-up.

---

## Bucket C — Spec-proc handled — VERIFIED (2026-06-18)

These work when the player is at the right mob/room (banker, postmaster, board).
Wiring confirmed: specs registered (`spec_assign.go`) and assigned — banker mob
8034, postmaster mobs 3010/21225, board objects 8064-8099 — and `ExecuteCommand`
intercepts mob/obj/room specs before registry lookup.

- **Banking (`balance`/`deposit`/`withdraw`) — FIXED.** Verification found the
  port ignored the bank account entirely: `balance` showed carried gold,
  `deposit` destroyed coins, and `withdraw` minted coins from nothing with no
  balance check (an **economy exploit**). Re-ported `specBank` from
  `src/spec_procs.c` to move coins between Gold and BankGold; added
  `Player.GetBankGold`/`SetBankGold` and `TestSpecBank`.
- **Postmaster (`mail`/`check`/`receive`) — verified functional.** Dispatch
  correct; `mail.go` implements level gate, stamp cost, recipient lookup,
  writing flags, check/receive. Minor polish: `$n` is not substituted with the
  mailman's name in tells (a broader act()-substitution gap, not a break).
- **Boards (`write`/`look`/`read`/`remove` via `gen_board`) — verified
  functional.** `genBoard` dispatches to `BoardSystem`. Follow-up to confirm by
  play: that a plain `look` near a board isn't over-intercepted.
- **Recharge (`recharge`) — FIXED.** Verification found `specRecharger`
  collapsed a wand/staff's max and current charges to the same value on
  recharge, instead of incrementing current and decrementing max
  independently as `src/new_cmds2.c` `SPECIAL(recharger)` does. The bug
  discarded the player's existing charges rather than adding one, and an
  extra (non-canonical) "worn out" guard at `maxCharges<=1` blocked legitimate
  C recharges. Fixed to match C exactly, including the quirky case where
  current charges can end up exceeding max (the recharger's own flavor text
  describes this). See `TestSpecRecharger`.
- **Stable/collect (`stable`/`collect`/`buy`/`list` via `specStableboy`) —
  verified functional.** Matches `src/spec_procs2.c` `SPECIAL(stableboy)`:
  300 gold to buy a horse, day-rate stabling, cost-on-collect. See
  `TestSpecStableboy`. Known minor gap: doesn't enforce C's follower-cap check
  (`num_followers(ch) >= GET_CHA(ch)/2`) before allowing a horse purchase —
  low-impact, not fixed yet.
- **Retrieve (`retrieve` via `specMortician`) — verified functional.**
  Matches `src/spec_procs3.c` `SPECIAL(mortician)`: corpse search by name,
  level-scaled cost, gold deduction.
- **`will`, `offer`, `rent` — dead in the original C game too, not a port
  gap.** `will`/`offer` only ever fire flavor-text tells (`recruiter` spec);
  `rent`'s real implementation (`gen_receptionist`,
  `src/objsave.c`) is never assigned to any mob in `src/spec_assign.c` — both
  are unreachable in the original codebase. No action item.
- **`donate`/`junk` — DONE (PR #30, bugfixed in PR #31).** Implemented per
  `src/act.item.c`'s `do_drop` `SCMD_DONATE`/`SCMD_JUNK` path (25% destroy
  chance, otherwise teleport to `donation_room_1=8053`/`donation_room_2=18204`
  regardless of current room — the dump-spec/donation-room VNum mismatch is
  preserved as a pre-existing C inconsistency, not "fixed"). PR #31 found and
  fixed 4 bugs uncovered after merge: NODROP/two-handed checks reading the
  wrong `ExtraFlags` bit, give/wear/put `all`-loops using a bogus flag check
  instead of `CAN_SEE_OBJ`, and `GetShortDesc`/`GetLongDesc` not consulting
  `Runtime.ShortDesc`/`Runtime.LongDesc` (broke corpse/money-pile display).

---

## Bucket D — Immortal / OLC tooling — RESOLVED by the web admin

**Do not port the OLC commands.** The building toolchain was rebuilt as the web
admin and is already mounted in the server:
- `pkg/admin` (router + handlers): full CRUD on zones/rooms/mobs/objects/shops
  (GET + PUT), `/admin/save-world` persistence, role-gated `builder`/`admin`,
  audit-logged. `admin-ui/` is the built React front-end, served from `main.go`
  (`http.Handle("/admin/", adminRouter)`).
- So `olc` `medit` `oedit` `redit` `sedit` `tedit` `zedit` `luaedit` `string`
  are **superseded** — drop them from the port.

**Open follow-up (verify, not port):** confirm `darkpawns.labz0rz.com/admin`
actually serves `admin-ui-dist` and the role auth works end-to-end. A `/verify`
or ops task.

**Wiz actions (separate from OLC):** `freeze` `thaw` `transfer` `poofin`
`poofout` `qecho` `wizlist` etc. Most moderation (kick/ban/penalties) already
exists in-game and in the admin players panel. Remaining wiz *actions* are
deferred; decide case-by-case whether each is worth an in-game command vs. admin
panel. (`admobs` `advance` `skillset` `slowns` `dns` `roomflags` `holler`
`holylight` `nohassle` `hire` `toh` `wnewbie` `nonewbie` `nosummon` `nograts`
`pardon` `wizhelp` `immlist` `imotd` `motd` `news` `nobroadcast`.)

---

## Bucket E — Preference toggles — DONE (2026-06-18)

Verified against `src/interpreter.c`: each toggle (`brief` `compact` `notell`
`noauction` `noshout` `nogossip` `nograts` `nowiz` `quest` `roomflags`
`norepeat` `holylight` `nohassle` `nonewbie` `noctell` `nobroadcast`
`nosummon`) is its own top-level command routed through `do_gen_tog` — there
is no unified `toggle <name>` dispatcher in the original.

This was a bigger gap than "likely a small unification": the correct,
already-ported `game.doGenTog` (`other_settings.go`) was reachable only
through the literal, never-typed command name `gentog`. None of the 16 real
toggle names above were registered at all. Separately, the two commands that
*were* registered and live — `color` and `autoexit` — never persisted
anything; `toggle <name>` only special-cased `autoexit`, rejecting every
other name.

**Fixed:**
- Registered all 16 toggle names as standalone commands → `doGenTog`, via a
  new `wrapToggle(key)` (`pkg/session/commands.go`); removed the dead
  `gentog`/`gentoggle` registration.
- Fixed `doGenTog`'s `cmdMap` to key by the *real* C command name — `nosummon`
  (was `summon`), `noshout` (was `deaf`), `nograts` (was `nogratz`),
  `nobroadcast` (was `nobroad`) — and dropped two invented entries
  (`autocxits`, `npcident`) that didn't correspond to any reachable C command.
- `cmdColor` (`pkg/session/act_informative.go`) was a placebo — `color on`
  printed "Color enabled." but never touched `PrfColor1`/`PrfColor2`. Ported
  the real 4-level `do_color` (off/sparse/normal/complete, `on` = shorthand
  for complete) — the display grid already expected this model
  (`colorLevelStr`), it just had nothing wired to it.
- `cmdAutoExit` was a no-op stub (`"Auto-exit toggled."` regardless of state).
  Now actually flips `Player.AutoExit`. Note: `autoexit` has no command-table
  entry in `src/interpreter.c` at all (dead in the original too) — this is a
  Go-only convenience command, just no longer a broken one.
- `cmdToggle` (`toggle`) had an invented `switch args[0]` that only handled
  `autoexit`; the real `do_toggle` ignores its argument entirely and always
  prints the grid. Removed the fake dispatch so `toggle <anything>` matches
  the original (display-only; real per-toggle commands now work directly).

See `TestBucketEToggleCommandsRegistered`, `TestDoGenTogRealCommandNames`,
`TestDoGenTogImmortalGated`, `TestCmdColorPersistsLevel`,
`TestCmdAutoExitTogglesPersistently`, `TestCmdToggleIgnoresArguments`.

Not done (low value, deferred): `ident`/`slowns` are global, non-player-flag
debug toggles (`SCMD_IDENT`/`SCMD_SLOWNS`, `LVL_IMPL-1`) for remote-username
lookups and slow-nameserver mode — no Go equivalent state exists, same spirit
as Bucket D's deferred wiz actions.

---

## Recommended order of attack

1. **Bucket A** — DONE. Wired the ~25 implemented skills.
2. **Bucket C verification** — DONE. Banking and recharge bugs fixed;
   postmaster/boards/stable/retrieve verified functional; `donate`/`junk`
   implemented (PR #30) and bugfixed (PR #31).
3. **Bucket E** — DONE. Toggle dispatch wired (16 commands); `color`/
   `autoexit`/`toggle` placebo bugs fixed along the way.
4. **Bucket B Part 1** — DONE (2026-06-18). Shared-handler aliases (`abilities`
   `glance` `mount` `reallyquit`), real drink/eat variants (`sip` `taste`),
   wired-but-uncalled logic (`insult` `dream`), new flavor/info commands
   (`think` `credits` `news` `policy` `handbook` `future` `whoami` `version`),
   and stricter-gated wizutil aliases (`reroll` `unaffect`).
   **Bucket B Part 2** — deferred: missing class skills (`charge` `track`,
   `shadow` `spike` `stake` `kabuki`), missing player actions (`fill` `grab`
   `leave` `search` `enter` `orgasm`), and `socials`/`ident`. Needs new game
   logic or a registry schema change — guided by `src/`; good candidate for
   the clawpatch→Daeron→coding-tool loop one skill at a time.
5. **Bucket D** — defer.

These buckets are derived from static analysis; (1) and (2) should be confirmed
by booting the server and actually invoking the commands.

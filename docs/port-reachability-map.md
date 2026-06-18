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

### Foundational blocker — timed affect → stat pipeline (PARTIALLY FIXED 2026-06-18)

Before implementing buff skills, note: most Bucket B class skills (berserk,
kuji-kiri seals rin/kyo/toh/kai/zai) buff **hitroll/damroll/AC** via timed
affects. The live path `cast → CallMagic → MagAffects → AddAffect` stores the
affect in `ActiveAffects` with the correct Location/Magnitude, **but base stats
are never modified and the stat getters ignored `ActiveAffects`** — so every such
buff (and the buff *spells* bless/armor/metalskin/psyshield) was silently inert.

**Fixed:** `GetHitroll`/`GetDamroll`/`GetAC` now fold in `ActiveAffects` via
`Player.sumAffectModsLocked(location)` (read-side summation, matching how
equipment affects already work; expiry via `RemoveAffectBySpell` auto-reverts).
Guarded by `TestActiveAffectsModifyStats` / `TestActiveAffectsStackAndExpire`.

**Still open (same pattern / related):**
- Extend summation to `ApplyStr`/saving throws/`ApplyHit`(maxHP)/etc. for full
  buff-spell fidelity (strength, bless's saving-throw component). One line each
  in the respective getters using the same helper.
- **Dual affect-path routing:** `cast` routes curse/poison/sleep/blindness/
  sanctuary to `AffectManager.ApplyAffect` with `engine.ApplyNone` (flags only),
  so curse's -hitroll/-damroll and poison's -str never apply, and those affects
  live in the manager's list, not `ActiveAffects`. Decide on ONE affect system.

### Missing skills/commands

No handler found in `pkg/`. These need implementation. Prioritize the class
skills (they make the combat system feel complete); verify each against `src/`.

- **Martial-arts forms (no handler):** `jin` `kai` `kyo` `retsu` `rin` `sha`
  `zai` `zhen` `shadow` `spike` `stake` `kabuki` — Dark-Pawns class skills.
  Cross-reference `src/` for which class/level and the skill formula.
- **Combat/movement skills:** `berserk` `charge` `mount` `track` — track has
  pathfinding scaffolding in `pkg/game/graph.go`; mount has ride logic in
  scripting. Verify before classing as full rewrites.
- **Player actions:** `fill` `grab` `junk` `leave` `search` `sip` `taste`
  `think` `dream` `glance` `insult` `orgasm` `reroll` `reallyquit` `enter`
  (enter has a spec_procs_missing.go stub).
- **Info commands:** `abilities` `credits` `version` `news` `policy` `handbook`
  `ident` `whoami` `future` `socials` (list socials) `unaffect`.

⚠ This list is from a name/handler probe; confirm each against `src/` and a
`grep` for alternate names before deciding rewrite vs. wire-up.

---

## Bucket C — Spec-proc handled (likely already reachable; verify in context)

These showed implementations in `spec_procs*.go`, meaning they work when the
player is at the right mob/room (e.g., a banker, a postmaster). Verify by
playing, not by registry presence:

- **Banking (banker spec proc):** `balance` `deposit` `withdraw` `gold` `coins`
- **Postmaster:** `mail` `check` `receive`
- **Other spec procs:** `collect` `donate` `stable` `retrieve` `will` `offer`
  `rent` `recharge`
- **Boards:** `read`

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

## Bucket E — Preference toggles (verify dispatch)

The `no*` family (`noauction` `nogossip` `notell` `noshout` `notitle` `nowiz`
`noctell` `norepeat` etc.) all probe to `other_settings.go`. Determine whether
they're reached via a unified `toggle`/`set` command or need individual
registration. `brief` `compact` `quest` `afk` are in the same family.

---

## Recommended order of attack

1. **Bucket A** — wire the ~25 implemented skills (one-liners). Biggest player-
   facing win per unit effort; restores whole class skill sets. Each needs a
   smoke check that the handler runs.
2. **Bucket C verification** — boot + play to confirm banking/postmaster/boards
   actually work in context; cheap, may close ~15 items with zero code.
3. **Bucket E** — confirm toggle dispatch; likely a small unification.
4. **Bucket B** — implement genuinely-missing class skills, guided by `src/`.
   This is the real remaining port *work*; good candidate for the
   clawpatch→Daeron→coding-tool loop one skill at a time.
5. **Bucket D** — defer.

These buckets are derived from static analysis; (1) and (2) should be confirmed
by booting the server and actually invoking the commands.

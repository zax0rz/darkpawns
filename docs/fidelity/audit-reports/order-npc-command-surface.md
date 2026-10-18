# Ordered NPC commands — actor-lane surface audit (DP-1453, 2026-10-18)

Incident: `order <pet> report` produced `WARN ordered NPC command unsupported
target=a horse command=rep` and nothing else. Egon's live report, 2026-10-18.

## The C call path (R5e)

`do_order` splits the line with `half_chop` and hands the unmodified message to
`command_interpreter(vict, message)` for a master-linked, charmed victim
(`src/act.offensive.c:321-333`) and for each same-room charmed follower
(`:335-352`). `command_interpreter` resolves prefixes against the ordered
`cmd_info[]` table (`src/interpreter.c:909-912`), so `rep` reaches `report`
(`src/interpreter.c:634`). There is no per-verb whitelist and no
mobile-specific dispatcher.

## The Go dispatch lane

`cmdOrder` → `manager.executeCharacterCommand` → `dispatchCharacterCommand`
(`pkg/session/character_dispatch.go`). The shared prelude already resolves the
abbreviation through `resolveCommandPrefix` (`pkg/session/command_order.go`,
generated from the C table), so `rep` resolved to `report` correctly. The drop
happened one step later: a resolved command with no `ActorHandler`
(`pkg/command/registry.go`) returned `CharacterCommandUnsupported`, which
`executeOrder` only logged. Silence was therefore a missing *actor body*, not a
missing abbreviation rule; this PR adds no string special-cases.

## Implemented in this PR

| command | C body | notes |
|---|---|---|
| `report` | `src/act.other.c:799-821` | Shared player/mobile body; the reporter's H/M/V and the recipient's `C_NRM` wrappers (`src/screen.h:47-52`); mobile `Kills/PKs/Deaths` read C's zeroed shared `dummy_mob` (`src/db.c:104,1281`). |
| `rest` | `src/act.movement.c:770-806` | Table-driven position body shared with the player path, including the `default` arm's `POS_SITTING` quirk. |
| `sleep` | `src/act.movement.c:808-839` | Same shared table (STANDING falls through to the sleep arm). |
| `wake` | `src/act.movement.c:841-881` | Shared self/target body; the target branch's room and victim lines are the only observable ones for a mob. |
| `stand` (mounted) | `src/act.movement.c:696-702` → `do_dismount` | A mounted NPC `stand` now dismounts instead of returning Unsupported. |
| `dismount` | `src/act.other.c:1597-1619` | Uses the existing `doDismountActor` mobile-rider path. |
| `mount` / `ride` | `src/act.other.c:1545-1595` | `CAN_MOUNT(mob)` is false (`src/utils.h:365-366`); every refusal but the non-mountable `TO_CHAR` is descriptorless, so the C-faithful body is a no-op for an NPC. |
| `follow` | `src/act.movement.c:883-951` | Non-quiet lane (`subcmd` 0 for order/force), reuses `followMobBody` for the state transition and audience lines. |

## Class audit (R5b) — how to regenerate

Level-0 `cmd_info[]` rows are the NPC-reachable surface. For each missing actor
lane, the script reads the named `do_*` function and flags it when its body
contains `TO_ROOM`/`TO_NOTVICT` — i.e. when an NPC's execution can change what
another character sees.

```bash
python3 - <<'EOF'
import re, glob
src = open('src/interpreter.c').read()
rows = re.findall(r'\{\s*"([^"]+)"\s*,\s*(POS_[A-Z_]+)\s*,\s*(do_[A-Za-z0-9_]+)\s*,\s*([^,]+),\s*([^}]+)\}', src)
texts = {f: open(f, errors='replace').read() for f in glob.glob('src/*.c')}
for name, pos, fn, lvl, sub in rows:
    body = ''
    for f, t in texts.items():
        m = re.search(r'(?:ACMD\()?' + re.escape(fn) + r'\)\s*\n?\{', t)
        if m:
            body = t[m.end():t.find('\n}', m.start())]
            break
    if re.search(r'TO_ROOM|TO_NOTVICT', body):
        print(name.strip().lower(), pos, fn, lvl.strip(), sep='\t')
EOF
```

Level-0 rows total 189. Socials are **not** in that count as gaps: the 183
`do_action` rows are handled by `dispatchCharacterSocial` → `NPCSocialCommand`
(`pkg/session/character_dispatch.go`), which is why they have no registry entry.

### Still without an actor lane: 166 rows

**56 with a C room-visible effect** (the remainder of the class this PR does not
claim):

```
afk aid alter ask behead bite carve disarm donate drink dream drop eat escape
fill flesh groinrip hit insult junk jin kill kai kyo mindlink murder neckbreak
parry point pour qui quit retreat retsu reallyquit rin roll scrounge sha sharpen
shoot sip sleeper spike stake subdue tag taste think toh turn ungroup whisper
yank zai zhen
```

Grouped, with the reason each is deliberately deferred:

- **Combat and offensive skills** (`alter`/`flesh`, `behead`, `bite`, `disarm`,
  `groinrip`, `hit`/`murder`, `kill`, `neckbreak`, `parry`, `retreat`/`escape`,
  `subdue`, `shoot`, `sleeper`, `spike`/`stake`, `carve`, `sharpen`, and the
  `do_kuji_kiri` rows `jin`/`kai`/`kyo`/`retsu`/`rin`/`sha`/`toh`/`zai`/`zhen`):
  these enter `hit()`/skill rolls and must reproduce C's draw order and wait
  states (R3). Copying a Player-only handler would invent a mobile combat path
  (R4).
- **Item transfer/consumption** (`drop`/`donate`/`junk`, `drink`/`sip`,
  `eat`/`taste`, `pour`/`fill`, `yank`, `scrounge`, `tag`): the item bodies are
  Player-inventory implementations; a mobile lane needs the mobile carry/weight
  rules ported as a family, not per verb.
- **Communication/dark-room extras** (`ask`, `whisper`, `think`, `point`,
  `dream`, `aid`, `mindlink`, `roll`, `turn`, `afk`, `ungroup`, `insult`):
  small, but several still draw RNG (`roll`: `src/act.other.c:1940`) or depend on
  Player-only preferences. `roll` is the sharpest R3 gap in this group: an NPC
  ordered to `roll` consumes a draw in C that Go does not.
- **Deliberate no-ops in C** — listed so they are not mistaken for gaps:
  `quit`/`qui`/`reallyquit` return immediately for `IS_NPC(ch) || !ch->desc`
  (`src/act.other.c:84-85`); `do_save` has the same shape. C produces no bytes
  and no state change for an ordered mob.

**110 with no room-visible C effect** (every message is `send_to_char` to the
descriptorless mobile, or the effect is state only the mob observes): these
remain `CharacterCommandUnsupported`, which logs a server warning but emits
nothing to any player — the same player-visible result C produces. They are not
claimed as ported; the warning is the deliberate gap marker.

Ordering any of the remaining verbs therefore still logs
`ordered NPC command unsupported` and drops the command. That is the honest
state: no player-visible bytes are invented (R4), and the gap is visible in the
log rather than silently green.


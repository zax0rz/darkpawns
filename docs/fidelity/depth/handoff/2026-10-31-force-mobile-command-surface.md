# Force's mobile command surface — audit list (2026-10-31, deepseek)

Found while fixing F9 (`group` could not run as a mobile). The rest of the class
is deferred: it is a command-surface question, not a colour one, and F9's own fix
is the piece the release needs.

## The defect

C's `do_force` resolves its target with `get_char_vis` — mobiles included — and
hands the line straight to `command_interpreter(vict, to_force)`
(`src/act.wizard.c:1869-1880`). Every command in C's table is therefore runnable
by a mobile, subject only to that command's own gates (position, level, and any
`IS_NPC` check it carries itself). `do_group` has no `IS_NPC` gate, which is why
`force <mob> group <player>` works in C.

The port dispatches through `Manager.executeCharacterCommand`, a **partial
registry** of `ActorHandler`s (`pkg/session/character_dispatch.go:87-121`). A
command with no actor handler returns `CharacterCommandUnsupported`, and
`cmdForceText` logs it and moves on (`pkg/session/wiz_communication.go:221-223`) —
**after** the forcer has been told `Okay.` C also says `Okay.` there, so the bytes
match and only the *effect* is missing. That is what makes it hard to notice: the
forcer gets a success message and nothing happens.

## Sites to check

The registry entries live in `pkg/session/commands.go`. Every command C lets a
mobile reach needs either an actor handler or a note saying why it does not.
Worth starting with the commands a scripted or forced mobile plausibly runs:

| command | why it matters |
|---|---|
| `ungroup` | `do_ungroup` has no `IS_NPC` gate (`src/act.other.c:744-794`); the port's player path is session-shaped, the same split F9 had |
| `split` | `do_split` bails on `IS_NPC` (`src/act.other.c:829-830`), so the port only needs to match that refusal |
| `follow` | `do_follow` has no `IS_NPC` gate and mobiles do follow |
| `flee`, `stand`, `sit`, `sleep`, `wake`, `rest` | position commands; several already have handlers |
| `get`, `drop`, `give`, `wear`, `wield` | item commands C runs for mobiles, with their own gates |
| `cast`, `recite`, `quaff` | spell and item use |

Two things to establish for each before changing anything (R5e):

1. Does C's command body have an `IS_NPC` gate? If it does, the port needs that
   gate, not an implementation.
2. Is the command reachable at all for a mobile — a mob has no descriptor, so
   `send_to_char(ch, …)` lines are invisible and only TO_VICT/TO_NOTVICT output
   can reach a player.

## What a fix looks like

Prefer moving the command body into `pkg/game` on `Actor` values and having both
the session handler and a new `registerActorCommand` delegate to it, which is what
F9 did for `group`. A second, mobile-only implementation would re-create exactly
the split this audit keeps finding.

Where a command genuinely cannot be implemented yet, the open question is what to
do about `Okay.`: C's byte is correct for a command that ran, and for a command
that does not exist C prints `Huh?!?` to the *mobile*, which no player sees. The
port's honest options are a louder server-side signal or a documented gap — not a
different message to the forcer, which would invent a byte C never sends.

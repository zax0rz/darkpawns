# 2026-09-24: Lua scripts the C game never attached

Field notes from the first inventory of the Lua scripting surface (DP-1333).
Claims are backed by PF-022.

## What C actually scripts

The C oracle links Lua 4.0.1 and registers 54 functions for scripts
(`src/scripts.c`, `cmdlib`). Its world files attach a script to exactly five
mobs, using four files. No room and no object is scripted. The rest of C's
script tree sits under `lib/scripts/mob/archive/`, unattached. Almost every
mob behaviour in C is a compiled `SPECIAL()` procedure: 235 assignments in
`spec_assign.c`.

## What the port scripts

The port registers all 54 bindings, plus about 140 more names: constants C
defines in `globals.lua`, Lua-4 compatibility shims for gopher-lua's Lua 5.1,
and a handful of functions C never had. Its world files attach scripts to
323 mobs. The 318 extra attachments came from the DP-165 "Lua script wiring"
work in May 2026, by an agent account that is not Zach and not a fidelity
agent. Its wiring plan matched archived scripts to mobs by keyword. The
largest batch landed under the commit title "fix: remove slog.SetDefault
deadlock", which describes none of it.

## Why nobody noticed

The census has seen these scripts misbehave for months, and 73 scenarios
answer it with a fixture, `strip-mob-script`, whose comment says it forces
"native special dispatch in both copies". Each scenario author met an
invented script and routed around it, one scenario at a time. None of them
asked why the port had a script where C had none.

That's the coincidence-green lesson in harness form. A fixture that makes
two servers comparable can also be the thing hiding why they weren't. The
question to ask of a fixture is the one to ask of a green result: would it
be needed if the port were faithful? For `strip-mob-script` on 318 mobs, the
answer is no.

## What this does to the question "is Lua ported?"

The question assumed Lua was an unported surface. The inventory says the
port runs far more Lua than C does. So the first fidelity step is removal:
put the world data back to C's five attachments and make sure each affected
mob's C special is wired. Only then does certifying the Lua engine against
the oracle mean what it says.

## The gate that would have caught it

The fix restores C's mob files and script tree byte for byte, and adds a CI
gate (`make world-fidelity`, `scripts/world_fidelity.py`). The gate compares
every world and script file against a hash manifest generated from the C
oracle, and any difference must be listed in
`docs/fidelity/world-exceptions.tsv` with a status and a reason. Run against
main as it stood, the gate reports 66 files that differ from C, 139 files C
does not have, and 170 C files that are missing. The whole DP-165 drift would
have failed CI the day it landed. It found one more thing too: four zone files
edited on 2026-05-03 to comment out loads of mobs that no longer exist. The
owner chose to restore C's files and make the port behave as C does.

With the data restored, the 73 `strip-mob-script` fixtures and the 5
`mirror-oracle-scripts` fixtures have nothing left to do, and they are gone.
None of the stripped mobs was one of C's four scripted mobs. Every one was
working around an attachment C never had (PF-025).

## Restoring four zones found a reset bug in a fifth

Doing that meant porting `renum_zone_table` (`db.c:968-1024`), the boot pass
that disables any reset command naming a missing room, mobile or object. Two
things came out of it. The four edited zones (58, 85, 95, 97) are not in C's
boot index, so neither server ever loaded them, and the May edit had changed
files nothing read. More importantly, the same C function also converts the
legacy `R` format, `R <if> <room> <vnum> -1`, into an object removal. Every one
of the shipped world's 30 `R` commands is in that format, all in the Haven
Lighthouse (zone 158), where each reset replaces a crystal chest with a stocked
one. The port read the legacy form as a mobile removal. The chest was never
removed, the load that follows found it at its maximum, and the loot command
behind it was skipped, so a looted chest never refilled. The port's own unit
tests used the port's argument order, so they could not catch it
(`zone-reset-remove-object` now proves the C behaviour; PF-026).

A decision that looked like housekeeping turned into reading a C function
nobody had ported, and that function held a live bug in a zone the decision
never touched.

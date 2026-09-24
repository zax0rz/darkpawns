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

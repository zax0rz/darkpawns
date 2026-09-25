# 2026-09-24: Porting the Lua bridge

Field notes from DP-1333 PR 2, which replaced the port's approximations of
C's script bindings with a port of the C↔Lua bridge in `src/scripts.c`.
Claims are backed by PF-027 to PF-031.

## Shims that never ran

gopher-lua implements Lua 5.1; C links Lua 4.0.1. The port covered the gap
with shims for Lua 4's global string functions. `strfind`, `strsub`, `gsub`
and `tonumber` were written in the shape of the C API:

```go
L.GetGlobal("string")
L.GetField(L.Get(-1), "find")
L.Push(L.Get(1))
L.Push(L.Get(2))
L.Call(2, 1)
```

In C's API `lua_getglobal` pushes its result. In gopher-lua `GetGlobal` and
`GetField` return theirs, so nothing was pushed and `Call` invoked the
script's first argument: "attempt to call a non-function object". Every
call failed. `format` was not defined at all, and gopher-lua's
`string.format` hands `%u` to Go's fmt, which prints `%!u(int64=2000)`.

All four scripted mobs in C's world depend on these functions: hisc's
`oncmd` through `no_move()`, the healer, blacksmith and high priest through
`assembler.lua`'s `return_obj()`. None of them could work in the port. The
code reads plausibly to anyone who knows the C API, and it compiled; the
only signal was an error log line per script run.

The fix binds the Lua 4 names to the Lua 5.1 library functions, which take
the same arguments and return the same results, and adds a Lua 4 `format`.

## What run_script actually writes back

C's `run_script` ends:

```c
if (ch && ch->in_room > 0) {
  lua_getglobal(L, "ch"); table_to_char(L);
  if (me && (me != ch)) { lua_getglobal(L, "me"); table_to_char(L); }
}
```

It reads as "write back ch, then me". But `table_to_char` reads its table
from stack slot 1 (`lua_pushvalue(L, 1)`) and pops nothing. At top level
the stack is empty when the write-back starts, so slot 1 is the ch table
both times. ch is written back twice; me never is. A script's changes to
`me` are lost, even for a pulse trigger where me and ch are the same
character, because the two globals are separate tables. `never_die.lua`
(`me.hp = me.maxhp` every pulse) would do nothing in C; it is attached to
no mobile.

The first port of the write-back wrote me as well, following the code's
evident intent. R5e: the call path, not the reading, is the law.

## Nesting

A script's `action(me, "give ...")` can reach another mobile's `ongive`,
and `raw_kill()` a `death` script, while the first script is still
running. C runs them nested on the one Lua state. The port held a plain
mutex for the whole of RunScript, so the same path would have deadlocked
the game loop. RunScript now recognises a run from the goroutine already
running a script and nests it on the outer run's lock, deadline and stack
frame.

## An invented line with a C citation

`MobInstance.RunScript` sent "You can't give that here." when an `ongive`
returned false. The string is in no C source file. An earlier audit report
(`docs/fidelity/audit-reports/scripts_audit.md`) and a brief
(`docs/briefs/brief-6-shop-system.md`) had instructed agents to "restore"
it as C behaviour. Agent prose attached C provenance to bytes C never had.

## Result

With the bridge in place, six oracle scenarios on the four C-scripted mobs
match C byte for byte. The healer's sound script over 8000 frozen-clock
pulses matches with weather messages interleaved, which pins where its
`number()` draws fall in `mobile_activity`. The healer's `ongive` given a
duplicate runestone tells the giver, remarks, then falls through to
`return_obj()` a second time and hands back the stone it had kept. That is
C's script, and the port now does it too.

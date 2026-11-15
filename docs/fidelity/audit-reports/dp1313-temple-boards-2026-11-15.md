# DP-1313 — the Temple of the Cross board does not duplicate

DP-1313 reports that the Temple of the Cross accumulated bulletin boards on
the live server. This session checked both engines' reset path.

## C

Zone 80's reset list loads the board with `O 0 8099 5 8008`. `reset_zone`'s
`O` arm (`src/db.c:2149-2150`) loads **one** instance per reset and only while
the world-wide prototype count is below the max:

```c
case 'O':
  if (obj_index[ZCMD.arg1].number < ZCMD.arg2) { ... read_object ... }
```

so the cap is global, not per room, and a room holding the maximum never gains
another board.

## Go

`Spawner`'s `O` case calls `canSpawnObject(vnum, max)` ->
`World.countObjectInstances(vnum)`, a world-wide count over the registered
object instances (`pkg/game/world_player.go:111-121`). That is C's check.

## Witness

`board-temple-reset-depth` puts a wizard in room 8008 and runs seven
`zreset 80` cycles, with a `look` after each. C and Go match byte-for-byte and
keep the same board count (one in this fixture; draw parity MATCH, 32229
entries). The reset's earlier commands gate the board line the same way in
both engines, so neither adds a duplicate.

## Separate finding

A `stat room 8008` variant used to count the boards exposed an unrelated
divergence: the room line reports `SpecProc: Exists` in C and `SpecProc: None`
in Go for this room. That is a room-spec-proc display gap, not a board or
reset bug, and is reported separately.

## Conclusion

The port matches C and does not duplicate the board; DP-1313 does not
reproduce on current main and needs no code change.

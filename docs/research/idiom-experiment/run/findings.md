# Latent port findings

## `give` does not resolve C's `me` / `self` recipient aliases

- Go: `pkg/game/item_transfer.go:503-506` sends `NoPersonHere` when the
  recipient is not a player name; this lookup does not special-case `me` or
  `self`.
- C: `src/handler.c:1285-1286` resolves either alias to the actor before
  `src/act.item.c:718-726` rejects self-gives with `What's the point of that?`.
- Difference: `give bread me` and `give bread self` produce the missing-person
  response in Go, while C produces the self-give response. `give-basic` covers
  the actor's character name, but not these aliases.
- Fidelity law: R1 and R2.
- Confidence: high; the differing branches and response strings are explicit
  in the source. Left unchanged per the experiment brief.

## `give` resolves a mob before C's shared room-character order

- Go: `pkg/game/item_transfer.go:602-620` searches `FindMobInRoom` before
  `giveFindVict` searches players.
- C: `src/act.item.c:798` calls `give_find_vict`, which uses
  `get_char_room_vis`; `src/handler.c:1293-1297` returns the first visible
  name match in the shared room list, and `src/handler.c:537-538` inserts new
  room occupants at its head.
- Difference: if a player's name also matches a mob keyword, Go always
  chooses the mob, while C chooses whichever matching character appears first
  in the room list. That can change the recipient and the mob-trigger path.
  Player-name validation does not compare against mob keywords, though a
  deployment's invalid-name list could reject a particular collision.
- Fidelity law: R1 and R2.
- Confidence: medium; the lookup order differs in source, but the exact
  collision is not covered by a current oracle scenario. Left unchanged per
  the experiment brief.

# GMCP in Dark Pawns

The telnet listener speaks GMCP (telnet option 201), the structured side
channel Mudlet and other rich MUD clients read. This is the reference for the
message set: what is sent, when, and in what shape.

**Message set version 1.** `session.GMCPMessageSetVersion`; module versions
are negotiated per module through `Core.Supports` (below).

## The rule

GMCP is out of band. A client that never negotiates option 201 receives
exactly the bytes it always did, and every message here is sent *after* the
C-faithful text for the same game moment, never instead of it (R1).

GMCP may restate what the text just told the player. It never tells them
something the text withheld (R4):

- A room is reported only when a room is rendered. In darkness or blindness
  `look_at_room` prints "Darkness" and no room name, so no `Room.Info` is sent.
- Exits follow `do_auto_exits`: closed doors are omitted for mortals and shown
  to immortals.
- A channel speaker is named exactly as the line named them: an unseen speaker
  is `"Someone"` in both.

`pkg/telnet/gmcp_test.go` holds the byte-level proof: a Mudlet-shaped client
and a plain client run the same session over a real socket, and their visible
text is identical.

## Negotiation

On connect the server offers `IAC WILL GMCP` (and `IAC WILL EOR`, see below).
A client that answers `IAC DO GMCP` has GMCP on for the connection.

| Client sends | Server does |
|---|---|
| `Core.Hello {"client","version"}` | records the client |
| `Core.Supports.Set ["Module n", ...]` | replaces the enabled-module list |
| `Core.Supports.Add [...]` / `.Remove [...]` | edits it; newly enabled modules get their current state at once |
| `Core.Ping` | answers `Core.Ping` |

Until a client sends its first `Core.Supports` message every module is sent.
After it, only enabled modules are: an entry enables the module itself, any
package inside it (`Char` covers `Char.Vitals`), or, for a parent entry, every
module under it (`Comm` covers `Comm.Channel`). Mudlet's default set enables
`Char` and `Room`; `Comm.Channel` must be added (the Dark Pawns package and
Mudlet's starter UI both add it).

| Module | Version | Packages |
|---|---|---|
| `Char` | 1 | `Char.Name`, `Char.Vitals`, `Char.Status` |
| `Room` | 1 | `Room.Info` |
| `Comm.Channel` | 1 | `Comm.Channel.Text` |

`Client.GUI` is a Mudlet core message rather than a module, and is sent to
every GMCP client when the server is configured with `DP_MUDLET_PACKAGE_URL`.

## Server messages

Field order and types are fixed within a module version.

### `Char.Name`

```json
{"name":"Walker"}
```

When the player enters the game, and again if the name changes.

### `Char.Vitals`

```json
{"hp":80,"maxhp":100,"mp":20,"maxmp":20,"mv":98,"maxmv":100}
```

Sent only when a value changed, at three moments:

- ahead of every prompt (C `make_prompt`, `comm.c:1028`; the prompt is where
  vitals are shown);
- when the player takes combat damage (the damage message is the text);
- after each `point_update` tick (`limits.c`), whose regeneration prints
  nothing but changes the numbers `score` and the prompt report.

### `Char.Status`

```json
{"name":"Walker","level":7,"race":"Elven","class":"Magic User","gold":12}
```

Sent with `Char.Vitals`, when a value changed. `race` and `class` are C's
`races[]` (`constants.c`) and `pc_class_types[]` (`class.c`) strings, the ones
`do_score` prints.

### `Room.Info`

```json
{"num":3001,"name":"The Temple Square","area":"Kir Drax'in","environment":"City","exits":{"n":3002,"s":3005}}
```

After every room render, where C calls `look_at_room` (entry, movement
`act.movement.c:266`, `look`, recall, teleport, summon, following a leader),
except dark and blind renders. Peeking through an exit (`look north`) is not a
render of the player's room and sends nothing.

- `num`: room vnum. The world map on the Dark Pawns site already publishes
  every vnum, name and exit.
- `area`: zone name.
- `environment`: the sector name `look_at_room` prints with `PRF_ROOMFLAGS`
  (`Inside`, `City`, `Field`, ..., `Unknown`).
- `exits`: direction (`n e s w u d`) to destination vnum, filtered as
  `do_auto_exits` filters them.

There is no `Room.WrongDir`. The game room numbers make it unnecessary: a
mapper never has to guess where a move went, because the next `Room.Info`
names the room.

### `Comm.Channel.Text`

```json
{"channel":"gossip","talker":"Someone","text":"Someone gossips, 'hi'"}
```

Once per delivered line, to each recipient, right after the line itself.
`text` is the line exactly as that player received it, without its line
ending; `talker` is the speaker as that player saw them.

| `channel` | C source |
|---|---|
| `say` | `do_say` (`act.comm.c:759`) |
| `tell` | `perform_tell` (`act.comm.c:873`), so `tell` and `reply` |
| `gossip`, `auction`, `grats`, `shout`, `holler`, `newbie` | `do_gen_comm` (`act.comm.c:1146`), and NPC gossip |
| `group` | `do_gsay` (`act.comm.c:824`) |
| `clan` | `do_ctell` (`act.comm.c:1451`) |

A `PRF_NOREPEAT` acknowledgement ("Okay.") is not a channel line and is not
sent.

### `Client.GUI`

```json
{"version":"1.0.0","url":"https://darkpawns.org/darkpawns.xml"}
```

On GMCP negotiation, when `DP_MUDLET_PACKAGE_URL` is set, and again as the
character enters the game. Mudlet installs the package at `url` and
reinstalls it whenever `version` (`mudlet/VERSION`) changes. The second offer
repairs a lost upgrade: Mudlet 5.0.1 can drop the new version when its
download lands while the profile is still saving after the old one was
removed, and an offer that finds no package installs it fresh. A client that
has this version already ignores it.

### `Client.Map`

```json
{"url":"https://darkpawns.org/darkpawns-map.xml","version":"3f9a0c1d2e4b"}
```

On GMCP negotiation, when `DP_MUDLET_MAP_URL` is set. Mudlet records `url`
as the game's map location; the Dark Pawns package also reads `version` and
loads the map (see `mudlet/README.md`).

The map is served by the game server at `/darkpawns-map.xml`
(`pkg/mudletmap`): the whole world as a Mudlet XML map, generated from the
**live** world so OLC edits reach it within five minutes, with an `ETag` of
the version so an unchanged map is never downloaded twice. Each zone is a
Mudlet area (zone *n* is area *n*+1, named as `Room.Info`'s `area`), rooms
are keyed by vnum, and each zone is laid out on the grid by walking its
exits. Dark Pawns has no coordinates of its own, so where exits don't fit a
grid (mazes, loops) a room is pushed further along its exit's direction
rather than drawn on top of another. Doors are marked but not given a state,
because resets and players change it. The site already publishes every room,
name and exit, so the map reveals nothing new.

Room.Info's `area` uses the same names, so a room mapped live lands in the
downloaded area. The two zones that share a name (`New Zone`) are told
apart as `New Zone (zone 34)` and `New Zone (zone 36)`.

## Prompt marking (EOR)

The server also offers `IAC WILL EOR`. A client that answers `IAC DO EOR`
gets `IAC EOR` after every prompt, which is how Mudlet tells a prompt from a
line still being written. It is a telnet command, never shown, and a client
that declines receives none.

## MSSP

MSSP advertises `GMCP 1`, `MCCP 1` and `ANSI 1` alongside the existing fields.
When the TLS telnet port runs (`-telnet-tls-port`), MSSP also carries
`TLS <port>` and `HOSTNAME <certificate name>`; Mudlet reads them to offer the
encrypted port to a player who connected in plaintext.

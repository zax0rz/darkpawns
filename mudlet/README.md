# Dark Pawns for Mudlet

A [Mudlet](https://www.mudlet.org/) package for Dark Pawns: hit point, mana
and movement gauges, a map that draws itself as you walk, and a chat window.
Everything it shows comes from GMCP data the game sends alongside its text;
the game text itself is exactly what any other client sees.

## Install

1. In Mudlet, make a new profile:
   - **Server address:** `darkpawns.org`
   - **Port:** `7777`
2. Connect. If the server offers the package, Mudlet downloads and
   installs it on its own, and upgrades it when a new version ships.
3. Otherwise, download [`darkpawns.xml`](https://darkpawns.org/darkpawns.xml) and import it:
   **Toolbox → Package Manager → Install**, then choose the file.

The dock appears on the right. Type `dp` for its commands.

### Secure connection

If the server runs its TLS port, Mudlet says so the first time you connect
("A more secure connection on port N is available") and offers to switch.
To set it up yourself instead, use the TLS port the server announced (the
examples in this repository use `7778`) and tick **Secure**. Connect by the
server's name, `darkpawns.org`, rather than an IP address: the certificate is
issued for the name.

Your password travels encrypted only on the secure port.

## What you get

| Part | Fed by | Notes |
|---|---|---|
| Character line | `Char.Status` | name, level, race, class, gold |
| Gauges | `Char.Vitals` | update with the prompt, on damage, and on the regeneration tick |
| Map | `Room.Info` | rooms keyed by the game's room numbers; double-click to walk |
| Chat | `Comm.Channel.Text` | each line exactly as the game printed it |

The map only learns rooms you can see. In darkness or while blind the game
names no room, and neither does GMCP; the map picks up again at the next lit
room. Closed doors are left out of a room's exits for the same reason the
game's own exit list leaves them out.

The package adds no game commands and no triggers on game text. Dark Pawns
already understands `n`/`e`/`s`/`w`/`u`/`d` and abbreviated commands such as
`con` for `consider`.

## Commands

| Command | Does |
|---|---|
| `dp` | show the version and these commands |
| `dp hide` / `dp show` | hide or restore the dock |
| `dp clear` | clear the chat window |

## For developers

The package is generated. Edit the Lua in [`src/`](src), then:

```bash
go run ./cmd/mudlet-package
```

Bump [`VERSION`](VERSION) and add a [`CHANGELOG.md`](CHANGELOG.md) entry for
every change: Mudlet only reinstalls a server-offered package when its version
changes. `go test ./cmd/mudlet-package` checks that the generated file is
current, loads every script into Lua 5.1 against a stub of Mudlet's API, and
replays a session through the gauges, map and chat.

The server offers the package through GMCP `Client.GUI` when
`DP_MUDLET_PACKAGE_URL` is set to a public URL of `darkpawns.xml` (Dark Pawns
itself serves it at `https://darkpawns.org/darkpawns.xml`); see
[`docs/gmcp.md`](../docs/gmcp.md) for the message set.

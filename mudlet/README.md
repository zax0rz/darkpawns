# Dark Pawns for Mudlet

A [Mudlet](https://www.mudlet.org/) package for Dark Pawns: hit point, mana
and movement gauges, a map that draws itself as you walk, and a chat window.
Everything it shows comes from GMCP data the game sends alongside its text;
the game text itself is exactly what any other client sees.

## Install

1. In Mudlet, make a new profile:
   - **Server address:** `darkpawns.org`
   - **Port:** `7778`
   - **Secure:** ticked
2. Connect. Mudlet downloads and installs this package on its own, and
   upgrades it when a new version ships.
3. If it doesn't, download [`darkpawns.xml`](https://darkpawns.org/darkpawns.xml)
   and import it: **Toolbox → Package Manager → Install**, then choose the file.

The dock appears on the right. Type `dp` for its commands.

If an upgrade leaves you without the dock, and `dp` answers `Huh?!?`,
disconnect and connect again: Mudlet installs the package fresh. Mudlet 5.0.1
can lose an upgrade when the download finishes while it is still saving the
profile after removing the old version; the next connection finds no package
and installs it without that step.

### Why port 7778

Port 7778 is encrypted, so your password and everything you type stay
private. Use the name `darkpawns.org`, not an IP address: the certificate is
issued for the name.

Port `7777` still works, for clients without TLS, but everything crosses the
network in the clear, your password included. If you connect there with
Mudlet, it tells you a more secure connection is available and offers to
switch. Say yes.

A self-hosted server may use different ports: its TLS port is whatever it was
started with (`-telnet-tls-port`), and Mudlet learns it the same way.

## What you get

| Part | Fed by | Notes |
|---|---|---|
| Character line | `Char.Status` | name, level, race, class, gold |
| Gauges | `Char.Vitals` | update with the prompt, on damage, and on the regeneration tick |
| Map | `Client.Map`, `Room.Info` | the whole world, downloaded on first connect; rooms added since are mapped as you walk; double-click to walk |
| Chat | `Comm.Channel.Text` | each line exactly as the game printed it |

The whole world map loads the first time you connect, if this profile has
no map yet, and updates itself when the world changes. If you already mapped
by hand, the package asks first: type `dp map` to swap in the full map.

Your position on the map follows the rooms you can see. In darkness or while
blind the game names no room, and neither does GMCP; the map picks up again
at the next lit room. Closed doors are left out of a room's exits for the same reason the
game's own exit list leaves them out.

The package adds no game commands and no triggers on game text. Dark Pawns
already understands `n`/`e`/`s`/`w`/`u`/`d` and abbreviated commands such as
`con` for `consider`.

## Commands

| Command | Does |
|---|---|
| `dp` | show the version and these commands |
| `dp hide` / `dp show` | hide or restore the dock |
| `dp clear` | clear the main window: the game's `clear` sends terminal codes Mudlet doesn't act on |
| `dp clear chat` | clear the chat window |
| `dp map` | load the whole world map (replaces this profile's map) |

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

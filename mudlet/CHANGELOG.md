# Dark Pawns for Mudlet — changelog

Every change to `src/` bumps `VERSION` and adds an entry here. The server
announces the version through GMCP `Client.GUI`, and Mudlet reinstalls the
package when it changes, so an unbumped change never reaches players.

## 1.1.6 (2026-09-23)

- Double-clicking a room on the map walks there. The package read the path
  from variables Mudlet only sets in its custom pathfinding mode, so in the
  default mode every double-click was a Lua error and nothing moved. It now
  sends the steps Mudlet has already found. Found in the Mudlet acceptance
  run (case 11).

## 1.1.5 (2026-09-23)

- The map shows after an upgrade. A profile has one map widget, made with
  the first dock; the dock 1.1.4 builds after an upgrade was drawn over it,
  leaving the map panel blank. The map is raised whenever the dock is built.
- `dp clear` clears the main window. The game's `clear` sends terminal
  clear-screen codes, which Mudlet's scrollback doesn't act on. `dp clear
  chat` clears the chat window, which `dp clear` used to do.
- `dp` takes more than one word after it, so `dp clear chat` reaches the
  package instead of the game.

## 1.1.4 (2026-09-23)

- An upgrade builds a new dock. Mudlet upgrades a package inside the running
  profile, and the new version found the old version's dock still in memory
  and showed it again, so 1.1.3's header never appeared until Mudlet was
  restarted. Each version now names its own widgets. Found in the Mudlet
  acceptance run (upgrade from 1.1.2).

## 1.1.3 (2026-09-23)

- The dock's header is the site header's lockup, as a picture: the pawn,
  then DARK over PAWNS in DM Serif Display. The package can't install fonts,
  so the lettering had fallen back to a sans-serif, and the pawn, stretched to
  fit its label, read as broken. The picture is drawn at its own size and
  never scaled; high-density screens get a 2x copy.

## 1.1.2 (2026-09-23)

- The world map offer is announced once, not twice: the load-time check
  and the `Client.Map` event could both see the same offer. A second map
  download can't start while one is running.

## 1.1.1 (2026-09-23)

- The world map offer is acted on even when it arrived before the package
  finished loading. The server offers the package and the map together, so
  on a first install or an upgrade the map offer landed while the package
  was still downloading, and neither the automatic load nor `dp map` saw
  it. Found in the Mudlet acceptance run (upgrade from 1.0.0).

## 1.1.0 (2026-09-22)

- The whole world as a map. When the server offers its world map (GMCP
  `Client.Map`), a profile with no map loads it on first connect, and a
  profile that loaded it takes each new version automatically. A map you
  built by hand is never replaced without asking: `dp map` loads the full
  one on request. Rooms the download doesn't have yet are still mapped as
  you walk.
- `dp map` command.

## 1.0.0 (2026-09-22)

First release, built against Dark Pawns GMCP message set 1 (`docs/gmcp.md`).

- Dock on the right, styled as the website's /play page: a Paper-Deep
  chassis with the Dark Pawns lockup (the pawn, DARK over PAWNS), the
  character line (`Char.Status`), hit point, mana and movement readings with
  thin gauges (`Char.Vitals`), then the map and a chat window on the game's
  dark canvas. DESIGN.md colours and fonts only; all text clears WCAG AA, and
  no reading depends on colour alone.
- Mapper built from `Room.Info` as you walk, keyed by the game's room
  numbers; double-click a room to walk there.
- Chat window fed by `Comm.Channel.Text`: say, tell, gossip, auction,
  grats, shout, holler, newbie, group and clan lines, as the game printed
  them.
- `dp` command: `dp hide`, `dp show`, `dp clear`.
- Asks Mudlet's built-in starter interface to stand aside, so gauges and
  chat are not drawn twice.
- Removes Mudlet's preinstalled generic mapper, which scrapes room text and,
  when a map window opens, sends a blank line and `look` to the game on its
  own: at the name prompt that ended the connection.

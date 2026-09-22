# Dark Pawns for Mudlet — changelog

Every change to `src/` bumps `VERSION` and adds an entry here. The server
announces the version through GMCP `Client.GUI`, and Mudlet reinstalls the
package when it changes, so an unbumped change never reaches players.

## 1.0.0 (2026-09-22)

First release, built against Dark Pawns GMCP message set 1 (`docs/gmcp.md`).

- Dock on the right: character line (`Char.Status`), hit point, mana and
  movement gauges (`Char.Vitals`), the map, and a chat window.
- Mapper built from `Room.Info` as you walk, keyed by the game's room
  numbers; double-click a room to walk there.
- Chat window fed by `Comm.Channel.Text`: say, tell, gossip, auction,
  grats, shout, holler, newbie, group and clan lines, as the game printed
  them.
- `dp` command: `dp hide`, `dp show`, `dp clear`.
- Asks Mudlet's built-in starter interface to stand aside, so gauges and
  chat are not drawn twice.

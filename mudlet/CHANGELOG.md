# Dark Pawns for Mudlet — changelog

Every change to `src/` bumps `VERSION` and adds an entry here. The server
announces the version through GMCP `Client.GUI`, and Mudlet reinstalls the
package when it changes, so an unbumped change never reaches players.

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

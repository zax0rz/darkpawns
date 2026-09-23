# Mudlet acceptance test plan

A hands-on gate for GMCP and the Mudlet package. The automated tests prove the
bytes on the wire (`pkg/telnet/gmcp_test.go`) and that the package loads
against a stub of Mudlet (`cmd/mudlet-package`); only a real Mudlet shows that
the map draws, the gauges move, and nothing errors in the console. Run this
before any change to GMCP or `mudlet/` reaches production, and post the
results table on the PR.

About 30 minutes. One tester, two clients.

## Setup

1. **Server.** From the branch under test, on a fresh database so the first
   character created becomes the immortal:

   ```bash
   python3 -m http.server 8088 --directory mudlet &
   ENVIRONMENT=development DP_MUDLET_PACKAGE_URL=http://localhost:8088/darkpawns.xml \
     DP_MUDLET_MAP_URL=http://localhost:4360/darkpawns-map.xml \
     go run ./cmd/server -port 4360 -telnet-port 7780 -db "$(mktemp -d)/mudlet-test.db"
   ```

2. **Mudlet.** The current stable release, with a new profile
   (`localhost`, port `7780`, **Secure** unticked: the local server has no
   certificate until the TLS cases). Record the version from **Help → About**.
3. **A second client.** Plain `telnet localhost 7780` in a terminal. It plays
   the other character and is the plain-text control.
4. **Inspection.** Mudlet's `lua` command prints GMCP tables, for example
   `lua display(gmcp.Room.Info)`. Keep **Toolbox → Errors** open throughout:
   any Lua error there fails the run.

Characters: create **Warden** first in the plain client (the first character
on a fresh database is the immortal), then **Tester** in Mudlet (a mortal).
Cases 9 and 10 swap them: Warden in Mudlet, Tester in the plain client.

## Cases

| # | Do | Expect |
|---|---|---|
| **Install** | | |
| 1 | Connect Mudlet for the first time, and stay at the name prompt. | Mudlet prints that it is downloading and installing `darkpawns`, then "Removed Mudlet's generic mapper". The dock appears on the right. The connection stays open at the name prompt: nothing is sent to the game on your behalf. No errors. |
| 2 | `lua display(gmcp)` before logging in. | `Client` present. No `Char` or `Room` yet. |
| 2a | Before logging in, open the map. | Mudlet reports downloading, then "World map loaded". The whole world is there: move around the map to check a few zones. |
| 3 | Log in as Tester. | The dock matches the website: a cream (Paper-Deep) panel with the site header's lockup at the top, identical to darkpawns.org: the ink pawn in one piece and **DARK** over **PAWNS** in the site's serif (PAWNS in oxblood), not stretched or blurred, then the character line, e.g. `Tester · level 1 Human Warrior · 0 gold`, then HP, MANA and MOVE readings (`20 / 20` style) over full bars, then MAP and CHAT on dark panels. Map shows the start room. |
| 4 | Type `dp`. | Version `1.0.0` and the command list. |
| **Map** | | |
| 5 | Walk out of the temple and around a few rooms, then back. | The map follows you: the current room stays centred, and no new rooms appear, because the downloaded map already has them. |
| 6 | Walk into a wall (`up` where there is no exit). | "Alas, you cannot go that way..." The map does not move. |
| 7 | Go to the Western Gate (8040, west end of Market Street). If the gate is open, `close gate`. Then `look`. | `west` is absent from `[ Exits: ]` and from `gmcp.Room.Info.exits`. |
| 8 | `open gate`, then `look`. (If it is locked, the city has shut it for the night: skip this case and rerun it by day.) | `west` appears in both. The map gains a stub, and a real exit once you walk through. |
| 9 | Close the gate again. Quit Tester and log Mudlet in as Warden. Type `autoexit` if the exit line is missing (the first character starts with it off, as in C), then `look` at the gate. | Warden's `[ Exits: ]` shows `(west)` in parentheses, and `gmcp.Room.Info.exits` includes `w`. Immortals see closed exits. |
| 10 | Still as Warden: `holylight` off if it is on, then `goto 3178` ("The Dark Room", flagged dark and indoors, so time of day can't light it), then `goto 8004`. | In the tunnel: "Darkness", `gmcp.Room.Info` unchanged, the map does not move. At 8004 the map recentres on the temple. |
| 11 | Log Mudlet back in as Tester for the rest of the run. Double-click a mapped room a few steps away. | Tester walks there. The path's commands appear as sent. |
| **Gauges** | | |
| 12 | Walk several rooms. | The Move gauge drops with each step, in step with the prompt's `V` value. |
| 13 | Stand still for two game hours (~2 minutes) after spending movement. | The Move gauge refills on the tick with no command typed. |
| 14 | Fight something weak until you take damage. | The HP gauge drops as damage messages arrive. |
| **Chat** | | |
| 15 | Warden: `say hello`. | Tester's main window and chat window both show `Warden says, 'hello'`. |
| 16 | Tester: `tell warden hi`, then Warden: `reply hey`. | Chat shows `You tell Warden, 'hi'` and `Warden tells you, 'hey'`, tagged `tell`. |
| 17 | Warden: `invis`, then `say boo`. | Tester's chat shows `Someone says, 'boo'`, never the name. |
| 18 | Warden: `gossip test`. | Tester's chat shows it tagged `gossip`, unless Tester's level is below the channel minimum, in which case both windows agree nothing arrived. |
| **Fidelity** | | |
| 19 | In both clients, as the same character in turn, run `help` with no argument. | Both show the first page and the pager prompt. Mudlet is paged exactly like telnet. |
| 20 | Compare the plain client's transcript for cases 5–8 with Mudlet's main window. | Same text. No stray characters, JSON or `[` fragments in either. |
| 21 | Watch the prompt in Mudlet. | Prompts render immediately, not after a pause, and never merge into the next line. |
| **Lifecycle** | | |
| 22 | Disconnect and reconnect. | No second dock, no duplicate chat lines (handlers did not stack). |
| 23 | Bump `mudlet/VERSION` to `1.0.1`, add a changelog line, run `go run ./cmd/mudlet-package`, restart the server, reconnect. | Mudlet reports upgrading from `1.0.0` to `1.0.1` and reinstalls. `dp` shows `1.0.1`, and the dock is rebuilt, map included. Revert the bump afterwards. On Mudlet 5.0.1 against a local server, the upgrade can leave no package (`dp` answers `Huh?!?`): a Mudlet race between its profile save and the download (see the README). Reconnect; it must install fresh. Note it, but it is not a package failure. |
| 24 | **Toolbox → Package Manager**, uninstall `darkpawns`. | The dock disappears and the window width is restored. Later GMCP produces no errors. |
| 25 | Re-import `mudlet/darkpawns.xml` by hand with the server's package URL unset. | The package installs from the file and behaves as in cases 3–5. Mudlet's built-in starter UI stays hidden. |
| 25a | New profile, map it by hand first: connect with the map URL unset, walk a few rooms, disconnect. Set `DP_MUDLET_MAP_URL` again, restart the server, reconnect. | Mudlet keeps the hand-made map and says a whole-world map is available via `dp map`. |
| 25b | `dp map`. | The download replaces the hand-made map with the whole world. |
| **TLS** (once the TLS listener is merged) | | |
| 26 | Run the server with `-telnet-tls-port 7781` and a certificate for a name that resolves to it. Connect in plaintext by that name. | Mudlet offers "A more secure connection on port 7781". Accept it. |
| 27 | Reconnect. | The connection is secure (padlock), and cases 3 and 15 behave the same. |

## Results

Copy into the PR:

```text
Mudlet version:
Server commit:
Result: PASS / FAIL
Failed cases (number, what happened):
Console errors (paste):
```

A failure in cases 19–21 is a fidelity failure (R1): it blocks the release
even if everything else passes.

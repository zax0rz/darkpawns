# Prompt boundary audit — 2026-09-24

DP-1326 showed that the ordinary oracle normalizer removes prompt-only lines
and trims their trailing spaces. Its green command blocks therefore cannot
prove `make_prompt` bytes (R1, R5f). This audit follows the reachable C paths
in `src/comm.c:643-648,1028-1185,1620-1646` and the Go paths in
`pkg/session/session_send.go`, `pkg/session/terminal.go`, and
`pkg/telnet/listener.go` (R5c, R5e).

| C prompt path | Go path | Raw-byte proof in this change |
| --- | --- | --- |
| First `CON_PLAYING` pass after entry | `sendWelcome` → `SendPrompt` | `prompt-entry-depth` compares the first trailing prompt frame with `no-settle` |
| Ordinary playing prompt after input | `TerminalLine` → `SendPrompt` | `prompt-playing-depth` covers bare RETURN and AFK transitions |
| Pager prompt and return to playing | `PageString` / `navigatePager` → `SendPrompt` | `prompt-pager-depth` covers entry, refresh, back, invalid input, final page, and quit, with color off; `prompt-pager-color-depth` covers color on |
| Descriptor string editor `] ` | `SendPrompt` editor branch | Existing editor tests cover text, but no raw oracle prompt fixture yet |
| Playing vitals, target, infobar, invisibility | `promptText` | Command oracle coverage exists; raw ANSI and prompt framing are not yet certified |
| Nanny, menu, and reconnect dialogue | Creation/menu handlers | Existing creation and lifecycle coverage is separate; raw prompt proof remains open |

The `lifecycle/descriptor link prompt lifecycle` row in
`docs/fidelity/depth/surface-inventory.tsv` remains blocked. These fixtures
prove the named paths only; they do not close the entire prompt surface.

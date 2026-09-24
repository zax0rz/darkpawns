# Browser play accessibility test plan

The `/play` client uses xterm for the game transcript and a separate dock for
character state. The **Screen reader** button enables xterm's
`screenReaderMode` and remembers the choice in this browser. Automated DOM
checks cannot establish what a screen reader actually speaks. Run the manual
checks below on the built page before treating this surface as accessible.

## Test setup

- Use a disposable test character or `guest`; never put a real password in a
  recording or issue report.
- Test desktop Chrome with NVDA on Windows, and Safari with VoiceOver on macOS.
  Record browser, OS, screen reader version, date, and character type.
- At each viewport (desktop around 1280px, tablet around 800px, mobile around
  375px), test at 100% and 200% browser zoom. Use keyboard only for one pass.
- Keep the browser console open for errors. Record a short excerpt of what was
  spoken for every failure; omit character secrets and private game text.

## Passes and expected results

| Pass | Action | Expected result |
| --- | --- | --- |
| Entry | Open `/play`, use the skip link, Tab to Screen reader, switch it on, then Tab to Terminal input | Toggle announces its on state; visible focus stays clear; terminal input has a name and accepts typing |
| Remembered setting | Reload, then switch the toggle off and reload again | State and spoken on/off match the remembered value; blocked storage still lets the toggle work for this visit |
| Secret input | Enter a character name and password with the mode on | Password characters are neither drawn nor announced as ordinary terminal input; an early paste cannot expose the password before the secret prompt |
| Play | As `guest` or a test character, type `look`, `help`, an invalid command, and RETURN | New game text is announced once, in order; the command prompt is usable; long output can be reviewed without losing the input |
| Busy output | Receive several room or channel lines quickly; start reading older output while another line arrives | Announcements remain intelligible; no surprise focus jump or repeated whole-transcript announcement |
| Dock | Navigate by headings/landmarks to Character, Map, In the Room, Inventory/Equipment, and Target when present | Vitals include numbers, not color alone; current room is named in text; decorative map dots do not flood the reading order |
| Inventory | Switch Inventory and Equipment with keyboard, then receive a state update | Pressed state and visible panel agree; focus stays on the chosen button; hidden items are absent from the reading order |
| Disconnect | Disconnect or interrupt the socket, then use Reconnect | Status announces the loss and return; stale dock data disappears; Reconnect returns keyboard focus to Terminal input |
| Layout | Repeat key actions at 200% zoom and a 375px viewport | No horizontal page scroll, clipped controls, overlapping text, or trapped focus; dock follows the terminal in reading order |
| Visual settings | Check reduced motion, forced colors/high contrast, and color-blind inspection | Controls and current values remain distinguishable without color; focus and text retain usable contrast |

## Recording results

For each run, record `pass / fail / not tested`, environment, date, and a link
to the issue for any failure. A pass requires observing the behavior with the
named screen reader; xterm's accessibility DOM and an automated audit are
supporting evidence only. In particular, test speech around password prompts
and rapid output manually before claiming the toggle is safe and useful.

The browser dock currently shows vitals, a local map, room contents, inventory,
equipment, and the current target when the server supplies those fields.
Mudlet's separate chat window relies on GMCP channel messages; the browser
WebSocket client does not receive that channel feed yet. Chat remains in the
canonical game transcript until a structured channel feed can be wired in
without parsing game prose (R1, R4).

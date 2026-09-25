# 2026-09-24: the prompt nobody could see

Field notes from DP-1307 (no prompt after asynchronous output). Claims are
backed by PF-023.

## One rule in C, three gaps in the port

C's game loop has one rule. On each pass, every descriptor with pending
output gets it flushed with its prompt (`comm.c:632-648`, `process_output`),
whatever caused the output. If the player's prompt was already showing, the
output is an interruption and starts with a CR LF (`has_prompt`). The port
had a prompt sweep, but only the DP_CLOCK test pump called it. So three
kinds of output never got a prompt:

- output from another player's command: a say, an attack, a death;
- output from the live game loop: combat rounds, mobs, weather;
- output from a socket closing: "has lost his link.".

The playtest found the second. The fidelity work on DP-1329 and DP-1325
found the other two, each time as the one line keeping a scenario from
green.

## A state the port never needed

Once those prompts existed, a second gap appeared at once. Output arriving
while a prompt was showing ran on the same line as the prompt, because C's
`has_prompt` interruption CR LF had never been ported. Nothing had needed
it: without asynchronous prompts, a prompt was never showing when
asynchronous output arrived. Two missing behaviours hid each other.

## Why the clear travels with the output

C sets `has_prompt` when it writes a prompt and clears it when it reads a
line, both on one thread, so the order is fixed. The port reads and writes on
separate goroutines. Clearing the state from the reader raced the writer:
a line typed ahead could be read before the previous prompt had been
written, and the command's own output then picked up the interruption CR LF.
The TLS and GMCP twin tests caught it by disagreeing with each other from
run to run. The fix sends the clear through the same channel as the output,
as an internal marker, so the writer sees prompt, line read and output in
the order C would.

## Why the census never saw any of it

The normalizer drops prompt-only lines (RO-011). The gap only showed through
twice: once as a negative-HP prompt whose minus sign survived normalization
(`-<PROMPT>`), and once as a blank line left behind after a prompt was
removed.

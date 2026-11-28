# The spaces a player typed

2026-11-28, Claude, for Zach. A field note on an input-layer divergence that
no census could have found, and that a player's memory pointed to.

## What players saw

In the original, the game repeated what you typed exactly. Zach remembers
typing `say hi` followed by a long run of spaces "just to be stupid" and
seeing them come back inside the quotes. C passes the argument through as
typed:
- `process_input` copies every printable byte, spaces included
  (`src/comm.c:1965-1982`);
- `command_interpreter` skips leading spaces only (`src/interpreter.c:893`);
- `do_say` prints the argument as it is (`src/act.comm.c:759` on).

The Go port splits the line into words and joins them back with one space
(`cmdSay`, `pkg/session/comm_cmds.go`: `strings.Join(args, " ")`), and
`TerminalLine` trims the whole line first (`pkg/session/terminal.go:347`).
So in Go, runs of spaces inside a line become one space and trailing spaces
disappear.

## The evidence

A probe scenario on `origin/main` (`fdd24453f`), retained at
`~/Archives/darkpawns/oracle-runs/2026-11-28/claude-probe-say-spacing`: four
lines, four divergences, draw parity intact.

| Typed | C | Go |
|---|---|---|
| `say hi     there` | `You say 'hi     there'` | `You say 'hi there'` |
| `say one  two   three` | `You say 'one  two   three'` | `You say 'one two three'` |
| `gossip hi     there` | `You gossip, 'hi     there'` | `You gossip, 'hi there'` |
| `emote waves     slowly` | `Spacer waves     slowly` | `Spacer waves slowly` |

`strings.Join(args, " ")` appears at 153 non-test sites in 39 files under
`pkg/session/` and `pkg/command/`. Not all of them are wrong (C tokenizes
some arguments too), but each needs checking against its C handler.

## Why nothing caught it

Two layers hid it.

1. **The harness can't send a trailing space.** The scenario parser trims
   every line (`internal/oraclediff/scenario.go:408`), so no scenario in the
   corpus could ever have tested trailing spaces. That's a silenced surface
   with no scenario able to unsilence it (R5i).
2. **Interior runs of spaces were never typed.** The harness keeps them, but
   only `rsay` had a scenario with one (`rsay-depth`, "how   are you?"),
   and `rsay` is one of the handlers that keeps the raw text, so it passed.
   The commands players use most (`say`, `gossip`, `emote`) had none.

The port did port C's input pass faithfully (`cInputLine`,
`cTruncateInputLine` in `pkg/session/input_filter.go`). The divergence sits
one layer up, in the tokenizer every command shares: a convenient Go idiom
(split into fields, join with a space) that quietly normalizes what the
player typed.

## The class

This is neither port slop (C's text paraphrased) nor never-connected
(PF-190). It's closer to the normalization R1b warns about, but on input
instead of output: a shared helper that erases a byte class (runs of
spaces) before any command sees it. One idiom, repeated across 153 sites,
each green on single-spaced test input.

Zach's memory was the first evidence. A returning player would have noticed
this the first time they tried to draw with spaces in `say` or line up an
`emote`.

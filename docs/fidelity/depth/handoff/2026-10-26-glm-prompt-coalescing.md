# Finding: Go prompts per message, C prompts once per game-loop pass (stop tier)

Date: 2026-10-26 · Found while shaping design #2152 wave 2 (#2171 review,
Claude's seed-1/seed-8 check) · Rules: R1, R3b (operation ordering), R5e, R5g

## The class

C emits **at most one prompt per descriptor per game-loop pass**. Go emits a
prompt immediately after every output batch. When output from two logical
events — an asynchronous act and the player's next command, or the reconnect
handoff and the first command — lands inside one C pass, C coalesces to a
single prompt at the pass end while Go has already prompted in between. The
player-visible byte stream differs (an extra `\r\n` + prompt frame). This is
session/transport behavior: **stop tier; the fix is a separate PR** (design
below, not implemented).

## C's law (src/comm.c, read 2026-10-26)

- `comm.c:613` — receiving a command line clears `d->has_prompt = 0`.
- `comm.c:1630-1643` — `process_output` appends `make_prompt(t)` to whatever
  output it flushes (one prompt rides the write; the flag decides whether the
  prepended CRLF is included).
- `comm.c:645-649` — at the **end of the pass**, after every descriptor's
  input and output are processed, each descriptor still lacking a prompt gets
  exactly one `write_to_descriptor(make_prompt(d))`.

So an asynchronous act (for example `close_socket`'s room act, `comm.c:2131`)
never produces a standalone prompt: whatever prompt the observer eventually
seems comes from the pass boundary. `make_prompt` itself is `comm.c:1028-1207`
(already ported; this finding is about *when* it runs, not what it renders).

## Repro A — link-loss act and the observer's command in one pass

`descriptor-linkloss-world` before its 2026-10-26 determinism fix: the
victim's transport drops between warmup and probe; the observer's `look`
follows immediately. Under `--draw-parity --seed 8` (three consecutive runs;
also the INFRA-then-pass flake Claude saw at seed 1):

```
--- [look [actor]] c-oracle
+++ [look [actor]] go-port
@@ -1,5 +1,6 @@
 Victim has lost his link.
-Temple Infirmary
+
+<PROMPT> Temple Infirmary
```

C delivers the act and the `look` output in one pass with one trailing
prompt; Go frames a prompt right after the asynchronous act. The draw stream
is MATCH (4086 entries) — this is purely a prompt-cadence transcript
divergence. The committed scenario now pumps one pulse between the link loss
and the look, so it proves the act and the persisting body without the race
(10/10 clean on seed 8, seeds 1/2/3/5 re-verified).

## Repro B — first command after a reconnect

From the #2171 menu-vehicle investigation (a diffed `<RELOGIN>` whose section
continued with an empty line after the dupe-check reattach):

```
--- [<RELOGIN>] c-oracle
+++ [<RELOGIN>] go-port
@@ -19,4 +19,5 @@
 Wrong password.
 Password:
 Reconnecting.
-Huh?!?
+
+> Huh?!?
```

C runs the reconnect handoff and answers the empty command inside one pass —
one prompt, and the `Huh?!?` arrives bare. Go has already emitted its
post-reconnect prompt, so the response is prefixed `> `. Same class, different
trigger: any output batch the session emits between the last command's
response and the next command's response is over-prompted relative to C.

## Go side today

`pkg/session/session_send.go` — `SendPrompt`/`sendPromptNow` (around :333-446)
writes a prompt as part of each output batch's completion; there is no
pass-equivalent barrier, and asynchronous world output (acts) goes through
the same immediate-prompt path.

## Fix design (for the separate PR; Zach approves before implementation)

Mirror C's flag-and-boundary cadence:

1. Per session, a `hasPrompt` flag cleared when a command line is accepted
   (C `comm.c:613`).
2. Output flush appends the prompt only when the batch is complete (C
   `comm.c:1635` ported semantics — Go already appends at flush; the bug is
   the extra standalone flush after async acts).
3. A pass-boundary step gives every session that produced output since its
   last prompt exactly one prompt (C `comm.c:645-649`). In Go's concurrent
   session model this is a manager-level tick barrier: after the world turn's
   output drain, sweep sessions once. The sweep point must be defined
   carefully: the oracle's frozen clock runs turns on `~dpclock` pumps, so
   the barrier belongs to the turn executor, not wall time.

Risks the PR must address: the websocket client and the INFOBAR path
(`InfobarOn` already replaces the text prompt — `session_send.go:443+`), the
`keep-prompts`/`no-settle` focused scenarios (R5f/DP-1326 — their expected
frames change only where a race previously existed, which is the point), and
a full census with particular attention to every `compare-close` and
`lifecycle-*` row. R5h: the red control is Repro A's diff itself — before the
fix it reproduces at seed 8, after it the same run is byte-clean.

Scenario authors must keep the pump discipline regardless: separate an
async-act block from the next command with `~dpclock pulse 1` when the
scenario's subject is the act, not the cadence.

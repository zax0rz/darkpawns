# DP-1371 D4 visibility/output train

Stop tier: common session/terminal output changes. R1/R5e/R5g/R5h.

## 1. Sleeping visibility

`src/utils.h:515-549` defines character/object visibility and PERS without AWAKE. Remove the extra awake guards from both Go visibility helpers. `SENDOK` retains its separate awake/TO_SLEEP and writing gates (`src/comm.c:2480-2555`). The sleeping-recipient row now retains its existing five-seed gsay-depth proof as oracle-green-multiseed. The old unit expecting Someone for a visible speaker with TO_SLEEP is corrected to Hero.

Other readers audited: Act names/object substitutions and hide-invisible delivery; character/object targeting; mobile AI (its explicit awake/position gates remain in mobact.go:163,194); item transfer, movement doors, skills and NPC lookup; session who/diagnose/assist/group/wizard/shop consumers of exported CanSee helpers. No new lock acquisition; removing the guards removes those position reads. Existing light, hide, holylight and wizinvis policy is otherwise unchanged; this proves the asleep boundary, not the complete visibility policy. Further policy questions require their own C-first trace.

## 2. Complete-color vitals prerequisite

Raw main output exposed plain peer vitals where `src/comm.c:1083-1121` emits green at >=75%, yellow at >=33%, red below, followed by reset before H/M/V. This is part of the make_prompt bytes reached by the D4 flush vehicle, so it is ported rather than masked in that vehicle. A new unit-green prompt.vitals-complete-color row accounts for it. The helper uses C float percentages, including IEEE division at zero maxima; only complete color enables these fields.

Other readers: all shared prompts for telnet/browser terminals, field display masks, infobar exclusion and wizinvis prefixes. GMCP values and transport envelopes are unchanged. Player flags and each current/maximum getter take and release their existing player read lock separately; no lock is held across another getter or send. This proves formatting for the provided player; active-switch body selection stays in the retained identity-consumer frontier.

## 3. Act flush framing

The existing async prompt sweep already sends listener prompts, but raw proof was absent. `src/act.comm.c:846-866` emits a complete act line then reset; `src/comm.c:1620-1643` adds one noncompact flush CRLF then prompt. Go's event framer appended another CRLF after reset. It now recognizes a line ending before trailing CSI sequences while preserving all original control bytes. No text normalization, command-only prompt patch or gsay-only exception is added.

`gsay-raw-flush-depth` compares exact actor/awake/sleeping peer bytes with complete color, visible/invisible speaker, gtell and norepeat. Both original blocked prompt rows become oracle-green at seed 1.

Other readers: the shared terminal event framer used by telnet and browser output, including acts, logs and multiline messages ending in color controls. Raw events still bypass framing; text and prompt frame kinds retain their existing paths. The helper is pure and adds no lock. Existing session output queue, prompt FIFO and async sweep are preserved.

## 4. Syslog consumer

`src/utils.c:212-238` applies the file flag independently of descriptor broadcast; writing, player level and two-bit log type filter recipients. It sends green, the CRLF-terminated bracket message, then reset. Go placed reset before CRLF; its order is repaired.

The consumer row becomes unit-green through the complete provided-Player type/log-level/minimum-level/writing/color/file-flag matrix. `syslog-consumer-raw` also proves a real colored quit producer at seed 1. File-flag evidence covers whether the line reaches the configured writer, not timestamp format.

Other readers: every game MudLog producer, player message sinks, shared terminal framing, manager session provider and server log writer. No new locking: provider/flag/level reads use their existing locks, released before delivery. Active-switch descriptor/body identity in the session provider remains the retained identity-consumer frontier; this consumer matrix does not certify that selection or NPC flag ownership.

## Proofs and census

Evidence: `~/Archives/darkpawns/oracle-runs/2026-10-03/dp-1371-visibility-proofs/`.

Main's gsay-depth and gsay-act-depth reproduce their existing EXPECTED divergences. Raw main capture independently exposes the sleeping-name, extra-line and missing-vitals-color differences. Every repaired unit has assertion revert/restoration checks: character guard, object guard, ANSI frame, vitals colors and both exact threshold comparisons, syslog reset order (0/1/0). Both raw vehicles pass, fail against original main-source server by content (exit 3), and pass restored (0/3/0). Their full C blocks are retained.

The three resolved gsay ledger associations and six obsolete fingerprints are removed, with the retired pins retained in evidence. No fingerprint is regenerated, no seed is dropped, and existing scenario files are unchanged. Normal gates and the one combined census at the final tip are reported in the PR.

Next train: D5 room index 3 and queued-close weather, then D6 wizlock/jail and entry.login-restrictions. Retain the browser-name routing/identity consumers, loaded-inventory ownership and legal-quit cleanup frontiers, and the separate multiseed teleport combat gap.

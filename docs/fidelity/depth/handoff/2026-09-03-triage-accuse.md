# Depth-fidelity triage handoff — `accuse` no-argument red

Date: 2026-09-03

Branch: `glm/depth-accuse-noarg`

Base: `origin/main` at `6a2d7ff9c` (the whirlpool merge)

## Verdict

`accuse-noarg-depth` remains blocked. No Go behavior change is confirmed or
made.

The required two isolated attempts were run against the current main baseline:

```text
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  /usr/local/go/bin/go run ./cmd/dp-oracle-diff \
  --scenario accuse-noarg-depth --show-oracle --seed 1
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  /usr/local/go/bin/go run ./cmd/dp-oracle-diff \
  --scenario accuse-noarg-depth --show-oracle --seed 2
```

Both runs reported a normalized divergence. The Go actor block was the
source-derived `Accuse who??`; the C actor block contained non-text,
pointer-like bytes rather than that literal. The malformed C output differed
between the two seeds but had the same shape. This is the same isolated red
recorded by `docs/fidelity/depth/accuse.tsv`, not a newly introduced failure.

## Call-path audit

The registered C command is `src/interpreter.c:330`:

```c
{ "accuse"   , POS_SITTING , do_action   , 0, 0 },
```

`src/act.social.c:102-121` resolves the social, applies the no-shout gate,
parses no target, and sends `action->char_no_arg` followed by `\r\n` to the
actor. The `accuse` record is the `lib/misc/socials` entry at lines 58-66.

The source path therefore says the expected actor bytes are the literal
`Accuse who??`. The live C oracle instead emits malformed bytes. There is no
confirmed Go-side divergence to fix; reproducing the oracle anomaly in Go
would violate R1 and R4, and the unresolved runtime/source disagreement fails
R5e's call-path proof boundary.

## Checks

`make fidelity-depth` passed on the base before this triage, reporting 4,111
cases: 4,013 proven/delegated, 45 blocked, and 53 excluded. The four
accuse rows affected by the malformed actor block remain blocked; the existing
self-target, not-found, and sleeping-target proofs remain green.

This triage follows R1/R4/R5e and advances after the two honest attempts as
required by the depth-loop objective.

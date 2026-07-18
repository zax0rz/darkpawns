# C-oracle determinism seam

This directory preserves the Dark Pawns C oracle's test-only determinism seam
outside the oracle clone. The patch was exported from oracle branch
`dp-oracle-seam`, commit `055437f`, against pristine base commit `d2cb13e`.

The seam adds three environment-gated behaviors:

- `DP_SEED=<uint32>` seeds the CMWC stream; when unset, the oracle still seeds
  from `time(0)`.
- Presence of `DP_CLOCK` freezes wall-clock heartbeats.
- With `DP_CLOCK` present, `~dpclock pulse <n>` advances the real C heartbeat
  exactly `n` times before normal command queuing and interpretation.

The control line accepts 1 through 100,000 pulses. It does not enter command
history or the input queue, alter wait-state, expand aliases, or pass through
`command_interpreter`; only the heartbeats it requests may draw or mutate game
state.

## Restore with Git

From a clean C-oracle checkout at `d2cb13e`:

```bash
git apply /path/to/darkpawns/tools/oracle-seam/dp-determinism.patch
cd src
make
```

The build installs the rebuilt executable at `../bin/circle`.

## Restore with patch

From the C-oracle repository root at `d2cb13e`:

```bash
patch -p1 < /path/to/darkpawns/tools/oracle-seam/dp-determinism.patch
cd src
make
```

Apply the patch only to the documented pristine base. To verify without
changing a checkout:

```bash
git apply --check /path/to/darkpawns/tools/oracle-seam/dp-determinism.patch
```

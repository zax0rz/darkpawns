# Server efficiency measurement

## Procedure (repeat exactly for the after measurement)

Run from the repository root of the selected revision on Linux/amd64:

```sh
go build -o docs/research/idiom-experiment/run/server-after ./cmd/server
python3 docs/research/idiom-experiment/run/measure_server.py docs/research/idiom-experiment/run/server-after after > docs/research/idiom-experiment/run/efficiency-after.tsv
```

For the baseline, substitute `server-baseline` and `baseline` in both command
arguments. `measure_server.py` starts five fresh processes from the built
binary, each under `/usr/bin/time -v`. It symlinks the checkout's
`lib/world` into a new scratch directory under `run/efficiency-<label>/`, and
uses a fresh SQLite database and isolated working directory for each trial.
The process receives a fixed `JWT_SECRET` and `DP_SEED=1`; `DP_CLOCK` and
`DP_FIXED_TIME` are unset. Each trial chooses an available loopback port from
`bind(127.0.0.1, 0)` (excluding 4000, 7777, 7778, and 7779), then starts HTTP
on that port with `-http-bind 127.0.0.1 -telnet-port 0 -telnet-tls-port 0
-whod-port 0`. Readiness is the first successful TCP connection to that
listener, timed from process launch with `time.monotonic_ns()`. The script
sends SIGTERM only to the exact server child PID it started, waits for the
`/usr/bin/time` wrapper, and reads peak RSS from its `Maximum resident set size`
field. Raw timing output and server logs remain in the corresponding
`efficiency-<label>/trial-NN/` directories. The TSV records each trial and the
median wall time and RSS.

## Baseline

Base SHA: `dce50f5bf398ec858a5f3bf92e8b4b61b2664357`.

Binary: **40,280,775 bytes** (`go build -o docs/research/idiom-experiment/run/server-baseline ./cmd/server`).

| Trial | Loopback port | Time to TCP accept (ms) | Peak RSS (bytes) |
|---:|---:|---:|---:|
| 1 | 58029 | 265.164 | 92,397,568 |
| 2 | 39421 | 187.322 | 88,903,680 |
| 3 | 45273 | 229.301 | 87,560,192 |
| 4 | 43729 | 198.539 | 90,750,976 |
| 5 | 45417 | 240.552 | 88,403,968 |
| **Median** | — | **229.301** | **88,903,680** |

Raw output: `efficiency-baseline.tsv`; `/usr/bin/time -v` records and server
logs: `efficiency-baseline/trial-*/`.

## After

Branch tip after the seven targeted refactors: `92eadcb96`.

Binary: **40,270,383 bytes** (−10,392 bytes, −0.026% from baseline).

| Trial | Loopback port | Time to TCP accept (ms) | Peak RSS (bytes) |
|---:|---:|---:|---:|
| 1 | 46823 | 154.005 | 91,004,928 |
| 2 | 53187 | 151.231 | 89,038,848 |
| 3 | 53327 | 161.900 | 87,523,328 |
| 4 | 45993 | 181.836 | 90,845,184 |
| 5 | 36593 | 156.461 | 88,834,048 |
| **Median** | — | **156.461** | **89,038,848** |

Compared with baseline, measured readiness was 31.8% faster and peak RSS was
0.15% higher. The host's memory pressure changed between measurements: the
census queue had observed 2,736 MiB available before Zach cleared accumulated
memory use. Treat the readiness difference as host-load-confounded rather than
evidence of a code speedup. RSS remains within the predicted ±5% band.

Raw output: `efficiency-after.tsv`; `/usr/bin/time -v` records and server
logs: `efficiency-after/trial-*/`.

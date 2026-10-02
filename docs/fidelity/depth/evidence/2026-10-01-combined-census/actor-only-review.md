# Actor-only first-response allowance: review follow-up

Fixed source commit: `a9a6ea47c2fc3d148dca137b1229a8428b20ba5a`. The real zreset CPU contention triple and
combined 36-worker run used this same clean commit and the reference oracle.
The combined schedule used the same frozen estimates as the previous 36-worker run.

The two-second allowance is armed by successful Send and consumed by that
connection's next read. RunAudienceProbe captures the issuing target first, then
reads passive peers concurrently, at plain 300ms quiescence. send:peer makes
that named peer the issuer and the primary its passive audience. Setup/pulse
commands receive the allowance on their own issuing connection; initial greeting
and unrelated drain reads keep 300ms. Peers are never read alongside the issuing
capture in RunAudienceProbe. Main's separate drainClients drains the primary
first and peers concurrently afterwards, without issuing commands.

Slow-command peer proof: 650ms CPU work delays the response and observer output.
Both ordinary actor and send:peer routes capture the observer's output after the
issuer returns. Moving peer reads before issuer capture fails an assertion; the
restored order passes (0/1/0). Passive-peer tests also verify the short silent wait.
The real zreset fixture rerun passes with the actor allowance, fails with the
legacy window using the original fingerprint on both attempts, then passes again.
All required Go, lint and fidelity gates pass.

| Configuration | Time |
|---|---:|
| Original main two-run baseline, 300ms-only capture | 37m 8.1s |
| Prior blanket first-byte allowance, separate runs | 55m 59.0s |
| Prior blanket first-byte allowance, combined 36 | 40m 18.6s |
| Actor-only allowance, combined 36 | 30m 22.3s |

This revision saves 596.260s (24.7%) against
blanket allowance at 36 workers, and 405.729s
(18.2%) against the original two-run baseline.
Prior separate-run summary times omit full-census infrastructure recheck wall time;
the combined elapsed includes its infrastructure rechecks. These are single
measurements, not a statistical speed estimate.

`oracle-combined: pairs=3132 deduplicated=946 full=CLEAN_AFTER_RECHECK claims=CLEAN_AFTER_RECHECK elapsed=1822.332s verdict=CLEAN_AFTER_RECHECK rechecked=retsu-failure-depth`

Both projections are clean. Every final classification matches the retained
baseline: all 1,058 full rows and all 3,020 claimed pairs, at their exact seeds.
No coverage, seed, expected-divergence pin or verdict rule changed. The union
still executes 3,132 distinct pairs, deduplicating 946 seed-1 reruns.
Evidence: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-02/dp-1371-actor-only-review` and the run paths in actor-only-review-results.json.

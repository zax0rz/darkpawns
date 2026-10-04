# DP-1371 D5 heartbeat output transaction (Train B)

Stop tier: session output, descriptor lifecycle, telnet/WebSocket delivery and persistence boundaries. Implements approved #1770 items 2–5 after Train A (#1772) and the Login@ harness prerequisite (#1773). Zach's live playtest remains the final merge gate.

## Invariant and commit order

Outside an active heartbeat turn or pumped burst, each send path executes its existing delivery branch. The only persistent suppression flag belongs to a descriptor already closed by the new immediate-idle path; it prevents a retained stale reference from sending on that closed channel. No ordinary movement or empty game-loop tick acquires the lifecycle lock. Ordinary command scheduling is unchanged.

The commits introduce the helper without production callers; route manager/text sinks and broadcasts; defer prompts; route raw/input/observation frames; route GMCP/agent vars; install live/pumped boundaries; move idle close to C's site; then complete snoop and orderly-close barriers. Each prerequisite has its own assertion-failing revert control. Existing expected divergences, pins and scenario seeds are unchanged.

Live staging begins after the existing initial input drain and ends after heartbeat callbacks. PumpPulses wraps its entire burst: each pulse still performs the existing input drain, but its output and prompt requests remain staged until the last pulse. The deferred end runs despite a recovered callback panic. End retains active state while committing: a concurrent enqueue joins the FIFO, and ordinary delivery resumes only when frames and prompt requests are empty under the batch mutex. Prompt state reads and rendering happen outside that mutex.

## C authority and idle ordering

Read from src/comm.c:636–679 (output/prompt/close pass precedes heartbeat), 825–830 (weather, affect, point order), 1624–1651 (flush framing/snoop), 2092–2148 (close, queues, snoop, save, act, descriptor removal); src/limits.c:438–451 (transfer, close, rent, extraction); src/weather.c:58–80 (sunset and darkness).

CheckIdling invokes the world-to-session callback after the RNUM-3 transfer, with world/player locks released. The manager closes only the descriptor attached to that concrete body. Immediate discard suppresses staged frames and prompt bookkeeping before the socket closes. It releases snoop links and notifies the snooper, cleans editors, saves, emits one lost-link act/log, then detaches any active PC switch. Lifecycle ownership ends before the existing rent-object pass and deferred extraction.

Transport teardown recognizes that idle close already ran, retaining the lifecycle owner until extraction and avoiding a second close/save/act. Extraction does not repeat the socket close. The close save and subsequent filtered rent save are separate C operations: preserving the latter keeps NORENT items from returning on login. Existing SQLite inventory/equipment round-trip tests verify that boundary.

An idling original has no attached descriptor and cannot close the borrowed body's descriptor. An idling borrowed body closes the acting descriptor and leaves both identities linkdead, without an explicit return message. Existing reconnect/extraction tests exercise both identities.

## Delivery and other readers

Staged frames copy the complete immutable transport envelope. Capacity accounts for queued plus staged frames and retains the existing nonblocking drop policy. Text bookkeeping runs only for delivered frames; raw controls do not create another prompt. Raw events, immutable input markers, observation state, GMCP, vars and token-refresh frames share the boundary without changing their envelopes.

Commit aggregates each surviving descriptor's text for C's percent-delimited snoop forwarding. Discarded text is never forwarded. Prompt requests are coalesced after surviving text and GMCP synchronization. An orderly CloseSend adds a FIFO close barrier, rejects later staged frames and lets the existing transport writer drain accepted output before closing. Immediate idle discard remains separate. FlushQueues cancels active staged output and stops when its channel is closed.

Production producers audited: World.MessageSink/MobileMessageSink, combat text/raw sinks and broadcasts, BroadcastToRoom, SendToAll/SendToOutdoor, SendMessage/sendText/sendGuarded, prompt queues, raw events, input markers, observation renderer, GMCP, dirty/full vars, token refresh, password timeout and registry takeover notice. The pre-existing ordinary branches remain in place. No harness, oracle, save-format, browser name-routing or identity-registry redesign is part of this train.

## Lock acquisitions

- Begin/stage/prompt deferral/close-barrier/active-state checks: outputBatch.mu only. Existing producers may enter while holding manager, weather, world or GMCP locks. The batch-locked path never acquires any of those in return; rendering there is pure JSON/terminal-envelope inspection.
- Commit: pop under outputBatch.mu, release; send under sendMu.RLock (atomic text bookkeeping only), release. Snoop lookup takes snoopMu.RLock and releases before forwarding, which re-enters staging without recursive delivery. Prompt sweep snapshots under manager.mu.RLock and releases before player/editor/GMCP reads and enqueue. Final empty-to-inactive transition takes outputBatch.mu.
- FIFO close commit: no batch lock held; entry-name release takes entryNameMu, releases; sendMu.Lock closes the channel. No writer acknowledgement is awaited under a lock. Close's orderly-drain query takes/releases outputBatch.mu before its existing transport close path.
- Immediate idle callback: playerLifecycleMu; attached-body lookup takes manager.mu.RLock then the existing leaf sendMu read check, and releases both. Discard takes/releases sendMu.Lock, then close's entryNameMu and sendMu.Lock separately. Socket close and transport detach run with no output-batch lock. Snoop cleanup takes/releases snoopMu before its notification enqueue. Existing editor cleanup, character snapshot/store save, act/mudlog and switch detach retain their existing lock order under lifecycle ownership. Switch detach takes/releases manager.mu before locking either character. The callback releases playerLifecycleMu before rent/extraction.
- FlushQueues through UnregisterAndClose: existing lifecycle lock; manager lookup/removal lock released first; batch-state check releases its mutex before discard/sendMu. Ordinary movement and send paths take no lifecycle lock.

## Proofs and evidence

Evidence root: ~/Archives/darkpawns/oracle-runs/2026-10-03/dp-1371-heartbeat-output-proofs/. Assertion-failing helper, manager/text, prompt, raw/observation, GMCP/vars, per-pulse boundary, idle-close and FIFO-close controls are retained there. Restored focused tests pass. The concurrent enqueue/commit/close tests and existing switch paths pass under -race; real telnet writeLoop and WebSocket writePump tests verify discard and orderly goodbye behavior.

The original outdoor 31-hour vehicle and a 25-hour burst companion pass at seed 1. The retained final-topology captures put the driver outdoors in room 8001: at pulse 19530 the driver receives `The suns slowly disappear in the west and south.`, and the actor receives only `<CLOSE>`. The earlier development census was 2/2 CLEAN (99.630 s). The burst contains earlier sunrise output as well as the terminal sunset, so committing per pulse can fail by leaking real weather rather than by a timer assertion. Normal build, vet, full tests, clean-cache lint, formatting, diff check, fidelity-depth, fidelity-units (1322/1322) and the string-census ratchet pass. The focused enqueue/commit/close, real-transport and existing switch paths pass under `-race`. The manifest claims both vehicles at seeds 1,2,3,5,8 for the final combined gate; its verdict and CI will be recorded before review readiness.

## Local playtest checklist (pending Zach)

Use the PR's local test server, with disposable player data. Check ordinary command prompts and output in telnet and the browser; same-room asynchronous weather/combat output; snoop output and target idle disconnect; legal quit/goodbye drain; switch/return and reconnect to both identities. The automated terminal-hour proof covers the long idle wait; the live check need not wait 31 hours. A short check is `look`, `score`, `say test` and `quit` in both clients, then wizard `switch`/`return` and the two reconnect identities used in the previous playtest. Do not deploy before the playtest and Claude's send-path review.

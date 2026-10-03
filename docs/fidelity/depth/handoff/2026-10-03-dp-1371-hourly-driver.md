# DP-1371 D5 Train A: one hourly point-update driver

Stop tier, explicitly approved in the goal's D5 sequenced decision. This train only unifies the driver; #1770 Train B owns output staging and synchronous idle close. The terminal-weather row remains blocked with that reference.

## Change and C call path

C `src/comm.c:825-830` calls weather_and_time, affect_update, point_update, hunt_items and player-file flush in that order, once every SECS_PER_MUD_HOUR * PASSES_PER_SEC. `src/utils.h:135` sets the hour to 63 seconds: 630 heartbeat pulses.

Go already dispatches that hourly block in the engine. The server formerly gated its point callback on DP_CLOCK, while NewWorld started an independent 63-second ticker for live play. That made live point updates independent of weather/affect order. The server now binds World.PointUpdate directly in both modes; the standalone startup and ticker method are removed. The engine cadence/order itself is unchanged. R1/R3b/R5e/R5g/R5h.

## Proofs

TestHourlyPointUpdateSingleDriverModes runs the actual live engine runner with a shortened test ticker through two hourly boundaries, and separately pumps through the near side and exact side of each boundary. Both assert exactly weather/affect/point/hunt/file once per hour. A duplicate point-dispatch mutation fails both modes on count/order assertions; restoring the source passes (0/1/0).

TestPointUpdateDriverWiring checks the actual server callback binding and production World source. This supplements the engine test: fake callbacks cannot detect the old production Frozen gate or the second World ticker. The test fails on main before repair. Restoring the Frozen gate independently fails, restoring the standalone ticker independently fails, and each restoration passes (0/1/0). These are structural wiring assertions, not live-world timing measurements.

The obsolete nil-callback 'single driver' test and literal-constant self-check are replaced by the real engine and wiring proofs. Direct PointUpdate empty-world tests remain. The DP_CLOCK scenario corpus exercises the same function and is validated by one combined census at the final tip; the live runner and wiring tests cover the changed live boundary the oracle cannot see.

## Other readers and locks

PointUpdate consumers include HMV regeneration, hunger/thirst/drunk conditions, poison/bleeding, affect-sensitive regeneration, jail timers, idling/rent/extraction, mobile state and object/corpse decay. They keep the same implementation and cadence; live scheduling now follows weather and affect_update on the heartbeat instead of a separately phased goroutine. GMCP's PointUpdated notification stays at the function's existing deferred boundary. Admin/persistence/session consumers retain their existing paths. The explicit wizard tick in pkg/session/wiz_system.go remains a manual command call, not an automatic driver.

No lock is introduced. The standalone goroutine is removed; the heartbeat calls PointUpdate without holding world/player/session/lifecycle locks. PointUpdate retains its existing snapshot and per-player/object locking; persistence callbacks, deferred extraction, output writers and GMCP are unchanged. Combat/event drivers are unchanged. No output queue transaction is present.

Evidence: `~/Archives/darkpawns/oracle-runs/2026-10-03/dp-1371-hourly-driver-proofs/`. Normal gates and final combined verdicts are recorded in the PR. After this train merges, implement Train B in the goal's approved commit order and outside-active-turn invariant.

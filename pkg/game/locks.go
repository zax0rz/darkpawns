// Package game — Lock Ordering Protocol
//
// This file documents the lock acquisition hierarchy for all mutexes in pkg/game/.
// 394 lock acquisitions across 57 functions depend on this ordering being correct.
// VIOLATING THIS ORDERING WILL CAUSE DEADLOCKS.
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │                      LOCK ACQUISITION ORDER                        │
// │                                                                     │
// │  Acquire locks from top to bottom only. Never hold a lower-numbered │
// │  lock while acquiring a higher-numbered one.                        │
// │                                                                     │
// │  1. World.mu           — top-level game state (rooms, mobs, objs)   │
// │  2. World.gossipMu     — gossip channel history                     │
// │  3. World.weatherMu    — weather state                             │
// │  4. World.mailWriteMu  — mail persistence                          │
// │  5. Clan.mu            — clan membership, ranks                    │
// │  6. Player.mu          — player stats, gold, exp, position         │
// │  7. Equipment.mu       — equipped item slots                       │
// │  8. Inventory.mu       — carried item list                         │
// │  9. MobInstance.mu     — mob state, HP, position                   │
// │  10. Spawner.mu        — zone reset scheduling                     │
// │  11. BoardState.mu     — bulletin board messages                   │
// │  12. Shop.mu           — shop inventory, pricing                   │
// │  13. ZoneDispatcher.mu — zone command routing                      │
// │  14. logWriterMu       — log file writes (independent)             │
// └─────────────────────────────────────────────────────────────────────┘
//
// Rules:
//   - Same-level locks (e.g., multiple Player.mu on different players):
//     always acquire in a consistent order (by Name/ID) to prevent ABBA deadlocks.
//   - Never upgrade RLock → Lock without releasing first.
//   - World.mu is always outermost. Never call World methods that acquire World.mu
//     while holding Player.mu, MobInstance.mu, or Clan.mu.
//   - Prefer defer Unlock() for simple critical sections. Use explicit Unlock()
//     for multi-lock or conditional unlock patterns.
//
// Observed nested patterns (verified safe):
//   World.mu → MobInstance.mu            (save.go — deserialization)
//   Player.mu → Equipment.mu             (death.go — death cleanup)
//   Clan.mu → Player.mu                  (item_transfer.go — gold transfer)
//   World.mu.RLock → Player/Mob.mu       (party.go — group handling)
//
// Audited: 2026-05-07 by BRENDA69. No violations found.

package game

// This file intentionally contains only documentation.
// The lock ordering above applies to all files in this package.

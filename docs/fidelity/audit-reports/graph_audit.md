# Port Fidelity Audit: Module 28 (`graph.c`)

This audit examines the port fidelity between the legacy C source file `src/graph.c` (BFS pathfinding, player `track` skill, and mob hunting/trash-talk routines) and its Go implementations in `pkg/game/graph.go` and `pkg/dreaming/graph.go`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/graph.c` (406 lines)
- **Core Functions**:
  - `bfs_enqueue`, `bfs_dequeue`, `bfs_clear_queue` (in-game queue structure tracking rooms and directions)
  - `find_first_step` (performs classic Breadth-First Search (BFS) to resolve the first direction on the shortest path between two rooms)
  - `do_track` (ACMD skill allowing players to sense trails toward visible characters)
  - `hunt_victim` (drives aggressive mob pathfinding, including opening closed doors, evasion checks, safe-room overrides, and trash-talk broadcasting)

### Go Port Files
- `pkg/game/graph.go` (implements the BFS queue, `findFirstStep()`, `validEdge()`, player `doTrack()`, and mob `huntVictim()` state machine)
- `pkg/dreaming/graph.go` (extends graph algorithms to implement a cognitive narrative memory graph consolidated on player dream ticks, tracking relationships and emotional valence)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Severe Track Skill Bug (Mobs Cannot Be Tracked Across Rooms)
- **Source Context**: `pkg/game/graph.go#L156-L193` (`doTrack`)
- **Fidelity Bug**: In legacy C, the `track` skill resolves targets using `get_char_vis()`, which scans all players and active NPCs (mobs) globally. 
  In the Go port, `w.getCharVis()` is hardcoded to *only* search the global active player slice, completely ignoring NPCs.
  To compensate, the Go port added a local loop:
  ```go
  for _, mob := range w.GetMobsInRoom(ch.GetRoom()) { ... }
  ```
  This loop only searches mobs **already standing in the player's current room**. If a player attempts to track a mob (NPC) located in another room, `getCharVis` returns `nil` and the track command fails.
- **Impact**: The `track` skill is **entirely broken for tracking mobs**! Since players already see mobs standing in their current room, the ability to trace trails to target mobs across zones is non-functional, which is a major deficit for warrior, paladin, and ranger gameplay.

### 2. Swapped Mob Communication Scopes (Swapped Gossip & Shout Channels)
- **Source Context**: `pkg/game/graph.go#L389-L414` (`mobGossip`, `mobAuction`, `mobShout`)
- **Fidelity Bug**: In the original MUD, gossip and auction channels were global channels broadcast to everyone on the server, while shout was restricted to the current and adjacent zones.
  In Go, the mob communication scopes are exactly swapped:
  - `mobGossip` and `mobAuction` send messages **only to the players in the mob's current room**.
  - `mobShout` broadcasts messages **globally to all players**.
- **Impact**: The immersive atmosphere is severely diminished. Players across the world will never see the mob bragging on Gossip about their upcoming kill, while shouts (meant for local adjacent areas) become global spam.

---

## 3. Go Improvements Over C

### 1. Thread-Safe BFS Marks (No Global Bit Manipulation)
- **Fidelity Improvement**: Legacy C set and cleared the `ROOM_BFS_MARK` bitvector directly on the global `world` room struct. This was highly prone to race conditions if multiple characters tracked simultaneously. Go isolates BFS searches using a clean, per-call local hash map `marks := make(map[int]bool)`, making BFS pathfinding completely thread-safe and safe for parallel executions.

### 2. Cognitive Memory consolidation Graph (`pkg/dreaming/graph.go`)
- **Fidelity Improvement**: Go beautifully bridges BFS algorithms with a high-level cognitive memory graph for AI agents. It models entities, rooms, events, and emotional valence. The system consolidated memories on dream ticks, decays salience over time, and generates summaries in natural language suitable for LLM injection. This is a magnificent modernization of pathfinding ideas.

---

## 4. Summary of Recommended Fixes / Enhancements

1. **Add Global Mob Searching in `doTrack`**:
   Refactor `doTrack` to search the world's active mobs when the player targets a non-player name:
   ```go
   // Instead of room-only mobs, search the global active mobs index
   for _, mob := range w.activeMobs {
       if strings.EqualFold(mob.GetName(), argument) {
           dir := w.findFirstStep(ch.GetRoom(), mob.GetRoom())
           // Resolve and print trail!
       }
   }
   ```
2. **Correct Mob Communication Scopes**:
   Fix channel scope mismatches in `graph.go`: route `mobGossip` and `mobAuction` globally (to all active players), and scope `mobShout` to the mob's room and adjacent zones.

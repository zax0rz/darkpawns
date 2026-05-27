# Port Fidelity Audit: Module 24 (`events.c`)

This audit examines the port fidelity between the legacy C source file `src/events.c` (timer-delayed event scheduling system) and its Go implementations in `pkg/events/queue.go`, `pkg/events/bus.go`, and `pkg/events/types.go`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/events.c` (126 lines)
- **Core Functions**:
  - `event_init` (initializes global event queue pointer)
  - `event_create` (allocates and inserts an event scheduled at `pulse + delay`)
  - `event_cancel` (removes and frees an event and its custom `void *event_obj`)
  - `event_process` (checks current pulse, triggers expired events, and re-schedules if retval > 0)
  - `event_time` (returns remaining pulses before event execution)
  - `event_free_all` (cancels and frees all scheduled events in the queue)

### Go Port Files
- `pkg/events/queue.go` (main thread-safe `EventQueue` implementing priority ordering using `container/heap`)
- `pkg/events/bus.go` (publish/subscribe in-memory `InProcessBus` for loose system decoupling)
- `pkg/events/types.go` (structured MUD events like `MobKilledEvent`, `PlayerLeveledEvent`, and `GoldEarnedEvent`)

---

## 2. Critical Logic Gaps & Severe Bugs

There are **no severe execution or logic bugs** in the `pkg/events/` port. The Go implementation is exceptionally solid and highly faithful. However, there are minor structural and performance discrepancies:

### 1. Deferred Heap Cancellation (Gargabe Collection Delay)
- **Source Context**: `pkg/events/queue.go#L127-L142` (`Cancel()`)
- **Fidelity Discrepancy**: In legacy C, calling `event_cancel` immediately removes the event node from the bucketed linked-list queue and reclaims its memory with `FREE(event)`.
  In Go, `Cancel` only flags `Cancelled = true`. The event is left residing in the heap, taking up memory, until its original execution time (`When`) is reached. At that point, `Process()` pops the node and skips it.
- **Impact**: While completely memory-safe due to Go's automatic garbage collection, a large volume of cancelled events (e.g. repeated spell casting cancellations, player logins/logouts) can cause the priority queue heap to grow unnecessarily large, increasing time-complexity for insertion (`O(log N)`).

### 2. Linear Heap Scan on Cancel
- **Source Context**: `pkg/events/queue.go#L136-L141`, `pkg/events/queue.go#L151-L156` (`CancelBySource`)
- **Fidelity Discrepancy**: To cancel an event by ID or Source, the Go queue performs a linear search `for _, evt := range eq.events`.
- **Impact**: For highly active queues, a linear sweep (`O(N)`) over all scheduled items for every cancellation can lead to CPU bottlenecking. Under heavy load (e.g., area-of-effect spells hitting dozens of targets with delayed events), this linear scanning is less efficient than C's direct node-pointer dequeue.

---

## 3. Go Improvements Over C

### 1. Type Safety over Unsafe Void Pointers
- **Fidelity Improvement**: In legacy C, the event queue carried a generic `void *event_obj` pointer, requiring unsafe casting inside callbacks and risking runtime crashes or segment faults. Go models events with explicit structural fields:
  ```go
  type Event struct {
      ID        uint64
      Source    int
      Target    int
      Obj       int
      Argument  int
      Trigger   string
      EventType int
      When      int64
      Func      EventFunc
  }
  ```
  This cleanly decouples scripts, players, and mobs, allowing the event system to naturally bridge with Lua scripts and MobPrograms.

### 2. Thread Safety and Concurrency
- **Fidelity Improvement**: C events executed on a single-threaded heart-beat tick. Go introduces a thread-safe mutex structure (`sync.Mutex`) allowing safe event insertion and cancellation from concurrent goroutines (e.g., Telnet input pumps or background AI agents).

### 3. Pub/Sub Decoupling Layer (`InProcessBus`)
- **Fidelity Improvement**: Go implements a modern publish-subscribe `Bus` interface entirely absent in the original MUD. This allows external metrics engines, moderation logs, and narrative memories to consume system state changes without coupling directly to core game loops.

---

## 4. Summary of Recommended Fixes / Enhancements

1. **Optimize Cancellation with a Lookup Map**:
   If profiling shows CPU spikes in event cancellations under heavy combat, add a lookup map tracking active events:
   ```go
   type EventQueue struct {
       ...
       active map[uint64]*Event // O(1) event cancellation index
   }
   ```

# Port Fidelity Audit: Module 43 (`queue.c`)

This audit examines the port fidelity between the legacy C source file `src/queue.c` and its Go counterparts in `pkg/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/queue.c` (175 lines)
- **Functions & Features**:
  - **Bucketed Circular Calendar Queue (Timing Wheel)**: Implements 10 parallel doubly-linked lists (`NUM_EVENT_QUEUES = 10`) sorted internally by execution pulse key (`key % NUM_EVENT_QUEUES`).
  - **Low Sorting Overhead**: By distributing events across circular buckets based on their execution pulse modulo, the MUD keeps list sizes small and reduces sorted insertion costs.
  - **O(1) Cancellation**: The MUD cancels/removes events via `queue_deq` using direct element pointers (`struct q_element *`), instantly unlinking the node in constant time.

### Go Port Files
- **Go Implementation**:
  - [pkg/events/queue.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/events/queue.go): Replaces the bucketed calendar queue with a central **Min-Heap** (`eventHeap []*Event`) implementing the `container/heap` interface.
  - [pkg/events/queue_test.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/events/queue_test.go): Comprehensive unit tests verifying event creation, cancellation, tick increments, and re-enqueueing.

---

## 2. High-Fidelity Validation & Gaps

While the Go Min-Heap is mathematically clean, it introduces significant **computational complexity gaps** and **concurrency bottlenecks** compared to the original timing wheel:

### 1. The O(N) Event Cancellation Search Bottleneck
- **The Gap**: Legacy C cancels events via `queue_deq` by passing the direct pointer to the `q_element` node. The unlinking is a pure constant-time $O(1)$ operation.
- In Go, `Cancel` (`pkg/events/queue.go#L132`) receives only a unique numeric `id`. Because the min-heap is represented as a flat slice, finding the event requires a linear search across the entire heap:
  ```go
  for _, evt := range eq.events {
      if evt.ID == id {
          evt.Cancelled = true
          return
      }
  }
  ```
- **Impact**: In a running MUD with hundreds or thousands of active combat ticks, mob scripts, and zone resets, a linear scan on every single event cancellation is highly inefficient, degrading to $O(N)$ where $N$ is the total active queue size.

### 2. Dead Event Bloat
- **The Gap**: In C, unlinked elements are immediately removed from list memory and freed.
- In Go, cancelling an event does not remove it from the min-heap. It only flags it as `Cancelled = true`. The dead event continues to reside in the heap, taking up memory and causing overhead in sorting comparison routines, until it finally bubbles to the top of the heap and is popped inside `Process()` when its target tick arrives.
- **Impact**: If many events are cancelled early, the heap remains bloated with dead nodes, increasing the performance cost ($O(\log N)$) of subsequent insertions and ticks.

### 3. O(N) Linear Scan in `CancelBySource`
- **The Gap**: When a mob or player dies, the event system must clear all pending actions scheduled by that entity.
- Go implements `CancelBySource` (`pkg/events/queue.go#L146`), which performs a complete linear scan of the heap slice. Under high-load multi-mob combat, frequent deaths will trigger high CPU spikes due to repeated linear scans.

---

## 3. Go's Architectural Improvements Over C

Despite the search bottlenecks, Go's min-heap introduces excellent modernize improvements:
1. **Thread Safety**: The legacy C queue was completely single-threaded and unsafe for concurrent execution. Go protects the event heap via `eq.mu sync.Mutex`, allowing thread-safe enqueueing from different concurrent systems.
2. **Asynchronous Ticker Mode**: Go's `Start()` runs a background goroutine ticker that automatically processes ticks. This allows event handling to run independently of connection handling, preventing connection lag from stalling game event execution.
3. **Type Safety and No Raw Pointers**: C used unsafe `void *data` casting for event payloads. Go defines strongly-typed `Event` structs and safe `EventFunc` callback signatures.

---

## 4. Concurrency & Thread Safety

- **Mutex Contention**: If many subsystems (combat, pathfinding, Lua scripting) dynamically enqueue events while the background goroutine ticker is calling `Process()`, high contention can occur on `eq.mu`.
- **Callback Side-Effects**: Event callback functions (`EventFunc`) are executed on the event queue's goroutine when `Start()` is used. If these callbacks modify game character, object, or room structs without proper state locks, they will trigger severe race conditions with the main session thread.

---

## 5. Summary of Recommended Next Steps

1. **Optimize Cancellation to O(1)**:
   Maintain a secondary index map (`idToHeapIdx map[uint64]int`) inside `EventQueue`. This map should keep track of each event's current index within the heap slice. When an event is pushed, popped, or swapped (in `Swap(i, j)`), update the map. This will enable immediate $O(1)$ lookups and allow removing items directly from the heap in $O(\log N)$ time rather than running a linear search.
2. **Implement Heap Compaction**:
   Periodically compact the heap or rebuild it (e.g. every 1000 ticks) to filter out and garbage collect all events marked as `Cancelled = true`, reducing the heap size and sorting overhead.
3. **Audit Thread Safety of Callbacks**:
   Ensure all functions registered as `EventFunc` callbacks utilize proper mutex locks when interacting with characters, rooms, and objects, or ensure they dispatch modifications back to a synchronized main thread queue.

# Port Fidelity Audit: Module 49 (`spec_assign.c`)

This audit examines the port fidelity between the legacy C source file `src/spec_assign.c` and its Go counterparts in `pkg/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/spec_assign.c` (643 lines)
- **Functions & Features**:
  - **Dynamic Boot-Time Function Assignments**: Implements helper routines `ASSIGNMOB`, `ASSIGNOBJ`, and `ASSIGNROOM` that dynamically bind C function pointers directly to active indexes (`mob_index[rnum].func = fname`) at boot time.
  - **Rigorous Global Mapping**: Groups assignments into three behavioral routines: `assign_mobiles()`, `assign_objects()`, and `assign_rooms()`. If an assignment refers to a non-existent VNum, it outputs an error log (`SYSERR: Attempt to assign spec to non-existant...`).

### Go Port Files
- **Go Implementation**:
  - [pkg/game/spec_assign.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/spec_assign.go): Defines static lookup maps (`MobSpecAssign`, `ObjSpecAssign`, `RoomSpecAssign`) translating VNums to string identifiers. Exposes the global `SpecRegistry` map where active special procedures are registered at boot.
  - [pkg/game/spec_assign_validation_test.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/spec_assign_validation_test.go): Implements `TestUnregisteredSpecProcs` to validate that all mapped specs are fully compiled and registered.

---

## 2. High-Fidelity Validation & Gaps

The Go port achieves a very high level of mapping fidelity:
1. **Accurate Mappings**: Mappings for element temple minions, tavern keepers, desert mages, player castles, cemetery bosses, and town citizens map perfectly to their legacy C specifications.
2. **Automated Registration Checking**: Unlike C which allowed silent binding, Go implements a robust automated test (`TestUnregisteredSpecProcs`) that runs during build time (`go test ./...`), checking that every spec name exists within `SpecRegistry` and raising compile-time validation errors if a registration is missing.

---

## 3. Go's Architectural Improvements Over C

- **String-Key Decoupling**: Legacy C used direct compiled function pointers, preventing external serialization or dynamic configurations. Go decouples this by mapping VNums to standard strings, making it possible to serialize specials or manage assignments dynamically via YAML/JSON.
- **Panic Protection**: C crashes instantly if a dynamic shift in zone indexing corrupts a function pointer. Go safely maps names to registered types, catching missing definitions gracefully during boot validation.

---

## 4. Summary of Recommended Next Steps

1. **Direct Pointer Resolution on Spawn**:
   Currently, looking up special procedures at runtime requires performing double map lookups (`GetMobSpec` looks up the name in `MobSpecAssign` and then looks up the function pointer in `SpecRegistry`) on every command and tick. Optimize this by resolving the spec function once during prototype instancing and attaching it directly as a field pointer on `MobInstance`, `ObjectInstance`, and `RoomInstance`.

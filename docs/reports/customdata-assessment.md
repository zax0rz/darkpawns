# CustomData Assessment — DP-63

## Current state

`ObjectInstance.CustomData` is `map[string]interface{}` — untyped key-value storage used for runtime-only properties that aren't part of the prototype.

## Keys in use

| Key | Type | Used by | Purpose |
|-----|------|---------|---------|
| `carryN` | int | mob.go | Mob carry weight (numeric) |
| `carryW` | int | mob.go | Mob carry weight (numeric) |
| `extra_descs` | map[string]string | object.go | Extra descriptions (keyword→desc) |
| `maxMove` | int | object.go | Max movement points |
| `move` | int | object.go | Current movement points |
| `restored_weight` | int | object.go | Restored weight from save |

Plus generic key-value from:
- `mob.go:646` — affect storage (mob.CustomData[key] = aff)
- `houses.go:127` — house save restore
- `objsave.go:787,804` — player item state restore

## Proposed typed alternative

```go
type ObjectState struct {
    // Numeric properties (from C obj_data values)
    CarryN    int               `json:"carry_n,omitempty"`
    CarryW    int               `json:"carry_w,omitempty"`
    Move      int               `json:"move,omitempty"`
    MaxMove   int               `json:"max_move,omitempty"`
    
    // Extra descriptions (keyword → description)
    ExtraDescs map[string]string `json:"extra_descs,omitempty"`
    
    // Mob-specific runtime state
    MobAffects map[string]interface{} `json:"mob_affects,omitempty"`
    
    // Saved state from player items (transitional)
    SavedState map[string]interface{} `json:"saved_state,omitempty"`
}
```

## Migration path

1. Add `ObjectState` struct alongside `CustomData`
2. Migrate known keys to typed fields
3. Keep `SavedState` as transitional `map[string]interface{}` for save/restore compatibility
4. Remove `CustomData` entirely once all callers are migrated

## Risk
- Save file compatibility: `CustomData` is serialized to JSON. The new struct must produce equivalent JSON.
- Mob affect storage uses arbitrary keys — this stays as `map[string]interface{}` under `MobAffects`.
- Low risk if done incrementally with JSON compatibility tests.

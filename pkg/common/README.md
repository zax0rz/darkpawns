# Common Package

This package contains shared interfaces and types to break circular dependencies.

## Current Circular Dependencies

1. `game` ↔ `world`
   - `game` imports `world` for `ShopManager`
   - `world` imports `game` for `ObjectInstance`

2. `session` ↔ `command`
   - `session` imports `command` for command functions
   - `command` imports `session` for `Session` type

3. `game` ↔ `engine`
   - `game` imports `engine` for `SkillManager`
   - `engine` imports `game` for `Affectable` interface (in example_integration.go)

## Solution

To fix these circular dependencies:

1. Move `ObjectInstance` type to `common` (done in `types.go`)
2. Create `ShopManagerInterface` in `common` (done in `common.go`)
3. Create `SessionInterface` in `common` (done in `common.go`)
4. Create `Affectable` interface in `common` (done in `common.go`)

## Next Steps

1. Update `game` package to use `common.ObjectInstance` instead of its own
2. Update `game` to use `common.ShopManagerInterface` instead of `*world.ShopManager`
3. Update `world` to use `*common.ObjectInstance` instead of `*game.ObjectInstance`
4. Update `command` to use `common.SessionInterface` instead of `*session.Session`
5. Update `session` to use command functions via interface or dependency injection
6. Update `engine` to use `common.Affectable` instead of its own interface

## Notes

- The `common.ObjectInstance` struct has all the methods from `game.ObjectInstance`
- The `common` package imports `parser` for `Obj` type
- Some methods return `interface{}` to avoid importing other packages
- Type assertions will be needed in some places
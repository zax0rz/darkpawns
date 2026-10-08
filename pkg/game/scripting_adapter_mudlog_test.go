package game

// The scripting engine reaches the file-TRUE producer sink by an inline type
// assertion on the bridge value (pkg/scripting/engine.go, scriptMudLogFile).
// This keeps WorldScriptableAdapter's method a compile-time fact, so the
// producers cannot silently stop firing if that assertion's shape changes.
var _ interface {
	MudLog(string, int, int, bool)
} = (*WorldScriptableAdapter)(nil)

// Same for the shop-keeper lookup the engine asks by inline assertion to place
// lua_item_check's "Unable to determine shop" producer.
var _ interface{ IsShopKeeper(int) bool } = (*WorldScriptableAdapter)(nil)

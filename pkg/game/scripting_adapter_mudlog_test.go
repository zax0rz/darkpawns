package game

// The scripting engine reaches the file-TRUE producer sink by an inline type
// assertion on the bridge value (pkg/scripting/engine.go, scriptMudLogFile).
// This keeps WorldScriptableAdapter's method a compile-time fact, so the
// producers cannot silently stop firing if that assertion's shape changes.
var _ interface {
	MudLog(string, int, int, bool)
} = (*WorldScriptableAdapter)(nil)

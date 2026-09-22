package admin

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/yuin/gopher-lua/parse"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/fileedit"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// fileEditWriteLevel is the owner's door for web file writes: code upload
// never rides the builder role. A tedit field whose C edit threshold is higher
// (credits, policies, wizlist, immlist are LVL_IMPL) keeps that higher bar.
const fileEditWriteLevel = game.LVL_HIGOD

// fileEditLevelLabel names the immortal tiers the file-edit surface gates on.
func fileEditLevelLabel(level int) string {
	switch {
	case level >= game.LVL_IMPL:
		return "IMPL"
	case level >= game.LVL_GRGOD:
		return "GRGOD"
	case level >= game.LVL_HIGOD:
		return "HIGOD"
	case level >= game.LVL_GOD:
		return "GOD"
	default:
		return "BUILDER"
	}
}

func fileEditRequirement(level int) string {
	return fmt.Sprintf("%s (%d)", fileEditLevelLabel(level), level)
}

// teditWriteLevel is the web write threshold for one tedit field.
func teditWriteLevel(field fileedit.TextField) int {
	return max(fileEditWriteLevel, field.MinLevel)
}

type fileRootInput struct {
	Root      string `path:"root" enum:"lua,tedit"`
	Directory string `query:"directory"`
}

type filePathInput struct {
	Root string `path:"root" enum:"lua,tedit"`
	Path string `query:"path" required:"true"`
}

type fileWriteInput struct {
	Root        string `path:"root" enum:"lua,tedit"`
	Path        string `query:"path" required:"true"`
	IfMatch     string `header:"If-Match" doc:"ETag of the content being replaced"`
	IfNoneMatch string `header:"If-None-Match" doc:"\"*\" to create a file that must not exist yet"`
	Body        struct {
		Content string `json:"content"`
	}
}

type fileDeleteInput struct {
	Root    string `path:"root" enum:"lua,tedit"`
	Path    string `query:"path" required:"true"`
	IfMatch string `header:"If-Match" required:"true"`
}

type fileListingResponse struct {
	Entries []fileedit.Entry   `json:"entries"`
	Actions []olc.SchemaAction `json:"actions"`
}
type fileListingOutput struct{ Body fileListingResponse }

type fileContentOutput struct {
	ETag string `header:"ETag"`
	Body fileedit.File
}

type scriptUsage struct {
	Kind       string `json:"kind"`
	VNum       int    `json:"vnum"`
	Name       string `json:"name"`
	ScriptName string `json:"script_name"`
}

type scriptResolveResponse struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

type (
	scriptUsageOutput   struct{ Body []scriptUsage }
	scriptResolveOutput struct{ Body scriptResolveResponse }
	noContentOutput     struct {
		Status int `status:"204"`
	}
)

func registerFileEdit(api huma.API, world *game.World, database *db.DB, auditLogger *audit.AuditLogger) {
	readGate := fileEditGate(database, false)
	writeGate := fileEditGate(database, true)

	huma.Register(api, huma.Operation{OperationID: "list-editable-files", Method: http.MethodGet, Path: "/admin/files/{root}/listing", Summary: "List files in a webOLC file-edit root", Middlewares: huma.Middlewares{readGate}},
		func(ctx context.Context, in *fileRootInput) (*fileListingOutput, error) {
			level := fileEditLevel(ctx)
			response := fileListingResponse{Actions: []olc.SchemaAction{{
				Key: "write", Label: "Save and delete files",
				RequiredLevel: fileEditWriteLevel, RequiredLabel: fileEditLevelLabel(fileEditWriteLevel),
				Allowed: level >= fileEditWriteLevel,
			}}}
			if in.Root == "tedit" {
				entries := make([]fileedit.Entry, 0, len(fileedit.TextFields))
				for _, field := range fileedit.TextFields {
					need := teditWriteLevel(field)
					entries = append(entries, fileedit.Entry{Name: field.Name, Path: field.Name, MaxBytes: field.MaxBytes, RequiredLevel: need, RequiredLabel: fileEditLevelLabel(need), Allowed: level >= need})
				}
				response.Entries = entries
				return &fileListingOutput{Body: response}, nil
			}
			entries, err := fileedit.List(luaFileRoot(world), in.Directory, luaListFilter)
			if err != nil {
				return nil, fileOperationError(err)
			}
			for i := range entries {
				entries[i].RequiredLevel = fileEditWriteLevel
				entries[i].RequiredLabel = fileEditLevelLabel(fileEditWriteLevel)
				entries[i].Allowed = level >= fileEditWriteLevel
			}
			response.Entries = entries
			return &fileListingOutput{Body: response}, nil
		})

	huma.Register(api, huma.Operation{OperationID: "get-editable-file", Method: http.MethodGet, Path: "/admin/files/{root}/content", Summary: "Read a webOLC editable file", Middlewares: huma.Middlewares{readGate}},
		func(ctx context.Context, in *filePathInput) (*fileContentOutput, error) {
			target, err := resolveFileTarget(world, in.Root, in.Path)
			if err != nil {
				return nil, err
			}
			file, err := fileedit.Read(target.root, target.name)
			if err != nil {
				return nil, fileOperationError(err)
			}
			file.Path = in.Path
			return &fileContentOutput{ETag: file.ETag, Body: file}, nil
		})

	huma.Register(api, huma.Operation{OperationID: "put-editable-file", Method: http.MethodPut, Path: "/admin/files/{root}/content", Summary: "Save a webOLC editable file", Description: "Replace with If-Match (the ETag you read) or create with If-None-Match: *. A stale If-Match is 412; the write never falls back to last-writer-wins.", Middlewares: huma.Middlewares{writeGate}},
		func(ctx context.Context, in *fileWriteInput) (*fileContentOutput, error) {
			target, err := resolveFileTarget(world, in.Root, in.Path)
			if err != nil {
				return nil, err
			}
			if err := target.authorizeWrite(fileEditLevel(ctx)); err != nil {
				return nil, err
			}
			content := in.Body.Content
			if in.Root == "tedit" {
				// C's strip_string removes carriage returns before fputs, so
				// tedit files are LF-delimited on disk (see finishTextEditLocked).
				content = strings.ReplaceAll(content, "\r", "")
				if n := len(strings.ReplaceAll(content, "\n", "\r\n")); n > target.maxBytes {
					return nil, huma.Error422UnprocessableEntity(fmt.Sprintf("%s is limited to %d bytes (%d with CRLF line endings)", in.Path, target.maxBytes, n))
				}
			} else if _, err := parse.Parse(strings.NewReader(content), in.Path); err != nil {
				return nil, huma.Error422UnprocessableEntity(formatLuaParseError(err, content))
			}

			var file fileedit.File
			action := "file_edit_save"
			switch {
			case in.IfNoneMatch == "*" && in.IfMatch == "":
				if in.Root == "tedit" {
					return nil, huma.Error400BadRequest("tedit files are fixed; save with If-Match")
				}
				action = "file_edit_create"
				file, err = fileedit.Create(target.root, target.name, []byte(content), 0o666)
			case in.IfMatch != "" && in.IfNoneMatch == "":
				file, err = fileedit.Write(target.root, target.name, []byte(content), in.IfMatch)
			default:
				return nil, huma.NewError(http.StatusPreconditionRequired, "send If-Match with the ETag you read, or If-None-Match: * to create")
			}
			if err != nil {
				return nil, fileOperationError(err)
			}
			file.Path = in.Path
			target.afterSave(world, content)
			auditFileEdit(ctx, auditLogger, action, fmt.Sprintf("%s:%s %s -> %s", in.Root, in.Path, in.IfMatch+in.IfNoneMatch, file.ETag))
			return &fileContentOutput{ETag: file.ETag, Body: file}, nil
		})

	huma.Register(api, huma.Operation{OperationID: "delete-editable-file", Method: http.MethodDelete, Path: "/admin/files/{root}/content", Summary: "Delete a webOLC Lua script", Middlewares: huma.Middlewares{writeGate}},
		func(ctx context.Context, in *fileDeleteInput) (*noContentOutput, error) {
			if in.Root == "tedit" {
				return nil, huma.Error400BadRequest("tedit files cannot be deleted")
			}
			target, err := resolveFileTarget(world, in.Root, in.Path)
			if err != nil {
				return nil, err
			}
			if err := fileedit.Delete(target.root, target.name, in.IfMatch); err != nil {
				return nil, fileOperationError(err)
			}
			forgetScriptFailures()
			auditFileEdit(ctx, auditLogger, "file_edit_delete", fmt.Sprintf("%s:%s %s", in.Root, in.Path, in.IfMatch))
			return &noContentOutput{Status: http.StatusNoContent}, nil
		})

	huma.Register(api, huma.Operation{OperationID: "get-script-usage", Method: http.MethodGet, Path: "/admin/files/lua/usage", Summary: "Find the rooms, mobs, and objects whose script_name runs this file", Middlewares: huma.Middlewares{readGate}},
		func(ctx context.Context, in *struct {
			Path string `query:"path" required:"true"`
		},
		) (*scriptUsageOutput, error) {
			if _, err := resolveFileTarget(world, "lua", in.Path); err != nil {
				return nil, err
			}
			return &scriptUsageOutput{Body: findScriptUsage(world, in.Path)}, nil
		})

	huma.Register(api, huma.Operation{OperationID: "resolve-script-name", Method: http.MethodGet, Path: "/admin/files/lua/resolve", Summary: "Resolve a script_name to the file the engine would run", Description: "Uses the engine's own lookup (flat, then mob/room/obj). When nothing matches, path proposes <kind>/<name> for creation.", Middlewares: huma.Middlewares{readGate}},
		func(ctx context.Context, in *struct {
			Name string `query:"name" required:"true"`
			Kind string `query:"kind" enum:"mob,obj,room,"`
		},
		) (*scriptResolveOutput, error) {
			name := filepath.Clean(filepath.FromSlash(in.Name))
			if _, err := resolveFileTarget(world, "lua", filepath.ToSlash(name)); err != nil {
				return nil, err
			}
			root := luaFileRoot(world)
			if found := scripting.ResolveScriptPath(root, name); found != "" {
				if rel, err := filepath.Rel(root, found); err == nil {
					return &scriptResolveOutput{Body: scriptResolveResponse{Path: filepath.ToSlash(rel), Exists: true}}, nil
				}
			}
			proposed := name
			if in.Kind != "" {
				proposed = filepath.Join(in.Kind, name)
			}
			return &scriptResolveOutput{Body: scriptResolveResponse{Path: filepath.ToSlash(proposed)}}, nil
		})
}

// formatLuaParseError names a line for every refusal. gopher-lua reports an
// unexpected end of input (an unclosed function, a missing "end") at Line -1,
// so that case names the file's last line.
func formatLuaParseError(err error, content string) string {
	var parseErr *parse.Error
	if !errors.As(err, &parseErr) {
		return "Lua parse error: " + strings.TrimSpace(err.Error())
	}
	if parseErr.Pos.Line > 0 {
		return fmt.Sprintf("Lua parse error at line %d, column %d near %q: %s", parseErr.Pos.Line, parseErr.Pos.Column, parseErr.Token, parseErr.Message)
	}
	lines := strings.Count(strings.TrimRight(content, "\n"), "\n") + 1
	return fmt.Sprintf("Lua parse error at line %d (end of file): %s; check for an unclosed block or a missing \"end\"", lines, parseErr.Message)
}

func fileEditGate(database *db.DB, write bool) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		claims, err := claimsFromBearerHeader(ctx.Header("Authorization"))
		if err != nil {
			writeOLCError(ctx, http.StatusUnauthorized, olcMiddlewareError{Error: "unauthorized"})
			return
		}
		if !claims.HasRole("builder") {
			writeOLCError(ctx, http.StatusForbidden, olcMiddlewareError{Error: "forbidden", Required: "builder"})
			return
		}
		if database == nil {
			writeOLCError(ctx, http.StatusServiceUnavailable, olcMiddlewareError{Error: "file-edit authorization unavailable"})
			return
		}
		record, err := database.GetPlayer(claims.PlayerName)
		if err != nil || record == nil {
			writeOLCError(ctx, http.StatusUnauthorized, olcMiddlewareError{Error: "unauthorized"})
			return
		}
		if write && record.Level < fileEditWriteLevel {
			need := fileEditRequirement(fileEditWriteLevel)
			writeOLCError(ctx, http.StatusForbidden, olcMiddlewareError{Error: "file writes require " + need, Required: need})
			return
		}
		requestContext := auth.SetClaimsOnContext(ctx.Context(), claims)
		requestContext = context.WithValue(requestContext, fileEditLevelKey{}, record.Level)
		ctx = huma.WithContext(ctx, requestContext)
		next(ctx)
	}
}

type fileEditLevelKey struct{}

func fileEditLevel(ctx context.Context) int {
	level, _ := ctx.Value(fileEditLevelKey{}).(int)
	return level
}

// fileTarget is one addressable file under one root. The two roots differ:
// Lua paths are free-form under the scripts tree; tedit names map onto C's
// fixed table under lib/text.
type fileTarget struct {
	kind       string
	root       string
	name       string
	maxBytes   int
	writeLevel int
	textFile   string
}

func (t fileTarget) authorizeWrite(level int) error {
	if level < t.writeLevel {
		return huma.Error403Forbidden("writing this file requires " + fileEditRequirement(t.writeLevel))
	}
	return nil
}

// afterSave converges on the telnet save's side effects: a script save clears
// the failed-scripts negative cache, a tedit save refreshes the live text.
func (t fileTarget) afterSave(world *game.World, content string) {
	if t.kind == "lua" {
		forgetScriptFailures()
		return
	}
	fileedit.NotifyTextSaved(world, t.textFile, content)
}

func forgetScriptFailures() {
	if game.ScriptEngine != nil {
		game.ScriptEngine.ForgetFailures()
	}
}

func resolveFileTarget(world *game.World, rootKind, name string) (fileTarget, error) {
	switch rootKind {
	case "lua":
		if !strings.HasSuffix(name, ".lua") {
			return fileTarget{}, huma.Error400BadRequest("Lua file must end in .lua")
		}
		return fileTarget{kind: "lua", root: luaFileRoot(world), name: name, writeLevel: fileEditWriteLevel}, nil
	case "tedit":
		field, ok := fileedit.TextFieldByName(name)
		if !ok {
			return fileTarget{}, huma.Error404NotFound("unknown tedit field")
		}
		return fileTarget{kind: "tedit", root: world.LibTextDir, name: field.Filename, maxBytes: field.MaxBytes, writeLevel: teditWriteLevel(field), textFile: field.Filename}, nil
	default:
		return fileTarget{}, huma.Error400BadRequest("unknown file root")
	}
}

func luaFileRoot(world *game.World) string {
	if world.ScriptsDir != "" {
		return filepath.Clean(world.ScriptsDir)
	}
	if world.WorldPath != "" {
		return filepath.Join(world.WorldPath, "scripts")
	}
	return "scripts"
}

// luaListFilter is luaedit's luafilter; hidden temp files never match it.
func luaListFilter(entry fs.DirEntry) bool {
	return fileedit.LuaFilter(entry.Name())
}

func fileOperationError(err error) error {
	switch {
	case errors.Is(err, fileedit.ErrConflict):
		return huma.NewError(http.StatusPreconditionFailed, "file changed since it was opened; reload to compare before saving")
	case errors.Is(err, fileedit.ErrExists):
		return huma.NewError(http.StatusPreconditionFailed, "a file with that name already exists; open it instead")
	case errors.Is(err, fileedit.ErrInvalidPath):
		return huma.Error400BadRequest("invalid file path")
	case errors.Is(err, fs.ErrNotExist):
		return huma.Error404NotFound("file not found")
	default:
		return huma.NewError(http.StatusInternalServerError, "file operation failed")
	}
}

// findScriptUsage lists every prototype whose script_name the engine would
// resolve to path. It resolves each distinct script_name once with the
// engine's own lookup, so "guard.lua" is attributed to the one file the next
// trigger actually loads, never to every guard.lua in the tree.
func findScriptUsage(world *game.World, path string) []scriptUsage {
	root := luaFileRoot(world)
	want := filepath.Join(root, filepath.Clean(filepath.FromSlash(path)))
	resolved := map[string]bool{}
	matches := func(scriptName string) bool {
		if scriptName == "" {
			return false
		}
		hit, seen := resolved[scriptName]
		if !seen {
			name := filepath.Clean(filepath.FromSlash(scriptName))
			hit = !strings.Contains(name, "..") && scripting.ResolveScriptPath(root, name) == want
			resolved[scriptName] = hit
		}
		return hit
	}
	result := make([]scriptUsage, 0)
	rooms := world.Rooms()
	for i := range rooms {
		if room := &rooms[i]; matches(room.ScriptName) {
			result = append(result, scriptUsage{Kind: "room", VNum: room.VNum, Name: room.Name, ScriptName: room.ScriptName})
		}
	}
	for _, mob := range world.GetAllMobPrototypes() {
		if matches(mob.ScriptName) {
			result = append(result, scriptUsage{Kind: "mob", VNum: mob.VNum, Name: mob.ShortDesc, ScriptName: mob.ScriptName})
		}
	}
	for _, obj := range world.GetAllObjPrototypes() {
		if matches(obj.ScriptName) {
			result = append(result, scriptUsage{Kind: "obj", VNum: obj.VNum, Name: obj.ShortDesc, ScriptName: obj.ScriptName})
		}
	}
	return result
}

func auditFileEdit(ctx context.Context, logger *audit.AuditLogger, action, details string) {
	if logger == nil {
		return
	}
	claims, _ := auth.GetClaimsFromContext(ctx)
	user := ""
	if claims != nil {
		user = claims.PlayerName
	}
	logger.Log(audit.AuditEvent{EventType: "olc", User: user, IPAddress: clientIPFrom(ctx), Action: action, Details: details, Success: true})
}

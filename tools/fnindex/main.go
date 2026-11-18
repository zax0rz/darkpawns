// Command fnindex builds a comment-backed index from Go functions in pkg/ to
// the C functions named by their source citations. It deliberately does not
// infer ports from similar function names: the index is an audit of evidence.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type cFunction struct {
	Name      string
	File      string
	StartLine int
	EndLine   int
	DeclStart int
	BodyOpen  int
	BodyEnd   int
	CallSites []string
}

type goFunction struct {
	Package    string
	Name       string
	File       string
	Line       int
	CFunctions []string
	Cites      []string
	Errors     []string
}

var (
	lineCitation      = regexp.MustCompile(`\bsrc/([A-Za-z0-9_.-]+\.c):(\d+)(?:-(\d+))?\b`)
	namedCitation     = regexp.MustCompile(`(?i)\b(?:C(?:['’]s)?|C\s+(?:function|helper|command|routine|handler)|ported\s+(?:verbatim\s+)?from|ports?|mirrors?|mirrored\s+from|based\s+on|implementation\s+of|port\s+of|same\s+as|equivalent\s+to)\s+(?:the\s+)?(?:C\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	plainCCitation    = regexp.MustCompile(`(?i)\bC(?:\s+equivalent)?\s*:\s*([A-Za-z_][A-Za-z0-9_]*)\s*\(|\bC\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	fileNameCitation  = regexp.MustCompile(`\b([A-Za-z0-9_.-]+\.c):([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	bareCNameCitation = regexp.MustCompile("(?i)\\bC(?:['’]s)?\\s+`([A-Za-z_][A-Za-z0-9_]*)`")
	macroCall         = regexp.MustCompile(`\b(?:SPECIAL|ASPELL|ACMD)\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)`)
	cMacroDefinition  = regexp.MustCompile(`(?m)^[ \t]*(?:ACMD|SPECIAL|ASPELL)\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)\s*\{`)
	cPlainDefinition  = regexp.MustCompile(`(?m)^[ \t]*(?:[A-Za-z_][A-Za-z0-9_]*\s+)+\**\s*([A-Za-z_][A-Za-z0-9_]*)\s*\([^;{}]*\)\s*\{`)
)

var cControlWords = map[string]bool{"do": true, "else": true, "for": true, "if": true, "switch": true, "while": true}

func main() {
	root := flag.String("root", ".", "repository root")
	output := flag.String("output", "docs/research/function-index/index.tsv", "index output path")
	cOutput := flag.String("c-output", "docs/research/function-index/c-functions.tsv", "C function inventory output path")
	flag.Parse()

	functions, cByFile, err := buildIndex(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	attachCallSites(*root, cByFile)
	goPorts := productionGoPorts(functions)
	goNameCandidates := productionGoNameCandidates(functions)
	path := filepath.Join(*root, *output)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	w := bufio.NewWriter(f)
	fmt.Fprintln(w, "go_package\tgo_function\tgo_file\tgo_line\tc_functions\tcitation_text\tresolution_errors")
	unresolved := 0
	for _, fn := range functions {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			clean(fn.Package), clean(fn.Name), clean(fn.File), fn.Line,
			clean(strings.Join(fn.CFunctions, "; ")),
			clean(strings.Join(fn.Cites, " | ")),
			clean(strings.Join(fn.Errors, " | ")))
		unresolved += len(fn.Errors)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := f.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := writeCInventory(filepath.Join(*root, *cOutput), cByFile, goPorts, goNameCandidates); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cf := 0
	for _, funcs := range cByFile {
		cf += len(funcs)
	}
	fmt.Printf("indexed %d Go functions; parsed %d C functions; unresolved citations: %d\n", len(functions), cf, unresolved)
}

func buildIndex(root string) ([]goFunction, map[string][]cFunction, error) {
	cByFile := make(map[string][]cFunction)
	var cFiles []string
	err := filepath.WalkDir(filepath.Join(root, "src"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".c") {
			return nil
		}
		cFiles = append(cFiles, path)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(cFiles)
	byName := make(map[string][]cFunction)
	for _, path := range cFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		rel, _ := filepath.Rel(root, path)
		parsed := parseCFunctions(string(data), filepath.ToSlash(rel))
		cByFile[filepath.ToSlash(rel)] = parsed
		for _, fn := range parsed {
			byName[fn.Name] = append(byName[fn.Name], fn)
		}
	}

	fset := token.NewFileSet()
	var out []goFunction
	err = filepath.WalkDir(filepath.Join(root, "pkg"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.AllErrors)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}
		rel, _ := filepath.Rel(root, path)
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			text := functionCommentText(fd)
			receiver := ""
			if fd.Recv != nil && len(fd.Recv.List) > 0 {
				receiver = astString(fset, fd.Recv.List[0].Type)
				receiver = strings.TrimPrefix(receiver, "*")
			}
			name := fd.Name.Name
			if receiver != "" {
				name = "(" + receiver + ")." + name
			}
			fn := goFunction{Package: file.Name.Name, Name: name, File: filepath.ToSlash(rel), Line: fset.Position(fd.Pos()).Line}
			resolveComments(text, byName, cByFile, &fn)
			out = append(out, fn)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, cByFile, nil
}

func functionCommentText(fd *ast.FuncDecl) string {
	if fd.Doc == nil {
		return ""
	}
	return fd.Doc.Text()
}

func astString(fset *token.FileSet, node ast.Node) string {
	var b strings.Builder
	if err := formatNode(&b, fset, node); err != nil {
		return ""
	}
	return b.String()
}

// Kept separate to make the AST formatting import explicit and easy to audit.
func formatNode(b *strings.Builder, fset *token.FileSet, node ast.Node) error {
	return format.Node(b, fset, node)
}

func resolveComments(comment string, byName map[string][]cFunction, byFile map[string][]cFunction, fn *goFunction) {
	seenC, seenCitation := map[string]bool{}, map[string]bool{}
	addC := func(c cFunction, cite string) {
		key := c.File + ":" + c.Name
		if !seenC[key] {
			seenC[key] = true
			fn.CFunctions = append(fn.CFunctions, key)
		}
		if !seenCitation[cite] {
			seenCitation[cite] = true
			fn.Cites = append(fn.Cites, cite)
		}
	}
	addError := func(reason string) {
		fn.Errors = append(fn.Errors, reason)
	}
	for _, m := range lineCitation.FindAllStringSubmatchIndex(comment, -1) {
		if negatesCitation(comment, m[0]) {
			continue
		}
		cite := comment[m[0]:m[1]]
		file := "src/" + comment[m[2]:m[3]]
		start, _ := strconv.Atoi(comment[m[4]:m[5]])
		end := start
		if m[6] >= 0 {
			end, _ = strconv.Atoi(comment[m[6]:m[7]])
		}
		funcs, exists := byFile[file]
		if !exists {
			addError(cite + " (source file not parsed)")
			continue
		}
		matched := false
		for _, c := range funcs {
			if c.StartLine <= end && c.EndLine >= start {
				addC(c, cite)
				matched = true
			}
		}
		if !matched {
			addError(cite + " (range does not intersect a parsed C function body)")
		}
	}
	for _, m := range namedCitation.FindAllStringSubmatchIndex(comment, -1) {
		if negatesCitation(comment, m[0]) {
			continue
		}
		name := comment[m[2]:m[3]]
		matches := byName[name]
		if len(matches) == 0 {
			continue
		}
		cite := comment[m[0]:m[1]]
		for _, c := range matches {
			addC(c, cite)
		}
	}
	for _, m := range plainCCitation.FindAllStringSubmatchIndex(comment, -1) {
		if negatesCitation(comment, m[0]) {
			continue
		}
		nameStart, nameEnd := m[2], m[3]
		if nameStart < 0 {
			nameStart, nameEnd = m[4], m[5]
		}
		name := comment[nameStart:nameEnd]
		cite := comment[m[0]:m[1]]
		for _, c := range byName[name] {
			addC(c, cite)
		}
	}
	for _, m := range fileNameCitation.FindAllStringSubmatchIndex(comment, -1) {
		if negatesCitation(comment, m[0]) {
			continue
		}
		base, name := comment[m[2]:m[3]], comment[m[4]:m[5]]
		cite := comment[m[0]:m[1]]
		for _, c := range byName[name] {
			if filepath.Base(c.File) == base {
				addC(c, cite)
			}
		}
	}
	for _, m := range bareCNameCitation.FindAllStringSubmatchIndex(comment, -1) {
		if negatesCitation(comment, m[0]) {
			continue
		}
		name := comment[m[2]:m[3]]
		cite := comment[m[0]:m[1]]
		for _, c := range byName[name] {
			addC(c, cite)
		}
	}
	for _, m := range macroCall.FindAllStringSubmatchIndex(comment, -1) {
		if negatesCitation(comment, m[0]) {
			continue
		}
		name := comment[m[2]:m[3]]
		cite := comment[m[0]:m[1]]
		for _, c := range byName[name] {
			addC(c, cite)
		}
	}
	sort.Strings(fn.CFunctions)
	sort.Strings(fn.Cites)
	sort.Strings(fn.Errors)
}

func negatesCitation(text string, start int) bool {
	if start <= 0 {
		return false
	}
	prior := strings.ToLower(strings.TrimSpace(text[:start]))
	for _, negation := range []string{"not", "is not", "isn't", "isn’t", "never", "no longer"} {
		if strings.HasSuffix(prior, negation) {
			return true
		}
	}
	return false
}

func parseCFunctions(source, file string) []cFunction {
	rawCleaned := stripCCommentsAndStrings(source)
	cleaned := stripPreprocessor(rawCleaned)
	lines := lineStarts(cleaned)
	cleanedBraceEnds := matchingBraceEnds(cleaned)
	rawBraceEnds := matchingBraceEnds(rawCleaned)
	var funcs []cFunction
	collect := func(re *regexp.Regexp) {
		for _, match := range re.FindAllStringSubmatchIndex(cleaned, -1) {
			name := cleaned[match[2]:match[3]]
			if cControlWords[name] {
				continue
			}
			open := match[1] - 1
			end := -1
			if open >= 0 && open < len(cleanedBraceEnds) {
				end = cleanedBraceEnds[open]
			}
			if end >= open {
				funcs = append(funcs, cFunction{Name: name, File: file, StartLine: lineAt(lines, match[0]), EndLine: lineAt(lines, end), DeclStart: match[0], BodyOpen: open, BodyEnd: end})
			}
		}
	}
	collect(cMacroDefinition)
	collect(cPlainDefinition)
	// The first-arm projection above prevents brace matching across inactive
	// alternatives. A definition can exist only in a later arm (for example,
	// comm.c's POSIX my_signal), so add any complete definition signatures found
	// in the raw source as well. Line-based citations still resolve to that
	// source definition regardless of the host build's preprocessor choice.
	for _, re := range []*regexp.Regexp{cMacroDefinition, cPlainDefinition} {
		for _, match := range re.FindAllStringSubmatchIndex(rawCleaned, -1) {
			name := rawCleaned[match[2]:match[3]]
			if cControlWords[name] {
				continue
			}
			open := match[1] - 1
			end := -1
			if open >= 0 && open < len(rawBraceEnds) {
				end = rawBraceEnds[open]
			}
			if end < open {
				continue
			}
			funcs = append(funcs, cFunction{Name: name, File: file, StartLine: lineAt(lines, match[0]), EndLine: lineAt(lines, end), DeclStart: match[0], BodyOpen: open, BodyEnd: end})
		}
	}
	seen := make(map[string]bool)
	unique := funcs[:0]
	for _, fn := range funcs {
		key := fn.Name + "\x00" + strconv.Itoa(fn.StartLine)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, fn)
		}
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i].StartLine < unique[j].StartLine })
	return unique
}

func productionGoPorts(functions []goFunction) map[string][]string {
	ports := make(map[string][]string)
	for _, fn := range functions {
		if strings.HasSuffix(fn.File, "_test.go") {
			continue
		}
		for _, c := range fn.CFunctions {
			ports[c] = append(ports[c], fn.File+":"+strconv.Itoa(fn.Line)+":"+fn.Name)
		}
	}
	for key := range ports {
		sort.Strings(ports[key])
	}
	return ports
}

func productionGoNameCandidates(functions []goFunction) map[string][]string {
	byName := make(map[string][]string)
	for _, fn := range functions {
		if strings.HasSuffix(fn.File, "_test.go") {
			continue
		}
		name := fn.Name
		if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
			name = name[dot+1:]
		}
		byName[normalizeName(name)] = append(byName[normalizeName(name)], fn.File+":"+strconv.Itoa(fn.Line)+":"+fn.Name)
	}
	for key := range byName {
		sort.Strings(byName[key])
	}
	return byName
}

func normalizeName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func attachCallSites(root string, byFile map[string][]cFunction) {
	paths := make([]string, 0, len(byFile))
	byName := make(map[string][]*cFunction)
	for file := range byFile {
		paths = append(paths, file)
	}
	sort.Strings(paths)
	for file, funcs := range byFile {
		for i := range funcs {
			byName[funcs[i].Name] = append(byName[funcs[i].Name], &funcs[i])
		}
		byFile[file] = funcs
	}
	identifier := regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\b`)
	for _, sourceFile := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourceFile)))
		if err != nil {
			continue
		}
		// Count C references in every source branch. The line-based body parser
		// selects a branch to match braces, but a caller in a later #else/#elif
		// arm is still a real source-level reference that belongs in the audit.
		source := stripCCommentsAndStrings(string(data))
		lines := lineStarts(source)
		for _, loc := range identifier.FindAllStringIndex(source, -1) {
			name := source[loc[0]:loc[1]]
			for _, fn := range byName[name] {
				if sourceFile == fn.File && loc[0] >= fn.DeclStart && loc[0] <= fn.BodyOpen {
					continue
				}
				fn.CallSites = append(fn.CallSites, sourceFile+":"+strconv.Itoa(lineAt(lines, loc[0])))
			}
		}
	}
	for _, funcs := range byFile {
		for i := range funcs {
			sort.Strings(funcs[i].CallSites)
		}
	}
}

func writeCInventory(path string, byFile map[string][]cFunction, goPorts, goNameCandidates map[string][]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	if _, err := fmt.Fprintln(w, "c_function\tc_file\tc_start_line\tc_end_line\tnon_definition_refs\tref_sites\tgo_ports\tgo_port_functions\tgo_name_candidates\tdisposition"); err != nil {
		_ = f.Close()
		return err
	}
	var funcs []cFunction
	for _, fileFuncs := range byFile {
		funcs = append(funcs, fileFuncs...)
	}
	sort.Slice(funcs, func(i, j int) bool {
		if funcs[i].File != funcs[j].File {
			return funcs[i].File < funcs[j].File
		}
		if funcs[i].StartLine != funcs[j].StartLine {
			return funcs[i].StartLine < funcs[j].StartLine
		}
		return funcs[i].Name < funcs[j].Name
	})
	for _, fn := range funcs {
		key := fn.File + ":" + fn.Name
		ports := goPorts[key]
		candidates := append([]string(nil), goNameCandidates[normalizeName(fn.Name)]...)
		// C helpers named foo() sometimes have a Go C-prefixed helper such as
		// cFoo(). Keep those as name-only leads, separate from cite-backed ports.
		for goName, goFunctions := range goNameCandidates {
			if strings.TrimPrefix(goName, "c") == normalizeName(fn.Name) && goName != normalizeName(fn.Name) {
				candidates = append(candidates, goFunctions...)
			}
		}
		if fn.File == "src/circle.c" && fn.Name == "main" {
			candidates = append(candidates, "cmd/server/main.go:283:main")
		}
		if fn.File == "src/comm.c" && fn.Name == "my_signal" {
			candidates = append(candidates, "cmd/server/main.go:1114:signal.Notify")
		}
		sort.Strings(candidates)
		candidates = uniqueStrings(candidates)
		disposition := "has-comment-cited-port"
		if len(ports) == 0 {
			switch {
			case fn.File == "src/comm.c" && fn.Name == "my_signal":
				disposition = "go-native-os-signal-registration"
			case fn.File == "src/circle.c" && fn.Name == "main":
				disposition = "go-native-server-entrypoint-not-game-routine"
			case len(fn.CallSites) == 0:
				disposition = "dead-candidate-no-src-identifier-references"
			case len(candidates) > 0:
				disposition = "possible-go-native-replacement-name-match-only"
			default:
				disposition = "active-c-reference-no-go-name-match-missing-lead"
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%s\t%d\t%s\t%s\t%s\n", clean(fn.Name), clean(fn.File), fn.StartLine, fn.EndLine, len(fn.CallSites), clean(strings.Join(fn.CallSites, ";")), len(ports), clean(strings.Join(ports, ";")), clean(strings.Join(candidates, ";")), disposition)
	}
	if err := w.Flush(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	unique := values[:1]
	for _, value := range values[1:] {
		if value != unique[len(unique)-1] {
			unique = append(unique, value)
		}
	}
	return unique
}

func stripPreprocessor(source string) string {
	lines := strings.SplitAfter(source, "\n")
	type conditional struct {
		parent     bool
		seenBranch bool
	}
	active := true
	var stack []conditional
	continuedDirective := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		isDirective := strings.HasPrefix(trimmed, "#") || continuedDirective
		if isDirective {
			if strings.HasPrefix(trimmed, "#if ") || strings.HasPrefix(trimmed, "#ifdef ") || strings.HasPrefix(trimmed, "#ifndef ") {
				stack = append(stack, conditional{parent: active, seenBranch: true})
			} else if strings.HasPrefix(trimmed, "#else") || strings.HasPrefix(trimmed, "#elif ") {
				if len(stack) > 0 {
					frame := &stack[len(stack)-1]
					active = frame.parent && !frame.seenBranch
					frame.seenBranch = true
				}
			} else if strings.HasPrefix(trimmed, "#endif") {
				if len(stack) > 0 {
					active = stack[len(stack)-1].parent
					stack = stack[:len(stack)-1]
				}
			}
			newline := ""
			if strings.HasSuffix(line, "\n") {
				newline = "\n"
			}
			lines[i] = strings.Repeat(" ", len(strings.TrimSuffix(line, "\n"))) + newline
			content := strings.TrimRight(strings.TrimSuffix(line, "\n"), " \t\r")
			continuedDirective = strings.HasSuffix(content, "\\")
			continue
		}
		if !active {
			newline := ""
			if strings.HasSuffix(line, "\n") {
				newline = "\n"
			}
			lines[i] = strings.Repeat(" ", len(strings.TrimSuffix(line, "\n"))) + newline
		}
	}
	return strings.Join(lines, "")
}

func stripCCommentsAndStrings(s string) string {
	b := []byte(s)
	state := byte(0)
	for i := 0; i < len(b); i++ {
		c := b[i]
		next := byte(0)
		if i+1 < len(b) {
			next = b[i+1]
		}
		switch state {
		case 0:
			if c == '/' && next == '/' {
				b[i], b[i+1], state = ' ', ' ', 'l'
				i++
			} else if c == '/' && next == '*' {
				b[i], b[i+1], state = ' ', ' ', 'b'
				i++
			} else if c == '"' || c == '\'' {
				state = c
				b[i] = ' '
			}
		case 'l':
			if c == '\n' {
				state = 0
			} else {
				b[i] = ' '
			}
		case 'b':
			if c == '*' && next == '/' {
				b[i], b[i+1], state = ' ', ' ', 0
				i++
			} else if c != '\n' {
				b[i] = ' '
			}
		default:
			if c == '\\' {
				b[i] = ' '
				if i+1 < len(b) && b[i+1] != '\n' {
					b[i+1] = ' '
					i++
				}
			} else if c == state {
				b[i] = ' '
				state = 0
			} else if c != '\n' {
				b[i] = ' '
			}
		}
	}
	return string(b)
}

func matchingBrace(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func matchingBraceEnds(s string) []int {
	ends := make([]int, len(s))
	for i := range ends {
		ends[i] = -1
	}
	stack := make([]int, 0, 64)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			stack = append(stack, i)
		case '}':
			if len(stack) == 0 {
				continue
			}
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			ends[open] = i
		}
	}
	return ends
}

func lineStarts(s string) []int {
	lines := []int{0}
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, i+1)
		}
	}
	return lines
}

func lineAt(lines []int, pos int) int {
	i := sort.Search(len(lines), func(i int) bool { return lines[i] > pos })
	return i
}

func clean(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

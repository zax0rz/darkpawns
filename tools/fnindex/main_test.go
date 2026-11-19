package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestParseCFunctionBodies(t *testing.T) {
	source := `
#define BODY { if (ignored) { } }
ACMD(do_test)
{
  if (condition) {
    while (condition) { }
  }
}
SPECIAL(room_test) { return FALSE; }
ASPELL(spell_test)
{
  return TRUE;
}
static int
plain_test(int value)
{
  return value;
}
#if defined(ONE)
int selected_branch(void) { return 1; }
#else
int alternate_branch(void) { return 0; }
#endif
`
	got := parseCFunctions(source, "src/sample.c")
	var names []string
	for _, fn := range got {
		names = append(names, fn.Name)
	}
	want := []string{"do_test", "room_test", "spell_test", "plain_test", "selected_branch", "alternate_branch"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("parsed C functions = %v, want %v", names, want)
	}
	if got[3].StartLine != 14 || got[3].EndLine != 18 {
		t.Fatalf("plain_test lines = %d-%d, want 14-18", got[3].StartLine, got[3].EndLine)
	}
}

func TestCParserIncludesDefinitionsInConditionalArms(t *testing.T) {
	var cFiles []string
	if err := filepath.WalkDir("../../src", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".c") {
			cFiles = append(cFiles, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, path := range cFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		raw := stripCCommentsAndStrings(string(data))
		parsed := parseCFunctions(string(data), filepath.ToSlash(path))
		seen := make(map[string]bool)
		for _, fn := range parsed {
			seen[fn.Name+":"+strconv.Itoa(fn.StartLine)] = true
		}
		for _, re := range []*regexp.Regexp{cMacroDefinition, cPlainDefinition} {
			for _, match := range re.FindAllStringSubmatchIndex(raw, -1) {
				name := raw[match[2]:match[3]]
				if cControlWords[name] {
					continue
				}
				open := match[1] - 1
				if matchingBrace(raw, open) < open {
					continue
				}
				line := lineAt(lineStarts(raw), match[0])
				key := name + ":" + strconv.Itoa(line)
				if !seen[key] {
					t.Errorf("%s:%d: definition %s is hidden by a conditional branch", path, line, name)
				}
			}
		}
	}
}

func TestResolveCCommentCitationForms(t *testing.T) {
	byName := map[string][]cFunction{
		"fname":     {{Name: "fname", File: "src/handler.c", StartLine: 68, EndLine: 79}},
		"isname":    {{Name: "isname", File: "src/handler.c", StartLine: 81, EndLine: 111}},
		"is_number": {{Name: "is_number", File: "src/interpreter.c", StartLine: 1175, EndLine: 1182}},
		"damage":    {{Name: "damage", File: "src/fight.c", StartLine: 1318, EndLine: 1706}},
	}
	var fn goFunction
	resolveComments("C: fname(), C isname(), C equivalent: is_number(), C's damage()", byName, nil, &fn)
	want := []string{"src/fight.c:damage", "src/handler.c:fname", "src/handler.c:isname", "src/interpreter.c:is_number"}
	if !reflect.DeepEqual(fn.CFunctions, want) {
		t.Fatalf("resolved citations = %v, want %v", fn.CFunctions, want)
	}
	var negated goFunction
	resolveComments("It is NOT C's damage(); see CFName instead.", byName, nil, &negated)
	if len(negated.CFunctions) != 0 {
		t.Fatalf("negated source mention resolved as a port: %v", negated.CFunctions)
	}
}

func TestResolveCommentsCountsRepeatedUnresolvedCitations(t *testing.T) {
	var fn goFunction
	resolveComments("src/config.c:82 and again src/config.c:82", nil, map[string][]cFunction{}, &fn)
	if len(fn.Errors) != 2 {
		t.Fatalf("unresolved citation occurrences = %d, want 2", len(fn.Errors))
	}
}

func TestFunctionCommentTextReadsDeclarationDoc(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", `package p
// C: outer()
func f() {}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	text := functionCommentText(file.Decls[0].(*ast.FuncDecl))
	if !strings.Contains(text, "outer()") {
		t.Fatalf("function comments omitted citation: %q", text)
	}
}

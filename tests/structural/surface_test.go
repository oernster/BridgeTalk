package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// boundSurface is every method the front end may call.
//
// Wails is handed the facade and binds every exported method on it, so in this
// application "exported" and "public API" are the same word. That makes adding an
// exported method to the facade a decision about the wire whether or not the author
// meant it as one; nothing about the language says so at the point of writing.
// The list is here so the decision has to be taken twice: once in the code and once
// against this.
var boundSurface = []string{
	"About", "Audition", "AuditionGroups", "ChooseJournalDir", "ChooseLibraryRoot",
	"CueBreakdown", "Licence", "MakeVoiceFolders", "MinimiseToTray", "Muted", "Quit",
	"Reactions", "RequestQuit", "Rescan", "SelectVoice", "SetLaunchOnBoot",
	"SetMuted",
	"SetVolume", "State", "StopAudition", "TakeKeyboard", "Voices", "Volume",
}

// The facade is found rather than listed.
//
// It used to be three named files. Adding a fourth put two exported methods on the
// facade that this test could not see: the front end could call them and nothing said
// so, which is the failure this test exists to prevent, arriving through the door the
// test itself left open. Every Go file at the repository root is read now, so a method
// on the facade is in scope wherever it is written.
func facadeFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var found []string
	for _, path := range goFiles(t) {
		if filepath.Dir(path) == root && !strings.HasSuffix(path, "_test.go") {
			found = append(found, path)
		}
	}
	if len(found) == 0 {
		t.Fatal("no files found at the repository root, the walk is wrong")
	}
	return found
}

// TestTheBoundSurfaceIsDeclared keeps the front end's reach to what was intended.
//
// Proved by adding an exported method to the facade and reading the exit code. It has
// been wrong once already: Report is on the Reporter port, so it had to be exported,
// and being exported put a way to write entries into the reaction log within reach of
// the page. It now lives on a type of its own and this test is what would have said so.
func TestTheBoundSurfaceIsDeclared(t *testing.T) {
	want := map[string]bool{}
	for _, name := range boundSurface {
		want[name] = true
	}

	var found []string
	for _, path := range facadeFiles(t) {
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !fn.Name.IsExported() {
				continue
			}
			if receiverIs(fn, "App") {
				found = append(found, fn.Name.Name)
			}
		}
	}
	sort.Strings(found)

	for _, name := range found {
		if !want[name] {
			t.Errorf(
				"%s is exported on the facade, so the front end can call it. "+
					"Add it to boundSurface if that is intended; otherwise move it off the facade",
				name,
			)
		}
		delete(want, name)
	}
	for name := range want {
		t.Errorf("boundSurface names %s, which the facade no longer has", name)
	}
}

// receiverIs reports whether a method hangs off the named type, by pointer or value.
func receiverIs(fn *ast.FuncDecl, name string) bool {
	if len(fn.Recv.List) == 0 {
		return false
	}
	switch expr := fn.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		ident, ok := expr.X.(*ast.Ident)
		return ok && ident.Name == name
	case *ast.Ident:
		return expr.Name == name
	}
	return strings.Contains(name, "")
}

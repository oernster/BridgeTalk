// Package structural enforces the architecture with tests rather than convention.
//
// Every assertion here has been proved to bite by planting a violation and watching
// it fail. An assertion never seen to fail is not yet a guard.
package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lineLimit is the module-size cap. dangerBand is five per cent below it: a file
// that lands between them is refactored down to safeLanding rather than left one
// edit away from breaching, because shaving a line or two buys nothing.
const (
	lineLimit   = 400
	dangerBand  = lineLimit - lineLimit/20
	safeLanding = 350
)

// modulePath prefixes every internal import.
const modulePath = "github.com/oernster/bridge-talk/"

// compositionRoot names the files allowed to import both application and
// infrastructure. main.go builds the adapters and app.go is the facade the window
// calls. Between them they are the composition root; nothing else may join them.
var compositionRoot = map[string]bool{"main.go": true, "app.go": true}

// forbiddenInDomain names the packages and calls that would make the domain impure.
// Time and randomness reach it only through injected interfaces.
var forbiddenInDomain = []string{
	"net", "net/http", "os", "path/filepath", "math/rand",
	"database/sql", "io/ioutil", "os/exec",
}

// forbiddenCallsInDomain names calls that read the wall clock or the global random
// source directly, which would make domain behaviour irreproducible.
var forbiddenCallsInDomain = []string{"time.Now", "rand.Intn", "rand.Float64"}

// repoRoot walks up from the test's directory to the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for range 6 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not find go.mod above the test directory")
	return ""
}

// goFiles returns every Go source file in the repository.
func goFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var found []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "frontend" || entry.Name() == "venv" || entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking repository: %v", err)
	}
	if len(found) == 0 {
		t.Fatal("no Go files found, the walk is wrong")
	}
	return found
}

// frontendSourceExtensions are the front-end files the size rule governs. Generated
// output and other people's code are not ours to keep small.
var frontendSourceExtensions = map[string]bool{".ts": true, ".tsx": true, ".css": true}

// frontendFiles returns every front-end source file.
func frontendFiles(t *testing.T) []string {
	t.Helper()
	source := filepath.Join(repoRoot(t), frontendSource)
	var found []string
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil //nolint:nilerr // an unreadable subtree is skipped, not fatal
		}
		if frontendSourceExtensions[strings.ToLower(filepath.Ext(path))] {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", source, err)
	}
	if len(found) == 0 {
		t.Fatalf("no front-end source found under %s, the walk is wrong", source)
	}
	return found
}

// importsOf parses a file and returns its import paths.
func importsOf(t *testing.T, path string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	out := make([]string, 0, len(parsed.Imports))
	for _, item := range parsed.Imports {
		out = append(out, strings.Trim(item.Path.Value, `"`))
	}
	return out
}

// layerOf returns which architectural layer a file belongs to.
func layerOf(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) >= 2 && parts[0] == "internal" {
		return parts[1]
	}
	return ""
}

func TestDomainHasNoOutwardImports(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if layerOf(root, path) != "domain" {
			continue
		}
		for _, imported := range importsOf(t, path) {
			if !strings.HasPrefix(imported, modulePath) {
				continue
			}
			inner := strings.TrimPrefix(imported, modulePath)
			if !strings.HasPrefix(inner, "internal/domain") {
				t.Errorf("%s imports %s: the domain depends on nothing", path, imported)
			}
		}
	}
}

func TestDomainIsPure(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if layerOf(root, path) != "domain" || strings.HasSuffix(path, "_test.go") {
			continue
		}
		for _, imported := range importsOf(t, path) {
			for _, banned := range forbiddenInDomain {
				if imported == banned {
					t.Errorf("%s imports %q: the domain performs no IO", path, banned)
				}
			}
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		for _, call := range forbiddenCallsInDomain {
			if strings.Contains(string(raw), call+"(") {
				t.Errorf("%s calls %s: inject the clock or the chooser instead", path, call)
			}
		}
	}
}

func TestApplicationDoesNotImportInfrastructure(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if layerOf(root, path) != "application" {
			continue
		}
		for _, imported := range importsOf(t, path) {
			if strings.Contains(imported, "internal/infrastructure") || strings.Contains(imported, "wails") {
				t.Errorf("%s imports %s: the application depends on ports only", path, imported)
			}
		}
	}
}

func TestCompositionRootIsWhitelisted(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		var application, infrastructure bool
		for _, imported := range importsOf(t, path) {
			application = application || strings.Contains(imported, "internal/application/services")
			infrastructure = infrastructure || strings.Contains(imported, "internal/infrastructure")
		}
		if !application || !infrastructure {
			continue
		}
		relative, _ := filepath.Rel(root, path)
		if !compositionRoot[filepath.ToSlash(relative)] {
			t.Errorf("%s wires application to infrastructure: only the composition root may", relative)
		}
	}
}

// sourceFiles is every file the size rule governs: the Go that runs the application
// and the TypeScript and CSS that draw it.
//
// The rule used to walk the Go alone, which left a third of the source outside the one
// check that says a file has grown too big to read. The front end is where the growth
// actually happens, because a component gathers markup, behaviour and the words on
// screen in the same place; the file holding the panes reached the band this week and
// nothing said so.
// The setup program's front end was outside it for the same reason again: the walk
// covers frontend/src; that is a second application's page. It reached 880 lines in
// one file holding its markup, its whole stylesheet and its whole script, with no
// rule anywhere having anything to say about it.
func sourceFiles(t *testing.T) []string {
	t.Helper()
	files := append(goFiles(t), frontendFiles(t)...)
	return append(files, setupFrontendFiles(t)...)
}

// setupFrontendDir is the setup program's front end. It has no build step, so the
// files there are the source rather than an output of one.
var setupFrontendDir = filepath.Join("installer", "frontend", "dist")

// setupFrontendExtensions are the files there that the size rule governs. The icons
// beside them are artwork.
var setupFrontendExtensions = map[string]bool{".html": true, ".css": true, ".js": true}

// setupFrontendFiles returns the setup program's own source files.
func setupFrontendFiles(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), setupFrontendDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(setupFrontendDir), err)
	}
	var found []string
	for _, entry := range entries {
		if entry.IsDir() || !setupFrontendExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			continue
		}
		found = append(found, filepath.Join(dir, entry.Name()))
	}
	if len(found) == 0 {
		t.Fatalf("no source found in %s, the walk is wrong", filepath.ToSlash(setupFrontendDir))
	}
	return found
}

func TestNoFileExceedsLineLimit(t *testing.T) {
	for _, path := range sourceFiles(t) {
		if count := lineCount(t, path); count > lineLimit {
			t.Errorf("%s has %d lines, over the %d limit", path, count, lineLimit)
		}
	}
}

func TestNoFileInDangerBand(t *testing.T) {
	for _, path := range sourceFiles(t) {
		count := lineCount(t, path)
		if count > dangerBand && count <= lineLimit {
			t.Errorf(
				"%s has %d lines, inside the danger band %d to %d: reduce it to %d or fewer",
				path, count, dangerBand+1, lineLimit, safeLanding,
			)
		}
	}
}

// lineCount counts the lines in a file.
func lineCount(t *testing.T, path string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return strings.Count(string(raw), "\n") + 1
}

// TestEveryExportedTypeIsDocumented keeps the package surface self-describing, which
// matters more here than usual because the cue vocabulary is a shipped contract.
func TestEveryExportedTypeIsDocumented(t *testing.T) {
	for _, path := range goFiles(t) {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE || general.Doc != nil {
				continue
			}
			for _, spec := range general.Specs {
				typed, ok := spec.(*ast.TypeSpec)
				if ok && typed.Name.IsExported() && typed.Doc == nil {
					t.Errorf("%s: exported type %s has no doc comment", path, typed.Name.Name)
				}
			}
		}
	}
}

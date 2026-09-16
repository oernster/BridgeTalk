package structural

// NFR-S-2: the application writes nowhere but the library root, its own per user data directories
// and the sign-in entry under HKCU. Where each write goes cannot be read off the source by a
// machine, so it is stated once in the list below and held there: every call that
// writes, moves or removes a file or changes the registry, in every package the application links,
// must be listed with where it writes. A new one fails until someone says where it goes.
//
// What this cannot see: a write made through COM or by Wails and its web view, neither of which is a
// call in this module's Go source.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/product"
)

// setupPackage is the package the setup program installs with. The application links it for the
// sign-in entry alone.
const setupPackage = "internal/infrastructure/setup"

// fileWrites names the calls in os that write, move or remove a file or folder.
var fileWrites = map[string]bool{
	"WriteFile": true, "Create": true, "CreateTemp": true, "OpenFile": true, "Mkdir": true,
	"MkdirAll": true, "MkdirTemp": true, "Remove": true, "RemoveAll": true, "Rename": true,
	"Chmod": true, "Chtimes": true, "Truncate": true, "Symlink": true, "Link": true,
	"Chown": true, "Lchown": true,
}

// registryWrites names the registry calls that make, change or remove a key or a value.
var registryWrites = map[string]bool{
	"CreateKey": true, "DeleteKey": true, "SetStringValue": true, "SetExpandStringValue": true,
	"SetDWordValue": true, "SetQWordValue": true, "SetBinaryValue": true, "SetStringsValue": true,
	"DeleteValue": true,
}

// localData is the application's own per user data folder, named once for the list below.
const localData = `%LOCALAPPDATA%\` + product.Slug

// writesByTheApplication are the places the application itself writes, each with where it writes.
var writesByTheApplication = map[string]string{
	"internal/infrastructure/wholefile/wholefile.go:Write":        "a part beside the file its caller names, then that file; each caller is listed here",
	"internal/infrastructure/config/settings.go:Save":             "the settings file under the user configuration directory",
	"internal/infrastructure/config/settings.go:Forget":           "the settings file and its folder under the user configuration directory",
	"internal/infrastructure/library/checklist.go:MomentFolder":   "a moment's folder under the library root (FR-314)",
	"internal/infrastructure/library/folders.go:MakeVoiceFolders": "a new voice's folders under the library root (FR-223)",
	"internal/infrastructure/library/folders.go:makeVoiceFolders": "a new voice's folders under the library root (FR-223)",
	"internal/infrastructure/library/root.go:DefaultRoot":         "the default recordings folder under " + localData,
	"internal/infrastructure/madelines/madelines.go:Write":        "a made line under " + localData + " (FR-523)",
	"internal/infrastructure/madelines/madelines.go:Delete":       "a made line under " + localData + " (FR-527)",
	"internal/infrastructure/runlog/output_windows.go:toTerminal": "the console the run was started from, which is no file",
	"internal/infrastructure/runlog/runlog.go:Open":               "Log.txt under " + localData + " (FR-715)",
	setupPackage + "/windows.go:SetLaunchOnBoot":                  "the sign-in entry under HKCU, the one registry write the application makes",
}

// writesBySetupAlone are writes in the setup package the application links but never reaches:
// TestTheApplicationCallsNoOtherSetupWrite holds it to that.
var writesBySetupAlone = map[string]string{
	setupPackage + "/location.go:probeWrite":          "a probe file in the folder chosen to install into, removed at once",
	setupPackage + "/plugins.go:MakePluginsFolder":    "the plugins folder in the install directory (FR-576)",
	setupPackage + "/setup.go:ExtractZip":             "the install directory",
	setupPackage + "/setup.go:extractEntry":           "the install directory",
	setupPackage + "/setup.go:RemoveTree":             "the saved window state, on uninstall",
	setupPackage + "/setup.go:CopyFile":               "the uninstaller in the install directory",
	setupPackage + "/windows.go:WriteUninstallEntry":  "the uninstall record under HKCU",
	setupPackage + "/windows.go:RemoveUninstallEntry": "the uninstall record under HKCU",
	setupPackage + "/windows.go:ApplyShortcuts":       "a Start Menu or Desktop shortcut",
}

// setupNamesTheApplicationUses are the only names the application reads out of the setup package.
var setupNamesTheApplicationUses = []string{"HiddenFlag", "IsLaunchOnBoot", "SetLaunchOnBoot"}

// linkedPackages answers the directory of every package in this module the application imports,
// directly or through another, starting from the files at the repository root. Test files are not
// followed, since they are not linked into the application.
func linkedPackages(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	seen := map[string]bool{".": true}
	queue := []string{"."}
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]
		for _, file := range sourceIn(t, filepath.Join(root, dir)) {
			for _, imported := range importsOf(t, file) {
				if !strings.HasPrefix(imported, modulePath) {
					continue
				}
				inner := strings.TrimPrefix(imported, modulePath)
				if !seen[inner] {
					seen[inner] = true
					queue = append(queue, inner)
				}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for dir := range seen {
		out = append(out, dir)
	}
	sort.Strings(out)
	return out
}

// sourceIn answers the Go files in dir that are not tests.
func sourceIn(t *testing.T, dir string) []string {
	t.Helper()
	all, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("listing %s: %v", dir, err)
	}
	var out []string
	for _, file := range all {
		if !strings.HasSuffix(file, "_test.go") {
			out = append(out, file)
		}
	}
	return out
}

// writesIn answers every write in file, each named by its package folder, its file and the function
// it sits in.
func writesIn(t *testing.T, root, file string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", file, err)
	}
	relative, _ := filepath.Rel(root, file)
	var out []string
	for _, declared := range parsed.Decls {
		function, ok := declared.(*ast.FuncDecl)
		if !ok {
			continue
		}
		ast.Inspect(function, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			named, _ := selector.X.(*ast.Ident)
			writes := registryWrites[selector.Sel.Name] ||
				(named != nil && named.Name == "os" && fileWrites[selector.Sel.Name]) ||
				(named != nil && named.Name == "wholefile" && selector.Sel.Name == "Write")
			if writes {
				out = append(out, filepath.ToSlash(relative)+":"+function.Name.Name)
			}
			return true
		})
	}
	return out
}

// Every write in the application's linked packages is listed with where it goes; nothing listed has
// gone.
func TestEveryWriteTheApplicationLinksSaysWhereItGoes(t *testing.T) {
	root := repoRoot(t)
	found := map[string]bool{}
	for _, dir := range linkedPackages(t) {
		for _, file := range sourceIn(t, filepath.Join(root, dir)) {
			for _, write := range writesIn(t, root, file) {
				found[write] = true
			}
		}
	}
	if len(found) == 0 {
		t.Fatal("no write was found anywhere, so the search is wrong")
	}
	for write := range found {
		_, own := writesByTheApplication[write]
		_, setups := writesBySetupAlone[write]
		if !own && !setups {
			t.Errorf("%s writes and says nowhere where; list it with the place it writes (NFR-S-2)", write)
		}
	}
	for _, listed := range []map[string]string{writesByTheApplication, writesBySetupAlone} {
		for write := range listed {
			if !found[write] {
				t.Errorf("%s is listed as a write but writes nothing now; take it off the list", write)
			}
		}
	}
}

// The application reaches the setup package for the sign-in entry and nothing else, so none of the
// writes setup makes on installing reach the application.
func TestTheApplicationCallsNoOtherSetupWrite(t *testing.T) {
	root := repoRoot(t)
	for _, dir := range linkedPackages(t) {
		if dir == setupPackage {
			continue
		}
		for _, file := range sourceIn(t, filepath.Join(root, dir)) {
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
			if err != nil {
				t.Fatalf("parsing %s: %v", file, err)
			}
			ast.Inspect(parsed, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if named, ok := selector.X.(*ast.Ident); ok && named.Name == "setup" &&
					!slices.Contains(setupNamesTheApplicationUses, selector.Sel.Name) {
					t.Errorf("%s uses setup.%s, which the application may not reach (NFR-S-2)", file, selector.Sel.Name)
				}
				return true
			})
		}
	}
}

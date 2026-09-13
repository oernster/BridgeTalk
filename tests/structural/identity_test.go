package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/product"
)

// identityHome is the one file allowed to write the product's name down.
var identityHome = filepath.Join("internal", "product", "product.go")

// forbiddenInFileName lists the characters a Windows file name refuses. The colon is
// the dangerous one: the file system does not reject it, it reads what follows as an
// alternate data stream, so the file silently takes the name of its first word.
const forbiddenInFileName = `<>:"/\|?*`

// TestTheProductIsNamedOnce keeps the identity in one place.
//
// The name reaches a reader in the window title, the tray, the About dialog, the setup
// program, the Start Menu and the Apps list. The same identity also names the install
// directory, the executable, the uninstall key and a window class. Written out at each
// of those it was nine files and three spellings held together by memory, which is how
// a rename leaves one of them behind.
//
// Only string literals are examined. A comment naming the product is prose and reads
// better for saying it.
//
// Proved by writing the name into a literal elsewhere and reading the exit code.
func TestTheProductIsNamedOnce(t *testing.T) {
	root := repoRoot(t)
	home := filepath.Join(root, identityHome)

	files := append(goFiles(t), frontendFiles(t)...)
	files = append(files, setupFrontendFiles(t)...)

	for _, path := range files {
		if path == home {
			continue
		}
		if strings.HasSuffix(path, ".go") {
			checkGoLiterals(t, root, path)
			continue
		}
		checkText(t, root, path)
	}
}

// The setup program's front end is read whole, through setupFrontendFiles.
//
// It is included because nothing else reaches it: the front-end walk covers
// frontend/src and this is a second application's page, so it sat outside every scan.
// It wrote the product out in sixteen places and a rename went through every other
// surface without touching one of them, leaving the setup program a user ran
// announcing a product that no longer existed, with nothing anywhere to say so.
//
// Reading the directory rather than naming its page is what keeps that fixed. The page
// has since been split into markup, a stylesheet and two scripts; naming index.html
// would have let every string in it walk quietly out of scope.

// checkGoLiterals looks in a Go file's string literals and nowhere else.
func checkGoLiterals(t *testing.T, root, path string) {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Errorf("parsing %s: %v", path, err)
		return
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		report(t, root, path, literal.Value)
		return true
	})
}

// checkText looks at a front-end file whole, since a name in a comment there is as
// much a second copy as a name in a string.
func checkText(t *testing.T, root, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("reading %s: %v", path, err)
		return
	}
	report(t, root, path, string(raw))
}

// report fails the test when a piece of text spells the product out.
func report(t *testing.T, root, path, text string) {
	t.Helper()
	for _, form := range []string{product.Name, product.Slug} {
		if !strings.Contains(text, form) {
			continue
		}
		relative, _ := filepath.Rel(root, path)
		t.Errorf(
			"%s writes %q: the product is named in %s and read from there",
			filepath.ToSlash(relative), form, filepath.ToSlash(identityHome),
		)
	}
}

// TestTheIdentityCanBeAFileName keeps both forms usable where only a path will do.
//
// The slug names the executable and the install directory; the display name names the
// Start Menu shortcut, because a shortcut's file name is the label a reader sees. Both
// therefore have to survive being a file name.
//
// Proved by putting a colon into each and reading the exit code. It went in once for
// real, in a display name that opened with a word and a colon: every shortcut the
// installer wrote arrived labelled with that first word alone, with no error anywhere.
func TestTheIdentityCanBeAFileName(t *testing.T) {
	for _, form := range []string{product.Name, product.Slug} {
		for _, bad := range forbiddenInFileName {
			if strings.ContainsRune(form, bad) {
				t.Errorf(
					"%q contains %q, which cannot appear in a file name: the identity "+
						"names the executable, the install directory and a shortcut",
					form, bad,
				)
			}
		}
	}
	if strings.ContainsAny(product.Slug, " \t") {
		t.Errorf("the slug %q contains whitespace, which a path has to quote", product.Slug)
	}
}

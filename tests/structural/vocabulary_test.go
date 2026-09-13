package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// cueTable is game truth: every cue the application knows, keyed by an id spelled in
// the game's own words.
var cueTable = filepath.Join("internal", "infrastructure", "config", "cues.toml")

// cueID matches an id line in the cue table.
var cueID = regexp.MustCompile(`(?m)^\s*id\s*=\s*"([^"]+)"`)

// cueEvent and cueFlag match the other two halves of the vocabulary in the cue table:
// the journal event a cue listens for and the status value it watches.
var (
	cueEvent = regexp.MustCompile(`(?m)^\s*event\s*=\s*"([^"]+)"`)
	cueFlag  = regexp.MustCompile(`(?m)^\s*flag\s*=\s*"([^"]+)"`)
)

// vocabularyHomes are the only places a game word may be written down. The cue table
// is game truth. The status package decodes the game's own flag names, so it declares
// them once as constants; everything else reaches the vocabulary through those.
var vocabularyHomes = []string{
	filepath.Join("internal", "infrastructure", "config"),
	filepath.Join("internal", "infrastructure", "status", "flags.go"),
}

// gameVocabulary reads every word the cue table treats as game truth: the cue ids, the
// journal events they listen for and the status values they watch. It is read from the
// table rather than described by a pattern, so the guard below knows the real words and
// never has to guess which strings look like one.
func gameVocabulary(t *testing.T, root string) map[string]string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, cueTable))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(cueTable), err)
	}
	table := string(raw)

	words := map[string]string{}
	for kind, pattern := range map[string]*regexp.Regexp{
		"cue id":        cueID,
		"journal event": cueEvent,
		"status value":  cueFlag,
	} {
		for _, match := range pattern.FindAllStringSubmatch(table, -1) {
			words[match[1]] = kind
		}
	}
	if len(words) == 0 {
		t.Fatalf("no vocabulary found in %s, the patterns are wrong", filepath.ToSlash(cueTable))
	}
	return words
}

// isVocabularyHome reports whether a file is one of the places allowed to spell a game
// word out. Test files are included: a test naming the word it exercises is the point.
func isVocabularyHome(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if strings.HasSuffix(relative, "_test.go") {
		return true
	}
	for _, home := range vocabularyHomes {
		if relative == home || strings.HasPrefix(relative, home+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// structTags returns the position of every field tag in a file. A tag is the wire
// format of a file the game writes, which is a different concern from cue logic, so
// the guard reads past it.
func structTags(parsed *ast.File) map[token.Pos]bool {
	tags := map[token.Pos]bool{}
	ast.Inspect(parsed, func(node ast.Node) bool {
		if field, ok := node.(*ast.Field); ok && field.Tag != nil {
			tags[field.Tag.Pos()] = true
		}
		return true
	})
	return tags
}

// TestGameVocabularyStaysInItsHome keeps game knowledge out of logic.
//
// A cue id, a journal event name and a status value are the game's words, not this
// application's. The two-layer cue model exists so they live in the cue table and reach
// the code as data; a literal anywhere else is a second, silent copy of the table that
// no reader of the table can see. It compiles, it runs and it is wrong the day the
// table is edited and the literal is not.
//
// The vocabulary is read from the table rather than matched by shape, so a harmless
// path or identifier that merely looks like a cue id is never flagged; only a real
// game word is.
//
// Proved by planting a cue id in the composition root and reading the exit code.
func TestGameVocabularyStaysInItsHome(t *testing.T) {
	root := repoRoot(t)
	words := gameVocabulary(t, root)

	for _, path := range goFiles(t) {
		if isVocabularyHome(root, path) {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		tags := structTags(parsed)

		var escaped []string
		ast.Inspect(parsed, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING || tags[literal.Pos()] {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			if kind, known := words[value]; known {
				escaped = append(escaped, kind+" "+strconv.Quote(value))
			}
			return true
		})

		sort.Strings(escaped)
		relative, _ := filepath.Rel(root, path)
		for _, found := range escaped {
			t.Errorf(
				"%s writes the %s down: game words belong in the cue table and reach "+
					"the code as data, so read it rather than repeating it",
				filepath.ToSlash(relative), found,
			)
		}
	}
}

// TestNoCueIdEndsInADotOrASpace holds FR-222.
//
// Windows strips a trailing dot or space from a name as it creates it, so a folder made
// for such an id arrives under another name and its recordings are never found. The
// domain refuses such an id when a table loads; this names it against the shipped table
// before anything has to run.
//
// Proved by planting such an id in the cue table and reading the exit code.
func TestNoCueIdEndsInADotOrASpace(t *testing.T) {
	root := repoRoot(t)

	table, err := os.ReadFile(filepath.Join(root, cueTable))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(cueTable), err)
	}
	ids := cueID.FindAllStringSubmatch(string(table), -1)
	if len(ids) == 0 {
		t.Fatalf("no cue ids found in %s, the pattern is wrong", filepath.ToSlash(cueTable))
	}

	for _, match := range ids {
		if strings.HasSuffix(match[1], ".") || strings.HasSuffix(match[1], " ") {
			t.Errorf(
				"cue id %q ends in a dot or a space, which Windows strips from a file name: "+
					"a folder made for it would never be found (FR-222)",
				match[1],
			)
		}
	}
}

// TestNoCueIdHoldsAnUnderscore holds FR-230.
//
// A cue folder's name writes each dot as an underscore (FR-229). An id already holding an
// underscore could share a folder name with another id, so a folder could no longer be
// read back as exactly one cue. The domain refuses such an id when a table loads; this
// names it against the shipped table before anything has to run.
//
// Proved by planting such an id in the cue table and reading the exit code.
func TestNoCueIdHoldsAnUnderscore(t *testing.T) {
	root := repoRoot(t)

	table, err := os.ReadFile(filepath.Join(root, cueTable))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(cueTable), err)
	}
	ids := cueID.FindAllStringSubmatch(string(table), -1)
	if len(ids) == 0 {
		t.Fatalf("no cue ids found in %s, the pattern is wrong", filepath.ToSlash(cueTable))
	}

	for _, match := range ids {
		if strings.Contains(match[1], "_") {
			t.Errorf(
				"cue id %q holds an underscore, which a cue folder's name writes for a dot: "+
					"two ids could share one folder (FR-230)",
				match[1],
			)
		}
	}
}

// finalDigits matches an id whose last dot separated segment is made of digits alone.
var finalDigits = regexp.MustCompile(`(^|\.)[0-9]+$`)

// TestNoCueIdEndsInDigits holds FR-219.
//
// A take in the flat form is told from another take by a trailing dot and digits, so
// docking.granted.2.wav is the second take of docking.granted. An id ending in a segment
// of digits would give such a name two meanings: another take of one cue or the first
// take of a different one. No id has that shape today, which is a property of the
// vocabulary rather than a law, so this makes it one.
//
// Proved by planting such an id in the cue table and reading the exit code.
func TestNoCueIdEndsInDigits(t *testing.T) {
	root := repoRoot(t)

	table, err := os.ReadFile(filepath.Join(root, cueTable))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(cueTable), err)
	}
	ids := cueID.FindAllStringSubmatch(string(table), -1)
	if len(ids) == 0 {
		t.Fatalf("no cue ids found in %s, the pattern is wrong", filepath.ToSlash(cueTable))
	}

	for _, match := range ids {
		if finalDigits.MatchString(match[1]) {
			t.Errorf(
				"cue id %q ends in a segment of digits, which the flat form reads as a take "+
					"number: a file named for it could mean two cues (FR-219)",
				match[1],
			)
		}
	}
}

package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// wireContract is the front end's hand-written statement of the shapes the facade
// returns. It is wire.ts rather than api.ts beside it: the shapes moved there when api.ts
// reached the size cap; the shapes are the contract while the calls are not.
//
// Wails generates the same shapes into frontend/wailsjs at build time. That file is
// gitignored and imported by nothing, so it is not the contract. This one is, typed out
// by a person. Nothing in the build compares the two halves: the type checker sees the
// interface, the marshaller sees the struct and neither can see the other.
var wireContract = filepath.Join("frontend", "src", "wire.ts")

// dtoSuffix marks a struct as part of the wire. Naming is the only thing that says so,
// because Wails binds whatever the facade returns rather than a declared set of types.
const dtoSuffix = "DTO"

// wireShapes pairs each DTO with the interface that restates it.
//
// The list is here so that adding a shape is a decision taken twice, once in the code
// and once against this, the way the bound method surface is. It also carries the one
// pair whose names differ: CueDTO is CueEntry on the page, because Cue there would sit
// beside the cue field on a reaction log line and read as the same thing.
var wireShapes = map[string]string{
	"AboutDTO":           "About",
	"AuditionDTO":        "Audition",
	"ChatterDTO":         "Chatter",
	"ChatterCategoryDTO": "ChatterCategory",
	"ChatterMomentDTO":   "ChatterMoment",
	"ChecklistDTO":       "Checklist",
	"CueBreakdownDTO":    "CueBreakdown",
	"CueDTO":             "CueEntry",
	"GroupDTO":           "Group",
	"LineFailureDTO":     "LineFailure",
	"MachineVoiceDTO":    "MachineVoice",
	"MakingDTO":          "Making",
	"PlaybackDTO":        "Playback",
	"PluginVoiceDTO":     "PluginVoice",
	"ReactionDTO":        "Reaction",
	"StateDTO":           "State",
	"UpdateDTO":          "Update",
	"VoiceDTO":           "Voice",
	"VoiceFoldersDTO":    "VoiceFolders",
}

// tsInterface captures one interface and its body; tsField matches one field inside
// that body, with its optional mark and its type; tsAlias matches a type alias, such as a
// union of string literals; tsComment matches a block comment, which is removed first so
// that prose naming a field is never read as a declaration of one.
var (
	tsInterface = regexp.MustCompile(`(?s)export interface (\w+) \{(.*?)\n\}`)
	tsField     = regexp.MustCompile(`(?m)^\s+([A-Za-z_]\w*)(\??):\s*(.+?)\s*;?\s*$`)
	tsAlias     = regexp.MustCompile(`(?m)^export type (\w+) = (.+?)\s*;?\s*$`)
	tsLiteral   = regexp.MustCompile(`^\s*('[^']*'|"[^"]*")\s*$`)
	tsComment   = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

// wireField is one field as it crosses: the TypeScript type it reads as and whether it may be
// absent. On the Go side the type is what the marshaller sends, stated in TypeScript; absent is
// a field tagged omitempty, which the page must declare with a question mark.
type wireField struct {
	tsType   string
	optional bool
}

// goScalars are the Go types the facade sends, by the TypeScript type each arrives as.
var goScalars = map[string]string{
	"string": "string", "bool": "boolean",
	"int": "number", "int32": "number", "int64": "number", "float64": "number",
}

// omitEmpty is the json option that leaves a field out of the object when it is empty.
const omitEmpty = "omitempty"

// tsTypeOf states a Go field type in TypeScript: a scalar, another DTO by the interface paired
// with it, a slice as an array and a pointer as its type or null. Empty for a type the wire test
// cannot state, which is refused rather than passed over.
func tsTypeOf(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		if shape, ok := wireShapes[typed.Name]; ok {
			return shape
		}
		return goScalars[typed.Name]
	case *ast.ArrayType:
		if inner := tsTypeOf(typed.Elt); inner != "" && typed.Len == nil {
			return inner + "[]"
		}
	case *ast.StarExpr:
		if inner := tsTypeOf(typed.X); inner != "" {
			return inner + " | null"
		}
	}
	return ""
}

// fieldNames answers the names of a shape's fields in order.
func fieldNames(fields map[string]wireField) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// facadeFields reads the wire fields off one struct, taking the json tag rather than the Go
// name because the tag is what actually reaches the page.
func facadeFields(t *testing.T, path, name string, structure *ast.StructType) map[string]wireField {
	t.Helper()

	fields := map[string]wireField{}
	for _, field := range structure.Fields.List {
		if len(field.Names) == 0 || !field.Names[0].IsExported() {
			continue
		}
		if field.Tag == nil {
			t.Errorf(
				"%s: %s.%s has no json tag, so it ships under its Go name",
				filepath.ToSlash(path), name, field.Names[0].Name,
			)
			continue
		}
		raw, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			t.Fatalf("%s: unreadable tag on %s.%s", path, name, field.Names[0].Name)
		}
		options := strings.Split(reflect.StructTag(raw).Get("json"), ",")
		tag := options[0]
		if tag == "" || tag == "-" {
			continue
		}
		stated := tsTypeOf(field.Type)
		if stated == "" {
			t.Errorf("%s: %s.%s has a Go type this test cannot state in TypeScript; add it to goScalars",
				filepath.ToSlash(path), name, field.Names[0].Name)
		}
		fields[tag] = wireField{tsType: stated, optional: hasOption(options[1:], omitEmpty)}
	}
	return fields
}

// hasOption reports whether a json tag's options hold one.
func hasOption(options []string, want string) bool {
	for _, option := range options {
		if option == want {
			return true
		}
	}
	return false
}

// facadeDTOs returns every wire struct the facade declares, by json field name.
//
// Only the files at the repository root are read. The setup program is a second Wails
// application with a page of its own, so its DTOs answer to a different contract.
func facadeDTOs(t *testing.T, root string) map[string]map[string]wireField {
	t.Helper()

	out := map[string]map[string]wireField{}
	for _, path := range goFiles(t) {
		if filepath.Dir(path) != root || strings.HasSuffix(path, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		relative, _ := filepath.Rel(root, path)
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, spec := range general.Specs {
				typed, ok := spec.(*ast.TypeSpec)
				if !ok || !strings.HasSuffix(typed.Name.Name, dtoSuffix) {
					continue
				}
				if structure, ok := typed.Type.(*ast.StructType); ok {
					out[typed.Name.Name] = facadeFields(t, relative, typed.Name.Name, structure)
				}
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("no DTOs found at the repository root, the walk is wrong")
	}
	return out
}

// wireInterfaces returns every interface the front end declares, by field name.
func wireInterfaces(t *testing.T, root string) map[string]map[string]wireField {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, wireContract))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(wireContract), err)
	}
	source := tsComment.ReplaceAll(raw, nil)

	// An alias every member of which is a string literal arrives as a string.
	literals := map[string]bool{}
	for _, match := range tsAlias.FindAllSubmatch(source, -1) {
		every := true
		for _, member := range strings.Split(string(match[2]), "|") {
			every = every && tsLiteral.MatchString(member)
		}
		literals[string(match[1])] = every
	}

	out := map[string]map[string]wireField{}
	for _, match := range tsInterface.FindAllSubmatch(source, -1) {
		fields := map[string]wireField{}
		for _, field := range tsField.FindAllSubmatch(match[2], -1) {
			stated := strings.Join(strings.Fields(string(field[3])), " ")
			if literals[stated] {
				stated = "string"
			}
			fields[string(field[1])] = wireField{tsType: stated, optional: len(field[2]) > 0}
		}
		out[string(match[1])] = fields
	}
	if len(out) == 0 {
		t.Fatalf("no interfaces found in %s, the pattern is wrong", filepath.ToSlash(wireContract))
	}
	return out
}

// absent returns the fields of want that got does not hold.
func absent(want, got map[string]wireField) []string {
	var out []string
	for _, name := range fieldNames(want) {
		if _, present := got[name]; !present {
			out = append(out, name)
		}
	}
	return out
}

// TestTheWireContractMatchesOnBothSides keeps the two statements of the wire in step.
//
// The shapes the facade returns are written twice, as Go structs with json tags and as
// TypeScript interfaces, in two languages, by hand on one side. Nothing else compares
// them. A field renamed or removed on the Go side alone leaves the interface declaring
// something that never arrives, which the type checker cannot see and the page reads as
// undefined; the reverse leaves a value crossing on every call that nothing collects.
// Both halves of that have happened here, which is what this was written for.
//
// Proved by removing a field on each side in turn and reading the exit code.
func TestTheWireContractMatchesOnBothSides(t *testing.T) {
	root := repoRoot(t)
	structs := facadeDTOs(t, root)
	interfaces := wireInterfaces(t, root)

	var undeclared []string
	for name := range structs {
		if _, ok := wireShapes[name]; !ok {
			undeclared = append(undeclared, name)
		}
	}
	sort.Strings(undeclared)
	for _, name := range undeclared {
		t.Errorf(
			"%s crosses the wire but is not paired in wireShapes: a shape the page "+
				"receives is a decision, so it is declared there as well",
			name,
		)
	}

	names := make([]string, 0, len(wireShapes))
	for name := range wireShapes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		checkShape(t, name, wireShapes[name], structs, interfaces)
	}
}

// checkShape compares one pair, kept apart from the loop so each failure reads as a
// statement about that shape rather than about the pass as a whole.
func checkShape(t *testing.T, name, shape string, structs, interfaces map[string]map[string]wireField) {
	t.Helper()

	fields, ok := structs[name]
	if !ok {
		t.Errorf("wireShapes pairs %s, which no longer exists in the facade", name)
		return
	}
	declared, ok := interfaces[shape]
	if !ok {
		t.Errorf("%s has no interface %s in %s", name, shape, filepath.ToSlash(wireContract))
		return
	}
	for _, field := range absent(fields, declared) {
		t.Errorf(
			"%s.%s is sent but interface %s does not declare it: the page cannot read "+
				"a field it has not been told about",
			name, field, shape,
		)
	}
	for _, field := range absent(declared, fields) {
		t.Errorf(
			"interface %s declares %s but %s does not send it: it would read as "+
				"undefined at runtime with nothing to say so",
			shape, field, name,
		)
	}
	for _, field := range fieldNames(fields) {
		sent, read := fields[field], declared[field]
		if _, both := declared[field]; !both || sent.tsType == "" {
			continue
		}
		if sent.tsType != read.tsType {
			t.Errorf("%s.%s is sent as %s but interface %s reads it as %s",
				name, field, sent.tsType, shape, read.tsType)
		}
		if sent.optional != read.optional {
			t.Errorf("%s.%s is optional on one side alone (omitempty %v, ? %v): a field left out "+
				"reads as undefined where the page expects a value; otherwise a value is guarded for nothing",
				name, field, sent.optional, read.optional)
		}
	}
}

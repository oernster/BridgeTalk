package structural

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Focus belongs to controls (FR-713). A container is never a ring stop and never wears a ring. A
// list shows where focus is by its current row, never by a ring round the whole of it. A region
// that scrolls with nothing to select rings when the keyboard lands on it, never under the
// pointer, which rests inside a region for as long as it is open. Each is read where a stray rule
// or a stray stop would be written: the style sheets and the markup.

// setupScrollingRegion is the setup page's one region that is a stop because it scrolls.
const setupScrollingRegion = ".body"

// setupRingScript gathers the setup page's ring stops.
const setupRingScript = "setup-ring.js"

// controlTags are the elements a ring stop may be without being a region: each is acted on.
var controlTags = []string{"button", "input", "select", "textarea"}

var (
	// pseudoClass matches one pseudo-class, so a selector can be compared with what it names.
	pseudoClass = regexp.MustCompile(`:[a-z-]+(\([^)]*\))?`)
	// ringState matches a state a ring answers: the pointer over an element or focus on or in it.
	ringState = regexp.MustCompile(`:(hover|focus|focus-visible|focus-within)\b`)
	// pointerState matches the pointer over an element.
	pointerState = regexp.MustCompile(`:hover\b`)
	// ringDrawn matches a declaration drawing a ring in the ring colour.
	ringDrawn = regexp.MustCompile(`(border|border-color|outline|outline-color|box-shadow)\s*:[^;]*var\(--ring\)`)
	// containerSubject matches a subject that is every element or a bare element that only holds.
	containerSubject = regexp.MustCompile(`^(\*|html|body|div|span|section|main|nav|aside|header|footer|form|fieldset|ul|ol|li|table|thead|tbody|tr|td)$`)
	// listMarkup finds a list in the markup: the class of an element given the listbox role.
	listMarkup = regexp.MustCompile(`className="([^"]+)"[^<>]*?role="listbox"`)
	// stopTag finds the tag of the element a stop mark sits in; stopClass finds its class.
	stopTag   = regexp.MustCompile(`<([a-zA-Z]+)[^<]*$`)
	stopClass = regexp.MustCompile(`className="([^"]+)"`)
	// setupStops reads the selector the setup page gathers its stops with.
	setupStops = regexp.MustCompile(`querySelectorAll\('([^']+)'\)`)
)

// ringRule is one selector of a rule that draws a ring in some state.
type ringRule struct {
	file, selector string
}

// ringRules answers every selector, in the window's parts and the setup page's sheet, whose
// subject is in a ring state and whose rule draws a ring.
func ringRules(t *testing.T) []ringRule {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), setupFrontendDir, setupSheet))
	if err != nil {
		t.Fatalf("reading %s: %v", setupSheet, err)
	}
	var rings []ringRule
	for _, rule := range append(themeRules(t), parseRules(setupSheet, raw)...) {
		if !ringDrawn.MatchString(rule.body) {
			continue
		}
		for _, selector := range rule.selectors {
			if ringState.MatchString(subject(selector)) {
				rings = append(rings, ringRule{file: rule.file, selector: selector})
			}
		}
	}
	if len(rings) == 0 {
		t.Fatal("no ring rule found, the scan is wrong")
	}
	return rings
}

// subject answers the last compound of a selector: the element the rule styles.
func subject(selector string) string {
	parts := strings.FieldsFunc(selector, func(r rune) bool { return strings.ContainsRune(" >+~", r) })
	return parts[len(parts)-1]
}

// named answers a selector with its pseudo-classes taken out, which is what it names.
func named(selector string) string { return pseudoClass.ReplaceAllString(selector, "") }

// windowMarkup answers the window's markup, file by file, leaving the tests out.
func windowMarkup(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), filepath.Dir(themeDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(filepath.Dir(themeDir)), err)
	}
	markup := make(map[string]string)
	for _, entry := range entries {
		name := entry.Name()
		if filepath.Ext(name) != ".tsx" || strings.Contains(name, ".test.") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		markup[name] = string(raw)
	}
	return markup
}

// listClasses answers the selector of every list in the window's markup.
func listClasses(t *testing.T) []string {
	t.Helper()
	var lists []string
	for _, text := range windowMarkup(t) {
		for _, match := range listMarkup.FindAllStringSubmatch(text, -1) {
			lists = append(lists, "."+match[1])
		}
	}
	if len(lists) == 0 {
		t.Fatal("no list found in the markup, the scan is wrong")
	}
	return lists
}

func TestNoListWearsARing(t *testing.T) {
	lists := listClasses(t)
	for _, ring := range ringRules(t) {
		if slices.Contains(lists, named(ring.selector)) {
			t.Errorf("%s: %s rings a list; its current row shows where focus is", ring.file, ring.selector)
		}
	}
}

func TestNoScrollingRegionRingsUnderThePointer(t *testing.T) {
	regions := append(slices.Clone(scrollingRegions), setupScrollingRegion)
	for _, ring := range ringRules(t) {
		if pointerState.MatchString(subject(ring.selector)) && slices.Contains(regions, named(ring.selector)) {
			t.Errorf("%s: %s rings a region under the pointer, which rests inside it", ring.file, ring.selector)
		}
	}
}

func TestNoContainerWearsARing(t *testing.T) {
	for _, ring := range ringRules(t) {
		if containerSubject.MatchString(named(subject(ring.selector))) {
			t.Errorf("%s: %s rings a container rather than a control", ring.file, ring.selector)
		}
	}
}

// TestOnlyControlsListsAndScrollingRegionsAreStops holds the ring to what can be acted on. A
// container among the stops costs a press that lands on nothing.
func TestOnlyControlsListsAndScrollingRegionsAreStops(t *testing.T) {
	allowed := listClasses(t)
	for _, region := range scrollingRegions {
		allowed = append(allowed, subject(region))
	}
	found := 0
	for file, text := range windowMarkup(t) {
		for _, at := range regexp.MustCompile(`data-stop`).FindAllStringIndex(text, -1) {
			found++
			tag := stopTag.FindStringSubmatch(text[:at[0]])
			if tag != nil && slices.Contains(controlTags, tag[1]) {
				continue
			}
			class := stopClass.FindStringSubmatch(text[strings.LastIndex(text[:at[0]], "<"):at[0]])
			if class == nil || !slices.Contains(allowed, "."+class[1]) {
				t.Errorf("%s: a stop at byte %d is neither a control, a list nor a region that scrolls", file, at[0])
			}
		}
	}
	if found == 0 {
		t.Fatal("no stop found in the markup, the scan is wrong")
	}

	raw, err := os.ReadFile(filepath.Join(repoRoot(t), setupFrontendDir, setupRingScript))
	if err != nil {
		t.Fatalf("reading %s: %v", setupRingScript, err)
	}
	gathered := setupStops.FindStringSubmatch(string(raw))
	if gathered == nil {
		t.Fatalf("%s gathers no stops, the scan is wrong", setupRingScript)
	}
	for _, part := range strings.Split(gathered[1], ",") {
		part = strings.TrimSpace(part)
		tag := regexp.MustCompile(`^[a-z]+`).FindString(part)
		if part != setupScrollingRegion && !slices.Contains(controlTags, tag) {
			t.Errorf("%s gathers %s as a stop, which is neither a control nor its region that scrolls", setupRingScript, part)
		}
	}
}

package structural

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Two ring rules a stylesheet can drop without a word from anything. No test in the front
// end computes a style, so a rule deleted or never written leaves every test green while the
// window shows the wrong thing. Both belong to the keyboard model (FR-713).

// scrollingRegions are the regions that become a ring stop while they overflow. None has
// anything but its ring to show that focus has landed on it.
var scrollingRegions = []string{".log", ".dialog .body", ".guide-body"}

// cssComment, cssRule and dangerRing take a style part apart: comments out, then each rule
// as its selector list and its declarations, then whether a border wears the danger token.
var (
	cssComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssRule    = regexp.MustCompile(`([^{}]+)\{([^{}]*)\}`)
	dangerRing = regexp.MustCompile(`border(-color)?\s*:[^;]*var\(--danger\)`)
)

// styleRule is one rule of a style part: every selector it lists and what it declares.
type styleRule struct {
	file      string
	selectors []string
	body      string
}

// themeRules reads every rule in the window's style parts, each selector with its spacing
// made single so a selector written across a line break still compares as written.
func themeRules(t *testing.T) []styleRule {
	t.Helper()
	dir := filepath.Join(repoRoot(t), themeDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(themeDir), err)
	}
	var rules []styleRule
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".css" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", entry.Name(), err)
		}
		text := cssComment.ReplaceAllString(string(raw), "")
		for _, match := range cssRule.FindAllStringSubmatch(text, -1) {
			var selectors []string
			for _, selector := range strings.Split(match[1], ",") {
				selectors = append(selectors, strings.Join(strings.Fields(selector), " "))
			}
			rules = append(rules, styleRule{file: entry.Name(), selectors: selectors, body: match[2]})
		}
	}
	if len(rules) == 0 {
		t.Fatalf("no rules found under %s, the scan is wrong", filepath.ToSlash(themeDir))
	}
	return rules
}

// TestEveryDisabledControlWearsTheDangerRing holds the third ring state. A disabled control
// wears the danger ring at all times, so a control that is there but inert reads as that at
// a glance; a disabled chooser that went quiet instead is the fault this was written for.
func TestEveryDisabledControlWearsTheDangerRing(t *testing.T) {
	found := 0
	for _, rule := range themeRules(t) {
		for _, selector := range rule.selectors {
			if !strings.Contains(selector, ":disabled") {
				continue
			}
			found++
			if !dangerRing.MatchString(rule.body) {
				t.Errorf("%s: %s is disabled without the danger ring", rule.file, selector)
			}
		}
	}
	if found == 0 {
		t.Fatal("no disabled rule found, the scan is wrong")
	}
}

// TestEveryScrollingRegionRingsForTheKeyboard holds the ring on a region that is a stop
// because it scrolls. Without one, Tab lands on the guide and nothing on screen moves, which
// reads as a dead press.
func TestEveryScrollingRegionRingsForTheKeyboard(t *testing.T) {
	rules := themeRules(t)
	for _, region := range scrollingRegions {
		wanted := region + ":focus-visible"
		ringed := false
		for _, rule := range rules {
			for _, selector := range rule.selectors {
				if selector == wanted && strings.Contains(rule.body, "var(--ring)") {
					ringed = true
				}
			}
		}
		if !ringed {
			t.Errorf("%s has no keyboard ring, so focus landing on it shows nothing", region)
		}
	}
}

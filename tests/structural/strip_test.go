package structural

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The rules the Status cards, the strip along the foot of the window and its live indicator
// rest on (FR-716, FR-717, FR-719). jsdom reads no stylesheet, so no front-end test computes a
// style: a rule edited away would leave every test green while the window drew the wrong
// thing. The parts are read here with the hand the ring rules are read with.

// panesPart, navbandPart and footerPart are the style parts holding those rules;
// indicatorSource declares the tones the live indicator can take.
var (
	panesPart       = filepath.Join(themeDir, "panes.css")
	navbandPart     = filepath.Join(themeDir, "navband.css")
	footerPart      = filepath.Join(themeDir, "footer.css")
	indicatorSource = filepath.Join("frontend", "src", "indicator.ts")
)

// stripShare is the strip's height as a share of the band's: three quarters (FR-717).
const stripShare = "0.75"

// customProperty captures one token with its value; toneList captures the indicator's list
// of tones and toneName one tone within it.
var (
	customProperty = regexp.MustCompile(`--([\w-]+)\s*:\s*([^;]+);`)
	toneList       = regexp.MustCompile(`export const tones = \[([^\]]*)\] as const`)
	toneName       = regexp.MustCompile(`'(\w+)'`)
)

// readSource reads one front-end file whole.
func readSource(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), path))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(path), err)
	}
	return raw
}

// declared answers what a part's rules for one selector declare, each value with its spacing
// made single; empty where no rule names the selector.
func declared(t *testing.T, part, selector string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, rule := range parseRules(filepath.Base(part), readSource(t, part)) {
		if !slices.Contains(rule.selectors, selector) {
			continue
		}
		for _, declaration := range strings.Split(rule.body, ";") {
			if property, value, found := strings.Cut(declaration, ":"); found {
				out[strings.TrimSpace(property)] = strings.Join(strings.Fields(value), " ")
			}
		}
	}
	return out
}

// customProperties answers every token a part declares, each value with its spacing made
// single.
func customProperties(t *testing.T, part string) map[string]string {
	t.Helper()
	text := cssComment.ReplaceAllString(string(readSource(t, part)), "")
	out := map[string]string{}
	for _, match := range customProperty.FindAllStringSubmatch(text, -1) {
		out[match[1]] = strings.Join(strings.Fields(match[2]), " ")
	}
	return out
}

// expectDeclared fails where a rule does not declare a property exactly as wanted.
func expectDeclared(t *testing.T, part, selector, property, want string) {
	t.Helper()
	if got := declared(t, part, selector)[property]; got != want {
		t.Errorf(
			"%s: %s declares %s as %q, want %q",
			filepath.ToSlash(part), selector, property, got, want,
		)
	}
}

// expectLeading fails where a shorthand does not open with the wanted size. A border or a
// padding goes on to name more than the size, which is not what is held here.
func expectLeading(t *testing.T, part, selector, property, want string) {
	t.Helper()
	if got := declared(t, part, selector)[property]; !strings.HasPrefix(got, want+" ") {
		t.Errorf(
			"%s: %s declares %s as %q, want it to open with %s",
			filepath.ToSlash(part), selector, property, got, want,
		)
	}
}

// TestTheStatusCardsWidenToShareTheRow holds FR-716. auto-fill keeps an empty track wherever
// the width holds more columns than there are cards, which left a block of space beside them
// in a wide window; auto-fit widens the cards to share the row.
func TestTheStatusCardsWidenToShareTheRow(t *testing.T) {
	columns := declared(t, panesPart, ".cards")["grid-template-columns"]
	if !strings.HasPrefix(columns, "repeat(auto-fit,") {
		t.Errorf("%s: .cards lays its columns as %q, want repeat(auto-fit, ...)", filepath.ToSlash(panesPart), columns)
	}
}

// TestTheStripIsAShareOfTheBandDrawnFromItsOwnSizes holds FR-717. The strip is three quarters
// of the band, read from the band's sizes. Those sizes are the ones the band's own rules draw
// it with, so the strip measures the band on screen rather than a copy of it.
func TestTheStripIsAShareOfTheBandDrawnFromItsOwnSizes(t *testing.T) {
	strip, band := customProperties(t, footerPart), customProperties(t, navbandPart)
	for _, token := range []struct {
		tokens     map[string]string
		part, name string
		want       string
	}{
		{strip, footerPart, "strip-share", stripShare},
		{strip, footerPart, "strip-height", "calc(var(--band-height) * var(--strip-share))"},
		{band, navbandPart, "band-button", "calc(var(--band-icon) + 2 * (var(--band-button-pad) + var(--band-button-rule)))"},
		{band, navbandPart, "band-height", "calc(var(--band-button) + 2 * var(--band-pad) + var(--band-rule))"},
	} {
		if got := token.tokens[token.name]; got != token.want {
			t.Errorf("%s: --%s is %q, want %q", filepath.ToSlash(token.part), token.name, got, token.want)
		}
	}
	expectDeclared(t, footerPart, ".strip", "height", "var(--strip-height)")
	expectLeading(t, navbandPart, ".navband", "padding", "var(--band-pad)")
	expectLeading(t, navbandPart, ".navband", "border-bottom", "var(--band-rule)")
	expectDeclared(t, navbandPart, ".navbtn", "padding", "var(--band-button-pad)")
	expectLeading(t, navbandPart, ".navbtn", "border", "var(--band-button-rule)")
	expectDeclared(t, navbandPart, ".navbtn .icon", "height", "var(--band-icon)")
}

// TestTheStripsLabelsOpenAboveTheirControls holds FR-717. The band's labels open below their
// controls, which at the foot of the window would fall outside it; the left-most control's
// label starts at its own left edge rather than hanging past the window's.
func TestTheStripsLabelsOpenAboveTheirControls(t *testing.T) {
	expectDeclared(t, footerPart, ".strip [data-label]::after", "top", "auto")
	expectDeclared(t, footerPart, ".strip [data-label]::after", "bottom", "calc(100% + var(--tip-gap))")
	expectDeclared(t, footerPart, ".strip > .navbtn:first-child::after", "left", "0")
	expectDeclared(t, footerPart, ".strip > .navbtn:first-child::after", "transform", "none")
}

// TestTheStripIsReadAfterTheBand keeps the strip's part after the band's in the manifest, so
// the band's sizes and labels are read before the rules that derive from them and override them.
func TestTheStripIsReadAfterTheBand(t *testing.T) {
	manifest := string(readSource(t, styleManifest))
	band := strings.Index(manifest, "'./theme/navband.css'")
	strip := strings.Index(manifest, "'./theme/footer.css'")
	if band < 0 || strip < band {
		t.Errorf("%s reads footer.css at %d and navband.css at %d: the strip must come after the band", filepath.ToSlash(styleManifest), strip, band)
	}
}

// TestEveryIndicatorToneHasItsColourToken holds FR-719: each tone the indicator can take is
// drawn in the colour token of the same name, so its colours stay in the palette.
func TestEveryIndicatorToneHasItsColourToken(t *testing.T) {
	list := toneList.FindSubmatch(readSource(t, indicatorSource))
	if list == nil {
		t.Fatalf("%s declares no tones list, the pattern is wrong", filepath.ToSlash(indicatorSource))
	}
	names := toneName.FindAllSubmatch(list[1], -1)
	if len(names) == 0 {
		t.Fatalf("%s lists no tones, the pattern is wrong", filepath.ToSlash(indicatorSource))
	}
	for _, name := range names {
		tone := string(name[1])
		expectDeclared(t, footerPart, ".strip .indicator.tone-"+tone, "color", "var(--"+tone+")")
	}
}

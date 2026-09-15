package structural

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// secondaryRule is the style part holding the rule the secondary text lines share;
// themeTokens is the palette that rule's colour is read from.
var (
	secondaryRule = filepath.Join("frontend", "src", "theme", "controls.css")
	themeTokens   = filepath.Join("frontend", "src", tokenFile)
)

// secondaryLines are the lines drawn in the theme's secondary text colour: the Missing
// takes purpose line (FR-318) and the Status cards' taglines (FR-716). They share one rule,
// so the one token measured here is the colour every one of them is drawn in.
var secondaryLines = []string{".row .purpose", ".card .tagline"}

// minimumSecondaryContrast is the ratio FR-318 requires: the WCAG 2 enhanced level for
// normal text.
const minimumSecondaryContrast = 7.0

// switchRules is the style part drawing Chatter's switches; switchParts are the selectors whose
// fills FR-736 measures, the thumb and the track while on.
var (
	switchRules = filepath.Join("frontend", "src", "theme", "chatter.css")
	switchParts = []string{".switch .thumb", ".switch[aria-checked='true']"}
)

// minimumSwitchContrast is the ratio FR-736 requires: WCAG 2.2 non-text contrast at level AA.
const minimumSwitchContrast = 3.0

// themeSelectors name the palette's two blocks, light then dark; backgroundTokens are the
// grounds a row can stand on.
var (
	themeSelectors   = []string{":root", ":root[data-theme='dark']"}
	backgroundTokens = []string{"surface", "panel"}
)

// The WCAG 2 relative luminance formula: an sRGB channel at or under the threshold is
// divided down; above it the offset, scale and exponent linearise it; the channels are
// then weighted. The flare offset is added to both sides of a contrast ratio.
const (
	channelMax      = 255.0
	hexDigits       = 2
	linearThreshold = 0.03928
	linearDivisor   = 12.92
	curveOffset     = 0.055
	curveScale      = 1.055
	curveExponent   = 2.4
	redWeight       = 0.2126
	greenWeight     = 0.7152
	blueWeight      = 0.0722
	flareOffset     = 0.05
)

// ruleColour captures the token a rule's own color declaration reads, passing over a
// background-color or a border-color; ruleFill captures the token its background reads;
// themeBlock captures one palette block by selector; tokenLine captures one token's hex value.
var (
	ruleColour = regexp.MustCompile(`(?:^|[\s;])color:\s*var\(--([\w-]+)\)`)
	ruleFill   = regexp.MustCompile(`(?:^|[\s;])background(?:-color)?:\s*var\(--([\w-]+)\)`)
	themeBlock = regexp.MustCompile(`(?ms)^(:root[^{\n]*?)\s*\{(.*?)^\}`)
	tokenLine  = regexp.MustCompile(`--([\w-]+):\s*(#[0-9a-fA-F]{6})\s*;`)
)

// luminance reads a six digit hex colour as WCAG 2 relative luminance.
func luminance(t *testing.T, hex string) float64 {
	t.Helper()
	weights := []float64{redWeight, greenWeight, blueWeight}
	total := 0.0
	for index, weight := range weights {
		start := 1 + index*hexDigits
		value, err := strconv.ParseUint(hex[start:start+hexDigits], 16, 8)
		if err != nil {
			t.Fatalf("reading %s: %v", hex, err)
		}
		channel := float64(value) / channelMax
		if channel <= linearThreshold {
			channel /= linearDivisor
		} else {
			channel = math.Pow((channel+curveOffset)/curveScale, curveExponent)
		}
		total += weight * channel
	}
	return total
}

// contrast returns the WCAG 2 contrast ratio between two colours, lighter over darker.
func contrast(t *testing.T, first, second string) float64 {
	t.Helper()
	lighter, darker := luminance(t, first), luminance(t, second)
	if darker > lighter {
		lighter, darker = darker, lighter
	}
	return (lighter + flareOffset) / (darker + flareOffset)
}

// palettes reads each block of the palette into its tokens, keyed by selector.
func palettes(t *testing.T, root string) map[string]map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, themeTokens))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(themeTokens), err)
	}
	out := map[string]map[string]string{}
	for _, block := range themeBlock.FindAllSubmatch(raw, -1) {
		tokens := map[string]string{}
		for _, line := range tokenLine.FindAllSubmatch(block[2], -1) {
			tokens[string(line[1])] = string(line[2])
		}
		out[string(block[1])] = tokens
	}
	return out
}

// secondaryToken reads the token drawn by the one rule that names every secondary line.
// A line given a rule of its own is not found here, since its colour would then be a second
// statement this test never measures.
func secondaryToken(t *testing.T, root string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, secondaryRule))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(secondaryRule), err)
	}
	for _, rule := range parseRules(filepath.Base(secondaryRule), raw) {
		if !namesEvery(rule.selectors, secondaryLines) {
			continue
		}
		if found := ruleColour.FindStringSubmatch(rule.body); found != nil {
			return found[1]
		}
	}
	t.Fatalf(
		"%s has no one rule naming %s that draws its colour from a token",
		filepath.ToSlash(secondaryRule), strings.Join(secondaryLines, " and "),
	)
	return ""
}

// namesEvery reports whether a rule's selectors include every one wanted.
func namesEvery(selectors, wanted []string) bool {
	for _, each := range wanted {
		if !slices.Contains(selectors, each) {
			return false
		}
	}
	return true
}

// TestTheSecondaryLinesContrastInBothThemes holds FR-318 for every line drawn in the
// secondary text colour, the Status cards' taglines of FR-716 among them: whatever token
// their shared rule draws in reads at 7 to 1 or better against every ground a row stands
// on, in each theme. The token is read from the rule rather than named here, so recolouring
// the lines is measured rather than trusted.
//
// Proved by pointing the rule at the muted token, which measures under 7 to 1 in the light
// theme, then reading the exit code. A paler light secondary token was caught too.
func TestTheSecondaryLinesContrastInBothThemes(t *testing.T) {
	root := repoRoot(t)
	holdsContrast(t, root, secondaryToken(t, root), "the secondary lines", minimumSecondaryContrast, "FR-318")
}

// fillToken reads the token the one rule naming selector in the switches' style part fills with.
func fillToken(t *testing.T, root, selector string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, switchRules))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(switchRules), err)
	}
	for _, rule := range parseRules(filepath.Base(switchRules), raw) {
		if !slices.Contains(rule.selectors, selector) {
			continue
		}
		if found := ruleFill.FindStringSubmatch(rule.body); found != nil {
			return found[1]
		}
	}
	t.Fatalf("%s has no rule for %s that fills from a token", filepath.ToSlash(switchRules), selector)
	return ""
}

// TestTheChatterSwitchesContrastInBothThemes holds FR-736: the thumb of every switch and the track
// of a switch while on each read at 3 to 1 or better against every ground a row stands on, in each
// theme. The tokens are read from the rules rather than named here, so recolouring a switch is
// measured rather than trusted.
func TestTheChatterSwitchesContrastInBothThemes(t *testing.T) {
	root := repoRoot(t)
	for _, part := range switchParts {
		holdsContrast(t, root, fillToken(t, root, part), part, minimumSwitchContrast, "FR-736")
	}
}

// holdsContrast measures one token, drawn for what, against every ground a row stands on in each
// theme, failing wherever it reads under minimum, the ratio the requirement named asks for.
func holdsContrast(t *testing.T, root, token, what string, minimum float64, requirement string) {
	t.Helper()
	themes := palettes(t, root)
	for _, selector := range themeSelectors {
		tokens, ok := themes[selector]
		if !ok {
			t.Errorf("%s has no %s block", tokenFile, selector)
			continue
		}
		fore, ok := tokens[token]
		if !ok {
			t.Errorf("%s defines no --%s for %s", selector, token, what)
			continue
		}
		for _, ground := range backgroundTokens {
			back, ok := tokens[ground]
			if !ok {
				t.Errorf("%s defines no --%s", selector, ground)
				continue
			}
			if ratio := contrast(t, fore, back); ratio < minimum {
				t.Errorf(
					"%s: %s, --%s on --%s, measures %.2f to 1, under the %.0f to 1 %s requires",
					selector, what, token, ground, ratio, minimum, requirement,
				)
			}
		}
	}
}

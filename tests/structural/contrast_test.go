package structural

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// purposeRule is the style part holding the Missing takes purpose line's rule;
// themeTokens is the palette that rule's colour is read from.
var (
	purposeRule = filepath.Join("frontend", "src", "theme", "controls.css")
	themeTokens = filepath.Join("frontend", "src", tokenFile)
)

// minimumPurposeContrast is the ratio FR-318 requires: the WCAG 2 enhanced level for
// normal text.
const minimumPurposeContrast = 7.0

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

// purposeColour captures the token the purpose rule draws its colour from; themeBlock
// captures one palette block by selector; tokenLine captures one token's hex value.
var (
	purposeColour = regexp.MustCompile(`(?s)\.row \.purpose \{[^}]*?color:\s*var\(--([\w-]+)\)`)
	themeBlock    = regexp.MustCompile(`(?ms)^(:root[^{\n]*?)\s*\{(.*?)^\}`)
	tokenLine     = regexp.MustCompile(`--([\w-]+):\s*(#[0-9a-fA-F]{6})\s*;`)
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

// TestThePurposeLineContrastsInBothThemes holds FR-318: whatever token the purpose line
// is drawn in reads at 7 to 1 or better against every ground a row stands on, in each
// theme. The token is read from the rule rather than named here, so recolouring the line
// is measured rather than trusted.
//
// Proved by pointing the rule at the muted token, which measures under 7 to 1 in the
// light theme, then reading the exit code. A paler light secondary token was caught too.
func TestThePurposeLineContrastsInBothThemes(t *testing.T) {
	root := repoRoot(t)
	rule, err := os.ReadFile(filepath.Join(root, purposeRule))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(purposeRule), err)
	}
	found := purposeColour.FindSubmatch(rule)
	if found == nil {
		t.Fatalf("%s has no .row .purpose rule drawing its colour from a token", filepath.ToSlash(purposeRule))
	}
	token := string(found[1])

	themes := palettes(t, root)
	for _, selector := range themeSelectors {
		tokens, ok := themes[selector]
		if !ok {
			t.Errorf("%s has no %s block", tokenFile, selector)
			continue
		}
		fore, ok := tokens[token]
		if !ok {
			t.Errorf("%s defines no --%s for the purpose line", selector, token)
			continue
		}
		for _, ground := range backgroundTokens {
			back, ok := tokens[ground]
			if !ok {
				t.Errorf("%s defines no --%s", selector, ground)
				continue
			}
			if ratio := contrast(t, fore, back); ratio < minimumPurposeContrast {
				t.Errorf(
					"%s: --%s on --%s measures %.2f to 1, under the %.0f to 1 FR-318 requires",
					selector, token, ground, ratio, minimumPurposeContrast,
				)
			}
		}
	}
}

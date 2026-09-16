package structural

// FR-813: the flatpak's sandbox is granted what the application uses and no more. build_flatpak.sh
// writes the manifest from its GRANTS list, so the list is held here to the requirement's grants,
// each once, with no network among them; its APP_ID is held to the product's own id, which the
// sign-in entry and the tray are named with too.
//
// What this cannot see: whether each grant is enough on a real desktop, which only running the
// flatpak shows.

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/product"
)

// flatpakScript is the script that writes the manifest.
const flatpakScript = "build_flatpak.sh"

// requiredGrants are FR-813's grants.
var requiredGrants = []string{
	"--share=ipc",
	"--socket=wayland",
	"--socket=fallback-x11",
	"--device=dri",
	"--socket=pulseaudio",
	"--filesystem=home",
	"--filesystem=~/.var/app/com.valvesoftware.Steam:ro",
	"--filesystem=xdg-config/autostart:create",
	"--talk-name=org.kde.StatusNotifierWatcher",
}

// grantsBlock is the GRANTS array in the script, from its opening line to its closing bracket.
var grantsBlock = regexp.MustCompile(`(?m)^GRANTS=\(\n((?:.*\n)*?)\)$`)

// appIDLine is the script's application id.
var appIDLine = regexp.MustCompile(`(?m)^APP_ID="([^"]*)"$`)

// flatpakGrants reads the grants a script's GRANTS array holds, in order, passing over blank lines and
// comments.
func flatpakGrants(t *testing.T, script string) []string {
	t.Helper()
	block := grantsBlock.FindStringSubmatch(script)
	if block == nil {
		t.Fatalf("%s holds no GRANTS array", flatpakScript)
	}
	var grants []string
	for _, line := range strings.Split(block[1], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		grants = append(grants, line)
	}
	return grants
}

func TestTheFlatpakIsGrantedWhatItUsesAndNoMore(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), flatpakScript))
	if err != nil {
		t.Fatalf("reading %s: %v", flatpakScript, err)
	}
	script := string(raw)

	grants := flatpakGrants(t, script)
	if !slices.Equal(sorted(grants), sorted(requiredGrants)) {
		t.Errorf("GRANTS = %v, want exactly %v", grants, requiredGrants)
	}
	for _, grant := range grants {
		if strings.Contains(grant, "network") {
			t.Errorf("GRANTS holds %s; the application makes no request (NFR-S-1)", grant)
		}
	}
	if !strings.Contains(script, `for grant in "${GRANTS[@]}"; do`) {
		t.Errorf("%s does not write its manifest's grants from GRANTS", flatpakScript)
	}

	id := appIDLine.FindStringSubmatch(script)
	if id == nil || id[1] != product.AppID {
		t.Errorf("%s names the application %v, want %q", flatpakScript, id, product.AppID)
	}
}

// sorted answers a sorted copy, so two lists can be compared as sets of lines that each appear once.
func sorted(lines []string) []string {
	out := slices.Clone(lines)
	slices.Sort(out)
	return out
}

package setup

import (
	"archive/zip"
	"bytes"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// Setup makes the plugins folder inside the install directory, so nobody has to create one by
// name in the right place (FR-576). An update or a repair makes it again over one already there,
// and whatever is in that one stays exactly as it was.
func TestThePluginsFolderIsMadeAndWhatIsInItIsLeftAlone(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := MakePluginsFolder(dir); err != nil {
		t.Fatalf("making the folder: %v", err)
	}
	folder := filepath.Join(dir, product.PluginsFolder)
	if info, err := os.Stat(folder); err != nil || !info.IsDir() {
		t.Fatalf("no plugins folder at %s: %v", folder, err)
	}

	plugin := filepath.Join(folder, "crew.dll")
	if err := os.WriteFile(plugin, []byte("the user's"), dirPerm); err != nil {
		t.Fatalf("writing a plugin: %v", err)
	}
	if err := MakePluginsFolder(dir); err != nil {
		t.Fatalf("making the folder a second time: %v", err)
	}
	if held, err := os.ReadFile(plugin); err != nil || string(held) != "the user's" {
		t.Errorf("the plugin reads %q after the folder was made again: %v", held, err)
	}
}

// A folder that cannot be made is refused naming it once, with the reason (FR-237).
func TestAPluginsFolderThatCannotBeMadeIsRefused(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A plain file where the install directory should be means nothing can be made inside it.
	blocked := filepath.Join(dir, "not a folder")
	if err := os.WriteFile(blocked, []byte("in the way"), dirPerm); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	err := MakePluginsFolder(blocked)

	if problems := refusal.Check(err, filepath.Join(blocked, product.PluginsFolder)); len(problems) != 0 {
		t.Errorf("the refusal reads wrong: %v", problems)
	}
}

// Uninstall offers to keep the plugins folder only where there is something in it to lose
// (FR-578). A subfolder counts as much as a file.
func TestOnlyAPluginsFolderHoldingSomethingIsOfferedToKeep(t *testing.T) {
	t.Parallel()

	absent := t.TempDir()
	if got := KeepablePlugins(absent); got != "" {
		t.Errorf("with no plugins folder, offered to keep %q", got)
	}

	empty := t.TempDir()
	mustMakeFolder(t, filepath.Join(empty, product.PluginsFolder))
	if got := KeepablePlugins(empty); got != "" {
		t.Errorf("with an empty plugins folder, offered to keep %q", got)
	}

	holding := t.TempDir()
	mustMakeFolder(t, filepath.Join(holding, product.PluginsFolder, "crew data"))
	if got, want := KeepablePlugins(holding), filepath.Join(holding, product.PluginsFolder); got != want {
		t.Errorf("with a subfolder in the plugins folder, offered %q, want %q", got, want)
	}
}

// A plugins folder that is there and cannot be read might hold anything, so keeping it is
// offered rather than risking the user's plugins going without a word (FR-578).
func TestAPluginsFolderThatCannotBeReadIsOfferedToKeep(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	refuses := func(string) ([]os.DirEntry, error) { return nil, errors.New("access is denied") }

	if got, want := keepablePlugins(dir, refuses), filepath.Join(dir, product.PluginsFolder); got != want {
		t.Errorf("a folder that cannot be read offered %q, want %q", got, want)
	}
}

// The uninstall delete keeps the plugins folder only where it holds something and removing it was
// not asked for (FR-578). An empty folder is never left behind in an install directory that has
// otherwise gone.
func TestTheUninstallKeepsThePluginsOnlyWhereThereIsSomethingToKeep(t *testing.T) {
	t.Parallel()
	holding := t.TempDir()
	mustMakeFolder(t, filepath.Join(holding, product.PluginsFolder, "crew data"))
	empty := t.TempDir()
	mustMakeFolder(t, filepath.Join(empty, product.PluginsFolder))

	cases := []struct {
		name          string
		dir           string
		removePlugins bool
		want          string
	}{
		{"holding something, kept", holding, false, product.PluginsFolder},
		{"holding something, removal asked for", holding, true, ""},
		{"empty, nothing to keep", empty, false, ""},
	}
	for _, each := range cases {
		if got := KeptOnUninstall(each.dir, each.removePlugins); got != each.want {
			t.Errorf("%s: kept %q, want %q", each.name, got, each.want)
		}
	}
}

// An update and a repair write the payload over the install directory, then make the plugins
// folder; every file already in that folder is left as it was found, with nothing added and
// nothing taken away (FR-577).
func TestAnUpdateLeavesThePluginsFolderAsItFoundIt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	held := map[string]string{
		"crew.dll":             "the user's plugin",
		"crew data/voices.txt": "what it keeps beside itself",
		"notes about crew.txt": "the user's own notes",
		ExeName + ".mine":      "a name close to the application's",
		"assets/readme.txt":    "a name the payload also carries",
	}
	for name, body := range held {
		plant(t, PluginsDir(dir), name, body)
	}
	payload := zipOf(t, map[string]string{ExeName: "the new program", "assets/readme.txt": "a readme"})

	if err := ExtractZip(payload, dir); err != nil {
		t.Fatalf("extracting: %v", err)
	}
	if err := MakePluginsFolder(dir); err != nil {
		t.Fatalf("making the plugins folder: %v", err)
	}

	found := map[string]string{}
	err := filepath.WalkDir(PluginsDir(dir), func(at string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		body, err := os.ReadFile(at)
		name, _ := filepath.Rel(PluginsDir(dir), at)
		found[filepath.ToSlash(name)] = string(body)
		return err
	})
	if err != nil {
		t.Fatalf("reading the plugins folder back: %v", err)
	}
	if !maps.Equal(found, held) {
		t.Errorf("the plugins folder holds %v after the update, want %v", found, held)
	}
}

// The payload never carries a plugins folder, whatever the built application's folder holds, so
// an update has nothing to write over the user's plugins with (FR-577).
func TestThePayloadNeverCarriesAPluginsFolder(t *testing.T) {
	t.Parallel()
	app := t.TempDir()
	plant(t, app, ExeName, "the program")
	plant(t, app, product.PluginsFolder+"/crew.dll", "a plugin left from trying the build")
	plant(t, app, "resources/"+product.PluginsFolder+"/icon.png", "not the plugins folder")
	var packed bytes.Buffer

	if err := Pack(&packed, Payload{App: app}); err != nil {
		t.Fatalf("packing: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(packed.Bytes()), int64(packed.Len()))
	if err != nil {
		t.Fatalf("reading the archive: %v", err)
	}
	var names []string
	for _, member := range reader.File {
		names = append(names, member.Name)
	}
	slices.Sort(names)
	want := []string{"resources/" + product.PluginsFolder + "/icon.png", ExeName}
	slices.Sort(want)
	if !slices.Equal(names, want) {
		t.Errorf("the payload carries %v, want %v", names, want)
	}
}

// mustMakeFolder makes folder and everything above it.
func mustMakeFolder(t *testing.T, folder string) {
	t.Helper()
	if err := os.MkdirAll(folder, dirPerm); err != nil {
		t.Fatalf("making %s: %v", folder, err)
	}
}

// Identity: the version, the credits and the attribution.
//
// Everything here is embedded in the binary, so the application is one file and the
// About dialog cannot disagree with what was built. The version is read from the
// VERSION file at build time rather than written as a literal anywhere.
package main

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed VERSION
var versionFile string

// version is the single version string, trimmed of the file's trailing newline.
var version = strings.TrimSpace(versionFile)

// Identity shown in the About dialog. None of it is duplicated in the front end.
const (
	appTagline   = "A ship's voice for Elite Dangerous, speaking with a machine voice of its own or with recordings you supply"
	appAuthor    = "Oliver Ernster"
	appCopyright = "© 2026 Oliver Ernster"
)

// appAuthorship and appAttribution are statements of fact. appLicence names the terms
// in a sentence; the full text is the LICENSE file, embedded below, which Help shows
// whole so the dialog and the source can never carry different terms.
const (
	appAuthorship  = "This application is the sole work of Oliver Ernster."
	appAttribution = "Its machine voices speak lines made on this computer by the Kokoro-82M " +
		"model; the application ships no recordings. It also plays recordings you supply, " +
		"from a directory you choose. It never changes or removes them; the one thing it " +
		"writes there is empty folders you ask it to make."
	appLicence = "Free software under the GNU General Public License, version 3. " +
		"Help, then Licence, shows the full terms."
)

//go:embed LICENSE
var licenceText string

// About returns the identity and dependency credits for the About dialog.
func (a *App) About() AboutDTO {
	return AboutDTO{
		Name:        appTitle,
		Tagline:     appTagline,
		Version:     version,
		Author:      appAuthor,
		Copyright:   appCopyright,
		Authorship:  appAuthorship,
		Attribution: appAttribution,
		Licence:     appLicence,
		Credits:     credits(),
	}
}

// Licence returns the full terms for the dialog under Help: the LICENSE file itself.
func (a *App) Licence() string { return licenceText }

// credit is one dependency the binary ships: the Go module it comes from where it is one, the name
// it is credited under, its licence and what it does here.
type credit struct{ module, name, licence, role string }

// shipped is every dependency the binary ships, maintained by hand so each credit names the right
// people in words a reader follows.
var shipped = []credit{
	{"", "Go standard library", "BSD-3-Clause", "the language and its runtime"},
	{"github.com/wailsapp/wails/v2", "Wails v2", "MIT", "the desktop shell"},
	{"", "React and React DOM", "MIT", "the user interface"},
	{"", "Vite", "MIT", "the front-end build and its module preload helper"},
	{"", "ONNX Runtime", "MIT, Microsoft Corporation", "running the voice model"},
	{"", "Kokoro-82M", "Apache-2.0, by hexgrad, converted to ONNX by onnx-community", "the voice model and its voices"},
	{"github.com/gopxl/beep/v2", "beep", "MIT", "audio decoding, mixing and playback"},
	{"github.com/ebitengine/oto/v3", "oto", "Apache-2.0", "the audio output device"},
	{"github.com/ebitengine/purego", "purego", "Apache-2.0", "calling the system audio API without cgo"},
	{"github.com/hajimehoshi/go-mp3", "go-mp3", "Apache-2.0", "MP3 decoding"},
	{"github.com/jfreymuth/oggvorbis", "oggvorbis", "MIT", "Ogg Vorbis decoding"},
	{"github.com/mewkiz/flac", "flac", "Unlicense", "FLAC decoding"},
	{"github.com/BurntSushi/toml", "BurntSushi/toml", "MIT", "the cue table and voice manifests"},
	{"golang.org/x/sys", "golang.org/x/sys", "BSD-3-Clause", "the Windows tray"},
	{"github.com/go-ole/go-ole", "go-ole", "MIT", "writing the Start Menu and Desktop shortcuts"},
	// Linked in by the modules above rather than named by this application; each role says which.
	{"github.com/wailsapp/go-webview2", "go-webview2", "MIT", "used by Wails"},
	{"github.com/wailsapp/mimetype", "mimetype", "MIT", "used by Wails"},
	{"git.sr.ht/~jackmordaunt/go-toast/v2", "go-toast", "MIT or Unlicense", "used by Wails"},
	{"github.com/bep/debounce", "debounce", "MIT", "used by Wails"},
	{"github.com/google/uuid", "google/uuid", "BSD-3-Clause", "used by Wails"},
	{"github.com/leaanthony/go-ansi-parser", "go-ansi-parser", "MIT", "used by Wails"},
	{"github.com/leaanthony/slicer", "slicer", "MIT", "used by Wails"},
	{"github.com/leaanthony/u", "leaanthony/u", "MIT", "used by Wails"},
	{"github.com/pkg/browser", "pkg/browser", "BSD-2-Clause", "used by Wails"},
	{"github.com/samber/lo", "lo", "MIT", "used by Wails"},
	{"github.com/tkrajina/go-reflector", "go-reflector", "Apache-2.0", "used by Wails"},
	{"golang.org/x/net", "golang.org/x/net", "BSD-3-Clause", "used by Wails and mimetype"},
	{"github.com/pkg/errors", "pkg/errors", "BSD-2-Clause", "used by beep and Wails"},
	{"github.com/rivo/uniseg", "uniseg", "MIT", "used by go-ansi-parser"},
	{"golang.org/x/text", "golang.org/x/text", "BSD-3-Clause", "used by lo"},
	{"github.com/jfreymuth/vorbis", "vorbis", "MIT", "used by oggvorbis"},
	{"github.com/icza/bitio", "bitio", "Apache-2.0", "used by flac"},
	{"github.com/mewkiz/pkg", "mewkiz/pkg", "Unlicense", "used by flac"},
}

// credits renders every credit the way the About dialog lists it.
func credits() []string {
	out := make([]string, 0, len(shipped))
	for _, each := range shipped {
		out = append(out, fmt.Sprintf("%s - %s (%s)", each.name, each.licence, each.role))
	}
	return out
}

// Identity: the version, the credits and the attribution.
//
// Everything here is embedded in the binary, so the application is one file and the
// About dialog cannot disagree with what was built. The version is read from the
// VERSION file at build time rather than written as a literal anywhere.
package main

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// version is the single version string, trimmed of the file's trailing newline.
var version = strings.TrimSpace(versionFile)

// Identity shown in the About dialog. None of it is duplicated in the front end.
const (
	appTagline   = "A ship's voice for Elite Dangerous, speaking with recordings you supply"
	appAuthor    = "Oliver Ernster"
	appCopyright = "© 2026 Oliver Ernster"
)

// appAuthorship and appAttribution are statements of fact. appLicence names the terms
// in a sentence; the full text is the LICENSE file, embedded below, which Help shows
// whole so the dialog and the source can never carry different terms.
const (
	appAuthorship  = "This application is the sole work of Oliver Ernster."
	appAttribution = "This application ships no audio. It plays recordings you " +
		"supply, from a directory you choose. It never writes to them."
	appLicence = "Free software under the GNU General Public License, version 3. " +
		"Help, then Licence, shows the full terms."
)

//go:embed LICENSE
var licenceText string

// credits names every dependency the binary ships, with its licence. The list is
// maintained by hand on purpose: a generated one would list the build graph rather
// than what is actually linked in; the point is to credit the right people.
func credits() []string {
	return []string{
		"Go standard library - BSD-3-Clause (the language and its runtime)",
		"Wails v2 - MIT (the desktop shell)",
		"React and React DOM - MIT (the user interface)",
		"Vite - MIT (the front-end build)",
		"beep - MIT (audio decoding, mixing and playback)",
		"oto - Apache-2.0 (the audio output device)",
		"purego - Apache-2.0 (calling the system audio API without cgo)",
		"go-mp3 - MIT (MP3 decoding)",
		"oggvorbis - MIT (Ogg Vorbis decoding)",
		"flac - Unlicense (FLAC decoding)",
		"BurntSushi/toml - MIT (the cue table)",
		"golang.org/x/sys - BSD-3-Clause (the Windows tray)",
	}
}

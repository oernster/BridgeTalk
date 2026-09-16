package modelfiles

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/oernster/bridge-talk/internal/infrastructure/tomlfile"
)

//go:embed models.toml
var embeddedList []byte

// EmbeddedList answers the shipped list as it is written, for a test reading another platform's files.
func EmbeddedList() []byte { return embeddedList }

// digestDigits is how many hexadecimal digits a SHA-256 is written with.
const digestDigits = sha256.Size * 2

// lowerHex is every digit a listed SHA-256 may be written with. Fetch and Check write a digest in
// lowercase, so an uppercase one could never match.
const lowerHex = "0123456789abcdef"

// secureScheme begins every source's address: a file is never downloaded over plain HTTP.
const secureScheme = "https://"

// listFile is the shape of models.toml: sources by name, then the files.
type listFile struct {
	Sources map[string]string `toml:"sources"`
	Files   []listEntry       `toml:"files"`
}

// listEntry is one file as models.toml gives it.
type listEntry struct {
	Name     string `toml:"name"`
	Platform string `toml:"platform"`
	Source   string `toml:"source"`
	Path     string `toml:"path"`
	Inside   string `toml:"inside"`
	Size     int64  `toml:"size"`
	SHA256   string `toml:"sha256"`
}

// Listed reads the shipped list (FR-535), answering the files this platform is made from.
func Listed() ([]File, error) {
	files, err := Parse(embeddedList)
	return For(files, runtime.GOOS), err
}

// For answers the files a platform is made from: every file naming no platform, with those naming
// that one.
func For(files []File, platform string) []File {
	var out []File
	for _, file := range files {
		if file.Platform == "" || file.Platform == platform {
			out = append(out, file)
		}
	}
	return out
}

// Parse reads a list, refusing an entry that could fetch the wrong thing or put it in the wrong
// place; a name listed twice is refused too.
func Parse(raw []byte) ([]File, error) {
	var parsed listFile
	if err := tomlfile.Decode(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parsing the model files list: %w", err)
	}
	files := make([]File, 0, len(parsed.Files))
	listed := make(map[string]bool, len(parsed.Files))
	for _, entry := range parsed.Files {
		file, err := entry.file(parsed.Sources)
		if err != nil {
			return nil, err
		}
		if listed[file.Name] {
			return nil, fmt.Errorf("%q is listed twice: %w", file.Name, ErrMisshapenEntry)
		}
		listed[file.Name] = true
		files = append(files, file)
	}
	return files, nil
}

// file checks the entry against the sources, then answers with the file it lists.
func (e listEntry) file(sources map[string]string) (File, error) {
	source, known := sources[e.Source]
	var why string
	switch {
	case e.Name != filepath.Base(e.Name) || e.Name == "." || e.Name == "..":
		why = "is not a plain file name"
	case !known:
		why = fmt.Sprintf("names the source %q, which is not listed", e.Source)
	case !strings.HasPrefix(source, secureScheme):
		why = "has a source not reached over https"
	case e.Path == "":
		why = "gives no path"
	case e.Size <= 0:
		why = "gives no size"
	case len(e.SHA256) != digestDigits || strings.Trim(e.SHA256, lowerHex) != "":
		why = "gives no SHA-256 in lowercase hexadecimal"
	default:
		return File{Name: e.Name, Platform: e.Platform, Address: source + e.Path, Inside: e.Inside, Size: e.Size, SHA256: e.SHA256}, nil
	}
	return File{}, fmt.Errorf("the entry %q %s: %w", e.Name, why, ErrMisshapenEntry)
}

package setup

import (
	"path/filepath"
	"strings"
)

// under reports whether a path sits inside a directory.
//
// It answers false rather than guessing: an empty path, an unrelated one and a
// sibling whose name merely starts the same way are all outside. The comparison
// folds case, because the paths it is given come from Windows and Windows does.
func under(path, dir string) bool {
	if path == "" || dir == "" {
		return false
	}
	relative, err := filepath.Rel(strings.ToLower(dir), strings.ToLower(path))
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

package window

import "fmt"

// linuxOpener is the program that opens a folder in the user's own file manager on Linux (FR-816).
const linuxOpener = "xdg-open"

// openWith runs the Linux opener over a folder, waiting for it so a failure reaches the reader. The
// folder is not named in the refusal, since the caller names it once (FR-237).
func openWith(dir string, run func(name string, args ...string) error) error {
	if err := run(linuxOpener, dir); err != nil {
		return fmt.Errorf("%s did not open it: %w", linuxOpener, err)
	}
	return nil
}

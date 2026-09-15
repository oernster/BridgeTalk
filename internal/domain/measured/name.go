package measured

import (
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// LineName names a voice's line as the script counts its lines, from one: the line at index 1 of
// Docked for bf_emma is bf_emma "Docked" line 2. Every message about a line names it this way.
func LineName(voice string, id cue.ID, index int) string {
	return fmt.Sprintf("%s %q line %d", voice, id, index+1)
}

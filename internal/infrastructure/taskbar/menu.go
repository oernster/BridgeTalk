package taskbar

// What the tray's menu and hover text say, on every platform that draws a tray. The Windows tray
// draws a Win32 menu and the Linux tray hands one to the desktop, so what the menu holds and how
// the hover text reads have one home here rather than one each.

import "fmt"

// The menu's own words (FR-710).
const (
	voiceMenu = "Voice"
	openItem  = "Open"
	muteItem  = "Mute"
	quitItem  = "Quit"
)

// commandBuffer is how many menu choices may queue before the tray blocks.
// A user cannot click faster than the main loop drains, so a small buffer is ample
// and a full one indicates the main loop has stopped rather than that it is busy.
const commandBuffer = 8

// mutedMark ends the hover text while playback is muted.
const mutedMark = " (muted)"

// voiceEntry is one entry of the Voice submenu as it is drawn.
type voiceEntry struct {
	label   string
	checked bool
	// separated marks the first voice of a kind after a voice of another, drawn under a separator.
	separated bool
}

// voiceEntries lists the Voice submenu: each voice under the label it is shown by, the cast one
// checked by everything that identifies it (FR-210, FR-540, FR-569, FR-710). The kinds follow one
// another in the order they were given, each group after the first under a separator (FR-509).
func voiceEntries(voices []Choice, active Voice) []voiceEntry {
	entries := make([]voiceEntry, 0, len(voices))
	for index, choice := range voices {
		entries = append(entries, voiceEntry{
			label:     choice.Label,
			checked:   choice.Voice == active,
			separated: index > 0 && choice.Kind != voices[index-1].Kind,
		})
	}
	return entries
}

// hoverText renders the hover text: the product, the cast voice where one is cast and whether
// playback is muted, so the state is readable without opening the menu (FR-710). The mute is
// said with no voice cast too, since the mute answers with nothing cast.
func hoverText(title string, cast Voice, shown string, muted bool) string {
	tip := title
	if cast.Name != "" {
		tip = fmt.Sprintf("%s: %s", tip, shown)
	}
	if muted {
		return tip + mutedMark
	}
	return tip
}

// labelOf answers with what a voice the menu holds is shown by. A voice the menu does not hold is
// shown by its name as it is.
func labelOf(voices []Choice, cast Voice) string {
	for _, choice := range voices {
		if choice.Voice == cast {
			return choice.Label
		}
	}
	return cast.Name
}

// offer hands one command to whoever reads commands, dropping it where nothing is reading.
//
// Dropping is deliberate: a tray that blocked here would freeze its icon and its menu, which is a
// worse answer to a busy moment than one lost click.
func offer(commands chan<- Command, command Command) {
	select {
	case commands <- command:
	default:
	}
}

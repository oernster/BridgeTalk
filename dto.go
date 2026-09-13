// The shapes the front end renders.
//
// They live beside the facade rather than inside it because they are the wire, not
// the logic. Every one of them exists to be marshalled, so keeping them together
// makes the whole surface the front end sees readable in one file.

package main

// VoiceDTO describes one selectable voice for the voice pane.
//
// InUse is the takes the voice holds, every one of which answers a cue: a file that
// matched no cue id never entered the catalogue and is named in the scan report
// instead. So this is both what the voice has and what it can reach.
//
// The cue coverage is not carried either, for a related reason. The row used to show
// how many of the game's moments a voice had its own recordings for, out of how many
// there are. Neither number means anything to somebody who has not read the cue
// table. The dialog behind the row's mark answers the same question in words.
//
// Nothing says whether a voice can be cast, because every voice can: a directory that
// resolved no take never became one. A flag that is always true is not information.
type VoiceDTO struct {
	Name  string `json:"name"`
	InUse int    `json:"inUse"`
}

// ReactionDTO is one line of the reaction log.
//
// It carried the cue's source and its priority for a while and the log never showed
// either, so both crossed on every reaction and were read by nothing. A field the
// page does not render is not diagnostic information, it is only the appearance of
// it. Either belongs back here the day a column wants it, which is one line.
type ReactionDTO struct {
	At      string `json:"at"`
	Cue     string `json:"cue"`
	Event   string `json:"event"`
	Clip    string `json:"clip"`
	Outcome string `json:"outcome"`
}

// StateDTO is everything the header and home pane need in one call.
type StateDTO struct {
	Voice       string `json:"voice"`
	Bound       int    `json:"bound"`
	Total       int    `json:"total"`
	Muted       bool   `json:"muted"`
	Silent      bool   `json:"silent"`
	JournalDir  string `json:"journalDir"`
	StatusPath  string `json:"statusPath"`
	LibraryRoot string `json:"libraryRoot"`
	Version     string `json:"version"`
	// LaunchOnBoot is read from the login entry itself rather than remembered
	// separately, so the toggle cannot disagree with what Windows will actually do.
	LaunchOnBoot bool `json:"launchOnBoot"`
	// Stalls counts the times the output device ran out of audio before it was
	// refilled, with the longest such wait in milliseconds. It is reported because a
	// break in the speech is otherwise something only the listener knows about;
	// only in words at that. A machine that keeps the device fed reports nothing.
	Stalls     int `json:"stalls"`
	WorstStall int `json:"worstStall"`
}

// PlaybackDTO reports whether the output device is sounding anything.
type PlaybackDTO struct {
	Playing bool `json:"playing"`
}

// AboutDTO carries the identity shown in the About dialog.
type AboutDTO struct {
	Name        string   `json:"name"`
	Tagline     string   `json:"tagline"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Copyright   string   `json:"copyright"`
	Authorship  string   `json:"authorship"`
	Attribution string   `json:"attribution"`
	Licence     string   `json:"licence"`
	Credits     []string `json:"credits"`
}

// CueDTO names one cue for a reader.
//
// The title is what reaches the screen, since an id is a name for the code rather
// than for a commander. The id travels with it because it keys the list and because
// the reaction log reports ids, so a silence seen there has something to match. The
// heading is the group in words, generated beside the title so the page never keeps a
// second list of names for the groups.
type CueDTO struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Group   string `json:"group"`
	Heading string `json:"heading"`
}

// CueBreakdownDTO is one voice's whole relationship with the cue table.
//
// Both halves travel together because the dialog shows them together; also because
// the pair is the only honest reading of either: a count of what a voice covers means
// nothing without the list of what it does not.
type CueBreakdownDTO struct {
	Voice    string   `json:"voice"`
	Served   []CueDTO `json:"served"`
	Unserved []CueDTO `json:"unserved"`
}

// The shapes the front end renders.
//
// They live beside the facade rather than inside it because they are the wire, not
// the logic. Every one of them exists to be marshalled, so keeping them together
// makes the whole surface the front end sees readable in one file.

package main

// VoiceDTO describes one voice for the Cast pane, with its two completeness figures
// (FR-215).
//
// Cues is the moments the voice has a recording for. How many moments there are is the same
// for every voice and already crosses as StateDTO.Total, so it is not repeated here. InUse
// is the distinct files the voice uses; Present is the recordings in its directory, whether
// they answer a moment or play at all. The pane words both figures in moments and
// recordings: a count of cues meant nothing to somebody who has not read the cue table,
// which is why the row once dropped its coverage altogether.
//
// Nothing says whether a voice can be cast, because every voice can: a directory that
// resolved no take never became one. A flag that is always true is not information.
//
// Name identifies the voice and is what a cast sends back; Display is what the page shows and
// Credit is the line beneath its figures, both from the voice's manifest (FR-210).
type VoiceDTO struct {
	Name    string `json:"name"`
	Display string `json:"display"`
	Credit  string `json:"credit"`
	Cues    int    `json:"cues"`
	InUse   int    `json:"inUse"`
	Present int    `json:"present"`
}

// ReactionDTO is one line of the reaction log.
//
// It carried the cue's source and its priority for a while and the log never showed
// either, so both crossed on every reaction and were read by nothing. A field the
// page does not render is not diagnostic information, it is only the appearance of
// it. Either belongs back here the day a column wants it, which is one line. The event
// name is not coming back: every cue id begins with it, so a column showing it only
// repeated the id beside it (FR-234).
//
// Title is the moment's full title (FR-233), which the live indicator says as the moment just
// played (FR-719). It is found here for any voice, recorded or machine, so the page never
// works a title out of an id.
type ReactionDTO struct {
	At      string `json:"at"`
	Cue     string `json:"cue"`
	Title   string `json:"title"`
	Clip    string `json:"clip"`
	Outcome string `json:"outcome"`
}

// StateDTO is everything the header and home pane need in one call.
type StateDTO struct {
	// Voice identifies the cast voice; VoiceDisplay is the name it is shown by (FR-210).
	Voice        string `json:"voice"`
	VoiceDisplay string `json:"voiceDisplay"`
	Bound        int    `json:"bound"`
	Total        int    `json:"total"`
	Muted        bool   `json:"muted"`
	Silent       bool   `json:"silent"`
	JournalDir   string `json:"journalDir"`
	StatusPath   string `json:"statusPath"`
	LibraryRoot  string `json:"libraryRoot"`
	Version      string `json:"version"`
	// LaunchOnBoot is read from the login entry itself rather than remembered
	// separately, so the toggle cannot disagree with what Windows will actually do.
	LaunchOnBoot bool `json:"launchOnBoot"`
	// Stalls counts the times the output device ran out of audio before it was
	// refilled, with the longest such wait in milliseconds. It is reported because a
	// break in the speech is otherwise something only the listener knows about;
	// only in words at that. A machine that keeps the device fed reports nothing.
	Stalls     int `json:"stalls"`
	WorstStall int `json:"worstStall"`
	// JournalProblem says why the journal directory is not being watched, naming it
	// once; empty while it is. The window opens either way, so the panes say it (FR-238).
	JournalProblem string `json:"journalProblem"`
	// MachineVoice says whether the cast voice is a machine voice, since a recordings folder may
	// carry a machine voice's id as its name (FR-540).
	MachineVoice bool `json:"machineVoice"`
	// Plugin names the plugin the cast voice came from; empty for every other kind. With Voice,
	// which holds the voice's id within that plugin, it says which plugin voice is cast, since an
	// id identifies a voice only within its own plugin (FR-569).
	Plugin string `json:"plugin"`
}

// MachineVoiceDTO is one machine voice the Cast pane offers: its id, which a cast sends back, the
// name the screen shows, the name alone its pill shows and the group whose panel it sits in (FR-508,
// FR-528, FR-720).
type MachineVoiceDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Given string `json:"given"`
	Group string `json:"group"`
}

// PluginVoiceDTO is one voice a plugin offers, as the Cast pane shows it.
//
// Plugin and ID are what a cast sends back, since an id identifies a voice only within the plugin
// that offered it (FR-569). Name is the name the plugin gave it; Display is what the screen shows,
// which is that name unless another plugin offers one like it (FR-568). Ready says whether the
// audio the voice needs is on this machine and Reason says why it is not, empty while it is
// (FR-570).
//
// No completeness figures travel with it. A recorded voice's figures are read off a folder this
// application owns; a plugin's audio is the plugin's own business and is never counted here.
type PluginVoiceDTO struct {
	Plugin  string `json:"plugin"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Display string `json:"display"`
	Ready   bool   `json:"ready"`
	Reason  string `json:"reason"`
}

// MakingDTO is how far making the cast machine voice's lines has got, as the Cast pane shows it.
//
// Current of Total lines are made (FR-515); CuesServed moments have a current made line, which
// the pane reads against StateDTO.Total (FR-522). Failed lists the lines that could not be made
// (FR-518). Stopped says why making stopped short (FR-520) and NotDeleted why the last cast
// could not delete what it had to (FR-530); each is empty where nothing went wrong.
type MakingDTO struct {
	Voice      string           `json:"voice"`
	Making     bool             `json:"making"`
	Current    int              `json:"current"`
	Total      int              `json:"total"`
	CuesServed int              `json:"cuesServed"`
	Failed     []LineFailureDTO `json:"failed"`
	Stopped    string           `json:"stopped"`
	NotDeleted string           `json:"notDeleted"`
}

// LineFailureDTO is one line that could not be made: its moment named for a reader, its place
// among that moment's lines counting from one and why (FR-518).
type LineFailureDTO struct {
	Cue    CueDTO `json:"cue"`
	Line   int    `json:"line"`
	Reason string `json:"reason"`
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
// the reaction log reports ids, so a silence seen there has something to match. No group
// or heading travels with it: a list shows each cue under its full title alone (FR-233).
//
// Folder is the name of the folder that holds the cue's takes, worked out on this side so
// the page never keeps the rule that writes a dot as an underscore (FR-229).
//
// Purpose is the sentence saying when the cue is heard, written in the cue table by hand
// (FR-231) and shown beneath the title on the Missing takes pane (FR-318).
type CueDTO struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Folder  string `json:"folder"`
	Purpose string `json:"purpose"`
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

// VoiceFoldersDTO reports what making a voice's folders did: where the voice is and how
// many folders were made.
type VoiceFoldersDTO struct {
	Path string `json:"path"`
	Made int    `json:"made"`
}

// ChecklistDTO is what one voice folder still has no recording for, with how many of the
// vocabulary's moments it does have one for. Folder is the voice folder's full path ending
// in the separator, so a missing moment's folder is Folder followed by its id.
type ChecklistDTO struct {
	Voice    string   `json:"voice"`
	Recorded int      `json:"recorded"`
	Total    int      `json:"total"`
	Missing  []CueDTO `json:"missing"`
	Folder   string   `json:"folder"`
}

// ChatterDTO is what the Chatter pane shows: every category in the table's order with its moments
// (FR-727). Problem says why the last switch pressed could not be kept, the switch applying all the
// same; empty while it was kept (FR-633).
type ChatterDTO struct {
	Categories []ChatterCategoryDTO `json:"categories"`
	Problem    string               `json:"problem"`
}

// ChatterCategoryDTO is one category Chatter lists, its moments in the table's order.
type ChatterCategoryDTO struct {
	Name    string             `json:"name"`
	Moments []ChatterMomentDTO `json:"moments"`
}

// ChatterMomentDTO is one moment named for a reader, with whether it is switched on (FR-727).
type ChatterMomentDTO struct {
	Cue CueDTO `json:"cue"`
	On  bool   `json:"on"`
}

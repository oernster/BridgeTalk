// Package taskbar owns the notification tray icon and its menu.
//
// The tray is the application's only visible surface until there is a window. It
// runs its own message loop on a locked OS thread, which is why it never calls back
// into the rest of the application: it reports what the user chose over a channel
// and the main loop acts on it. A callback invoked from the tray thread would be
// running on the wrong thread for everything it wanted to touch.
package taskbar

// CommandKind names what the user chose from the tray menu.
type CommandKind int

const (
	// CommandToggleMute asks for playback to be silenced or unsilenced.
	CommandToggleMute CommandKind = iota
	// CommandSelectVoice asks for the voice it carries to be cast, of whichever kind.
	CommandSelectVoice
	// CommandQuit asks the application to stop.
	CommandQuit
	// CommandShow asks for the window back. The icon is the only way to reach a
	// window that has been put away, so it has to answer a plain click as well as
	// the menu: an icon that does nothing on the usual gesture reads as broken.
	CommandShow
	// CommandNoTray says the desktop never took the icon, so there is no tray after all. Only a
	// platform where the icon is offered to the desktop rather than drawn sends it: on Linux a
	// desktop may draw no tray; whether one does is known only once it has had time to answer
	// (FR-814).
	CommandNoTray
)

// Kind says which sort of voice a choice is.
//
// A name identifies a voice only within its own kind: a recordings folder may carry a machine
// voice's id (FR-540) and an id inside a plugin is unique only there (FR-569). So the kind travels
// with the name everywhere the menu speaks about a voice.
type Kind int

const (
	// Recorded is a voice read from a folder of recordings.
	Recorded Kind = iota
	// Machine is a voice the application speaks itself (FR-509).
	Machine
	// Plugin is a voice a plugin offers (FR-565).
	Plugin
)

// Voice is what identifies one voice to the menu.
//
// It is one type rather than a field for each kind, because every place that speaks about a voice
// here asks the same question: is this the one that is cast? A comparison of three fields written
// out at each of those places is three chances for them to disagree.
type Voice struct {
	Kind Kind
	// Plugin is the name of the plugin that offered the voice; empty for every other kind.
	Plugin string
	// Name identifies the voice within its kind; within its plugin too where it has one.
	Name string
}

// Command is one choice made from the tray menu.
type Command struct {
	Kind CommandKind
	// Chosen is the voice to cast for CommandSelectVoice; empty for every other kind. The
	// application casts it by its kind, which is the one thing the tray need not know how to do.
	Chosen Voice
}

// Choice is one voice the menu offers: what identifies it and the label it is shown by, which
// differ where the voice's manifest names it (FR-210).
type Choice struct {
	Voice
	// Label is what the menu and the hover text show.
	Label string
}

// Options configures a tray at construction.
type Options struct {
	// Title is the tooltip shown when hovering the icon.
	Title string
	// Voices lists the selectable voices, in the order they appear in the menu.
	Voices []Choice
	// Active is the voice shown as chosen; its zero value is no voice at all, since a name is
	// what a voice must have.
	Active Voice
	// Muted is the starting state of the mute item.
	Muted bool
	// Icon is the application's .ico file, which a tray handed a picture takes its frame from. Windows
	// reads the icon out of the binary instead, so it is left empty there (FR-814).
	Icon []byte
}

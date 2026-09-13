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
	// CommandSelectVoice asks for a different voice.
	CommandSelectVoice
	// CommandQuit asks the application to stop.
	CommandQuit
	// CommandShow asks for the window back. The icon is the only way to reach a
	// window that has been put away, so it has to answer a plain click as well as
	// the menu: an icon that does nothing on the usual gesture reads as broken.
	CommandShow
)

// Command is one choice made from the tray menu.
type Command struct {
	Kind CommandKind
	// Voice carries the voice name for CommandSelectVoice; empty otherwise.
	Voice string
}

// Choice is one voice the menu offers: the name that identifies it and the label it is shown
// by, which differ where the voice's manifest names it (FR-210).
type Choice struct {
	// Name identifies the voice; a choice from the menu carries it.
	Name string
	// Label is what the menu and the hover text show.
	Label string
}

// Options configures a tray at construction.
type Options struct {
	// Title is the tooltip shown when hovering the icon.
	Title string
	// Voices lists the selectable voices, in the order they appear in the menu.
	Voices []Choice
	// ActiveVoice names the voice shown as chosen, by the name that identifies it.
	ActiveVoice string
	// Muted is the starting state of the mute item.
	Muted bool
}

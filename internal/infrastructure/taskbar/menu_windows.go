//go:build windows

package taskbar

import "unsafe"

// menuVoice is one entry of the Voice submenu as it is drawn.
type menuVoice struct {
	id      uint32
	label   string
	checked bool
	// separated marks the first machine voice after a recorded one, drawn under a separator.
	separated bool
}

// voiceItems lists the Voice submenu: each voice under the label it is shown by, the cast one
// checked by the name that identifies it and its kind (FR-210, FR-540, FR-710). The machine voices
// follow the recorded voices, the first of them under a separator (FR-509). It is apart from
// showMenu so the entries can be read without a menu to draw them in.
func (t *Tray) voiceItems() []menuVoice {
	active, _ := t.activeVoice.Load().(string)
	machine := t.activeMachine.Load()
	items := make([]menuVoice, 0, len(t.options.Voices))
	for index, choice := range t.options.Voices {
		items = append(items, menuVoice{
			id:        uint32(idVoiceBase + index),
			label:     choice.Label,
			checked:   choice.Name == active && choice.Machine == machine,
			separated: choice.Machine && index > 0 && !t.options.Voices[index-1].Machine,
		})
	}
	return items
}

// showMenu builds the context menu, tracks it and dispatches what was chosen.
func (t *Tray) showMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer func() { _, _, _ = procDestroyMenu.Call(menu) }()

	voices, _, _ := procCreatePopupMenu.Call()
	for _, item := range t.voiceItems() {
		flags := uintptr(0)
		if item.checked {
			flags = mfChecked
		}
		if item.separated {
			appendSeparator(voices)
		}
		appendMenuItem(voices, item.id, item.label, flags)
	}
	if len(t.options.Voices) > 0 {
		appendSubmenu(menu, voices, "Voice")
		appendSeparator(menu)
	}

	muteFlags := uintptr(0)
	if t.muted.Load() {
		muteFlags = mfChecked
	}
	appendMenuItem(menu, idShow, "Open", 0)
	appendSeparator(menu)
	appendMenuItem(menu, idMute, "Mute", muteFlags)
	appendSeparator(menu)
	appendMenuItem(menu, idQuit, "Quit", 0)

	var cursor point
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	// The menu will not dismiss on a click elsewhere unless its owner window is
	// foreground first, which is a documented quirk of tray menus.
	_, _, _ = procSetForegroundWindow.Call(uintptr(t.window))

	chosen, _, _ := procTrackPopupMenu.Call(
		menu,
		tpmRightButton|tpmNonotify|tpmReturnCmd,
		uintptr(cursor.x), uintptr(cursor.y),
		0, uintptr(t.window), 0,
	)
	// Posting a null message lets the menu close cleanly, another documented quirk.
	_, _, _ = procPostMessage.Call(uintptr(t.window), wmNull, 0, 0)

	t.dispatch(uint32(chosen))
}

// dispatch turns a menu command identifier into a Command on the channel.
//
// A send that would block is dropped rather than stalling the tray thread: a full
// buffer means the main loop has stopped reading; a frozen menu would be a worse
// symptom than a lost click.
//
// The hover text is not sent again here. The choice has not been acted on when this
// returns, so the text would describe the state before it; measured, it did in 1000 of
// 1000 rounds. The main loop's change posts the refresh instead (FR-710).
func (t *Tray) dispatch(chosen uint32) {
	var command Command
	switch {
	case chosen == 0:
		return
	case chosen == idMute:
		command = Command{Kind: CommandToggleMute}
	case chosen == idQuit:
		command = Command{Kind: CommandQuit}
	case chosen == idShow:
		command = Command{Kind: CommandShow}
	case chosen >= idVoiceBase:
		index := int(chosen - idVoiceBase)
		if index >= len(t.options.Voices) {
			return
		}
		choice := t.options.Voices[index]
		command = Command{Kind: CommandSelectVoice, Voice: choice.Name}
		if choice.Machine {
			command.Kind = CommandSelectMachineVoice
		}
	default:
		return
	}
	t.send(command)
}

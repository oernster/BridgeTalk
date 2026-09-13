//go:build windows

package taskbar

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/oernster/bridge-talk/internal/product"
)

// className is the hidden owner window's class. It is unique to this application so
// two programs registering a tray class cannot collide.
const className = product.Slug + "TrayWindow"

// commandBuffer is how many menu choices may queue before the tray thread blocks.
// A user cannot click faster than the main loop drains, so a small buffer is ample
// and a full one indicates the main loop has stopped rather than that it is busy.
const commandBuffer = 8

// Tray is the notification-area icon and its menu.
//
// Everything inside the message loop runs on one locked OS thread. The only things
// crossing that boundary are the command channel outward and two atomics inward,
// so no Win32 handle is ever touched from another goroutine.
type Tray struct {
	commands chan Command
	options  Options

	muted       atomic.Bool
	activeVoice atomic.Value

	window windows.HWND
	icon   windows.Handle

	started sync.Once
	stopped sync.Once
	ready   chan error
}

// executablePath returns this program's own path, used to load its embedded icon.
func executablePath() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return path
}

// New builds a tray. It does not appear until Start is called.
func New(options Options) *Tray {
	tray := &Tray{
		commands: make(chan Command, commandBuffer),
		options:  options,
		ready:    make(chan error, 1),
	}
	tray.muted.Store(options.Muted)
	tray.activeVoice.Store(options.ActiveVoice)
	return tray
}

// Commands yields the user's menu choices. The channel is closed when the tray stops.
func (t *Tray) Commands() <-chan Command { return t.commands }

// SetMuted updates the state the menu shows. Safe from any goroutine.
func (t *Tray) SetMuted(muted bool) { t.muted.Store(muted) }

// SetActiveVoice updates which voice the menu shows as chosen. Safe from any goroutine.
func (t *Tray) SetActiveVoice(name string) { t.activeVoice.Store(name) }

// Start shows the icon and runs the message loop on its own locked thread.
//
// It returns once the icon is visible; or with the error that stopped it appearing.
// A tray that cannot be created is not fatal to the application: the caller may
// carry on headless, which is exactly what happens when it runs as a service.
func (t *Tray) Start() error {
	var err error
	t.started.Do(func() {
		go t.run()
		err = <-t.ready
	})
	return err
}

// Stop removes the icon and ends the message loop.
func (t *Tray) Stop() {
	t.stopped.Do(func() {
		if t.window != 0 {
			_, _, _ = procPostMessage.Call(uintptr(t.window), wmClose, 0, 0)
		}
	})
}

// run owns the tray thread from creation to teardown.
func (t *Tray) run() {
	// The message loop and every window it owns must stay on one thread for the
	// lifetime of the window, which is what LockOSThread guarantees.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(t.commands)

	if err := t.create(); err != nil {
		t.ready <- err
		return
	}
	t.ready <- nil
	t.pump()
	t.destroy()
}

// create registers the window class, makes the hidden owner window and adds the icon.
func (t *Tray) create() error {
	instance, _, _ := procGetModuleHandle.Call(0)
	name, err := windows.UTF16PtrFromString(className)
	if err != nil {
		return fmt.Errorf("encoding the tray class name: %w", err)
	}

	class := wndClassEx{
		style:         0,
		lpfnWndProc:   windows.NewCallback(t.windowProc),
		hInstance:     windows.Handle(instance),
		lpszClassName: name,
	}
	class.cbSize = uint32(unsafe.Sizeof(class))
	if ret, _, callErr := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&class))); ret == 0 {
		return fmt.Errorf("registering the tray window class: %w", callErr)
	}

	handle, _, callErr := procCreateWindowEx.Call(
		0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(name)),
		0, 0, 0, 0, 0, 0, 0, instance, 0,
	)
	if handle == 0 {
		return fmt.Errorf("creating the tray window: %w", callErr)
	}
	t.window = windows.HWND(handle)
	t.icon = ownIcon()

	data := t.iconData()
	if ret, _, callErr := procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&data))); ret == 0 {
		_, _, _ = procDestroyWindow.Call(handle)
		t.window = 0
		return fmt.Errorf("adding the tray icon: %w", callErr)
	}
	return nil
}

// iconData builds the NOTIFYICONDATAW describing the icon and its tooltip.
func (t *Tray) iconData() notifyIconData {
	data := notifyIconData{
		hWnd:             t.window,
		uID:              trayIconID,
		uFlags:           nifMessage | nifIcon | nifTip,
		uCallbackMessage: wmTrayCallback,
		hIcon:            t.icon,
	}
	data.cbSize = uint32(unsafe.Sizeof(data))
	copyUTF16(data.szTip[:], t.tooltip())
	return data
}

// tooltip renders the hover text, which names the active voice so the state is
// readable without opening the menu.
func (t *Tray) tooltip() string {
	voice, _ := t.activeVoice.Load().(string)
	if voice == "" {
		return t.options.Title
	}
	voice = t.label(voice)
	if t.muted.Load() {
		return fmt.Sprintf("%s: %s (muted)", t.options.Title, voice)
	}
	return fmt.Sprintf("%s: %s", t.options.Title, voice)
}

// refreshTooltip re-sends the icon data so the hover text follows the state.
func (t *Tray) refreshTooltip() {
	if t.window == 0 {
		return
	}
	data := t.iconData()
	_, _, _ = procShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&data)))
}

// pump runs the message loop until the window closes.
func (t *Tray) pump() {
	var message msg
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		// GetMessage returns 0 on WM_QUIT and -1 on error; both end the loop.
		if int32(ret) <= 0 {
			return
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		_, _, _ = procDispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
}

// destroy removes the icon and the hidden window.
func (t *Tray) destroy() {
	if t.window == 0 {
		return
	}
	data := t.iconData()
	_, _, _ = procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
	_, _, _ = procDestroyWindow.Call(uintptr(t.window))
	t.window = 0
}

// windowProc handles the messages the tray window receives.
func (t *Tray) windowProc(hwnd windows.HWND, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case wmTrayCallback:
		// The right button opens the menu; the left one asks for the window back,
		// on a single click and on a double. Both buttons opened the menu while the
		// window could not be hidden, so a left click had nothing to show. It has
		// now: a window put away in the notification area is reachable only here,
		// and the gesture people try first is a plain click.
		switch lParam {
		case wmRButtonUp:
			t.showMenu()
		case wmLButtonUp, wmLButtonDblClk:
			t.send(Command{Kind: CommandShow})
		}
		return 0
	case wmClose:
		_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case wmDestroy:
		_, _, _ = procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(uintptr(hwnd), uintptr(message), wParam, lParam)
	return ret
}

// label answers with what a voice is shown by, given the name that identifies it. A name the
// menu does not hold is shown as it is.
func (t *Tray) label(name string) string {
	for _, choice := range t.options.Voices {
		if choice.Name == name {
			return choice.Label
		}
	}
	return name
}

// menuVoice is one entry of the Voice submenu as it is drawn.
type menuVoice struct {
	id      uint32
	label   string
	checked bool
}

// voiceItems lists the Voice submenu: each voice under the label it is shown by, the cast one
// checked by the name that identifies it (FR-210, FR-710). It is apart from showMenu so the
// entries can be read without a menu to draw them in.
func (t *Tray) voiceItems() []menuVoice {
	active, _ := t.activeVoice.Load().(string)
	items := make([]menuVoice, 0, len(t.options.Voices))
	for index, choice := range t.options.Voices {
		items = append(items, menuVoice{
			id:      uint32(idVoiceBase + index),
			label:   choice.Label,
			checked: choice.Name == active,
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
		command = Command{Kind: CommandSelectVoice, Voice: t.options.Voices[index].Name}
	default:
		return
	}
	t.send(command)
	t.refreshTooltip()
}

// send offers one command to the main loop, dropping it where nothing is reading.
//
// Dropping is deliberate: this runs on the tray's own locked thread and blocking it
// would freeze the icon and its menu, which is a worse answer to a busy moment than
// one lost click.
func (t *Tray) send(command Command) {
	select {
	case t.commands <- command:
	default:
	}
}

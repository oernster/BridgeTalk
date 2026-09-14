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
// crossing that boundary are the command channel outward, two atomics inward and a
// message posted to say they changed; no other goroutine does anything else with a
// Win32 handle.
type Tray struct {
	commands chan Command
	options  Options

	muted         atomic.Bool
	activeVoice   atomic.Value
	activeMachine atomic.Bool

	// window belongs to the tray thread. posted holds the same handle for every other
	// goroutine, which may only post to it: zero before the window exists and again once
	// it is being taken down.
	window windows.HWND
	posted atomic.Uintptr
	icon   windows.Handle

	// notify hands icon data to the shell. A test replaces it to read what would be sent
	// without an icon appearing.
	notify func(message uintptr, data *notifyIconData) error

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
		notify:   shellNotify,
		ready:    make(chan error, 1),
	}
	tray.muted.Store(options.Muted)
	tray.activeVoice.Store(options.ActiveVoice)
	tray.activeMachine.Store(options.ActiveMachine)
	return tray
}

// shellNotify sends icon data to the shell, answering with the reason when it refuses.
func shellNotify(message uintptr, data *notifyIconData) error {
	if ret, _, callErr := procShellNotifyIcon.Call(message, uintptr(unsafe.Pointer(data))); ret == 0 {
		return callErr
	}
	return nil
}

// Commands yields the user's menu choices. The channel is closed when the tray stops.
func (t *Tray) Commands() <-chan Command { return t.commands }

// SetMuted updates the state the menu and the hover text show. Safe from any goroutine.
func (t *Tray) SetMuted(muted bool) {
	t.muted.Store(muted)
	t.post(wmRefreshTip)
}

// SetActiveVoice updates which voice the menu and the hover text show, by the name that
// identifies it and whether it is a machine voice. Safe from any goroutine.
func (t *Tray) SetActiveVoice(name string, machine bool) {
	t.activeVoice.Store(name)
	t.activeMachine.Store(machine)
	t.post(wmRefreshTip)
}

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
	t.stopped.Do(func() { t.post(wmClose) })
}

// post hands a message to the tray thread, doing nothing while there is no window to take
// it. PostMessage may be called from any thread, which is why it is the one call made from
// outside the tray thread.
func (t *Tray) post(message uintptr) {
	if window := t.posted.Load(); window != 0 {
		_, _, _ = procPostMessage.Call(window, message, 0, 0)
	}
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
	if err := t.notify(nimAdd, &data); err != nil {
		_, _, _ = procDestroyWindow.Call(handle)
		t.window = 0
		return fmt.Errorf("adding the tray icon: %w", err)
	}
	t.posted.Store(handle)
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
	voice = t.label(voice, t.activeMachine.Load())
	if t.muted.Load() {
		return fmt.Sprintf("%s: %s (muted)", t.options.Title, voice)
	}
	return fmt.Sprintf("%s: %s", t.options.Title, voice)
}

// refreshTooltip re-sends the icon data so the hover text follows the state. It runs on
// the tray thread, on the message SetMuted and SetActiveVoice post once the state has
// changed (FR-710).
func (t *Tray) refreshTooltip() {
	if t.window == 0 {
		return
	}
	data := t.iconData()
	_ = t.notify(nimModify, &data)
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
	t.posted.Store(0)
	data := t.iconData()
	_ = t.notify(nimDelete, &data)
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
	case wmRefreshTip:
		t.refreshTooltip()
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

// label answers with what a voice is shown by, given the name that identifies it and its kind. A
// voice the menu does not hold is shown by its name as it is.
func (t *Tray) label(name string, machine bool) string {
	for _, choice := range t.options.Voices {
		if choice.Name == name && choice.Machine == machine {
			return choice.Label
		}
	}
	return name
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

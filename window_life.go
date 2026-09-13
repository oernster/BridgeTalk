// The window's own life: showing it, putting it away and ending the run.
//
// It sits beside app.go for the reason settings.go and cast.go do. These are the few
// methods that act on the window rather than on the application's state. The close
// choice belongs with them: the cross is the one control whose meaning the window
// cannot settle on its own.

package main

import (
	"context"

	"github.com/oernster/bridge-talk/internal/infrastructure/window"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// closeRequestEvent asks the front end for the close choice, since the cross is
// ambiguous in a resident application: it means "put it away" as often as it means
// "stop it". The window closing outright was the whole of the answer before, which
// silently ended an application the commander expected to still be listening.
const closeRequestEvent = "close-request"

// quitWails ends the run loop. Before startup there is no context to end, so the
// request is dropped rather than panicking.
func (a *App) quitWails() {
	if a.ctx == nil {
		return
	}
	runtime.Quit(a.ctx)
}

// hideInWails puts the window away without ending the run. Before startup there is no
// context to hide, so the request is dropped rather than panicking.
func (a *App) hideInWails() {
	if a.ctx == nil {
		return
	}
	runtime.WindowHide(a.ctx)
}

// beforeClose answers the window's close button.
//
// Returning true cancels the close, so the cross asks rather than acts. The question
// is only worth asking where there is somewhere to go: with no tray icon, hiding the
// window would leave the application running with nothing on screen to bring it back,
// so the close is allowed through instead.
//
// A quit that has already been decided, from the File menu, the tray or the dialog
// itself, passes straight through. Without that flag the dialog would ask again about
// the quit it was just told to perform.
func (a *App) beforeClose(context.Context) bool {
	if a.quitting.Load() || a.session.tray == nil {
		return false
	}
	// The cross can be pressed on a window that is behind others; it can also come
	// from the taskbar button's own menu. Either way the window is raised first. A dialog drawn on a window the
	// commander cannot see reads as the button having done nothing at all.
	a.show()
	a.emit(closeRequestEvent, nil)
	return true
}

// MinimiseToTray puts the window away and leaves the application listening.
func (a *App) MinimiseToTray() { a.hide() }

// RequestQuit ends the application from the close dialog's own Quit.
func (a *App) RequestQuit() { a.Quit() }

// showInWails is the production window raise. Before startup there is no context to
// raise into, so the request is dropped rather than panicking.
//
// It focuses the WebView2 child directly, which is what a mouse click does and the
// only route that does not depend on the framework's own focus race. See
// focus_windows.go for why that race exists and what was measured. Raising the window
// remains the fallback for the platforms where the child cannot be found.
func (a *App) showInWails() {
	if a.ctx == nil {
		return
	}
	// Unhidden first, always. Taking focus does not make a hidden window visible:
	// SetForegroundWindow acts on a window that is already there, so a window put
	// away in the notification area stayed away while this reported success. Showing
	// a window that is already shown costs nothing.
	runtime.WindowShow(a.ctx)
	window.TakeFocus()
}

// restoreInWails brings the window back from the notification area, in the middle of
// the screen with the keyboard.
//
// Centred rather than wherever it was: a window is put away for minutes or hours; the
// screen it comes back to may not be the one it left, so the middle is the one
// place it is certainly reachable. Centring while it is still hidden means it does not
// appear and then jump.
func (a *App) restoreInWails() {
	if a.ctx == nil {
		return
	}
	runtime.WindowCenter(a.ctx)
	a.showInWails()
}

// Quit ends the application from the File menu.
//
// The intent is recorded before the run ends, so the close the framework then performs
// passes through beforeClose rather than raising the dialog again.
func (a *App) Quit() {
	a.quitting.Store(true)
	a.quit()
}

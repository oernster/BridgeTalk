//go:build linux

package taskbar

// The notification-area icon on Linux (FR-814).
//
// A Linux tray icon is not drawn by the application: it is published over D-Bus as a
// StatusNotifierItem for the desktop's StatusNotifierWatcher to draw. Not every desktop runs a
// watcher. So the icon is offered only once a watcher answers, asked for over a grace period
// because a sign-in start comes up before the panel does. Where none answers the tray says so on
// its command channel and the application carries on without one.
//
// The menu and the hover text read as they do on Windows; menu.go holds both.

import (
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/systray"
	"github.com/godbus/dbus/v5"

	"github.com/oernster/bridge-talk/internal/infrastructure/iconfile"
)

// trayGrace is how long the desktop is given to take the icon before the application treats itself
// as having no tray: o7 Debrief measured an autostart launch coming up before the panel and waits
// this long (FR-814).
const trayGrace = 15 * time.Second

// watcherAskEvery is how often the watcher is asked for during the grace period.
const watcherAskEvery = time.Second

// linuxTraySide is the side of the picture a Linux tray hands the desktop, a size the committed icon
// holds so nothing is resampled.
const linuxTraySide = 64

// watcherName is the bus name the desktop's tray watcher owns.
const watcherName = "org.kde.StatusNotifierWatcher"

// nameHasOwner is the bus's own question of whether a name is owned.
const nameHasOwner = "org.freedesktop.DBus.NameHasOwner"

// Tray is the notification-area icon and its menu.
type Tray struct {
	commands chan Command
	options  Options

	muted       atomic.Bool
	active      atomic.Value
	activeLabel atomic.Value

	// mu guards the menu, which exists only once the desktop has taken the icon.
	mu     sync.Mutex
	shown  bool
	voices []*systray.MenuItem
	mute   *systray.MenuItem
	end    func()

	started sync.Once
	stopped sync.Once
	stop    atomic.Bool
}

// New builds a tray. It is not offered to the desktop until Start is called.
func New(options Options) *Tray {
	tray := &Tray{commands: make(chan Command, commandBuffer), options: options}
	tray.muted.Store(options.Muted)
	tray.active.Store(options.Active)
	tray.activeLabel.Store(labelOf(options.Voices, options.Active))
	return tray
}

// Commands yields the user's menu choices, then CommandNoTray where the desktop never takes the
// icon.
func (t *Tray) Commands() <-chan Command { return t.commands }

// SetMuted updates the state the menu and the hover text show. Safe from any goroutine.
func (t *Tray) SetMuted(muted bool) {
	t.muted.Store(muted)
	t.refresh()
}

// SetActiveVoice updates which voice the menu and the hover text show. A blank label shows the
// voice by its name. Safe from any goroutine.
func (t *Tray) SetActiveVoice(cast Voice, label string) {
	if label == "" {
		label = cast.Name
	}
	t.active.Store(cast)
	t.activeLabel.Store(label)
	t.refresh()
}

// Start offers the icon to the desktop without waiting for it to be taken, so the window is not
// held up for the grace period. It never fails here: a desktop with no tray is said on the command
// channel once the grace period has passed.
func (t *Tray) Start() error {
	t.started.Do(func() { go t.offer() })
	return nil
}

// Stop takes the icon away.
func (t *Tray) Stop() {
	t.stopped.Do(func() {
		t.stop.Store(true)
		t.mu.Lock()
		end := t.end
		t.mu.Unlock()
		if end != nil {
			end()
		}
	})
}

// offer waits for the desktop's watcher, then publishes the icon; with none it says so.
func (t *Tray) offer() {
	bus, err := dbus.SessionBus()
	present := err == nil && awaitWatcher(func() bool { return !t.stop.Load() && owned(bus, watcherName) }, watcherAskEvery, trayGrace, time.Sleep)
	if t.stop.Load() {
		return
	}
	if !present {
		offer(t.commands, Command{Kind: CommandNoTray})
		return
	}
	start, end := systray.RunWithExternalLoop(t.build, nil)
	t.mu.Lock()
	t.end = end
	t.mu.Unlock()
	start()
}

// owned asks the bus whether a name is owned, answering no where the bus cannot say.
func owned(bus *dbus.Conn, name string) bool {
	var has bool
	return bus.BusObject().Call(nameHasOwner, 0, name).Store(&has) == nil && has
}

// build draws the icon and its menu once the desktop is ready for them.
func (t *Tray) build() {
	if frame, err := iconfile.Frame(t.options.Icon, linuxTraySide); err == nil {
		systray.SetIcon(frame)
	}
	systray.SetTitle(t.options.Title)
	systray.SetOnTapped(func() { offer(t.commands, Command{Kind: CommandShow}) })

	active, _ := t.active.Load().(Voice)
	var voices []*systray.MenuItem
	if len(t.options.Voices) > 0 {
		menu := systray.AddMenuItem(voiceMenu, "")
		for index, entry := range voiceEntries(t.options.Voices, active) {
			if entry.separated {
				menu.AddSeparator()
			}
			item := menu.AddSubMenuItemCheckbox(entry.label, "", entry.checked)
			voices = append(voices, item)
			go t.relay(item.ClickedCh, Command{Kind: CommandSelectVoice, Chosen: t.options.Voices[index].Voice})
		}
		systray.AddSeparator()
	}
	go t.relay(systray.AddMenuItem(openItem, "").ClickedCh, Command{Kind: CommandShow})
	systray.AddSeparator()
	mute := systray.AddMenuItemCheckbox(muteItem, "", t.muted.Load())
	go t.relay(mute.ClickedCh, Command{Kind: CommandToggleMute})
	systray.AddSeparator()
	go t.relay(systray.AddMenuItem(quitItem, "").ClickedCh, Command{Kind: CommandQuit})

	t.mu.Lock()
	t.voices, t.mute, t.shown = voices, mute, true
	t.mu.Unlock()
	t.refresh()
}

// relay turns every click on one menu item into its command.
func (t *Tray) relay(clicked <-chan struct{}, command Command) {
	for range clicked {
		offer(t.commands, command)
	}
}

// refresh makes the checks and the hover text follow the state, once there is a menu to follow it.
func (t *Tray) refresh() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.shown {
		return
	}
	active, _ := t.active.Load().(Voice)
	shown, _ := t.activeLabel.Load().(string)
	systray.SetTooltip(hoverText(t.options.Title, active, shown, t.muted.Load()))
	for index, entry := range voiceEntries(t.options.Voices, active) {
		checkTo(t.voices[index], entry.checked)
	}
	checkTo(t.mute, t.muted.Load())
}

// checkTo sets a menu item's check to the state given.
func checkTo(item *systray.MenuItem, checked bool) {
	if checked {
		item.Check()
		return
	}
	item.Uncheck()
}

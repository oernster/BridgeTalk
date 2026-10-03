# Technical debt

What is still open, what is deliberately left and what only looks like debt.

Every item in this file is a behaviour-preserving internal concern. Nothing here reverts a feature or
changes what the user sees. Read it against `ARCHITECTURE.md` and the structural tests, which are the
authority on the invariants an item might threaten.

Open items are numbered sections. A numbered heading is the definition of an open item, so a scan for
`## <number>.` is the machine check for whether this file is clear. The two standing sections below are
deliberately unnumbered and are not open items.

History is not recorded here. A resolved item is deleted outright, never rewritten as done and never
archived. A resolution worth remembering belongs in the release notes.

## 1. The Linux tray code outlives the Linux tray

No tray is put up on Linux since FR-814 (`offersTray` in `voices.go` answers true on Windows alone), yet
the code that drew one is still compiled into every Linux build and never runs:

- `internal/infrastructure/taskbar/tray_linux.go`, `watcher.go` and `trayicon.go` with
  `trayicon_test.go`, plus `awaitWatcher`, `trayPicture` and `linuxTraySide`;
- `icon_other.go` (which embeds the icon only to hand it to that tray) beside its empty Windows twin
  `icon_windows.go`, with `Options.Icon` and the `Icon: applicationIcon` voices.go passes;
- `CommandNoTray` with its handler in `loop.go`, `App.trayGone` and the `hasTray` check it feeds, plus
  `TestATrayTheDesktopNeverTookIsNoTray`, which FR-814's Verified-by cites; nothing sends that command;
- `fyne.io/systray` and `godbus/dbus` in `go.mod`, needed by the taskbar package alone and credited in
  `identity.go`.

The cost of carrying it is small: two dependencies and some dead code in the Linux binary. The cost of
removing it is the risk. Taking it out touches the close logic as well as the taskbar package; the
change can be checked from Windows only by a cgo-off `go vet` of the taskbar package; the root package
cannot be vetted for Linux there. Blocked on a Linux build to test it: remove it all at once, retag
`tray_other.go` as `!windows`, run `go mod tidy`, update ARCHITECTURE.md, DEVELOPMENT.md, TESTING.md and
FR-814's Verified-by, then build and run on Linux.

## Looks like debt, not worth touching

Nothing yet.

## Not debt (do not "fix" these)

**The setup program's drawn header mark, now that the artwork exists.** The setup page still carries an
inline SVG mark behind the real icon; it swaps to the image only once that has actually loaded. That is
not a leftover. The page has no bundler, so it loads its icon as a plain file; if that file were ever
missing the drawing is what keeps the header looking finished rather than showing a broken image.

**The setup program holding no install logic of its own.** Every act the setup window performs goes
through `internal/infrastructure/setup`, which is where the extraction, the registry writes, the
shortcuts and the process handling live. `installer/app.go` decides only which screen to open and when
to refuse, such as while the application is running, then calls it. The portable half of `setup` is
unit tested; `installer` has no tests of its own, since every method on it acts on the machine.

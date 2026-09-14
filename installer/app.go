package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/madelines"
	"github.com/oernster/bridge-talk/internal/infrastructure/runlog"
	"github.com/oernster/bridge-talk/internal/infrastructure/setup"
	"github.com/oernster/bridge-talk/internal/infrastructure/window"
)

// uninstallExeName is the copy of setup left inside the install directory, so the
// Apps list has something to call even after the downloaded setup file is gone.
const uninstallExeName = "uninstall.exe"

// App is the Wails facade for the setup program. Everything the front end can do
// goes through a method here.
type App struct {
	ctx           context.Context
	payload       string
	version       string
	uninstallMode bool
	prefersDark   bool
}

// NewApp builds the facade. Started with -uninstall, setup opens on the removal
// screen rather than on the manage one.
func NewApp(payload string, version string, prefersDark bool) *App {
	uninstall := len(os.Args) > 1 && os.Args[1] == setup.UninstallFlag
	return &App{
		payload:       payload,
		version:       version,
		uninstallMode: uninstall,
		prefersDark:   prefersDark,
	}
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// domReady takes the keyboard once the page exists.
//
// Wails hands the webview its keyboard from the main window's WM_SETFOCUS, which
// Windows raises only on a change of focus; the handler for it is bound inside an
// asynchronous callback. Whether the window's first focus arrives before there is a
// handler for it is a race, one that setup loses. Measured on a cold launch: the
// primary button was focused by the page yet drew no ring and Enter did nothing at
// all, which reads as a dead keyboard rather than as focus sitting elsewhere.
func (a *App) domReady(context.Context) { a.show() }

// show gives the webview the keyboard, falling back to asking Wails for the window.
func (a *App) show() {
	if window.TakeFocus() {
		return
	}
	if a.ctx != nil {
		wailsruntime.WindowShow(a.ctx)
	}
}

// TakeKeyboard is called by the page when it finds it has no keyboard.
//
// The page is the only thing that can tell: from Go the window looks focused either
// way. It is the second half of the repair, because domReady runs before the webview
// is necessarily ready to keep what it is given.
func (a *App) TakeKeyboard() { a.show() }

// StateDTO describes what setup should offer, given what is already on the machine.
//
// AppName is sent because the page must not write the product's name down. It did
// once, in sixteen places; a rename then went through every other surface without
// touching it, so the setup program a user ran announced a product that no longer
// existed. Nothing said a word: it is a string in a page nobody compiles.
type StateDTO struct {
	AppName          string `json:"appName"`
	Mode             string `json:"mode"`
	Relation         string `json:"relation"`
	Installed        bool   `json:"installed"`
	InstalledVersion string `json:"installedVersion"`
	ThisVersion      string `json:"thisVersion"`
	InstallDir       string `json:"installDir"`
	LaunchOnBoot     bool   `json:"launchOnBoot"`
	StartMenu        bool   `json:"startMenu"`
	Desktop          bool   `json:"desktop"`
	PrefersDark      bool   `json:"prefersDark"`
}

// OptionsDTO carries the choices made on the install or reinstall screen.
type OptionsDTO struct {
	StartMenu    bool `json:"startMenu"`
	Desktop      bool `json:"desktop"`
	LaunchOnBoot bool `json:"launchOnBoot"`
}

// Progress is emitted on the "progress" event while a long operation runs.
type Progress struct {
	Pct int    `json:"pct"`
	Msg string `json:"msg"`
}

// relationNames turn the comparison into the word the front end routes on.
var relationNames = map[setup.Relation]string{
	setup.Newer: "newer",
	setup.Same:  "same",
	setup.Older: "older",
}

// DetectState inspects the machine and returns the mode setup should open in.
func (a *App) DetectState() StateDTO {
	dir, _ := setup.InstallDir()
	installedVersion, installed := setup.InstalledVersion()
	shortcuts := setup.CurrentShortcuts()

	mode := "install"
	switch {
	case a.uninstallMode:
		mode = "uninstall"
	case installed:
		mode = "manage"
	}
	relation := setup.Same
	if installed {
		relation = setup.Compare(a.version, installedVersion)
	}
	return StateDTO{
		AppName:          setup.AppName,
		Mode:             mode,
		Relation:         relationNames[relation],
		Installed:        installed,
		InstalledVersion: installedVersion,
		ThisVersion:      a.version,
		InstallDir:       dir,
		LaunchOnBoot:     setup.IsLaunchOnBoot(),
		StartMenu:        shortcuts.StartMenu,
		Desktop:          shortcuts.Desktop,
		PrefersDark:      a.prefersDark,
	}
}

// AppRunning reports whether the application is open, so the front end can offer to
// close it rather than failing later on a locked executable.
func (a *App) AppRunning() bool { return setup.IsAppRunning() }

// CloseRunningApp ends the running application so setup can proceed.
func (a *App) CloseRunningApp() error { return setup.CloseRunningApp() }

// Install performs a fresh install, an update, a downgrade or a reinstall. All four
// are the same act: overwrite the files, then apply the options as given.
func (a *App) Install(choices OptionsDTO) error {
	return a.write(choices)
}

// Repair re-extracts and re-registers the application, leaving every option exactly
// as it stands. It is the quick fix for a damaged install, as distinct from a
// reinstall, which asks for the options again.
//
// The sign-in entry is kept wherever it exists, not only where it names a program that is
// there: the program being missing is what a damaged install is (FR-804).
func (a *App) Repair() error {
	shortcuts := setup.CurrentShortcuts()
	return a.write(OptionsDTO{
		StartMenu:    shortcuts.StartMenu,
		Desktop:      shortcuts.Desktop,
		LaunchOnBoot: setup.HasLaunchOnBootEntry(),
	})
}

// write is the single install path behind Install and Repair.
func (a *App) write(choices OptionsDTO) error {
	// Refuse to overwrite a running instance: extracting over a locked executable
	// fails part way and leaves a half-written install.
	if setup.IsAppRunning() {
		return setup.ErrAppRunning
	}
	dir, err := setup.InstallDir()
	if err != nil {
		return err
	}

	a.progress(10, "Extracting files...")
	if err := setup.ExtractZip(a.payload, dir); err != nil {
		return fmt.Errorf("extract files: %w", err)
	}
	exePath := filepath.Join(dir, setup.ExeName)

	a.progress(55, "Registering the application...")
	if err := a.register(dir, exePath); err != nil {
		return err
	}

	a.progress(80, "Applying your choices...")
	setup.ApplyShortcuts(exePath, dir, setup.Shortcuts{
		StartMenu: choices.StartMenu,
		Desktop:   choices.Desktop,
	})
	if err := setup.SetLaunchOnBoot(exePath, choices.LaunchOnBoot); err != nil {
		return fmt.Errorf("configure launch at sign-in: %w", err)
	}

	a.progress(100, "Done.")
	return nil
}

// register leaves a copy of setup beside the application and writes the Apps list
// entry that points at it.
func (a *App) register(dir, exePath string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate setup: %w", err)
	}
	uninstallExe := filepath.Join(dir, uninstallExeName)
	if err := setup.CopyFile(self, uninstallExe); err != nil {
		return fmt.Errorf("write the uninstaller: %w", err)
	}
	sizeKB, _ := setup.DirSizeKB(dir)
	if err := setup.WriteUninstallEntry(setup.UninstallInfo{
		Version:      a.version,
		InstallDir:   dir,
		UninstallExe: uninstallExe,
		IconPath:     exePath,
		EstimatedKB:  sizeKB,
	}); err != nil {
		return fmt.Errorf("register the application: %w", err)
	}
	return nil
}

// Uninstall removes the shortcuts, the login entry, the registry record, the lines made
// for machine voices, the log and the installed files. When the user asks to forget their
// settings, the application's stored choices go too, along with the theme and volume
// the window keeps.
func (a *App) Uninstall(removeState bool) error {
	// The scheduled deletion cannot remove a locked executable, so a running
	// application has to close first.
	if setup.IsAppRunning() {
		return setup.ErrAppRunning
	}
	dir, err := setup.InstallDir()
	if err != nil {
		return err
	}

	a.progress(20, "Removing shortcuts...")
	setup.RemoveShortcuts()
	_ = setup.SetLaunchOnBoot("", false)

	a.progress(50, "Removing registry entries...")
	_ = setup.RemoveUninstallEntry()

	// The made lines and the log go whatever is ticked, since the application wrote them
	// (FR-525, FR-715); the window's state goes only when forgetting is asked for. One that
	// cannot be found arrives empty, which removes nothing.
	a.progress(60, "Removing the lines made for machine voices and the log...")
	madeLines, _ := madelines.Dir()
	log, _ := runlog.Path()
	state, _ := setup.StateDir()
	_ = setup.RemoveLeftovers(setup.Leftovers{MadeLines: madeLines, Log: log, State: state}, removeState)

	if removeState {
		a.progress(70, "Removing your saved settings...")
		_ = config.NewSettings().Forget()
	}

	a.progress(90, "Removing files...")
	setup.ScheduleDirDeletion(dir)

	a.progress(100, "Done.")
	return nil
}

// LaunchApp starts the installed application, backing the "launch when setup
// finishes" option.
func (a *App) LaunchApp() error { return setup.LaunchApp() }

// SetLaunchOnBoot toggles the login entry live from the manage screen.
func (a *App) SetLaunchOnBoot(enabled bool) error {
	dir, err := setup.InstallDir()
	if err != nil {
		return err
	}
	return setup.SetLaunchOnBoot(filepath.Join(dir, setup.ExeName), enabled)
}

// SetShortcuts applies the shortcut boxes live from the manage screen, so unticking
// one takes the shortcut away there and then.
func (a *App) SetShortcuts(startMenu, desktop bool) error {
	dir, err := setup.InstallDir()
	if err != nil {
		return err
	}
	setup.ApplyShortcuts(filepath.Join(dir, setup.ExeName), dir, setup.Shortcuts{
		StartMenu: startMenu,
		Desktop:   desktop,
	})
	return nil
}

// Quit closes the setup program.
func (a *App) Quit() { wailsruntime.Quit(a.ctx) }

// progress reports how far a long operation has got.
func (a *App) progress(pct int, msg string) {
	wailsruntime.EventsEmit(a.ctx, "progress", Progress{Pct: pct, Msg: msg})
}

// Where plugins are looked for and what is said about the ones passed over. It is the
// composition root's business: nothing above it knows a plugin exists.
package main

import (
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/runlog"
	"github.com/oernster/bridge-talk/internal/infrastructure/setup"
	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// windowsOS is runtime.GOOS on Windows, the one platform whose plugins sit beside the application.
const windowsOS = "windows"

// pluginsFolder answers where plugins are looked for on the platform goos, given how to find the
// running executable and the product's own data folder. Both are parameters, so every platform's
// rule is exercised on every platform.
//
// On Windows the folder is beside the application, which for an installed build is the install
// directory itself, since that is where the setup program writes the executable. Reading the
// recorded install location instead would have a build run from anywhere else look in a folder it
// is not in. The model files are found the same way, beside the application (FR-539), so this is
// one habit rather than two.
//
// Elsewhere it is inside the product's own data folder (FR-818). A flatpak's install directory is
// read only, so a folder beside the application is one nobody could put a plugin in.
func pluginsFolder(goos string, executable, dataDir func() (string, error)) (string, error) {
	if goos == windowsOS {
		found, err := executable()
		if err != nil {
			return "", err
		}
		return filepath.Join(filepath.Dir(found), product.PluginsFolder), nil
	}
	data, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, product.PluginsFolder), nil
}

// makesPluginsFolder reports whether the application makes the plugins folder itself on the platform
// goos: everywhere but Windows, where the setup program makes it (FR-576). Elsewhere no setup program
// runs, so without this a user would have to create the folder by name in the right place (FR-818).
func makesPluginsFolder(goos string) bool { return goos != windowsOS }

// loadPlugins loads every plugin in folder, writing to the log every one it passed over and
// why (FR-567). Where makeFolder is set, the folder is made first where it is not there yet; the
// maker is setup's own, so the folder is made one way wherever it is made.
//
// Not knowing where the folder is stops nothing: it means no plugins, which is the ordinary
// case anyway. Saying so is worth one line, since a user who installed a plugin would
// otherwise have nothing at all to read. A folder that cannot be made stops nothing either; the
// reason goes to the log and the loader then finds no folder, which is no plugins.
func loadPlugins(folder string, notFound error, makeFolder bool, log runlog.Lines) *plugin.Set {
	if notFound != nil {
		log.Log("note: the application cannot tell where its plugins folder is, so no plugin is loaded: " + notFound.Error())
		return &plugin.Set{}
	}
	if makeFolder {
		if err := setup.MakePluginsFolder(filepath.Dir(folder)); err != nil {
			log.Log("note: the plugins folder could not be made, so no plugin is loaded: " + err.Error())
		}
	}
	set := plugin.Load(folder, plugin.OpenLibrary)
	reportPlugins(set, log)
	return set
}

// reportPlugins writes to the log every plugin passed over and every voice passed over inside a
// plugin that otherwise loaded (FR-567).
//
// A voice whose audio is not on this machine is passed over as surely as a file that would not
// load: it cannot be cast (FR-570) and the notification area does not offer it (FR-509). The Cast
// pane says so beside the voice, which is gone the moment the window closes; the log is what is
// still there when the user is asked afterwards what happened.
//
// The plugin is named by its file rather than by the name it gave itself, as a refused plugin is:
// the file is the one thing about a plugin the user can see by opening the folder; two
// plugins may honestly choose one name (FR-568).
func reportPlugins(set *plugin.Set, log runlog.Lines) {
	for _, passed := range set.Refusals {
		log.Log(refusal.PassedOver("the plugin "+passed.File, passed.Why))
	}
	for _, loaded := range set.Plugins {
		for _, voice := range loaded.Voices() {
			if voice.Ready {
				continue
			}
			log.Log(refusal.PassedOver("the voice "+voice.Name+" in the plugin "+loaded.File, voice.Reason))
		}
	}
}

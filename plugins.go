// Where plugins are looked for and what is said about the ones passed over. It is the
// composition root's business: nothing above it knows a plugin exists.
package main

import (
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/runlog"
	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// pluginsBeside answers where plugins are looked for, given the running executable.
//
// The folder is beside the application, which for an installed build is the install
// directory itself, since that is where the setup program writes the executable. Reading
// the recorded install location instead would have a build run from anywhere else look in a
// folder it is not in. The model files are found the same way, beside the application
// (FR-539), so this is one habit rather than two.
func pluginsBeside(executable string) string {
	return filepath.Join(filepath.Dir(executable), product.PluginsFolder)
}

// loadPlugins loads every plugin beside the application, writing to the log every one it
// passed over and why (FR-567).
//
// Not knowing where the application is stops nothing: it means no plugins, which is the
// ordinary case anyway. Saying so is worth one line, since a user who installed a plugin
// would otherwise have nothing at all to read. Where plugins are not available yet, which missingOn
// names, the folder is not looked in at all (FR-818).
func loadPlugins(executable string, notFound error, missingOn string, log runlog.Lines) *plugin.Set {
	if missingOn != "" {
		log.Log("note: plugins are not loaded on " + missingOn + " yet")
		return &plugin.Set{}
	}
	if notFound != nil {
		log.Log("note: the application cannot tell where it is, so no plugin is loaded: " + notFound.Error())
		return &plugin.Set{}
	}
	set := plugin.Load(pluginsBeside(executable), plugin.OpenLibrary)
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

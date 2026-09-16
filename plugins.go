// Where plugins are looked for and what is said about the ones passed over. It is the
// composition root's business: nothing above it knows a plugin exists.
package main

import (
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/runlog"
)

// pluginsFolder is the folder inside the install directory that holds plugins (FR-560).
const pluginsFolder = "plugins"

// pluginsBeside answers where plugins are looked for, given the running executable.
//
// The folder is beside the application, which for an installed build is the install
// directory itself, since that is where the setup program writes the executable. Reading
// the recorded install location instead would have a build run from anywhere else look in a
// folder it is not in. The model files are found the same way, beside the application
// (FR-539), so this is one habit rather than two.
func pluginsBeside(executable string) string {
	return filepath.Join(filepath.Dir(executable), pluginsFolder)
}

// loadPlugins loads every plugin beside the application, writing to the log every one it
// passed over and why (FR-567).
//
// Not knowing where the application is stops nothing: it means no plugins, which is the
// ordinary case anyway. Saying so is worth one line, since a user who installed a plugin
// would otherwise have nothing at all to read.
func loadPlugins(executable string, notFound error, log runlog.Lines) *plugin.Set {
	if notFound != nil {
		log.Log("note: the application cannot tell where it is, so no plugin is loaded: " + notFound.Error())
		return &plugin.Set{}
	}
	set := plugin.Load(pluginsBeside(executable), plugin.OpenLibrary)
	for _, passed := range set.Refusals {
		log.Log("note: the plugin " + passed.File + " was passed over: " + passed.Why)
	}
	return set
}

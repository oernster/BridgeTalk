// The plugin voice surface of the facade: which voices the loaded plugins offer, which of them
// can speak and casting one from the window.
//
// It sits beside machine.go for the reason machine.go sits beside cast.go: a kind of voice's own
// questions are a slice that comes out whole.

package main

import (
	"fmt"

	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
)

// PluginVoices lists every voice every loaded plugin offers, in the order the plugins were loaded
// and each offered them (FR-565).
//
// A voice whose audio is not on this machine is listed with the reason it gave rather than left
// out (FR-570): a voice the user installed and cannot see is a fault they have no way to read.
// The list is empty rather than missing where no plugin is loaded, which is the ordinary case
// (FR-562).
func (a *App) PluginVoices() []PluginVoiceDTO {
	if a.session.plugins == nil {
		return []PluginVoiceDTO{}
	}
	offered := a.session.plugins.Voices()
	shown := pluginDisplays(offered)
	out := make([]PluginVoiceDTO, 0, len(offered))
	for index, voice := range offered {
		out = append(out, PluginVoiceDTO{
			Plugin:  voice.Plugin().Name,
			ID:      voice.ID,
			Name:    voice.Name,
			Display: shown[index],
			Ready:   voice.Ready,
			Reason:  voice.Reason,
		})
	}
	return out
}

// pluginDisplays answers the name each voice is shown by, in the order given (FR-568).
//
// A voice is shown by the name its plugin gave it. Two plugins may honestly choose one name, so a
// name offered more than once is shown with the plugin offering it rather than one of the two
// being dropped; where two plugins share a name as well, the file each was loaded from tells them
// apart, since the file is the one thing about a plugin the user can see by opening the folder.
//
// The names are worked out over the whole list rather than per voice, because whether a name is
// shared is a fact about the list and not about either voice in it.
func pluginDisplays(voices []*plugin.Voice) []string {
	sharedName, sharedPlugin := shared(voices)
	out := make([]string, 0, len(voices))
	for _, voice := range voices {
		if !sharedName[voice.Name] {
			out = append(out, voice.Name)
			continue
		}
		out = append(out, fmt.Sprintf("%s (%s)", voice.Name, offeredBy(voice.Plugin(), sharedPlugin)))
	}
	return out
}

// shared answers which voice names are offered more than once and which plugin names are carried
// by more than one file.
func shared(voices []*plugin.Voice) (map[string]bool, map[string]bool) {
	names := map[string]int{}
	files := map[string]map[string]bool{}
	for _, voice := range voices {
		names[voice.Name]++
		from := voice.Plugin()
		if files[from.Name] == nil {
			files[from.Name] = map[string]bool{}
		}
		files[from.Name][from.File] = true
	}
	sharedName := map[string]bool{}
	for name, count := range names {
		sharedName[name] = count > 1
	}
	sharedPlugin := map[string]bool{}
	for name, carried := range files {
		sharedPlugin[name] = len(carried) > 1
	}
	return sharedName, sharedPlugin
}

// offeredBy names the plugin a shared voice name is shown with: its own name; the file it was
// loaded from where another file carries that name too.
func offeredBy(from *plugin.Plugin, sharedPlugin map[string]bool) string {
	if sharedPlugin[from.Name] {
		return from.File
	}
	return from.Name
}

// CastPluginVoice casts the voice a plugin offers, by the plugin's own name and the voice's id
// within it (FR-569).
//
// A voice whose audio is not on this machine is refused with the reason it gave rather than cast
// (FR-570); so is one no loaded plugin offers. Otherwise the voice confirms in a take of its own
// as a recorded voice does, the window is told and the pair is kept for the next run. As with
// every other kind, a failure to keep it is reported without the cast being undone: the cast
// succeeded and only the memory of it did not.
func (a *App) CastPluginVoice(from, id string) error {
	if err := a.session.castPlugin(from, id); err != nil {
		return err
	}
	a.acknowledge()
	a.emitState()
	return a.rememberPluginVoice(from, id)
}

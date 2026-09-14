package config

// ParseScript, ParseSounds, ParsePauses and VoiceScript expose the readers to the tests, so a file
// the shipped ones never are can be read.
var (
	ParseScript = parseScript
	ParseSounds = parseSounds
	ParsePauses = parsePauses
	VoiceScript = voiceScript
)

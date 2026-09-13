// Package status watches Status.json, the file the game rewrites in place to
// describe the ship's current state.
//
// It needs a different reader from the journal. Nothing is appended, so there is no
// offset to resume from; instead the whole file is read, the values are compared
// with the previous reading, then one synthetic event is emitted per changed value.
// Without this source every cue describing ship state is unreachable, because none
// of it appears in the journal.
package status

// Flag names the state values this application can react to. The names are the
// vocabulary the cue table uses, so they are part of the shipped contract.
const (
	FlagDocked            = "Docked"
	FlagLanded            = "Landed"
	FlagLandingGearDown   = "LandingGearDown"
	FlagShieldsUp         = "ShieldsUp"
	FlagSupercruise       = "Supercruise"
	FlagFlightAssistOff   = "FlightAssistOff"
	FlagHardpointsDeploy  = "HardpointsDeployed"
	FlagInWing            = "InWing"
	FlagLightsOn          = "LightsOn"
	FlagCargoScoopDeploy  = "CargoScoopDeployed"
	FlagSilentRunning     = "SilentRunning"
	FlagScoopingFuel      = "ScoopingFuel"
	FlagSrvHandbrake      = "SrvHandbrake"
	FlagSrvTurret         = "SrvTurret"
	FlagSrvDriveAssist    = "SrvDriveAssist"
	FlagMassLocked        = "FsdMassLocked"
	FlagFsdCharging       = "FsdCharging"
	FlagFsdCooldown       = "FsdCooldown"
	FlagLowFuel           = "LowFuel"
	FlagOverHeating       = "OverHeating"
	FlagIsInDanger        = "IsInDanger"
	FlagBeingInterdicted  = "BeingInterdicted"
	FlagInMainShip        = "InMainShip"
	FlagInFighter         = "InFighter"
	FlagInSRV             = "InSRV"
	FlagAnalysisMode      = "HudInAnalysisMode"
	FlagNightVision       = "NightVision"
	FlagFsdJump           = "FsdJump"
	FlagSrvHighBeam       = "SrvHighBeam"
	FlagOnFoot            = "OnFoot"
	FlagInTaxi            = "InTaxi"
	FlagLowOxygen         = "LowOxygen"
	FlagLowHealth         = "LowHealth"
	FlagGlideMode         = "GlideMode"
	FlagBreathableAtmos   = "BreathableAtmosphere"
	FlagHyperdriveCharged = "FsdHyperdriveCharging"
)

// flagBits maps each reactable bit of the Flags field to its name. The values are
// the game's own bit assignments and are therefore data, not magic numbers.
var flagBits = map[uint32]string{
	1:          FlagDocked,
	2:          FlagLanded,
	4:          FlagLandingGearDown,
	8:          FlagShieldsUp,
	16:         FlagSupercruise,
	32:         FlagFlightAssistOff,
	64:         FlagHardpointsDeploy,
	128:        FlagInWing,
	256:        FlagLightsOn,
	512:        FlagCargoScoopDeploy,
	1024:       FlagSilentRunning,
	2048:       FlagScoopingFuel,
	4096:       FlagSrvHandbrake,
	8192:       FlagSrvTurret,
	32768:      FlagSrvDriveAssist,
	65536:      FlagMassLocked,
	131072:     FlagFsdCharging,
	262144:     FlagFsdCooldown,
	524288:     FlagLowFuel,
	1048576:    FlagOverHeating,
	4194304:    FlagIsInDanger,
	8388608:    FlagBeingInterdicted,
	16777216:   FlagInMainShip,
	33554432:   FlagInFighter,
	67108864:   FlagInSRV,
	134217728:  FlagAnalysisMode,
	268435456:  FlagNightVision,
	1073741824: FlagFsdJump,
	2147483648: FlagSrvHighBeam,
}

// flag2Bits maps the reactable bits of the Odyssey Flags2 field to their names.
var flag2Bits = map[uint32]string{
	1:      FlagOnFoot,
	2:      FlagInTaxi,
	64:     FlagLowOxygen,
	128:    FlagLowHealth,
	4096:   FlagGlideMode,
	65536:  FlagBreathableAtmos,
	524288: FlagHyperdriveCharged,
}

// Derived value names, emitted when a non-bit field changes.
const (
	// ValueGuiFocus is the panel the commander is looking at.
	ValueGuiFocus = "GuiFocus"
	// ValueFireGroup is the selected fire group.
	ValueFireGroup = "FireGroup"
	// ValuePips is the power distribution, reported as its dominant capacitor.
	ValuePips = "Pips"
)

// GUI focus values, named so the cue table never carries a bare number.
const (
	GuiNoFocus         = "NoFocus"
	GuiInternalPanel   = "InternalPanel"
	GuiExternalPanel   = "ExternalPanel"
	GuiCommsPanel      = "CommsPanel"
	GuiRolePanel       = "RolePanel"
	GuiStationServices = "StationServices"
	GuiGalaxyMap       = "GalaxyMap"
	GuiSystemMap       = "SystemMap"
	GuiOrrery          = "Orrery"
	GuiFssMode         = "FssMode"
	GuiSaaMode         = "SaaMode"
	GuiCodex           = "Codex"
)

// guiFocusNames maps the game's numeric focus to its name.
var guiFocusNames = map[int]string{
	0:  GuiNoFocus,
	1:  GuiInternalPanel,
	2:  GuiExternalPanel,
	3:  GuiCommsPanel,
	4:  GuiRolePanel,
	5:  GuiStationServices,
	6:  GuiGalaxyMap,
	7:  GuiSystemMap,
	8:  GuiOrrery,
	9:  GuiFssMode,
	10: GuiSaaMode,
	11: GuiCodex,
}

// Pip distribution names.
const (
	PipsBalanced = "Balanced"
	PipsSystems  = "Systems"
	PipsEngines  = "Engines"
	PipsWeapons  = "Weapons"
)

// pipIndex names the three capacitors in the order the game reports them.
const (
	pipSystems = 0
	pipEngines = 1
	pipWeapons = 2
	pipCount   = 3
)

// dominantPips reduces the three pip values to the capacitor holding the most;
// balanced when no one capacitor leads. The game reports pips in half-pip units, so
// the values are compared rather than interpreted.
func dominantPips(pips []int) string {
	if len(pips) < pipCount {
		return PipsBalanced
	}
	systems, engines, weapons := pips[pipSystems], pips[pipEngines], pips[pipWeapons]
	switch {
	case systems > engines && systems > weapons:
		return PipsSystems
	case engines > systems && engines > weapons:
		return PipsEngines
	case weapons > systems && weapons > engines:
		return PipsWeapons
	default:
		return PipsBalanced
	}
}

// guiFocusName renders a numeric focus, falling back to no focus for a value this
// application does not know, which is how a game update adding a panel behaves.
func guiFocusName(value int) string {
	if name, ok := guiFocusNames[value]; ok {
		return name
	}
	return GuiNoFocus
}

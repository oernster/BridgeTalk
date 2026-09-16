//go:build !windows && !linux

package setup

// SetLaunchOnBoot has no sign-in entry to write where neither Windows nor Linux runs.
func SetLaunchOnBoot(string, bool) error { return ErrUnsupported }

// IsLaunchOnBoot reports no sign-in entry where neither Windows nor Linux runs.
func IsLaunchOnBoot() bool { return false }

// HasLaunchOnBootEntry reports no sign-in entry where neither Windows nor Linux runs.
func HasLaunchOnBootEntry() bool { return false }

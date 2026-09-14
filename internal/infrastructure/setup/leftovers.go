package setup

import "errors"

// Leftovers names what the application keeps outside its install directory that an uninstall
// deletes (FR-525, FR-805).
type Leftovers struct {
	// MadeLines is the folder the lines made for machine voices are kept in.
	MadeLines string
	// State is the folder the window keeps its theme and volume in.
	State string
}

// RemoveLeftovers deletes the made lines' folder whatever is asked (FR-525), then the window's state
// where forget is set (FR-805). Each folder is deleted whole with nothing beside it touched, so the
// recordings kept next to the made lines stay. A folder that cannot be deleted does not stop the
// other; every refusal comes back joined. An empty folder name removes nothing, as os.RemoveAll has
// it.
func RemoveLeftovers(left Leftovers, forget bool) error {
	failed := []error{RemoveTree(left.MadeLines)}
	if forget {
		failed = append(failed, RemoveTree(left.State))
	}
	return errors.Join(failed...)
}

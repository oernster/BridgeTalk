package config

// A settings file another program holds with no read sharing cannot be read; it can be let go a
// moment later. A choice made from what was read then is built from nothing, so saving it would put
// one choice in place of all of them. The save is refused while the last read failed.

import (
	"os"
	"reflect"
	"syscall"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
)

// noSharing opens a file the way a program that lets no one else read it does.
const noSharing = 0

func TestASettingsFileThatCouldNotBeReadIsNotSavedOver(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	if err := store.Save(ports.Settings{Voice: "Alice", SwitchedOff: []string{"FSDJump"}}); err != nil {
		t.Fatalf("seeding: %v", err)
	}
	whole, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	name, err := syscall.UTF16PtrFromString(store.path)
	if err != nil {
		t.Fatalf("naming: %v", err)
	}
	held, err := syscall.CreateFile(name, syscall.GENERIC_READ, noSharing, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatalf("holding the file: %v", err)
	}

	chosen := store.Load()
	_ = syscall.CloseHandle(held)
	chosen.SkippedUpdate = "9.9.9"
	if err := store.Save(chosen); err == nil {
		t.Error("a save built from a file that could not be read reported success")
	}
	if after, _ := os.ReadFile(store.path); !reflect.DeepEqual(after, whole) {
		t.Errorf("the file reads %q, want it as it was", after)
	}
	if store.Problem() == nil {
		t.Error("the store says nothing is wrong, want it to say the file could not be read")
	}
}

// readSharing lets others read a held file but neither write nor move it.
const readSharing = syscall.FILE_SHARE_READ

// A damaged file that can be read but not moved aside, since another program holds it, is not
// saved over either: it is the only copy of the choices it held.
func TestADamagedSettingsFileThatCannotBeKeptAsideIsNotSavedOver(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	damaged := seededThenDamaged(t, store)
	name, err := syscall.UTF16PtrFromString(store.path)
	if err != nil {
		t.Fatalf("naming: %v", err)
	}
	held, err := syscall.CreateFile(name, syscall.GENERIC_READ, readSharing, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatalf("holding the file: %v", err)
	}

	chosen := store.Load()
	_ = syscall.CloseHandle(held)
	chosen.SkippedUpdate = "9.9.9"
	if err := store.Save(chosen); err == nil {
		t.Error("a save over a damaged file that could not be kept aside reported success")
	}
	if after, _ := os.ReadFile(store.path); !reflect.DeepEqual(after, damaged) {
		t.Errorf("the file reads %q, want the damaged file as it was", after)
	}
}

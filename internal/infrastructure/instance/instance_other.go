//go:build !windows

package instance

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync/atomic"
	"syscall"

	"golang.org/x/sys/unix"
)

// The suffixes naming the file whose lock is the claim and the pipe a later start writes to summon.
const (
	claimSuffix  = ".lock"
	summonSuffix = ".summon"
)

// Permissions for the data folder, the lock file and the pipe: the user's own.
const (
	dirPerm  = 0o755
	filePerm = 0o600
)

// summonByte is what a later start writes down the pipe; any byte would do.
var summonByte = []byte{1}

// take claims the one copy by locking a file in dir: the first start holds the lock for the run; a
// later start finds it held. The kernel lets the lock go when the holder's file is closed, however
// the holder ended. A pipe beside the file carries a later start's summons.
func take(name, dir string, summon bool) (*Claim, error) {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return inert(), fmt.Errorf("claiming the one copy: %w", err)
	}
	lock, err := os.OpenFile(filepath.Join(dir, name+claimSuffix), os.O_CREATE|os.O_RDWR, filePerm)
	if err != nil {
		return inert(), fmt.Errorf("claiming the one copy: %w", err)
	}
	pipe := filepath.Join(dir, name+summonSuffix)
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = lock.Close()
		if !errors.Is(err, unix.EWOULDBLOCK) {
			return inert(), fmt.Errorf("claiming the one copy: %w", err)
		}
		if summon {
			return nil, summonHolder(pipe)
		}
		return nil, ErrRunning
	}
	// A leftover that is not a pipe would be read at once and forever, so it is replaced.
	if info, err := os.Lstat(pipe); err == nil && info.Mode()&fs.ModeNamedPipe == 0 {
		_ = os.Remove(pipe)
	}
	if err := unix.Mkfifo(pipe, filePerm); err != nil && !errors.Is(err, unix.EEXIST) {
		_ = lock.Close()
		return inert(), fmt.Errorf("making the summons: %w", err)
	}

	var stopping atomic.Bool
	done := make(chan struct{})
	claim := newClaim(func() {
		stopping.Store(true)
		// Opening the pipe to write wakes a listener waiting to open it to read.
		if wake, err := openPipe(pipe, os.O_WRONLY|syscall.O_NONBLOCK); err == nil {
			_ = wake.Close()
		}
		<-done
		_ = lock.Close()
	})
	go listen(claim, pipe, &stopping, done)
	return claim, nil
}

// listen passes each summons down the pipe to the claim until the claim is released. A fault here
// ends the listening and is said in the run log, never the application.
func listen(claim *Claim, pipe string, stopping *atomic.Bool, done chan<- struct{}) {
	defer close(done)
	defer func() {
		if fault := recover(); fault != nil {
			fmt.Fprintf(os.Stderr, "listening for a second start stopped: %v\n%s\n", fault, debug.Stack())
		}
	}()
	for {
		reader, err := openPipe(pipe, os.O_RDONLY)
		if err != nil {
			fmt.Fprintf(os.Stderr, "listening for a second start stopped: %v\n", err)
			return
		}
		_, _ = io.Copy(io.Discard, reader)
		_ = reader.Close()
		if stopping.Load() {
			return
		}
		claim.summon()
	}
}

// summonHolder writes to the running copy's pipe, then answers ErrRunning. A summons that cannot be
// written is said alongside it, since the running copy then stays as it is.
func summonHolder(pipe string) error {
	writer, err := openPipe(pipe, os.O_WRONLY|syscall.O_NONBLOCK)
	if err != nil {
		return fmt.Errorf("%w; asking it for its window: %v", ErrRunning, err)
	}
	defer writer.Close()
	if _, err := writer.Write(summonByte); err != nil {
		return fmt.Errorf("%w; asking it for its window: %v", ErrRunning, err)
	}
	return ErrRunning
}

// openPipe opens the summons pipe, the one place it is opened. Opening to read waits for a writer;
// opening to write without waiting fails at once where nobody is reading.
func openPipe(pipe string, flag int) (*os.File, error) {
	return os.OpenFile(pipe, flag, 0)
}

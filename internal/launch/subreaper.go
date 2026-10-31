package launch

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

const (
	prSetPdeathsig      = 1
	prSetChildSubreaper = 36
	prGetChildSubreaper = 37
)

// BecomeSubreaper marks the calling process as a child subreaper (prctl PR_SET_CHILD_SUBREAPER).
// From then on every orphaned descendant (Wine detaches wineserver, services.exe, and the game itself
// from the short-lived `proton`/steam.exe launchers) is re-parented to THIS process instead of PID 1.
// Because they stay descendants, Yama ptrace_scope=1 lets us open /proc/<pid>/mem with no capability.
//
// Call it once, early in main, before starting any Wine process. It affects the whole process
// (thread group), so any thread may call it. VERIFIED on the worker, see docs/launch.md.
func BecomeSubreaper() error {
	if _, _, e := syscall.Syscall(syscall.SYS_PRCTL, prSetChildSubreaper, 1, 0); e != 0 {
		return fmt.Errorf("prctl(PR_SET_CHILD_SUBREAPER): %w", e)
	}
	return nil
}

// IsSubreaper reports whether the calling process is a child subreaper.
func IsSubreaper() (bool, error) {
	var v int32
	if _, _, e := syscall.Syscall(syscall.SYS_PRCTL, prGetChildSubreaper, uintptr(unsafe.Pointer(&v)), 0); e != 0 {
		return false, e
	}
	return v != 0, nil
}

// CloseOnExecInherited marks every fd above 2 that this process inherited as close-on-exec. A live run holds
// the per-instance flock on such an fd (`flock ... cmd`); without this the Wine processes started from here
// keep it for their whole life and block every later run (d2game does the same, see its main).
func CloseOnExecInherited() {
	ents, _ := os.ReadDir("/proc/self/fd")
	for _, e := range ents {
		if n, err := strconv.Atoi(e.Name()); err == nil && n > 2 {
			syscall.CloseOnExec(n)
		}
	}
}

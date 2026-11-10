//go:build !linux

package launch

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

// ErrUnsupportedOS is returned by every Wine/Xvfb lifecycle entry point that needs Linux process
// control. It exists so d2bot and d2dsetup compile for windows and darwin; their d2d path never calls it.
var ErrUnsupportedOS = fmt.Errorf("launch: Wine/Xvfb lifecycle is not supported on %s", runtime.GOOS)

func sysKill(int, syscall.Signal) error { return ErrUnsupportedOS }

func tryFlock(*os.File) (bool, error) { return false, ErrUnsupportedOS }
func unFlock(*os.File)                {}
func wait4NoHang(int) error           { return ErrUnsupportedOS }

func setSession(*exec.Cmd)   {}
func setGroup(*exec.Cmd)     {}
func setPdeathsig(*exec.Cmd) {}

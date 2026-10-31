package launch

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// Display is the X display a game instance renders on.
//
// HostDisplay is the user's desktop (i0, the original game): never created or stopped by us.
// XvfbDisplay is a private virtual display per instance (i1..i5): created on demand, stopped with the
// instance. The interface is the seam for other hosts later (container Xvfb, Xvnc, a real GPU session).
type Display interface {
	Name() string // ":101"
	// Env is the environment that makes a child render here (and nothing else of the desktop).
	Env() []string
	// Ensure makes the display exist.
	Ensure(ctx context.Context) error
	// Stop removes the display; stopped reports whether we actually stopped one.
	Stop(ctx context.Context) (stopped bool, err error)
}

// HostDisplay is the desktop session (XWayland). A bare ssh shell has none and the game dies at once
// (VERIFIED LIVE, tools/game.sh history), so the session variables are passed explicitly.
type HostDisplay struct {
	Display    string // ":1"
	Wayland    string // "wayland-0"
	RuntimeDir string // "/run/user/1000"
	XAuthority string // first /run/user/<uid>/xauth_*; "" if none
}

func (h HostDisplay) Name() string { return h.Display }
func (h HostDisplay) Env() []string {
	e := []string{"DISPLAY=" + h.Display, "WAYLAND_DISPLAY=" + h.Wayland, "XDG_RUNTIME_DIR=" + h.RuntimeDir}
	if h.XAuthority != "" {
		e = append(e, "XAUTHORITY="+h.XAuthority)
	}
	return e
}
func (HostDisplay) Ensure(context.Context) error       { return nil }
func (HostDisplay) Stop(context.Context) (bool, error) { return false, nil }

// XvfbDisplay is a private Xvfb on display :N (no window manager, no TCP).
type XvfbDisplay struct {
	Sys    System
	Num    int    // 101 -> ":101"
	Size   string // "800x600x24"
	LogDir string // xvfb-<id>.out goes here
	ID     string
}

func (x XvfbDisplay) Name() string { return fmt.Sprintf(":%d", x.Num) }

// Env clears the desktop session variables: nothing of the user's session may leak into the game.
func (x XvfbDisplay) Env() []string { return []string{"DISPLAY=" + x.Name()} }
func (x XvfbDisplay) socket() string {
	return fmt.Sprintf("/tmp/.X11-unix/X%d", x.Num)
}

func (x XvfbDisplay) size() string {
	if x.Size == "" {
		return "800x600x24"
	}
	return x.Size
}

func (x XvfbDisplay) Ensure(ctx context.Context) error {
	if x.Sys.Exists(x.socket()) {
		return nil
	}
	log := filepath.Join(x.LogDir, "xvfb-"+x.ID+".out")
	if err := x.Sys.Spawn(nil, "", log, "Xvfb", x.Name(), "-screen", "0", x.size(), "-nolisten", "tcp"); err != nil {
		return fmt.Errorf("Xvfb %s: %w", x.Name(), err)
	}
	x.Sys.Sleep(ctx, 1e9) // the socket appears a moment after the process starts
	return nil
}

func (x XvfbDisplay) pid() int {
	ps, err := x.Sys.Procs()
	if err != nil {
		return 0
	}
	for _, p := range ps {
		if len(p.Argv) >= 2 && filepath.Base(p.Argv[0]) == "Xvfb" && p.Argv[1] == x.Name() {
			return p.PID
		}
	}
	return 0
}

func (x XvfbDisplay) Stop(ctx context.Context) (bool, error) {
	p := x.pid()
	if p == 0 {
		return false, nil
	}
	if err := x.Sys.Signal(p, 15); err != nil {
		return false, err
	}
	return true, nil
}

// envWithout returns env minus the given variable names (later entries of base are NOT deduplicated:
// exec keeps the last one, so extra entries appended afterwards win).
func envWithout(env []string, names ...string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		drop := false
		for _, n := range names {
			if strings.HasPrefix(kv, n+"=") {
				drop = true
			}
		}
		if !drop {
			out = append(out, kv)
		}
	}
	return out
}

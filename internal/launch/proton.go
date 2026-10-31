package launch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

// Env returns the environment for a Proton invocation. Proton is run directly (no umu): it only needs
// STEAM_COMPAT_DATA_PATH (prefix parent) and STEAM_COMPAT_CLIENT_INSTALL_PATH. VERIFIED with GE-Proton11-11.
func (c Config) Env() []string {
	env := os.Environ()
	env = append(env,
		"STEAM_COMPAT_CLIENT_INSTALL_PATH="+c.SteamRoot,
		"STEAM_COMPAT_DATA_PATH="+c.PrefixDir,
		"PROTON_LOG=1",
		"PROTON_LOG_DIR="+c.LogDir,
	)
	if c.Instance != "" {
		env = append(env, "D2BOT_INSTANCE="+c.Instance)
	}
	return append(env, c.ExtraEnv...)
}

// ProtonCmd builds `proton <verb> args...` (verb is "run" for normal use). The command is not started.
// cwd is the working directory ("" for the current one). Do not pass a context that you cancel to stop
// the game: ctx cancellation would kill only the python launcher.
func (c Config) ProtonCmd(ctx context.Context, cwd, verb string, args ...string) *exec.Cmd {
	// Proton does not create STEAM_COMPAT_DATA_PATH itself (it dies on the missing pfx.lock). VERIFIED.
	os.MkdirAll(c.PrefixDir, 0o755)
	os.MkdirAll(c.LogDir, 0o755)
	name, argv, werr := c.Wrap(cwd, c.ProtonScript, append([]string{verb}, args...)...)
	cmd := exec.CommandContext(ctx, name, argv...)
	if werr != nil {
		cmd.Err = werr // Start returns it
	}
	cmd.Dir = cwd
	cmd.Env = c.Env()
	// Own process group: a ^C in the app's terminal must not hit Wine directly; we shut it down ourselves.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// WineserverKill runs `wineserver -k` for the prefix: the only reliable way to stop everything in it
// (game, helper, services.exe...). Uses the runner's own wineserver binary.
func (c Config) WineserverKill() error {
	ws := filepath.Join(filepath.Dir(c.ProtonScript), "files", "bin", "wineserver")
	cmd := exec.Command(ws, "-k")
	cmd.Env = append(c.Env(), "WINEPREFIX="+filepath.Join(c.PrefixDir, "pfx"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wineserver -k: %w: %s", err, out)
	}
	return nil
}

// WineserverWait runs `wineserver -w` for the prefix: it returns when the prefix's wineserver has exited (at once when
// none runs) or when ctx ends. UNVERIFIED LIVE: the -w flag is documented in wineserver(1) and not yet run on the runner's binary.
func (c Config) WineserverWait(ctx context.Context) error {
	ws := filepath.Join(filepath.Dir(c.ProtonScript), "files", "bin", "wineserver")
	cmd := exec.CommandContext(ctx, ws, "-w")
	cmd.Env = append(c.Env(), "WINEPREFIX="+filepath.Join(c.PrefixDir, "pfx"))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("wineserver -w: %w: %s", err, out)
	}
	return nil
}

// Spawner starts child processes from one dedicated, locked OS thread that lives until the process
// exits. Go's SysProcAttr.Pdeathsig is delivered when the *thread* that forked dies, and Go retires
// threads freely, so Pdeathsig set from an ordinary goroutine can fire spuriously. Starting from a
// locked thread makes it mean "this app died".
//
// Pdeathsig only covers the direct child (the `proton` python launcher), not the Wine processes under
// it, so it is a minor safety net. The real cleanup is WineserverKill on shutdown.
type Spawner struct {
	reqs chan spawnReq
}

type spawnReq struct {
	cmd  *exec.Cmd
	done chan error
}

// NewSpawner starts the locked thread.
func NewSpawner() *Spawner {
	s := &Spawner{reqs: make(chan spawnReq)}
	go func() {
		runtime.LockOSThread() // never unlocked: the thread dies with the process
		for r := range s.reqs {
			r.done <- r.cmd.Start()
		}
	}()
	return s
}

// Start starts cmd from the locked thread with Pdeathsig=SIGTERM (cmd.SysProcAttr must be set).
func (s *Spawner) Start(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGTERM
	r := spawnReq{cmd, make(chan error, 1)}
	s.reqs <- r
	return <-r.done
}

package launch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Step is one install action with an independent verifier, so a half-finished setup resumes where it
// stopped: the runner skips steps whose Verify already passes, and re-verifies after Do.
type Step struct {
	Name string
	// Interactive means the user must act in a Wine window (type the CD key). The app never sees or
	// stores keys.
	Interactive bool
	Do          func(ctx context.Context) error
	Verify      func() error
}

// Installer builds the step list for one setup.
type Installer struct {
	Cfg Config
	// StubD2 and StubLoD are the user's Blizzard downloader stubs (Downloader_Diablo2_enUS.exe and
	// Downloader_Diablo2_Lord_of_Destruction_enUS.exe).
	StubD2, StubLoD string
	// PatchDir is where the patch is cached (default Cfg.Root/downloads).
	PatchDir string
	// StepTimeout bounds each interactive wait (default 45 min: the stubs download 0.5-1.5 GB).
	StepTimeout time.Duration
	// Log receives progress lines; may be nil.
	Log func(string)
}

func (in *Installer) logf(f string, a ...any) {
	if in.Log != nil {
		in.Log(fmt.Sprintf(f, a...))
	}
}

func (in *Installer) desktop() string {
	// Proton's default user inside the prefix is "steamuser". VERIFIED.
	return filepath.Join(in.Cfg.PrefixDir, "pfx", "drive_c", "users", "steamuser", "Desktop")
}

func (in *Installer) patchPath() string {
	d := in.PatchDir
	if d == "" {
		d = filepath.Join(in.Cfg.Root, "downloads")
	}
	return filepath.Join(d, "LODPatch_114d.exe")
}

// Steps returns the ordered install plan.
func (in *Installer) Steps() []Step {
	cfg := in.Cfg
	return []Step{
		{
			Name: "prefix",
			// Any Proton invocation creates the prefix on first use. A no-op program keeps it quiet.
			Do: func(ctx context.Context) error {
				return in.runProton(ctx, "", "cmd.exe", "/c", "exit")
			},
			Verify: func() error { return needFile(filepath.Join(cfg.PrefixDir, "pfx", "system.reg")) },
		},
		{
			Name: "install-d2", Interactive: true,
			Do:     func(ctx context.Context) error { return in.runStub(ctx, in.StubD2, in.verifyBase) },
			Verify: in.verifyBase,
		},
		{
			Name: "install-lod", Interactive: true,
			Do:     func(ctx context.Context) error { return in.runStub(ctx, in.StubLoD, in.verifyLoD) },
			Verify: in.verifyLoD,
		},
		{
			Name: "patch-download",
			Do: func(ctx context.Context) error {
				if err := os.MkdirAll(filepath.Dir(in.patchPath()), 0o755); err != nil {
					return err
				}
				return Fetch(ctx, nil, Download{URL: PatchURL, Dest: in.patchPath(), SHA256: PatchSHA256, Size: PatchSize})
			},
			Verify: func() error {
				ok, err := fileMatches(in.patchPath(), PatchSHA256)
				if err != nil || !ok {
					return fmt.Errorf("patch missing or wrong hash: %v", err)
				}
				return nil
			},
		},
		{
			Name: "patch-apply", Interactive: true,
			// The patch is a Blizzard GUI updater that finds the install through the registry InstallPath
			// the installer wrote (VERIFIED string in the exe). A silent switch is UNVERIFIED, so this
			// step shows its window; the user clicks through. It may launch the game when done.
			Do: func(ctx context.Context) error {
				return in.runExeUntil(ctx, in.patchPath(), nil, in.verifyVersion)
			},
			Verify: in.verifyVersion,
		},
	}
}

// Run executes the plan: for each step, skip if Verify passes, else Do then Verify.
func (in *Installer) Run(ctx context.Context) error {
	for _, s := range in.Steps() {
		if s.Verify() == nil {
			in.logf("%s: already done", s.Name)
			continue
		}
		in.logf("%s: running", s.Name)
		if err := s.Do(ctx); err != nil {
			return fmt.Errorf("%s: %w", s.Name, err)
		}
		if err := s.Verify(); err != nil {
			return fmt.Errorf("%s: verify: %w", s.Name, err)
		}
		in.logf("%s: ok", s.Name)
	}
	return nil
}

func needFile(p string) error {
	if _, err := os.Stat(p); err != nil {
		return err
	}
	return nil
}

// verifyBase: the base installer's own log says it finished, and the core MPQs exist. VERIFIED file
// names from a real install.
func (in *Installer) verifyBase() error {
	return in.verifyInstall("Diablo II Install Log.html", "d2data.mpq", "d2char.mpq", "d2sfx.mpq", "Game.exe")
}

func (in *Installer) verifyLoD() error {
	return in.verifyInstall("Diablo II LOD Install Log.html", "d2exp.mpq", "d2xmusic.mpq", "d2xtalk.mpq", "d2xvideo.mpq")
}

func (in *Installer) verifyInstall(log string, files ...string) error {
	b, err := os.ReadFile(filepath.Join(in.Cfg.GameDir, log))
	if err != nil {
		return err
	}
	if !strings.Contains(string(b), "Installation completed successfully") {
		return errors.New(log + ": no 'Installation completed successfully'")
	}
	for _, f := range files {
		if err := needFile(filepath.Join(in.Cfg.GameDir, f)); err != nil {
			return err
		}
	}
	return nil
}

func (in *Installer) verifyVersion() error {
	v, err := FileVersion(in.Cfg.RealGameExe())
	if err != nil {
		return err
	}
	if v != Want114d {
		return fmt.Errorf("Game.exe is %s, want %s", v, Want114d)
	}
	return nil
}

func (in *Installer) runProton(ctx context.Context, cwd string, args ...string) error {
	cmd := in.Cfg.ProtonCmd(ctx, cwd, "run", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, tail(string(out), 400))
	}
	return nil
}

// runStub runs a downloader stub and waits until done() passes (or the timeout/ctx ends). The stub
// downloads its installer into Desktop/<name>, starts it, and the installer asks for the CD key in its own
// window. The stub process may exit before the installer finishes (UNVERIFIED), so completion is judged
// by done(), not by the exit of `proton run`.
func (in *Installer) runStub(ctx context.Context, stub string, done func() error) error {
	if stub == "" {
		return errors.New("stub path not configured")
	}
	if err := needFile(stub); err != nil {
		return err
	}
	return in.runExeUntil(ctx, stub, nil, done)
}

func (in *Installer) runExeUntil(ctx context.Context, exe string, args []string, done func() error) error {
	timeout := in.StepTimeout
	if timeout == 0 {
		timeout = 45 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := in.Cfg.ProtonCmd(context.Background(), filepath.Dir(exe), "run", append([]string{exe}, args...)...)
	if err := cmd.Start(); err != nil {
		return err
	}
	exited := make(chan struct{})
	go func() { cmd.Wait(); close(exited) }()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for %s: %w", filepath.Base(exe), ctx.Err())
		case <-tick.C:
			if done() == nil {
				return nil
			}
		case <-exited:
			// The launcher chain ended; give the installer/patch a last chance to be judged finished.
			exited = nil
			if done() == nil {
				return nil
			}
		}
	}
}

func tail(s string, n int) string {
	if len(s) > n {
		return "..." + s[len(s)-n:]
	}
	return s
}

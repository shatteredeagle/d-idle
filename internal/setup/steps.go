package setup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/paths"
)

// Step ids, stable (they are in the JSON contract and in the persisted state).
const (
	StepDataDir   = "data-dir"
	StepSources   = "sources"
	StepPrefix    = "prefix"
	StepInstallD2 = "install-d2"
	StepInstallLo = "install-lod"
	StepImport    = "import-game"
	StepPatch     = "patch"
	StepExtract   = "extract"
	StepCharacter = "character"
)

// Deps are the effects of the default steps; tests replace them. NewDeps returns the real ones.
type Deps struct {
	Env     *Env
	Display string // X display for installer windows (":99"), "" = inherit

	// Config builds the resolved launch.Config for a data dir (finds the Proton runner).
	Config func(s *State) (launch.Config, error)
	// Prefix creates the base prefix (first Proton run).
	Prefix func(ctx context.Context, cfg launch.Config) error
	// RunWindowed runs a Windows exe in the prefix and returns when done() passes, the exe chain exits
	// and done() passes, or ctx ends. The user acts in its window (CD keys are typed there).
	RunWindowed func(ctx context.Context, cfg launch.Config, cwd, exe string, done func() error) error
	// VerifyPieces checks a pre-downloaded installer folder against the torrent embedded in the user's stub.
	VerifyPieces func(stub, dir string) error
	// Copy copies a tree (regular files only; symlinks are refused).
	Copy func(ctx context.Context, src, dst string, logf func(string, ...any)) error
	// TablesDir is the extraction target for the install, CacheCurrent and Extract as in internal/data.
	TablesDir    func(dataDir, gameDir string) (string, error)
	CacheCurrent func(gameDir, tablesDir string) bool
	Extract      func(gameDir, tablesDir string) error
	// CreateCharacter starts the game, creates the softcore expansion character and saves and exits.
	CreateCharacter func(ctx context.Context, cfg launch.Config, name, class string) error
}

// NewDeps returns the real implementations.
func NewDeps(env *Env) *Deps {
	d := &Deps{Env: env}
	d.Config = d.defaultConfig
	d.Prefix = func(ctx context.Context, cfg launch.Config) error {
		in := &launch.Installer{Cfg: cfg}
		return stepNamed(in, "prefix").Do(ctx)
	}
	d.RunWindowed = runWindowed
	d.VerifyPieces = func(stub, dir string) error {
		m, err := launch.ReadStubMeta(stub)
		if err != nil {
			return err
		}
		return m.VerifyDir(dir, true)
	}
	d.Copy = copyTree
	d.TablesDir = defaultTablesDir
	d.CacheCurrent = data.CacheCurrent
	d.Extract = data.Extract
	d.CreateCharacter = createCharacter
	return d
}

func (d *Deps) defaultConfig(s *State) (launch.Config, error) {
	l := paths.Layout{Root: s.DataDir}
	c := launch.Config{Root: s.DataDir, PrefixDir: BasePrefix(s.DataDir), LogDir: l.Logs()}
	if d.Display != "" {
		c.ExtraEnv = append(c.ExtraEnv, "DISPLAY="+d.Display)
	}
	if err := c.Resolve(); err != nil {
		return c, fmt.Errorf("%w (doctor check host.runner)", err)
	}
	return c, nil
}

func stepNamed(in *launch.Installer, name string) launch.Step {
	for _, s := range in.Steps() {
		if s.Name == name {
			return s
		}
	}
	panic("setup: launch.Installer has no step " + name)
}

// TablesDirFor is the extraction directory of the install in gameDir: <data>/cache/data/<schema>-<stamp>
// (paths.Layout.DataTables), where stamp is the first 12 hex digits of the Patch_D2.mpq sha256.
func TablesDirFor(dataDir, gameDir string) (string, error) { return defaultTablesDir(dataDir, gameDir) }

func defaultTablesDir(dataDir, gameDir string) (string, error) {
	st, err := data.InstallStamp(gameDir)
	if err != nil {
		return "", err
	}
	stamp := "nopatch"
	if len(st.PatchSHA256) >= 12 {
		stamp = st.PatchSHA256[:12]
	}
	return paths.Layout{Root: dataDir}.DataTables(fmt.Sprint(st.Schema), stamp), nil
}

// NewSteps is the default plan:
//
//	data-dir -> sources -> prefix -> install-d2 -> install-lod (or import-game) -> patch -> extract -> character
func NewSteps(d *Deps) []StepDef {
	game := func(s *State) (string, error) {
		c, err := d.Config(s)
		return c.GameDir, err
	}
	installed := func(s *State) bool { // base and LoD present in the prefix's game dir
		g, err := game(s)
		return err == nil && d.gameFilesOK(g) == nil
	}
	installer := func(s *State) (*launch.Installer, launch.Config, error) {
		c, err := d.Config(s)
		return &launch.Installer{Cfg: c}, c, err
	}
	notImport := func(s *State) bool { return s.Mode != ModeGameDir }

	installStep := func(id, title, stubOf string, dirOf func(*State) string, verifyName string) StepDef {
		stub := func(s *State) string {
			if stubOf == "d2" {
				return s.StubD2
			}
			return s.StubLoD
		}
		return StepDef{
			ID: id, Title: title, Interactive: true, Applies: notImport,
			Ready: func(s *State) error {
				switch s.Mode {
				case ModeStubs:
					return needFile(stub(s), "the Blizzard downloader stub you supplied")
				case ModeInstallerDirs:
					return needFile(filepath.Join(dirOf(s), "Installer.exe"), "Installer.exe in the installer folder you supplied")
				}
				return fmt.Errorf("choose a source mode first (%s, %s or %s)", ModeStubs, ModeInstallerDirs, ModeGameDir)
			},
			Verify: func(ctx context.Context, s *State) error {
				in, _, err := installer(s)
				if err != nil {
					return err
				}
				return stepNamed(in, verifyName).Verify()
			},
			Do: func(ctx context.Context, s *State, r *Run) error {
				in, cfg, err := installer(s)
				if err != nil {
					return err
				}
				exe, cwd := stub(s), filepath.Dir(stub(s))
				if s.Mode == ModeInstallerDirs {
					cwd = dirOf(s)
					exe = filepath.Join(cwd, "Installer.exe")
					if stub(s) != "" { // optional: prove the user's download is complete and unmodified
						r.Logf("verifying installer folder against the stub's piece hashes")
						if err := d.VerifyPieces(stub(s), cwd); err != nil {
							return fmt.Errorf("installer folder %s: %w", cwd, err)
						}
					}
				}
				r.WaitingUser("Installer window is open: click through it and type your own CD key there. This app never sees it.")
				r.Logf("running %s in the prefix", filepath.Base(exe))
				return d.RunWindowed(ctx, cfg, cwd, exe, stepNamed(in, verifyName).Verify)
			},
		}
	}

	return []StepDef{
		{
			ID: StepDataDir, Title: "Choose data directory",
			Ready: func(s *State) error {
				if !filepath.IsAbs(s.DataDir) {
					return errors.New("dataDir must be an absolute path")
				}
				return nil
			},
			Verify: func(ctx context.Context, s *State) error {
				if !filepath.IsAbs(s.DataDir) {
					return errors.New("no data dir chosen")
				}
				if st, err := d.Env.Stat(s.DataDir); err != nil || !st.IsDir() {
					return errors.New("data dir does not exist yet")
				}
				// fs.prefix must not fail: NTFS and noexec mounts cannot host a Wine prefix.
				o := Options{DataDir: s.DataDir, Slots: s.Slots}
				o.Normalize()
				if r := checkFS(ctx, d.Env, &o); r.Status == Fail {
					return errors.New(r.Detail + ". " + r.Fix)
				}
				return nil
			},
			Do: func(ctx context.Context, s *State, r *Run) error {
				l := paths.Layout{Root: s.DataDir}
				for _, p := range []string{l.Root, l.PrefixesDir(), l.CacheData(), l.Logs(), filepath.Join(l.Root, "downloads"), filepath.Join(l.Root, "helper")} {
					if err := os.MkdirAll(p, 0o755); err != nil {
						return err
					}
				}
				f, err := os.CreateTemp(l.Root, ".writetest-*")
				if err != nil {
					return fmt.Errorf("data dir not writable: %w", err)
				}
				f.Close()
				return os.Remove(f.Name())
			},
		},
		{
			ID: StepSources, Title: "Locate your Diablo II files",
			Verify: func(ctx context.Context, s *State) error {
				if installed(s) {
					return nil // nothing to locate: the game is already in the prefix
				}
				return d.validateSources(s)
			},
			Do: func(ctx context.Context, s *State, r *Run) error { return d.validateSources(s) },
		},
		{
			ID: StepPrefix, Title: "Create base Wine prefix",
			Verify: func(ctx context.Context, s *State) error {
				in, _, err := installer(s)
				if err != nil {
					return err
				}
				return stepNamed(in, "prefix").Verify()
			},
			Do: func(ctx context.Context, s *State, r *Run) error {
				_, cfg, err := installer(s)
				if err != nil {
					return err
				}
				r.Logf("first Wine run builds the prefix (about 25 s)")
				return d.Prefix(ctx, cfg)
			},
		},
		installStep(StepInstallD2, "Install Diablo II", "d2", func(s *State) string { return s.InstallerD2Dir }, "install-d2"),
		installStep(StepInstallLo, "Install Lord of Destruction", "lod", func(s *State) string { return s.InstallerLoDDir }, "install-lod"),
		{
			ID: StepImport, Title: "Import existing game folder",
			Applies: func(s *State) bool { return s.Mode == ModeGameDir },
			Ready:   func(s *State) error { return needDir(s.GameDirSrc, "the game folder you supplied") },
			Verify: func(ctx context.Context, s *State) error {
				g, err := game(s)
				if err != nil {
					return err
				}
				return d.gameFilesOK(g)
			},
			Do: func(ctx context.Context, s *State, r *Run) error {
				g, err := game(s)
				if err != nil {
					return err
				}
				if err := d.gameFilesOKIn(s.GameDirSrc); err != nil {
					return fmt.Errorf("source folder: %w", err)
				}
				// U: whether the game starts from a copied folder without the installer's registry keys
				// (InstallPath) was not tested; see docs/setup.md live checks.
				return d.Copy(ctx, s.GameDirSrc, g, r.Logf)
			},
		},
		{
			ID: StepPatch, Title: "Apply the 1.14d patch",
			Interactive: true,
			Ready: func(s *State) error {
				if s.PatchFile == "" {
					return errors.New("patchFile: supply your own LODPatch_114d.exe (this app never downloads Blizzard files)")
				}
				return needFile(s.PatchFile, "the patch file you supplied")
			},
			Verify: func(ctx context.Context, s *State) error {
				g, err := game(s)
				if err != nil {
					return err
				}
				return d.versionOK(g)
			},
			Do: func(ctx context.Context, s *State, r *Run) error {
				_, cfg, err := installer(s)
				if err != nil {
					return err
				}
				sum, err := d.Env.SHA256(s.PatchFile)
				if err != nil {
					return err
				}
				if sum != launch.PatchSHA256 {
					return fmt.Errorf("patch file sha256 %s is not the official LODPatch_114d.exe (%s)", sum, launch.PatchSHA256)
				}
				// The patch lives beside the prefix so the Wine side can reach it by a short path.
				dst := filepath.Join(s.DataDir, "downloads", "LODPatch_114d.exe")
				if err := d.Copy(ctx, s.PatchFile, dst, r.Logf); err != nil {
					return err
				}
				r.WaitingUser("Patch window is open: click through it. It may start the game when done; close that.")
				return d.RunWindowed(ctx, cfg, filepath.Dir(dst), dst, func() error { return d.versionOK(cfg.GameDir) })
			},
		},
		{
			ID: StepExtract, Title: "Extract data tables",
			Verify: func(ctx context.Context, s *State) error {
				g, err := game(s)
				if err != nil {
					return err
				}
				t, err := d.TablesDir(s.DataDir, g)
				if err != nil {
					return err
				}
				if !d.CacheCurrent(g, t) {
					return errors.New("no current data cache")
				}
				return nil
			},
			Do: func(ctx context.Context, s *State, r *Run) error {
				g, err := game(s)
				if err != nil {
					return err
				}
				if err := d.versionOK(g); err != nil {
					return fmt.Errorf("refusing to extract from a game that is not 1.14d: %w", err)
				}
				t, err := d.TablesDir(s.DataDir, g)
				if err != nil {
					return err
				}
				r.Logf("extracting tables to %s", t)
				return d.Extract(g, t)
			},
		},
		{
			ID: StepCharacter, Title: "Create the first throwaway test character",
			Hold: func(s *State) string {
				if s.CharName == "" {
					return "choose a character name and class"
				}
				return ""
			},
			Ready: func(s *State) error {
				o := Options{DataDir: s.DataDir}
				o.Normalize()
				if _, err := d.Env.Stat(o.HelperExe); err != nil {
					return fmt.Errorf("d2input.exe missing at %s (doctor check helper.d2input)", o.HelperExe)
				}
				return validCharName(charName(s))
			},
			Verify: func(ctx context.Context, s *State) error {
				c, err := d.Config(s)
				if err != nil {
					return err
				}
				return needFile(filepath.Join(c.SaveDir(), charName(s)+".d2s"), "character save")
			},
			Do: func(ctx context.Context, s *State, r *Run) error {
				c, err := d.Config(s)
				if err != nil {
					return err
				}
				class := s.CharClass
				if class == "" {
					class = "sorceress"
				}
				if _, ok := launch.ClassClick[class]; !ok {
					return fmt.Errorf("unknown class %q", class)
				}
				r.Logf("creating %s (%s), softcore expansion, normal difficulty", charName(s), class)
				return d.CreateCharacter(ctx, c, charName(s), class)
			},
		},
	}
}

func charName(s *State) string {
	if s.CharName != "" {
		return s.CharName
	}
	return "Setupbot"
}

var nameRe = regexp.MustCompile(`^[A-Za-z]{2,15}$`)

func validCharName(n string) error {
	if !nameRe.MatchString(n) {
		return fmt.Errorf("character name %q must be 2 to 15 letters", n)
	}
	return nil
}

// validateSources checks the user-supplied paths for the chosen mode. It reads nothing big (piece hashes
// are verified in the install steps).
func (d *Deps) validateSources(s *State) error {
	switch s.Mode {
	case ModeStubs:
		for _, p := range []struct{ v, what string }{{s.StubD2, "stubD2 (Diablo II downloader)"}, {s.StubLoD, "stubLoD (Lord of Destruction downloader)"}} {
			if err := needFile(p.v, p.what); err != nil {
				return err
			}
			if _, err := launch.ReadStubMeta(p.v); err != nil {
				return fmt.Errorf("%s is not a Blizzard downloader stub: %w", p.what, err)
			}
		}
	case ModeInstallerDirs:
		for _, p := range []struct{ v, what string }{{s.InstallerD2Dir, "installerD2Dir"}, {s.InstallerLoDDir, "installerLoDDir"}} {
			if err := needFile(filepath.Join(p.v, "Installer.exe"), p.what+" (needs Installer.exe)"); err != nil {
				return err
			}
		}
	case ModeGameDir:
		if err := needDir(s.GameDirSrc, "gameDirSrc"); err != nil {
			return err
		}
		return d.gameFilesOKIn(s.GameDirSrc)
	default:
		return fmt.Errorf("mode must be %s, %s or %s", ModeStubs, ModeInstallerDirs, ModeGameDir)
	}
	return nil
}

func (d *Deps) gameFilesOK(dir string) error { return d.gameFilesOKIn(dir) }

func (d *Deps) gameFilesOKIn(dir string) error {
	ents, err := d.Env.ReadDir(dir)
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, e := range ents {
		have[strings.ToLower(e.Name())] = true
	}
	var miss []string
	for _, f := range requiredGameFiles {
		if !have[strings.ToLower(f)] {
			miss = append(miss, f)
		}
	}
	if len(miss) > 0 {
		return fmt.Errorf("missing %s in %s", strings.Join(miss, ", "), dir)
	}
	return nil
}

func (d *Deps) versionOK(gameDir string) error {
	v, err := d.Env.GameVersion(findFold(d.Env, gameDir, "Game.exe"))
	if err != nil {
		return err
	}
	if v != launch.Want114d.String() {
		return fmt.Errorf("Game.exe is %s, want %s", v, launch.Want114d)
	}
	return nil
}

func needFile(p, what string) error {
	if p == "" {
		return fmt.Errorf("%s: no path given", what)
	}
	if !filepath.IsAbs(p) {
		return fmt.Errorf("%s: %q is not an absolute path", what, p)
	}
	st, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if st.IsDir() {
		return fmt.Errorf("%s: %s is a directory", what, p)
	}
	return nil
}

func needDir(p, what string) error {
	if p == "" {
		return fmt.Errorf("%s: no path given", what)
	}
	if !filepath.IsAbs(p) {
		return fmt.Errorf("%s: %q is not an absolute path", what, p)
	}
	st, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if !st.IsDir() {
		return fmt.Errorf("%s: %s is not a directory", what, p)
	}
	return nil
}

// copyTree copies regular files; symlinks and special files are refused so a hostile folder cannot make
// the copy write or read outside the tree.
func copyTree(ctx context.Context, src, dst string, logf func(string, ...any)) error {
	st, err := os.Stat(src)
	if err != nil {
		return err
	}
	n := 0
	copyFile := func(from, to string, mode fs.FileMode) error {
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		in, err := os.Open(from)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(to+".part", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		return os.Rename(to+".part", to)
	}
	if !st.IsDir() {
		return copyFile(src, dst, st.Mode())
	}
	return filepath.WalkDir(src, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		switch {
		case e.IsDir():
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		case e.Type().IsRegular():
			if n++; n%50 == 0 {
				logf("copied %d files", n)
			}
			return copyFile(p, filepath.Join(dst, rel), 0o644)
		}
		return fmt.Errorf("refusing %s: not a regular file", p)
	})
}

// runWindowed is launch.Installer.runExeUntil for a caller outside the package: start the exe under
// Proton and poll done() every 2 s; the launcher chain exiting is not completion (installers detach).
func runWindowed(ctx context.Context, cfg launch.Config, cwd, exe string, done func() error) error {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Minute)
	defer cancel()
	cmd := cfg.ProtonCmd(context.Background(), cwd, "run", exe)
	if err := cmd.Start(); err != nil {
		return err
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
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
			exited = nil
			if done() == nil {
				return nil
			}
		}
	}
}

// typer adds the WM_CHAR path to the input.OOG adapter (as in cmd/oog).
type typer struct {
	*input.OOG
	c *input.Client
}

func (t typer) TypeChar(r rune) error { return t.c.TypeChar(r) }

// createCharacter drives the real game through the OOG flow. U: composed from verified parts (docs/oog.md,
// docs/launch.md) but never run end to end by this package; the calling process must be the subreaper
// (launch.BecomeSubreaper) so it can read game memory.
func createCharacter(ctx context.Context, cfg launch.Config, name, class string) error {
	sp := launch.NewSpawner()
	h := launch.NewHelper(cfg, sp)
	if err := h.Ensure(ctx, 60*time.Second); err != nil {
		return fmt.Errorf("helper: %w", err)
	}
	g := launch.NewGame(cfg, sp)
	if _, err := g.Launch(ctx, 60*time.Second); err != nil {
		return fmt.Errorf("game: %w", err)
	}
	defer g.Shutdown(5 * time.Second)
	var src *mem.Proc
	var lastErr error
	for i := 0; i < 100 && src == nil; i++ {
		p, err := mem.OpenGame()
		if err == nil {
			src = p
			break
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
	if src == nil {
		return fmt.Errorf("cannot read game memory (is this process the subreaper, doctor check kernel.subreaper?): %w", lastErr)
	}
	defer src.Close()
	cli := input.NewClient(cfg.HelperAddr)
	f := &launch.Flow{Mem: src, In: typer{input.NewOOG(cli), cli}}
	if err := f.CreateCharacter(ctx, name, class); err != nil {
		return err
	}
	return f.LeaveGame(ctx, 30*time.Second)
}

package launch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Sandbox: every Wine process of an instance (Game.exe, the d2input helper, the prefix wineserver and services) runs
// inside a private mount namespace made by bubblewrap in which ONLY an explicit allowlist of paths exists. Everything
// else (/mnt/*, /media, /run/media, gvfs and portal FUSE mounts, other homes, the NAS CIFS shares) is absent. A hung
// network mount then cannot stall Wine at startup (V live 2026-10-29: Game.exe never got a window while /mnt/media
// hung; with the sandbox the window appears). Design, allowlist, failure modes: docs/sandbox.md.
//
// Namespaces: user (needed to make the mount namespace unprivileged) and mount ONLY. Never pid, net or ipc: the bot
// must still see the Wine pids, read /proc/<pid>/mem (V: works across the user namespace boundary, same uid
// mapped to itself), reach the helper on 127.0.0.1 and `wineserver -k` through /tmp/.wine-UID from outside.

// SandboxConfig is the per-instance switch and override; the zero value means "auto".
type SandboxConfig struct {
	// Mode is "" or "auto" (on when bwrap works, except i0), "on" (fail the launch if bwrap is unusable, also i0) or "off".
	Mode string
	// Bwrap overrides the bubblewrap binary (default: bwrap from PATH).
	Bwrap string
	// ExtraRO and ExtraRW add host paths to the allowlist (read-only / read-write), after the environment lists
	// D2_SANDBOX_RO and D2_SANDBOX_RW (colon separated).
	ExtraRO, ExtraRW []string
}

// Entry kinds. They map 1:1 to bwrap options.
const (
	kRO      = "ro-bind"  // read-only bind of Src at Dst
	kRW      = "bind"     // read-write bind
	kDev     = "dev-bind" // bind that keeps device nodes usable
	kTmpfs   = "tmpfs"    // empty tmpfs at Dst
	kDir     = "dir"      // empty directory at Dst (mode 0700 when Perm is set)
	kSymlink = "symlink"  // Dst -> Src (Src is the link target)
)

// Entry is one line of the allowlist, rendered to one bwrap option.
type Entry struct {
	Kind     string
	Src, Dst string
	Perm     string // "0700" for kDir, else ""
}

// Sandbox is a resolved allowlist for one instance. Argv turns it into a bwrap command line.
type Sandbox struct {
	Bwrap   string
	Entries []Entry
	// MkDirs are host directories that must exist before the launch (a bind of a missing source fails).
	MkDirs []string
}

// SandboxHost is what the allowlist expansion asks of the machine; tests fake it.
type SandboxHost struct {
	Home string
	UID  int
	Env  func(string) string
	// Stat reports whether a host path exists; Readlink resolves a symlink ("" if it is none).
	Exists   func(string) bool
	Readlink func(string) string
}

// RealSandboxHost is the running machine.
func RealSandboxHost() SandboxHost {
	home, _ := os.UserHomeDir()
	return SandboxHost{
		Home: home, UID: os.Getuid(), Env: os.Getenv,
		Exists:   func(p string) bool { _, err := os.Stat(p); return err == nil },
		Readlink: func(p string) string { l, _ := os.Readlink(p); return l },
	}
}

// The allowlist. THIS IS THE ONE PLACE to change what a Wine process may see (docs/sandbox.md, "How to add a path").
// Paths derived from the instance (its root, prefix, runner, helper) are added in BuildSandbox.
var (
	// sandboxRO are mounted read-only when they exist.
	sandboxRO = []string{"/usr", "/etc"}
	// sandboxUsrMerge are the classic top-level dirs: recreated as the same symlink the host has (usr-merge), else bound read-only.
	sandboxUsrMerge = []string{"/bin", "/sbin", "/lib", "/lib32", "/lib64"}
	// sandboxDev is bound with device nodes (graphics, /dev/shm, the Wine futex/eventfd paths). It carries no file system mounts of interest.
	sandboxDev = "/dev"
	// sandboxProc is the host /proc: a fresh procfs cannot be mounted without a pid namespace and we must not unshare that.
	sandboxProc = "/proc"
	// sandboxTmpfs are empty tmpfs mounts; TMP below gets two sockets bound back in.
	sandboxTmpfs = []string{"/tmp", "/run"}
	// sandboxTmpBinds are bound read-write into the private /tmp: the X11 socket dir and (via HostMkDirs) the wineserver
	// socket dir, so wineserver -k / the bot reach servers from outside the sandbox.
	sandboxTmpBinds = []string{"/tmp/.X11-unix"}
)

// wineSocketDir is /tmp/.wine-UID, the wineserver socket directory.
func wineSocketDir(uid int) string { return "/tmp/.wine-" + strconv.Itoa(uid) }

// BuildSandbox expands the allowlist for cfg. extra are absolute host paths from the command being run (an installer exe,
// its directory) that are mounted read-only too. Paths that do not exist are skipped, never created (except MkDirs).
func BuildSandbox(cfg Config, h SandboxHost, bwrap string, extra ...string) Sandbox {
	s := Sandbox{Bwrap: bwrap}
	var tmpfs, binds []Entry
	for _, p := range sandboxTmpfs {
		tmpfs = append(tmpfs, Entry{Kind: kTmpfs, Dst: p})
	}
	// The private home: empty tmpfs, instance dirs bound into it below. Skipped when the home is "/" or a prefix of nothing.
	if h.Home != "" && h.Home != "/" {
		tmpfs = append(tmpfs, Entry{Kind: kTmpfs, Dst: h.Home})
	}
	for _, p := range sandboxRO {
		if h.Exists(p) {
			binds = append(binds, Entry{Kind: kRO, Src: p, Dst: p})
		}
	}
	for _, p := range sandboxUsrMerge {
		if l := h.Readlink(p); l != "" {
			binds = append(binds, Entry{Kind: kSymlink, Src: l, Dst: p})
		} else if h.Exists(p) {
			binds = append(binds, Entry{Kind: kRO, Src: p, Dst: p})
		}
	}
	binds = append(binds, Entry{Kind: kDev, Src: sandboxDev, Dst: sandboxDev}, Entry{Kind: kRW, Src: sandboxProc, Dst: sandboxProc})
	runDir := "/run/user/" + strconv.Itoa(h.UID)
	binds = append(binds, Entry{Kind: kDir, Dst: runDir, Perm: "0700"}) // XDG_RUNTIME_DIR exists and is empty
	for _, p := range sandboxTmpBinds {
		if h.Exists(p) {
			binds = append(binds, Entry{Kind: kRW, Src: p, Dst: p})
		}
	}
	ws := wineSocketDir(h.UID)
	s.MkDirs = append(s.MkDirs, ws)
	binds = append(binds, Entry{Kind: kRW, Src: ws, Dst: ws})

	// Instance-derived paths.
	rw := []string{cfg.Root, cfg.PrefixDir, cfg.LogDir, cfg.GameDir}
	rw = append(rw, cfg.Sandbox.ExtraRW...)
	rw = append(rw, splitList(h.Env("D2_SANDBOX_RW"))...)
	ro := []string{}
	if cfg.HelperExe != "" {
		ro = append(ro, filepath.Dir(cfg.HelperExe))
	}
	if cfg.ProtonScript != "" {
		ro = append(ro, filepath.Dir(cfg.ProtonScript)) // the runner dir only, not all of compatibilitytools.d
	}
	ro = append(ro, cfg.Sandbox.ExtraRO...)
	ro = append(ro, splitList(h.Env("D2_SANDBOX_RO"))...)
	ro = append(ro, extra...)
	if x := h.Env("XAUTHORITY"); x != "" && (cfg.Instance == "" || cfg.Instance == "i0") { // the real desktop game only
		ro = append(ro, x)
	}
	if cfg.SteamRoot != "" { // Proton only needs the dir to exist
		binds = append(binds, Entry{Kind: kDir, Dst: cfg.SteamRoot})
	}
	for _, p := range rw {
		binds = appendBind(binds, h, kRW, p)
	}
	for _, p := range ro {
		binds = appendBind(binds, h, kRO, p)
	}
	// Parents before children: tmpfs first, then everything ordered by destination, stable for equal ones.
	sort.SliceStable(binds, func(i, j int) bool { return binds[i].Dst < binds[j].Dst })
	s.Entries = dedupe(append(tmpfs, binds...))
	return s
}

func appendBind(binds []Entry, h SandboxHost, kind, p string) []Entry {
	if p == "" || !filepath.IsAbs(p) || !h.Exists(p) || covered(binds, p) {
		return binds
	}
	return append(binds, Entry{Kind: kind, Src: p, Dst: p})
}

// covered reports whether an existing bind already contains p (a bound directory above it), so the exe of a game
// that lives in the instance root does not get its own mount line.
func covered(binds []Entry, p string) bool {
	for _, b := range binds {
		switch b.Kind {
		case kRO, kRW, kDev:
			if b.Dst == p || strings.HasPrefix(p, strings.TrimSuffix(b.Dst, "/")+"/") {
				return true
			}
		}
	}
	return false
}

// dedupe drops an entry equal to an earlier one and keeps the writable one when a path is listed both ways.
func dedupe(es []Entry) []Entry {
	var out []Entry
	idx := map[string]int{}
	for _, e := range es {
		k := e.Dst
		if e.Kind == kTmpfs || e.Kind == kDir || e.Kind == kSymlink {
			k = e.Kind + ":" + e.Dst
		}
		if i, ok := idx[k]; ok {
			if e.Kind == kRW && out[i].Kind == kRO {
				out[i].Kind = kRW
			}
			continue
		}
		idx[k] = len(out)
		out = append(out, e)
	}
	return out
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ":") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Argv returns the full command line: bwrap, namespaces, allowlist, "--", then name and args. cwd "" keeps the caller's.
func (s Sandbox) Argv(cwd, name string, args ...string) []string {
	a := []string{s.Bwrap, "--unshare-user"}
	for _, e := range s.Entries {
		switch e.Kind {
		case kTmpfs:
			a = append(a, "--tmpfs", e.Dst)
		case kDir:
			if e.Perm != "" {
				a = append(a, "--perms", e.Perm)
			}
			a = append(a, "--dir", e.Dst)
		case kSymlink:
			a = append(a, "--symlink", e.Src, e.Dst)
		default:
			a = append(a, "--"+e.Kind, e.Src, e.Dst)
		}
	}
	if cwd != "" {
		a = append(a, "--chdir", cwd)
	}
	a = append(a, "--", name)
	return append(a, args...)
}

// Prepare creates the host directories the allowlist binds from (the wineserver socket dir).
func (s Sandbox) Prepare() error {
	for _, d := range s.MkDirs {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	return nil
}

// ---- decision: on or off ----

// sandboxProbe checks that bwrap can build a user+mount namespace here (unprivileged userns can be off, or blocked by
// AppArmor). Swapped by tests; the real one caches its answer per bwrap path.
var sandboxProbe = realProbe

var probeCache sync.Map // bwrap path -> error (nil = works)

func realProbe(bwrap string) error {
	if v, ok := probeCache.Load(bwrap); ok {
		err, _ := v.(error)
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// The same allowlist as a real launch (minus the instance paths), so a host layout it cannot express fails here.
	sb := BuildSandbox(Config{}, RealSandboxHost(), bwrap)
	if err := sb.Prepare(); err != nil {
		return err
	}
	argv := sb.Argv("", "/usr/bin/true")
	out, err := exec.CommandContext(ctx, argv[0], argv[1:]...).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("bwrap probe failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	probeCache.Store(bwrap, err)
	return err
}

var lookPath = exec.LookPath

// sandboxChoice is the decision for one Config.
type sandboxChoice struct {
	On     bool
	Bwrap  string
	Why    string // one line for logs, doctor and error messages
	Forced bool   // mode "on": an unusable bwrap is an error, not a fallback
}

// ParseSandboxSwitch maps D2_SANDBOX text: 0/off/false/no -> "off", 1/on/true/yes -> "on", anything else "".
func ParseSandboxSwitch(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "0", "off", "false", "no":
		return "off"
	case "1", "on", "true", "yes":
		return "on"
	}
	return ""
}

// decideSandbox is the on/off rule: D2_SANDBOX beats cfg.Sandbox.Mode; "auto" is on for i1..i9 when bwrap works and off for
// i0 (the user's real desktop game: its X authority and session are not verified inside the sandbox).
func decideSandbox(cfg Config, getenv func(string) string) sandboxChoice {
	mode := ParseSandboxSwitch(getenv("D2_SANDBOX"))
	if mode == "" {
		mode = strings.ToLower(cfg.Sandbox.Mode)
	}
	if mode == "off" {
		return sandboxChoice{Why: "sandbox off (D2_SANDBOX=0 or mode off)"}
	}
	forced := mode == "on"
	if !forced && (cfg.Instance == "" || cfg.Instance == "i0") {
		return sandboxChoice{Why: "sandbox off for i0 (real desktop game; opt in with D2_SANDBOX=1)"}
	}
	bw := cfg.Sandbox.Bwrap
	if bw == "" {
		bw = getenv("D2_BWRAP")
	}
	if bw == "" {
		bw = "bwrap"
	}
	p, err := lookPath(bw)
	if err != nil {
		return sandboxChoice{Forced: forced, Why: "sandbox off: bwrap not found (install bubblewrap)"}
	}
	if err := sandboxProbe(p); err != nil {
		return sandboxChoice{Forced: forced, Why: "sandbox off: " + err.Error()}
	}
	return sandboxChoice{On: true, Bwrap: p, Forced: forced, Why: "sandbox on (" + p + ")"}
}

var warnOnce sync.Map

// resolveSandbox decides and, when on, builds the allowlist. A forced sandbox that cannot work is an error; an unforced
// one falls back to no sandbox with a single logged warning (the old behavior, still guarded by the hung-mount check).
func (c Config) resolveSandbox(extra ...string) (Sandbox, bool, error) {
	ch := decideSandbox(c, os.Getenv)
	if !ch.On {
		if ch.Forced {
			return Sandbox{}, false, fmt.Errorf("D2_SANDBOX=1 but %s", ch.Why)
		}
		if strings.Contains(ch.Why, "bwrap") { // a real fallback worth a warning, not a plain "off"
			if _, seen := warnOnce.LoadOrStore(ch.Why, true); !seen {
				slog.Warn("running Wine WITHOUT the mount sandbox: a hung network mount can stall it", "why", ch.Why)
			}
		}
		return Sandbox{}, false, nil
	}
	return BuildSandbox(c, RealSandboxHost(), ch.Bwrap, extra...), true, nil
}

// SandboxActive reports whether launches of this Config run sandboxed (doctor, the hung-mount guard).
func (c Config) SandboxActive() bool {
	return decideSandbox(c, os.Getenv).On
}

// SandboxStatus is a one-line explanation for doctor output and logs.
func (c Config) SandboxStatus() string { return decideSandbox(c, os.Getenv).Why }

// Wrap returns the command to run for name+args: unchanged when the sandbox is off, else the bwrap command line.
// The host directories the allowlist needs are created. cwd is the working directory of the program ("" = inherit).
func (c Config) Wrap(cwd, name string, args ...string) (string, []string, error) {
	sb, on, err := c.resolveSandbox(argPaths(cwd, args)...)
	if err != nil {
		return "", nil, err
	}
	if !on {
		return name, args, nil
	}
	if err := sb.Prepare(); err != nil {
		return "", nil, err
	}
	argv := sb.Argv(cwd, name, args...)
	return argv[0], argv[1:], nil
}

// argPaths picks the absolute paths among a command's arguments (an installer exe, a patch) and the working directory:
// they are mounted read-only so the sandboxed program can open what it was told to run.
func argPaths(cwd string, args []string) []string {
	var out []string
	if cwd != "" {
		out = append(out, cwd)
	}
	for _, a := range args {
		if filepath.IsAbs(a) {
			out = append(out, a)
		}
	}
	return out
}

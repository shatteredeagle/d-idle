package setup

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"d2lod-bot/internal/data/mpq"
	"d2lod-bot/internal/launch"
)

// Tag legend used in comments: V verified, S secondary source, U unverified.

// Checks is the doctor's rule list, in dependency order.
func Checks() []Check {
	return []Check{
		{ID: "host.container", Title: "Container detection", Run: checkContainer},
		{ID: "host.runner", Title: "Wine / Proton runner", Run: checkRunner},
		{ID: "host.wine32", Title: "32-bit support for plain Wine", Run: checkWine32},
		{ID: "kernel.yama", Title: "Kernel ptrace scope (Yama)", Run: checkYama},
		{ID: "host.sandbox", Title: "Wine mount sandbox (bubblewrap)", Run: checkSandbox},
		{ID: "kernel.subreaper", Title: "Subreaper and memory read feasibility", Slow: true, Needs: []string{"kernel.yama"}, Run: checkSubreaper},
		{ID: "display.xvfb", Title: "Virtual or host display", Run: checkDisplay},
		{ID: "resources.disk", Title: "Free disk space", Run: checkDisk},
		{ID: "resources.cpu", Title: "CPU for planned slots", Run: checkCPU},
		{ID: "resources.ram", Title: "RAM for planned slots", Run: checkRAM},
		{ID: "fs.prefix", Title: "Filesystem for the Wine prefix", Run: checkFS},
		{ID: "net.ports", Title: "Ports free", Run: checkPorts},
		{ID: "helper.d2input", Title: "d2input.exe helper present", Run: checkHelper},
		{ID: "game.files", Title: "Game files present", Run: checkGameFiles},
		{ID: "game.version", Title: "Game.exe is 1.14d", Needs: []string{"game.files"}, Run: checkGameVersion},
		{ID: "game.mpq", Title: "MPQ archives readable", Slow: true, Needs: []string{"game.files"}, Run: checkMPQ},
		{ID: "data.extract", Title: "Data tables can be extracted", Slow: true, Needs: []string{"game.files", "game.mpq"}, Run: checkExtract},
		{ID: "config.parse", Title: "Config files parse", Run: checkConfig},
		{ID: "instance.paths", Title: "Instance paths exist", Run: checkInstancePaths},
		{ID: "data.tables", Title: "Extracted data tables present", Run: checkTables},
		{ID: "helper.reach", Title: "d2input helper reachable", Run: checkHelperReach},
		{ID: "tokens.perms", Title: "Token files are private", Run: checkTokenPerms},
		{ID: "resources.load", Title: "Machine load", Run: checkLoad},
		{ID: "clock.sanity", Title: "System clock", Run: checkClock},
		{ID: "char.difficulty", Title: "Character difficulty and resist penalty", Run: checkCharacter},
	}
}

func res(s Status, detail, fix string) Result { return Result{Status: s, Detail: detail, Fix: fix} }

// missing is the status for something that exists only after installation.
func missing(o *Options) Status {
	if o.GameExpected {
		return Fail
	}
	return Warn
}

func checkContainer(_ context.Context, e *Env, _ *Options) Result {
	kind := ""
	switch {
	case exists(e, "/.dockerenv"):
		kind = "docker"
	case exists(e, "/run/.containerenv"):
		kind = "podman"
	}
	if kind == "" {
		if c := e.Getenv("container"); c != "" {
			kind = c
		}
	}
	if kind == "" {
		if b, err := e.ReadFile("/proc/1/cgroup"); err == nil {
			s := string(b)
			for _, k := range []string{"docker", "lxc", "kubepods", "libpod"} {
				if strings.Contains(s, k) {
					kind = k
					break
				}
			}
		}
	}
	if kind == "" {
		return res(Pass, "not in a container", "")
	}
	return res(Pass, "running in a container ("+kind+"): kernel settings such as Yama are host-wide and cannot be changed from inside", "")
}

func checkRunner(ctx context.Context, e *Env, o *Options) Result {
	if p := e.Getenv("PROTON"); p != "" {
		if _, err := e.Stat(p); err != nil {
			return res(Fail, "PROTON="+p+" does not exist", "Unset PROTON or point it at a proton script.")
		}
		return res(Warn, "PROTON override "+p+" (not the verified runner "+o.RunnerName+")", "")
	}
	rs := launch.FindRunners(existingDirs(e, e.RunnerDirs))
	for _, r := range rs {
		if r.Name == o.RunnerName {
			return res(Pass, "Proton "+r.Name+" found at "+r.Dir+" (the verified runner)", "")
		}
	}
	if r, err := launch.PickRunner(rs, o.RunnerName); err == nil {
		return res(Warn, "Proton "+r.Name+" found; only "+o.RunnerName+" is verified for Diablo II", "Install "+o.RunnerName+" (ProtonUp-Qt or the GE-Proton release) into compatibilitytools.d.")
	}
	if w, err := e.LookPath("wine"); err == nil {
		out, _ := e.Run(ctx, w, "--version")
		return res(Warn, "plain Wine "+strings.TrimSpace(string(out))+" at "+w+"; Diablo II 1.14d on plain Wine is unverified (U)", "Prefer GE-Proton; plain Wine is the container image's runtime.")
	}
	return res(Fail, "no Proton and no wine found", "Install GE-Proton (ProtonUp-Qt) into ~/.local/share/Steam/compatibilitytools.d, or wine (Debian: apt install wine wine32:i386).")
}

// checkSandbox reports whether Wine processes will run in the private mount namespace (docs/sandbox.md). Never a Fail:
// without bwrap everything still works, only a hung network mount can stall Wine again.
func checkSandbox(_ context.Context, e *Env, _ *Options) Result {
	switch launch.ParseSandboxSwitch(e.Getenv("D2_SANDBOX")) {
	case "off":
		return res(Warn, "mount sandbox disabled (D2_SANDBOX=0): a hung network mount (NAS down) stalls Wine at startup", "Unset D2_SANDBOX.")
	}
	if e.GOOS != "linux" {
		return res(Pass, "mount sandbox is Linux only", "")
	}
	bw := e.Getenv("D2_BWRAP")
	if bw == "" {
		bw = "bwrap"
	}
	if _, err := e.LookPath(bw); err != nil {
		return res(Warn, "bubblewrap not found: Wine runs without the mount sandbox, so a hung network mount can stall it", "Install bubblewrap (Arch/Debian/Fedora: package bubblewrap).")
	}
	if b, err := e.ReadFile("/proc/sys/user/max_user_namespaces"); err == nil && strings.TrimSpace(string(b)) == "0" {
		return res(Warn, "user namespaces are disabled (max_user_namespaces=0): the mount sandbox cannot start", "sysctl -w user.max_user_namespaces=65536 (or accept running without the sandbox).")
	}
	if b, err := e.ReadFile("/proc/sys/kernel/unprivileged_userns_clone"); err == nil && strings.TrimSpace(string(b)) == "0" {
		return res(Warn, "unprivileged user namespaces are disabled (kernel.unprivileged_userns_clone=0): the mount sandbox cannot start", "sysctl -w kernel.unprivileged_userns_clone=1.")
	}
	return res(Pass, "bubblewrap present: i1..i9 run Wine in a private mount namespace with an explicit path allowlist (i0 opt-in: D2_SANDBOX=1)", "")
}

func checkWine32(_ context.Context, e *Env, _ *Options) Result {
	if _, err := e.LookPath("wine"); err != nil {
		return res(Pass, "plain Wine not used (Proton ships its own 32-bit libraries)", "")
	}
	for _, p := range []string{"/usr/lib/i386-linux-gnu/libc.so.6", "/lib/i386-linux-gnu/libc.so.6", "/usr/lib32/libc.so.6", "/lib32/libc.so.6"} {
		if exists(e, p) {
			return res(Pass, "32-bit libc present ("+p+")", "")
		}
	}
	// Debian-style wine64-only installs and WoW64 builds may still run 32-bit exes (U), so only warn.
	return res(Warn, "no 32-bit libc found; Diablo II is a 32-bit program", "Debian/Ubuntu: dpkg --add-architecture i386 && apt update && apt install wine32:i386. Fedora: dnf install wine.i686. Arch: enable multilib.")
}

func checkYama(_ context.Context, e *Env, _ *Options) Result {
	b, err := e.ReadFile("/proc/sys/kernel/yama/ptrace_scope")
	if err != nil {
		return res(Pass, "Yama not present (ptrace_scope unrestricted)", "")
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	switch n {
	case 0, 1:
		return res(Pass, fmt.Sprintf("ptrace_scope=%d; the app reads game memory as the subreaper (V docs/launch.md 1)", n), "")
	case 2:
		if hasCap(e, 19) {
			return res(Warn, "ptrace_scope=2 but CAP_SYS_PTRACE is held", "Prefer scope 1: sysctl kernel.yama.ptrace_scope=1")
		}
		return res(Fail, "ptrace_scope=2 (admin-only attach): the app cannot open /proc/<pid>/mem of the game", "Run as root with CAP_SYS_PTRACE, or lower it on the host: sysctl -w kernel.yama.ptrace_scope=1 (persist in /etc/sysctl.d/10-ptrace.conf).")
	default:
		return res(Fail, fmt.Sprintf("ptrace_scope=%d: ptrace is disabled system-wide until reboot", n), "Reboot the host with kernel.yama.ptrace_scope=1 (3 cannot be lowered at runtime).")
	}
}

// hasCap reports whether bit is in CapEff of /proc/self/status.
func hasCap(e *Env, bit uint) bool {
	b, err := e.ReadFile("/proc/self/status")
	if err != nil {
		return false
	}
	for _, l := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(l, "CapEff:"); ok {
			n, err := strconv.ParseUint(strings.TrimSpace(v), 16, 64)
			return err == nil && n&(1<<bit) != 0
		}
	}
	return false
}

func checkSubreaper(ctx context.Context, e *Env, _ *Options) Result {
	if e.Probe == nil {
		return res(Skip, "no probe available", "")
	}
	r, err := e.Probe(ctx)
	if err != nil {
		return res(Fail, "probe could not run: "+err.Error(), "Check that /bin/sh and sleep exist and that the binary can re-execute itself (noexec /tmp, seccomp).")
	}
	switch {
	case r.MemRead:
		return res(Pass, "became subreaper, adopted an orphaned child and read its memory with no capability", "")
	case !r.Subreaper:
		return res(Fail, "prctl(PR_SET_CHILD_SUBREAPER) failed: "+r.Error, "Kernel or seccomp profile blocks prctl; use a profile that allows it (Docker default allows it).")
	case !r.Reparented:
		return res(Fail, "orphans are not re-parented to the app: "+r.Error, "Run the app as PID 1 of the container or under --init, not behind a wrapper that keeps another subreaper.")
	}
	return res(Fail, "orphan adopted but /proc/<pid>/mem is not readable: "+r.Error, "Lower Yama: sysctl -w kernel.yama.ptrace_scope=1; check AppArmor/SELinux policies denying ptrace.")
}

func checkDisplay(_ context.Context, e *Env, _ *Options) Result {
	x, xerr := e.LookPath("Xvfb")
	host := e.Getenv("DISPLAY") != "" || e.Getenv("WAYLAND_DISPLAY") != ""
	switch {
	case xerr == nil:
		return res(Pass, "Xvfb found at "+x, "")
	case host:
		return res(Warn, "no Xvfb; only the desktop display can be used (one visible game, no headless instances)", "Install Xvfb: Debian/Ubuntu apt install xvfb; Fedora dnf install xorg-x11-server-Xvfb; Arch pacman -S xorg-server-xvfb.")
	}
	return res(Fail, "no Xvfb and no DISPLAY/WAYLAND_DISPLAY: the game and the installer need a display", "Install Xvfb (Debian/Ubuntu: apt install xvfb).")
}

// Disk sizes: S (docs/launch.md 7: Tomes 1.5 + 0.5 GB, install about 2 GB, prefix about 1 GB) and V
// (docs/multi.md: about 125 MB unique per extra instance). Real needs vary, so the numbers are U.
func checkDisk(_ context.Context, e *Env, o *Options) Result {
	need := uint64(8<<30) + uint64(o.Slots)*(256<<20)
	p := nearest(e, o.DataDir)
	free, err := e.FreeBytes(p)
	if err != nil {
		return res(Warn, "cannot read free space of "+p+": "+err.Error(), "")
	}
	d := fmt.Sprintf("%.1f GB free at %s, about %.1f GB needed for %d slot(s)", gb(free), p, gb(need), o.Slots)
	switch {
	case free < need:
		return res(Fail, d, "Free disk space or choose a data dir on a larger volume (D2BOT_DATA).")
	case free < need+5<<30:
		return res(Warn, d, "Space is tight; logs, fixtures and save backups grow over time.")
	}
	return res(Pass, d, "")
}

func checkCPU(_ context.Context, e *Env, o *Options) Result {
	n := float64(e.NumCPU)
	src := "host"
	if q, ok := cgroupCPU(e); ok && q < n {
		n, src = q, "cgroup limit"
	}
	d := fmt.Sprintf("%.1f CPUs (%s) for %d slot(s); each game instance busy-loops about one core (V docs/multi.md)", n, src, o.Slots)
	switch {
	case n < float64(o.Slots):
		return res(Fail, d, "Reduce the number of slots or raise the CPU limit.")
	case n < float64(o.Slots)+1:
		return res(Warn, d, "Leave one more core for the bot, Xvfb and the web UI.")
	}
	return res(Pass, d, "")
}

func cgroupCPU(e *Env) (float64, bool) {
	b, err := e.ReadFile("/sys/fs/cgroup/cpu.max")
	if err != nil {
		return 0, false
	}
	f := strings.Fields(string(b))
	if len(f) != 2 || f[0] == "max" {
		return 0, false
	}
	q, err1 := strconv.ParseFloat(f[0], 64)
	p, err2 := strconv.ParseFloat(f[1], 64)
	if err1 != nil || err2 != nil || p == 0 {
		return 0, false
	}
	return q / p, true
}

// RAM: about 265 MB PSS per instance (V docs/multi.md) plus about 1 GB for the app, Wine base and UI (U).
func checkRAM(_ context.Context, e *Env, o *Options) Result {
	need := uint64(1024+300*o.Slots) << 20
	avail, src, ok := memAvailable(e)
	if !ok {
		return res(Warn, "cannot read available memory", "")
	}
	d := fmt.Sprintf("%.1f GB available (%s), about %.1f GB needed for %d slot(s)", gb(avail), src, gb(need), o.Slots)
	switch {
	case avail < need:
		return res(Fail, d, "Add RAM, raise the container memory limit or reduce slots.")
	case avail < need*2:
		return res(Warn, d, "Little headroom; Wine's first run and the installer use more than a running game.")
	}
	return res(Pass, d, "")
}

func memAvailable(e *Env) (uint64, string, bool) {
	var avail uint64
	if b, err := e.ReadFile("/proc/meminfo"); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(l, "MemAvailable:"); ok {
				kb, _ := strconv.ParseUint(strings.Fields(v)[0], 10, 64)
				avail = kb << 10
			}
		}
	}
	src := "host"
	if b, err := e.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		if lim, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64); err == nil && (avail == 0 || lim < avail) {
			avail, src = lim, "cgroup limit"
		}
	}
	return avail, src, avail != 0
}

// Filesystems Wine prefixes and hardlink clones cannot live on (no unix permissions, no hardlinks, no case rules).
var badFS = map[string]bool{"ntfs": true, "ntfs3": true, "fuseblk": true, "exfat": true, "vfat": true, "msdos": true, "fat": true}
var iffyFS = map[string]bool{"nfs": true, "nfs4": true, "cifs": true, "smb3": true, "9p": true, "virtiofs": true, "fuse.sshfs": true}

func checkFS(_ context.Context, e *Env, o *Options) Result {
	p := nearest(e, o.DataDir)
	b, err := e.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return res(Warn, "cannot read mountinfo", "")
	}
	mnt, fstype, opts := mountFor(string(b), p)
	if mnt == "" {
		return res(Warn, "no mount found for "+p, "")
	}
	d := fmt.Sprintf("%s is on %s (%s)", p, mnt, fstype)
	switch {
	case badFS[fstype]:
		return res(Fail, d+": a Wine prefix needs unix permissions and hardlinks", "Choose a data dir on ext4, xfs, btrfs or zfs (D2BOT_DATA). NTFS/exFAT/FAT drives only work for bulky cold data, never for the prefix or game dir.")
	case hasOpt(opts, "noexec"):
		return res(Fail, d+" is mounted noexec: Wine cannot run the game", "Remount without noexec or pick another data dir.")
	case iffyFS[fstype]:
		return res(Warn, d+": network filesystems are slow and unreliable for Wine (locks, hardlinks) (S)", "Prefer a local disk for the prefix; keep only cold data on the share.")
	}
	return res(Pass, d, "")
}

// mountFor finds the longest mount point containing p in /proc/self/mountinfo text.
func mountFor(info, p string) (mnt, fstype string, opts []string) {
	for _, l := range strings.Split(info, "\n") {
		pre, post, ok := strings.Cut(l, " - ")
		if !ok {
			continue
		}
		f := strings.Fields(pre)
		g := strings.Fields(post)
		if len(f) < 6 || len(g) < 1 {
			continue
		}
		m := unescapeMount(f[4])
		if (p == m || m == "/" || strings.HasPrefix(p, m+"/")) && len(m) >= len(mnt) {
			mnt, fstype, opts = m, g[0], append(strings.Split(f[5], ","), mountSuper(g)...)
		}
	}
	return
}

func mountSuper(g []string) []string {
	if len(g) >= 3 {
		return strings.Split(g[2], ",")
	}
	return nil
}

func unescapeMount(s string) string {
	return strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`).Replace(s)
}

func hasOpt(opts []string, o string) bool {
	for _, x := range opts {
		if x == o {
			return true
		}
	}
	return false
}

func checkPorts(_ context.Context, e *Env, o *Options) Result {
	type pp struct{ addr, what string }
	ports := []pp{{o.UIAddr, "web UI"}}
	for i := 0; i < o.Slots; i++ {
		ports = append(ports, pp{"127.0.0.1:" + strconv.Itoa(47012+i), fmt.Sprintf("d2input helper %d", i)})
	}
	if o.Slots > 1 {
		ports = append(ports, pp{"127.0.0.1:4000", "host game for party play"})
	}
	var busy []string
	for _, p := range ports {
		if err := e.Listen(p.addr); err != nil {
			busy = append(busy, p.addr+" ("+p.what+")")
		}
	}
	if len(busy) > 0 {
		return res(Warn, "in use: "+strings.Join(busy, ", "), "Stop what uses them (an already running d2bot is fine), or change the UI address (-addr) / free the helper ports (ss -ltnp).")
	}
	return res(Pass, fmt.Sprintf("%d port(s) free", len(ports)), "")
}

func checkHelper(_ context.Context, e *Env, o *Options) Result {
	st, err := e.Stat(o.HelperExe)
	if err != nil || st.IsDir() {
		return res(missing(o), "d2input.exe not found at "+o.HelperExe, "Build it: GOOS=windows GOARCH=386 go build -o "+o.HelperExe+" ./cmd/d2input (it is part of this project, not Blizzard's).")
	}
	return res(Pass, o.HelperExe, "")
}

// Files whose presence proves a base + LoD install (names VERIFIED on a real install, docs/launch.md 4).
var requiredGameFiles = []string{"Game.exe", "Diablo II.exe", "d2data.mpq", "d2char.mpq", "d2sfx.mpq", "d2exp.mpq", "d2xmusic.mpq", "d2xtalk.mpq", "d2xvideo.mpq"}

func checkGameFiles(_ context.Context, e *Env, o *Options) Result {
	ents, err := e.ReadDir(o.GameDir)
	if err != nil {
		return res(missing(o), "game dir "+o.GameDir+" not readable: "+err.Error(), "Run the setup wizard and provide your own Diablo II and Lord of Destruction installers or an existing install folder. Nothing is downloaded for you.")
	}
	have := map[string]bool{}
	for _, d := range ents {
		have[strings.ToLower(d.Name())] = true
	}
	var miss []string
	for _, f := range requiredGameFiles {
		if !have[strings.ToLower(f)] {
			miss = append(miss, f)
		}
	}
	if len(miss) > 0 {
		return res(missing(o), "missing in "+o.GameDir+": "+strings.Join(miss, ", "), "Finish the install in the wizard; the Lord of Destruction expansion is required (d2exp.mpq and the d2x* archives).")
	}
	if !have["patch_d2.mpq"] {
		return res(missing(o), "Patch_D2.mpq missing: the 1.14d patch is not applied", "Apply the 1.14d patch you downloaded yourself (wizard step patch).")
	}
	return res(Pass, "all required files in "+o.GameDir, "")
}

func checkGameVersion(_ context.Context, e *Env, o *Options) Result {
	exe := findFold(e, o.GameDir, "Game.exe")
	v, err := e.GameVersion(exe)
	if err != nil {
		return res(Fail, "cannot read the Game.exe version: "+err.Error(), "Re-copy Game.exe from your install.")
	}
	sum, err := e.SHA256(exe)
	if err != nil {
		return res(Fail, "cannot hash Game.exe: "+err.Error(), "")
	}
	if label, ok := o.KnownGameExe[sum]; ok {
		return res(Pass, "Game.exe "+v+", sha256 matches the verified build "+label, "")
	}
	if v == launch.Want114d.String() {
		return res(Warn, "Game.exe version is "+v+" but sha256 "+sum+" is not a verified build (repacked, cracked or a no-CD exe?); memory offsets were verified for the known build only", "Use the unmodified Game.exe produced by the official 1.14d patch.")
	}
	return res(Fail, "Game.exe is "+v+", want "+launch.Want114d.String()+" (1.14d); every memory offset is specific to that build", "Apply the official 1.14d patch (LODPatch_114d.exe, your own copy) in the wizard step patch.")
}

// Needed to prove the tables are reachable: present in d2data.mpq on a real install (data package reads these).
func checkMPQ(_ context.Context, e *Env, o *Options) Result {
	var bad []string
	n := 0
	for _, name := range []string{"d2data.mpq", "d2exp.mpq", "patch_d2.mpq"} {
		p := findFold(e, o.GameDir, name)
		if _, err := e.Stat(p); err != nil {
			if name == "patch_d2.mpq" {
				continue
			}
			bad = append(bad, name+": missing")
			continue
		}
		a, err := mpq.Open(p)
		if err != nil {
			bad = append(bad, name+": "+err.Error())
			continue
		}
		n++
		if len(a.ListFile()) == 0 && !a.Has("(listfile)") && !a.Has("data\\global\\excel\\armor.txt") {
			bad = append(bad, name+": no files readable")
		}
		a.Close()
	}
	if len(bad) > 0 {
		return res(Fail, strings.Join(bad, "; "), "The archive is truncated or corrupt: copy it again from your install media or re-run the installer step (do not use a re-downloaded or modified MPQ).")
	}
	return res(Pass, fmt.Sprintf("%d archive(s) opened and indexed", n), "")
}

func checkExtract(_ context.Context, e *Env, o *Options) Result {
	if err := e.TryExtract(o.GameDir); err != nil {
		return res(Fail, "table extraction failed: "+err.Error(), "Check game.mpq results first; the install must be 1.14d with Patch_D2.mpq.")
	}
	return res(Pass, "all data tables were read from the MPQs", "")
}

// --- helpers

func exists(e *Env, p string) bool { _, err := e.Stat(p); return err == nil }

func existingDirs(e *Env, dirs []string) []string {
	var out []string
	for _, d := range dirs {
		if st, err := e.Stat(d); err == nil && st.IsDir() {
			out = append(out, d)
		}
	}
	return out
}

// nearest returns p or its closest existing ancestor.
func nearest(e *Env, p string) string {
	p = filepath.Clean(p)
	for p != "/" && p != "." {
		if _, err := e.Stat(p); err == nil {
			return p
		}
		p = filepath.Dir(p)
	}
	return p
}

// findFold returns dir/name matched case-insensitively (the install is a Windows tree), else dir/name.
func findFold(e *Env, dir, name string) string {
	ents, _ := e.ReadDir(dir)
	for _, d := range ents {
		if strings.EqualFold(d.Name(), name) {
			return filepath.Join(dir, d.Name())
		}
	}
	return filepath.Join(dir, name)
}

func gb(n uint64) float64 { return float64(n) / (1 << 30) }

// GameFiles lists the file names a Diablo II folder must contain (plus Patch_D2.mpq once patched); for fakes.
func GameFiles() []string { return append([]string{"Patch_D2.mpq"}, requiredGameFiles...) }

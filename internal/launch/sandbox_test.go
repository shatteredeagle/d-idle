package launch

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func fakeHost(exist ...string) SandboxHost {
	set := map[string]bool{}
	for _, p := range exist {
		set[p] = true
	}
	links := map[string]string{"/bin": "usr/bin", "/sbin": "usr/bin", "/lib": "usr/lib", "/lib64": "usr/lib"}
	return SandboxHost{
		Home: "/h/u", UID: 1000,
		Env:      func(k string) string { return map[string]string{}[k] },
		Exists:   func(p string) bool { return set[p] },
		Readlink: func(p string) string { return links[p] },
	}
}

func testCfg() Config {
	return Config{
		Instance: "i4", Root: "/h/u/Games/d2lod-inst/i4", PrefixDir: "/h/u/Games/d2lod-inst/i4/proton",
		LogDir: "/h/u/Games/d2lod-inst/i4/logs", GameDir: "/h/u/Games/d2lod-inst/i4/proton/pfx/drive_c/Program Files (x86)/Diablo II",
		SteamRoot: "/h/u/.local/share/Steam", HelperExe: "/h/u/Games/d2lod/helper/d2input.exe",
		ProtonScript: "/h/u/.local/share/Steam/compatibilitytools.d/GE-Proton11-11/proton",
	}
}

var testExist = []string{"/usr", "/etc", "/tmp/.X11-unix",
	"/h/u/Games/d2lod-inst/i4", "/h/u/Games/d2lod-inst/i4/proton", "/h/u/Games/d2lod-inst/i4/logs",
	"/h/u/Games/d2lod/helper", "/h/u/.local/share/Steam/compatibilitytools.d/GE-Proton11-11",
	"/tmp/.wine-1000", "/h/u/Games/d2lod/helper/d2input.exe"}

// The golden command line: if this changes, docs/sandbox.md (the allowlist) changes too.
func TestSandboxGoldenArgv(t *testing.T) {
	sb := BuildSandbox(testCfg(), fakeHost(testExist...), "/usr/bin/bwrap")
	got := strings.Join(sb.Argv("/h/u/Games/d2lod-inst/i4/proton/pfx/drive_c/Program Files (x86)/Diablo II", "/p/proton", "run", "/x/Diablo II.exe", "-w", "-ns"), "\n")
	want := strings.Join([]string{
		"/usr/bin/bwrap", "--unshare-user",
		"--tmpfs", "/tmp", "--tmpfs", "/run", "--tmpfs", "/h/u",
		"--symlink", "usr/bin", "/bin",
		"--dev-bind", "/dev", "/dev",
		"--ro-bind", "/etc", "/etc",
		"--dir", "/h/u/.local/share/Steam",
		"--ro-bind", "/h/u/.local/share/Steam/compatibilitytools.d/GE-Proton11-11", "/h/u/.local/share/Steam/compatibilitytools.d/GE-Proton11-11",
		"--bind", "/h/u/Games/d2lod-inst/i4", "/h/u/Games/d2lod-inst/i4",
		"--ro-bind", "/h/u/Games/d2lod/helper", "/h/u/Games/d2lod/helper",
		"--symlink", "usr/lib", "/lib",
		"--symlink", "usr/lib", "/lib64",
		"--bind", "/proc", "/proc",
		"--perms", "0700", "--dir", "/run/user/1000",
		"--symlink", "usr/bin", "/sbin",
		"--bind", "/tmp/.X11-unix", "/tmp/.X11-unix",
		"--bind", "/tmp/.wine-1000", "/tmp/.wine-1000",
		"--ro-bind", "/usr", "/usr",
		"--chdir", "/h/u/Games/d2lod-inst/i4/proton/pfx/drive_c/Program Files (x86)/Diablo II",
		"--", "/p/proton", "run", "/x/Diablo II.exe", "-w", "-ns",
	}, "\n")
	if got != want {
		t.Fatalf("argv\n%s\n--- want\n%s", got, want)
	}
	if !reflect.DeepEqual(sb.MkDirs, []string{"/tmp/.wine-1000"}) {
		t.Fatal(sb.MkDirs)
	}
}

// Nothing of the network shares may appear in any form.
func TestSandboxHidesMounts(t *testing.T) {
	exist := append([]string{"/mnt", "/mnt/media", "/media", "/run/media", "/run/user/1000/gvfs"}, testExist...)
	sb := BuildSandbox(testCfg(), fakeHost(exist...), "bwrap")
	for _, a := range sb.Argv("", "x") {
		for _, bad := range []string{"/mnt", "/media", "/run/media", "gvfs", "/run/user/1000/doc"} {
			if strings.Contains(a, bad) {
				t.Fatalf("%q leaked into the allowlist: %v", bad, sb.Argv("", "x"))
			}
		}
	}
}

func TestSandboxExtraPaths(t *testing.T) {
	cfg := testCfg()
	cfg.Sandbox = SandboxConfig{ExtraRO: []string{"/opt/ro"}, ExtraRW: []string{"/srv/rw", "/nope"}}
	h := fakeHost(append([]string{"/opt/ro", "/srv/rw", "/env/ro", "/h/u/Downloads/stub.exe"}, testExist...)...)
	base := h.Env
	h.Env = func(k string) string {
		if k == "D2_SANDBOX_RO" {
			return "/env/ro:/gone"
		}
		return base(k)
	}
	sb := BuildSandbox(cfg, h, "bwrap", "/h/u/Downloads/stub.exe", cfg.GameDir+"/Game.exe")
	argv := strings.Join(sb.Argv("", "x"), " ")
	for _, w := range []string{"--ro-bind /opt/ro /opt/ro", "--bind /srv/rw /srv/rw", "--ro-bind /env/ro /env/ro", "--ro-bind /h/u/Downloads/stub.exe /h/u/Downloads/stub.exe"} {
		if !strings.Contains(argv, w) {
			t.Errorf("missing %q in %s", w, argv)
		}
	}
	for _, bad := range []string{"/nope", "/gone", "Game.exe"} { // missing paths skipped; a file under the bound root is covered
		if strings.Contains(argv, bad) {
			t.Errorf("unexpected %q in %s", bad, argv)
		}
	}
}

func TestSandboxDedupeKeepsWritable(t *testing.T) {
	cfg := testCfg()
	cfg.Sandbox.ExtraRO = []string{cfg.LogDir}
	sb := BuildSandbox(cfg, fakeHost(testExist...), "bwrap")
	n := 0
	for _, e := range sb.Entries {
		if e.Dst == cfg.LogDir {
			n++
		}
	}
	if n != 0 { // logs sit under the bound instance root: no own line at all
		t.Fatalf("log dir listed %d times", n)
	}
}

func TestDecideSandbox(t *testing.T) {
	oldLook, oldProbe := lookPath, sandboxProbe
	defer func() { lookPath, sandboxProbe = oldLook, oldProbe }()
	lookPath = func(s string) (string, error) { return "/usr/bin/" + s, nil }
	sandboxProbe = func(string) error { return nil }
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	i4, i0 := Config{Instance: "i4"}, Config{Instance: "i0"}

	if c := decideSandbox(i4, env(nil)); !c.On || c.Bwrap != "/usr/bin/bwrap" {
		t.Fatalf("default on: %+v", c)
	}
	if c := decideSandbox(i0, env(nil)); c.On {
		t.Fatalf("i0 default must stay off: %+v", c)
	}
	if c := decideSandbox(Config{}, env(nil)); c.On {
		t.Fatalf("no instance = i0: %+v", c)
	}
	if c := decideSandbox(i0, env(map[string]string{"D2_SANDBOX": "1"})); !c.On || !c.Forced {
		t.Fatalf("i0 opt in: %+v", c)
	}
	for _, off := range []string{"0", "off", "false", "No"} {
		if c := decideSandbox(i4, env(map[string]string{"D2_SANDBOX": off})); c.On {
			t.Fatalf("D2_SANDBOX=%s must disable: %+v", off, c)
		}
	}
	if c := decideSandbox(Config{Instance: "i4", Sandbox: SandboxConfig{Mode: "off"}}, env(nil)); c.On {
		t.Fatal("config off")
	}
	if c := decideSandbox(Config{Instance: "i4", Sandbox: SandboxConfig{Mode: "off"}}, env(map[string]string{"D2_SANDBOX": "1"})); !c.On {
		t.Fatal("env beats config")
	}
	lookPath = func(string) (string, error) { return "", errors.New("nf") }
	if c := decideSandbox(i4, env(nil)); c.On || c.Forced || !strings.Contains(c.Why, "bwrap not found") {
		t.Fatalf("no bwrap, auto = fall back: %+v", c)
	}
	if c := decideSandbox(i4, env(map[string]string{"D2_SANDBOX": "1"})); c.On || !c.Forced {
		t.Fatalf("no bwrap, forced: %+v", c)
	}
	lookPath = func(s string) (string, error) { return s, nil }
	sandboxProbe = func(string) error { return errors.New("userns denied") }
	if c := decideSandbox(i4, env(nil)); c.On || !strings.Contains(c.Why, "userns denied") {
		t.Fatalf("probe fail: %+v", c)
	}
}

func TestWrapOffAndForcedError(t *testing.T) {
	t.Setenv("D2_SANDBOX", "0")
	n, a, err := testCfg().Wrap("/c", "/p/proton", "run", "x.exe")
	if err != nil || n != "/p/proton" || !reflect.DeepEqual(a, []string{"run", "x.exe"}) {
		t.Fatalf("%s %v %v", n, a, err)
	}
	oldLook := lookPath
	defer func() { lookPath = oldLook }()
	lookPath = func(string) (string, error) { return "", errors.New("nf") }
	t.Setenv("D2_SANDBOX", "1")
	if _, _, err := testCfg().Wrap("", "/p/proton"); err == nil {
		t.Fatal("forced sandbox without bwrap must fail the launch")
	}
	cmd := testCfg().ProtonCmd(t.Context(), "", "run", "x.exe")
	if cmd.Err == nil {
		t.Fatal("ProtonCmd must carry the error so Start fails")
	}
}

func TestProtonCmdWrapsWhenOn(t *testing.T) {
	oldLook, oldProbe := lookPath, sandboxProbe
	defer func() { lookPath, sandboxProbe = oldLook, oldProbe }()
	lookPath = func(string) (string, error) { return "/usr/bin/bwrap", nil }
	sandboxProbe = func(string) error { return nil }
	t.Setenv("D2_SANDBOX", "")
	cfg := testCfg()
	cfg.PrefixDir, cfg.LogDir = t.TempDir(), t.TempDir() // ProtonCmd creates them
	cmd := cfg.ProtonCmd(t.Context(), "", "run", "x.exe")
	if cmd.Path != "/usr/bin/bwrap" || cmd.Args[1] != "--unshare-user" || cmd.Args[len(cmd.Args)-1] != "x.exe" {
		t.Fatalf("%v", cmd.Args)
	}
}

// The lifecycle (d2game, tools/game.sh) spawns the game and the helper through the sandbox, wineserver -k does not.
func TestLifeSpawnsAreWrapped(t *testing.T) {
	oldLook, oldProbe := lookPath, sandboxProbe
	defer func() { lookPath, sandboxProbe = oldLook, oldProbe }()
	lookPath = func(string) (string, error) { return "/usr/bin/bwrap", nil }
	sandboxProbe = func(string) error { return nil }
	t.Setenv("D2_SANDBOX", "")
	l, f, _ := testLife(t)
	f.exists["/tmp/.X11-unix/X104"] = true
	f.spawnGame = false // the fake only knows unwrapped argv; here we inspect the argv instead
	_ = l.Start(t.Context(), "i4")
	_ = l.EnsureHelper(t.Context(), "i4")
	var game, helper []string
	for _, s := range f.spawns {
		if len(s.argv) > 0 && s.argv[0] == "/usr/bin/bwrap" {
			if strings.Contains(strings.Join(s.argv, " "), "Diablo II.exe") {
				game = s.argv
			}
			if strings.Contains(strings.Join(s.argv, " "), "d2input.exe") {
				helper = s.argv
			}
		}
	}
	for name, a := range map[string][]string{"game": game, "helper": helper} {
		if a == nil || a[1] != "--unshare-user" {
			t.Fatalf("%s spawn not sandboxed: %v", name, f.spawns)
		}
	}
	for _, r := range f.runs {
		if strings.Contains(r, "bwrap") {
			t.Fatalf("wineserver -k must run on the host: %s", r)
		}
	}
}

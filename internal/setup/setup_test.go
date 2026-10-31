package setup

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/launch"
)

// fakeEnv: files is path -> content; dirs lists directories; listing derives from files.
func fakeEnv(files map[string]string) *Env {
	e := &Env{
		GOOS: "linux", NumCPU: 8,
		Getenv: func(string) string { return "" },
		ReadFile: func(p string) ([]byte, error) {
			if s, ok := files[p]; ok {
				return []byte(s), nil
			}
			return nil, fs.ErrNotExist
		},
		Stat: func(p string) (fs.FileInfo, error) {
			if _, ok := files[p]; ok {
				return nil, nil
			}
			return nil, fs.ErrNotExist
		},
		ReadDir:   func(string) ([]fs.DirEntry, error) { return nil, fs.ErrNotExist },
		LookPath:  func(string) (string, error) { return "", errors.New("no") },
		FreeBytes: func(string) (uint64, error) { return 100 << 30, nil },
		Listen:    func(string) error { return nil },
	}
	return e
}

func one(t *testing.T, id string, e *Env, o Options) Result {
	t.Helper()
	o.DataDir = "/data"
	rs := RunChecks(context.Background(), e, o, Checks())
	for _, r := range rs {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no check %s", id)
	return Result{}
}

func TestYama(t *testing.T) {
	for scope, want := range map[string]Status{"0": Pass, "1": Pass, "2": Fail, "3": Fail} {
		e := fakeEnv(map[string]string{"/proc/sys/kernel/yama/ptrace_scope": scope + "\n"})
		if r := one(t, "kernel.yama", e, Options{}); r.Status != want || (want == Fail && r.Fix == "") {
			t.Errorf("scope %s: %+v", scope, r)
		}
	}
	if r := one(t, "kernel.yama", fakeEnv(nil), Options{}); r.Status != Pass {
		t.Error("no Yama file should pass")
	}
	e := fakeEnv(map[string]string{"/proc/sys/kernel/yama/ptrace_scope": "2", "/proc/self/status": "CapEff:\t0000000000080000\n"})
	if r := one(t, "kernel.yama", e, Options{}); r.Status != Warn {
		t.Errorf("cap held: %+v", r)
	}
}

func TestFSAndDisk(t *testing.T) {
	info := "30 1 8:1 / / rw - ext4 /dev/sda1 rw\n40 30 8:2 / /data rw,noexec - ext4 /dev/sdb rw\n"
	if r := one(t, "fs.prefix", fakeEnv(map[string]string{"/proc/self/mountinfo": info, "/data": ""}), Options{}); r.Status != Fail || !strings.Contains(r.Detail, "noexec") {
		t.Errorf("%+v", r)
	}
	info = "30 1 8:1 / / rw - ntfs3 /dev/sda1 rw\n"
	if r := one(t, "fs.prefix", fakeEnv(map[string]string{"/proc/self/mountinfo": info, "/data": ""}), Options{}); r.Status != Fail {
		t.Errorf("ntfs: %+v", r)
	}
	e := fakeEnv(map[string]string{"/data": ""})
	e.FreeBytes = func(string) (uint64, error) { return 3 << 30, nil }
	if r := one(t, "resources.disk", e, Options{}); r.Status != Fail {
		t.Errorf("%+v", r)
	}
	e.FreeBytes = func(string) (uint64, error) { return 10 << 30, nil }
	if r := one(t, "resources.disk", e, Options{}); r.Status != Warn {
		t.Errorf("%+v", r)
	}
}

func TestCPUPortsContainer(t *testing.T) {
	e := fakeEnv(map[string]string{"/sys/fs/cgroup/cpu.max": "200000 100000\n", "/.dockerenv": ""})
	if r := one(t, "resources.cpu", e, Options{Slots: 3}); r.Status != Fail {
		t.Errorf("%+v", r)
	}
	if r := one(t, "host.container", e, Options{}); !strings.Contains(r.Detail, "docker") {
		t.Errorf("%+v", r)
	}
	e.Listen = func(a string) error {
		if strings.HasSuffix(a, ":47013") {
			return errors.New("in use")
		}
		return nil
	}
	if r := one(t, "net.ports", e, Options{Slots: 2}); r.Status != Warn || !strings.Contains(r.Detail, "47013") {
		t.Errorf("%+v", r)
	}
}

func TestSkipAndExit(t *testing.T) {
	e := fakeEnv(nil)
	rs := RunChecks(context.Background(), e, Options{DataDir: "/data", GameExpected: true, Quick: true}, Checks())
	by := map[string]Result{}
	for _, r := range rs {
		by[r.ID] = r
	}
	if by["game.files"].Status != Fail || by["game.version"].Status != Skip || by["kernel.subreaper"].Status != Skip {
		t.Errorf("%+v", by)
	}
	if ExitCode(rs) != 2 || NewReport(rs).Counts.Fail == 0 {
		t.Error("exit")
	}
	rs = RunChecks(context.Background(), e, Options{DataDir: "/data", Quick: true}, Checks())
	if by := (map[string]Result{})[""]; by.Status != "" || rs[0].ID == "" {
		t.Error()
	}
	for _, id := range []string{"host.runner", "kernel.yama", "kernel.subreaper", "display.xvfb", "resources.disk", "fs.prefix", "net.ports", "helper.d2input", "game.files", "game.version", "game.mpq", "data.extract", "resources.cpu", "resources.ram", "host.container", "host.wine32"} {
		found := false
		for _, x := range IDs(Checks()) {
			found = found || x == id
		}
		if !found {
			t.Errorf("missing id %s", id)
		}
	}
}

func TestGameVersion(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Game.exe"), []byte("x"), 0o644)
	e := NewEnv()
	ver := "1.14.3.71"
	e.GameVersion = func(string) (string, error) { return ver, nil }
	e.SHA256 = func(string) (string, error) { return "abc", nil }
	o := &Options{GameDir: dir, KnownGameExe: map[string]string{"known": "x"}}
	if r := checkGameVersion(context.Background(), e, o); r.Status != Warn {
		t.Errorf("unknown hash right version: %+v", r)
	}
	e.SHA256 = func(string) (string, error) { return "known", nil }
	if r := checkGameVersion(context.Background(), e, o); r.Status != Pass {
		t.Errorf("%+v", r)
	}
	ver = "1.13.0.0"
	e.SHA256 = func(string) (string, error) { return "abc", nil }
	if r := checkGameVersion(context.Background(), e, o); r.Status != Fail {
		t.Errorf("%+v", r)
	}
}

func TestRealProbe(t *testing.T) {
	r, err := RunProbeChild(context.Background())
	if err != nil {
		t.Skip("cannot re-exec test binary:", err)
	}
	_ = r // result depends on host Yama; only the plumbing is asserted
}

func TestMain(m *testing.M) {
	MaybeProbe()
	os.Exit(m.Run())
}

// --- wizard

func fakeSteps(log *[]string, failOn string) []StepDef {
	done := map[string]bool{}
	mk := func(id string) StepDef {
		return StepDef{
			ID: id, Title: id,
			Verify: func(context.Context, *State) error {
				if done[id] {
					return nil
				}
				return errors.New("not done")
			},
			Do: func(_ context.Context, _ *State, r *Run) error {
				*log = append(*log, id)
				if id == failOn {
					return errors.New("boom 1234-ABCDEF-5678")
				}
				done[id] = true
				return nil
			},
		}
	}
	skip := mk("skipme")
	skip.Applies = func(s *State) bool { return s.Mode == "x" }
	return []StepDef{mk("a"), skip, mk("b"), mk("c")}
}

func TestWizardResume(t *testing.T) {
	path := filepath.Join(t.TempDir(), "setup.json")
	var log []string
	steps := fakeSteps(&log, "b")
	w, err := New(path, steps)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Run(context.Background()); err == nil || strings.Contains(err.Error(), "ABCDEF") {
		t.Fatalf("want failure with redacted key, got %v", err)
	}
	v := w.View()
	if v.Steps[0].Status != StepDone || v.Steps[1].Status != StepSkipped || v.Steps[2].Status != StepFailed || v.Current != "b" {
		t.Fatalf("%+v", v.Steps)
	}
	// "crash": a new wizard from disk; step b was fixed
	log = nil
	steps2 := fakeSteps(&log, "")
	w2, _ := New(path, steps2)
	if w2.View().Steps[0].Status != StepDone {
		t.Fatal("progress not persisted")
	}
	if err := w2.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !w2.View().Done {
		t.Fatal("not done")
	}
	// a is not re-run: its fake Verify state is fresh, so it is re-verified and re-done; b and c ran
	if len(log) < 2 {
		t.Fatalf("%v", log)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "ABCDEF") {
		t.Error("key-shaped text persisted")
	}
}

func TestWizardInterruptedAndCancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.json")
	block := make(chan struct{})
	steps := []StepDef{{ID: "w", Title: "w", Interactive: true,
		Do: func(ctx context.Context, _ *State, r *Run) error {
			r.WaitingUser("click it")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-block:
				return nil
			}
		}}}
	w, _ := New(path, steps)
	if err := w.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100 && w.View().Steps[0].Status != StepWaitingUser; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if w.View().Steps[0].Status != StepWaitingUser {
		t.Fatal("never waiting")
	}
	if err := w.SetInputs(Inputs{Slots: 2}); !errors.Is(err, ErrBusy) {
		t.Error("inputs allowed while running")
	}
	w.Cancel()
	w.Wait()
	if s := w.View().Steps[0].Status; s != StepPending {
		t.Errorf("after cancel %s", s)
	}
	// simulate a crash mid-run
	w.st.Steps["w"] = StepState{Status: StepWaitingUser}
	w.save()
	w2, _ := New(path, steps)
	if s := w2.View().Steps[0].Status; s != StepPending {
		t.Errorf("after crash %s", s)
	}
}

func TestRedact(t *testing.T) {
	for _, in := range []string{"key ABCD-123456-WXYZ ok", "AAAABBBBCCCCDDDD", "1234-5678-9012-3456"} {
		if got := Redact(in); strings.Contains(got, "ABCD-") || strings.Contains(got, "AAAA") || strings.Contains(got, "1234") {
			t.Errorf("%q -> %q", in, got)
		}
	}
	if Redact("GE-Proton11-11 d2data.mpq") != "GE-Proton11-11 d2data.mpq" {
		t.Error("over-redacted")
	}
}

// --- default steps with fakes

func installTree(t *testing.T, dir string) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	for _, f := range append([]string{"Patch_D2.mpq"}, requiredGameFiles...) {
		os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644)
	}
}

func TestDefaultStepsGameDirMode(t *testing.T) {
	t.Run("with patch file", func(t *testing.T) { gameDirMode(t, true) })
	// An install that is already 1.14.3.71 needs no patch file (container: no /patch mount).
	t.Run("already patched, no patch file", func(t *testing.T) { gameDirMode(t, false) })
}

func gameDirMode(t *testing.T, withPatch bool) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	src := filepath.Join(dir, "myd2")
	installTree(t, src)
	patch := filepath.Join(dir, "patch.exe")
	os.WriteFile(patch, []byte("p"), 0o644)

	env := NewEnv()
	ver := "1.14.3.71"
	env.GameVersion = func(string) (string, error) { return ver, nil }
	env.SHA256 = func(string) (string, error) { return launch.PatchSHA256, nil }
	d := NewDeps(env)
	game := GameDirIn(BasePrefix(data))
	d.Config = func(s *State) (launch.Config, error) {
		return launch.Config{Root: s.DataDir, PrefixDir: BasePrefix(s.DataDir), GameDir: game}, nil
	}
	var extracted, created bool
	d.Prefix = func(ctx context.Context, c launch.Config) error {
		return os.MkdirAll(filepath.Join(c.PrefixDir, "pfx"), 0o755)
	}
	d.TablesDir = func(dd, g string) (string, error) { return filepath.Join(dd, "cache", "t"), nil }
	d.CacheCurrent = func(string, string) bool { return extracted }
	d.Extract = func(string, string) error { extracted = true; return nil }
	d.CreateCharacter = func(ctx context.Context, c launch.Config, n, cl string) error {
		created = true
		os.MkdirAll(c.SaveDir(), 0o755)
		return os.WriteFile(filepath.Join(c.SaveDir(), n+".d2s"), []byte("x"), 0o644)
	}
	d.RunWindowed = func(context.Context, launch.Config, string, string, func() error) error {
		t.Error("no window expected in game-dir mode at 1.14d")
		return nil
	}
	helper := filepath.Join(data, "helper", "d2input.exe")
	os.MkdirAll(filepath.Dir(helper), 0o755)
	os.WriteFile(helper, []byte("h"), 0o644)
	// prefix verify needs system.reg
	d.Prefix = func(ctx context.Context, c launch.Config) error {
		p := filepath.Join(c.PrefixDir, "pfx")
		os.MkdirAll(p, 0o755)
		return os.WriteFile(filepath.Join(p, "system.reg"), []byte("r"), 0o644)
	}

	w, err := New(filepath.Join(dir, "setup.json"), NewSteps(d))
	if err != nil {
		t.Fatal(err)
	}
	// without inputs the run stops at the first step
	if err := w.Run(context.Background()); err == nil {
		t.Fatal("want error for missing data dir")
	}
	pf := patch
	if !withPatch {
		pf = ""
	}
	if err := w.SetInputs(Inputs{DataDir: data, Mode: ModeGameDir, GameDirSrc: src, PatchFile: pf}); err != nil {
		t.Fatal(err)
	}
	// the web UI holds the run before the character step until a name is set
	w.HoldForInput = true
	if err := w.Run(context.Background()); err != nil {
		t.Fatalf("%v\n%v", err, w.View().Log)
	}
	if v := w.View(); v.Done || created || v.Current != StepCharacter || v.Running || v.Steps[len(v.Steps)-1].Note == "" {
		t.Fatalf("run did not hold at character: %+v", v)
	}
	if err := w.SetInputs(Inputs{CharName: "Wizbot"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Run(context.Background()); err != nil {
		t.Fatalf("%v\n%v", err, w.View().Log)
	}
	v := w.View()
	if !v.Done || !extracted || !created {
		t.Fatalf("%+v", v.Steps)
	}
	for _, s := range v.Steps {
		if (s.ID == StepInstallD2 || s.ID == StepInstallLo) && s.Status != StepSkipped {
			t.Errorf("%s should be skipped", s.ID)
		}
	}
}

func TestSourcesValidation(t *testing.T) {
	d := NewDeps(NewEnv())
	for name, s := range map[string]State{
		"no mode":      {},
		"relative":     {Inputs: Inputs{Mode: ModeGameDir, GameDirSrc: "rel/path"}},
		"missing stub": {Inputs: Inputs{Mode: ModeStubs, StubD2: "/nonexistent/a.exe", StubLoD: "/nonexistent/b.exe"}},
		"no installer": {Inputs: Inputs{Mode: ModeInstallerDirs, InstallerD2Dir: t.TempDir(), InstallerLoDDir: t.TempDir()}},
		"empty game":   {Inputs: Inputs{Mode: ModeGameDir, GameDirSrc: t.TempDir()}},
	} {
		if err := d.validateSources(&s); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestCopyTreeRefusesSymlink(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "a"), []byte("1"), 0o644)
	os.Symlink("/etc/passwd", filepath.Join(src, "l"))
	if err := copyTree(context.Background(), src, filepath.Join(t.TempDir(), "o"), func(string, ...any) {}); err == nil {
		t.Error("symlink copied")
	}
}

// --- HTTP

func TestHTTP(t *testing.T) {
	var log []string
	w, _ := New(filepath.Join(t.TempDir(), "s.json"), fakeSteps(&log, ""))
	a := &API{W: w, Env: fakeEnv(map[string]string{"/data": ""}), Opts: func() Options { return Options{DataDir: "/data"} }}
	srv := httptest.NewServer(a.Handler())
	defer srv.Close()
	do := func(method, path, body string) (int, string) {
		req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b := new(strings.Builder)
		buf := make([]byte, 4096)
		for {
			n, e := resp.Body.Read(buf)
			b.Write(buf[:n])
			if e != nil {
				break
			}
		}
		return resp.StatusCode, b.String()
	}
	if c, _ := do("GET", "/state", ""); c != 200 {
		t.Error(c)
	}
	if c, b := do("PUT", "/inputs", `{"mode":"stubs","cdKey":"AAAA-BBBBBB-CCCC"}`); c != 400 || strings.Contains(b, "AAAA") {
		t.Errorf("cdKey accepted: %d %s", c, b)
	}
	if c, _ := do("PUT", "/inputs", `{"mode":"stubs","slots":2}`); c != 200 {
		t.Error(c)
	}
	if c, _ := do("GET", "/inputs", ""); c != 405 {
		t.Error(c)
	}
	if c, b := do("GET", "/checks?quick=1&pre=1", ""); c != 200 || !strings.Contains(b, `"kernel.yama"`) {
		t.Errorf("%d %s", c, b)
	}
	if c, _ := do("POST", "/start", ""); c != 202 {
		t.Error(c)
	}
	w.Wait()
	if c, _ := do("POST", "/reset?step=zzz", ""); c != 400 {
		t.Error(c)
	}
	if c, _ := do("GET", "/nope", ""); c != 404 {
		t.Error(c)
	}
}

// Real Game.exe, skipped without the dev copy.
func TestRealGameExe(t *testing.T) {
	const dir = "/tmp/d2mpq"
	if _, err := os.Stat(dir + "/Game.exe"); err != nil {
		t.Skip("no " + dir)
	}
	e := NewEnv()
	o := &Options{GameDir: dir, KnownGameExe: KnownGameExe}
	if r := checkGameVersion(context.Background(), e, o); r.Status != Pass {
		t.Errorf("%+v", r)
	}
}

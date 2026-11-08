package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/selfupdate"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time        { return c.t }
func (c *fakeClock) Sleep(d time.Duration) { c.t = c.t.Add(d) }

// script describes one fake child: when (after start) it exits and with what code; a TERM makes it exit at once.
type script struct {
	exitAfter  time.Duration // 0: never on its own
	code       int
	ignoreTerm bool
}

type fakeChild struct {
	clk     *fakeClock
	started time.Time
	s       script
	termed  bool
	killed  bool
}

func (c *fakeChild) Exited() (int, bool) {
	if c.killed {
		return 137, true
	}
	if c.termed && !c.s.ignoreTerm {
		return 0, true
	}
	if c.s.exitAfter > 0 && c.clk.Now().Sub(c.started) >= c.s.exitAfter {
		return c.s.code, true
	}
	return 0, false
}
func (c *fakeChild) Terminate() { c.termed = true }
func (c *fakeChild) Kill()      { c.killed = true }

type fakeRunner struct {
	clk     *fakeClock
	scripts map[string][]script // by release version, consumed in order
	started []string
	failFor string
}

func (r *fakeRunner) Start(rel Release) (Child, error) {
	r.started = append(r.started, rel.Version)
	if rel.Version == r.failFor {
		return nil, errors.New("exec format error")
	}
	q := r.scripts[rel.Version]
	var s script
	if len(q) > 0 {
		s, r.scripts[rel.Version] = q[0], q[1:]
	}
	return &fakeChild{clk: r.clk, started: r.clk.Now(), s: s}, nil
}

type fakeProbes struct {
	clk       *fakeClock
	apiAfter  time.Duration // -1: never
	tickAfter time.Duration // -1: never; after this the counter advances with time
	watch     bool
	t0        time.Time
}

func (p *fakeProbes) BotAPI() bool {
	return p.apiAfter >= 0 && p.clk.Now().Sub(p.t0) >= p.apiAfter
}
func (p *fakeProbes) Tick() (uint64, bool) {
	if p.tickAfter < 0 {
		return 7, true // a frozen counter
	}
	el := p.clk.Now().Sub(p.t0)
	if el < p.tickAfter {
		return 0, false
	}
	return uint64(el / time.Second), true
}
func (p *fakeProbes) Watch() bool { return p.watch }

type rig struct {
	t      *testing.T
	b      *Boot
	clk    *fakeClock
	run    *fakeRunner
	probe  *fakeProbes
	pub    ed25519.PublicKey
	priv   ed25519.PrivateKey
	layout selfupdate.Layout
	stop   bool
}

func newRig(t *testing.T) *rig {
	clk := &fakeClock{t: time.Unix(1_800_000_000, 0)}
	pub, priv, _ := selfupdate.GenerateKey(rand.Reader)
	l := selfupdate.Layout{Root: t.TempDir()}
	r := &rig{t: t, clk: clk, pub: pub, priv: priv, layout: l}
	r.run = &fakeRunner{clk: clk, scripts: map[string][]script{}}
	r.probe = &fakeProbes{clk: clk, apiAfter: 10 * time.Second, tickAfter: 30 * time.Second, watch: true, t0: clk.t}
	r.b = &Boot{
		Layout: l,
		Image: Image{ABI: 2, BaseVersion: "0.3.0", Local: selfupdate.Compat{BotAPI: 1, D2DCtl: 1, DataSchema: 1},
			Keys: []ed25519.PublicKey{pub}, WatchWanted: true,
			Base: Release{Base: true, BinDir: "/opt/d2lod", D2DDir: "/opt/d2d", Entry: "/opt/d2lod/entrypoint.sh"}},
		Policy: DefaultPolicy(), Clock: clk, Probe: r.probe, Runner: r.run,
		Stop: func() bool { return r.stop },
	}
	return r
}

// install writes a signed release directory like Engine.Apply does.
func (r *rig) install(version string, mut func(*selfupdate.Manifest)) {
	r.t.Helper()
	dir := r.layout.Release(version)
	os.MkdirAll(filepath.Join(dir, "bin"), 0o755)
	os.WriteFile(filepath.Join(dir, "bin", "d2bot"), []byte("x"), 0o755)
	sum := sha256.Sum256([]byte("x"))
	m := selfupdate.Manifest{Schema: 1, Version: version, Seq: 9, Issued: "2026-11-20T00:00:00Z", Channel: "stable", MinImage: 1,
		Compat: selfupdate.Compat{BotAPI: 1, D2DCtl: 1, DataSchema: 1},
		Files:  []selfupdate.File{{Name: "a.tar.gz", SHA256: hex.EncodeToString(sum[:]), Size: 1, Dest: "bin"}}}
	if mut != nil {
		mut(&m)
	}
	raw, err := m.Marshal()
	if err != nil {
		r.t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, selfupdate.ManifestName), raw, 0o644)
	os.WriteFile(filepath.Join(dir, selfupdate.ManifestName+".sig"), selfupdate.Sign(r.priv, raw), 0o644)
}

func (r *rig) pendingUpdate(newV, oldV string) {
	r.install(newV, nil)
	if oldV != "" {
		r.install(oldV, nil)
	}
	r.layout.SwapCurrent(newV)
	r.layout.SaveState(selfupdate.State{Current: newV, Previous: oldV, Pending: newV})
}

func TestSelectHighestCompatible(t *testing.T) {
	r := newRig(t)
	r.install("0.4.0", nil)
	r.install("0.3.5", nil)
	got, sk := r.b.Select(selfupdate.State{Current: "0.4.0", Previous: "0.3.5"})
	if got.Version != "0.4.0" || got.Base || len(sk) != 0 {
		t.Fatalf("got %+v %v", got, sk)
	}
	// the baked base is newer (image upgrade over an older downloaded release): the base wins
	got, _ = r.b.Select(selfupdate.State{Current: "0.2.0"})
	if got.Version != "0.3.0" || !got.Base {
		t.Fatalf("base should win: %+v", got)
	}
	r.install("0.2.0", nil)
	got, _ = r.b.Select(selfupdate.State{Current: "0.2.0"})
	if !got.Base {
		t.Fatalf("older stored release beat the base: %+v", got)
	}
	// same version as the base: the immutable base wins the tie
	r.install("0.3.0", nil)
	if got, _ = r.b.Select(selfupdate.State{Current: "0.3.0"}); !got.Base {
		t.Fatal("tie should go to the base")
	}
}

func TestSelectSkipsUnusable(t *testing.T) {
	r := newRig(t)
	r.install("0.5.0", func(m *selfupdate.Manifest) { m.MinImage = 3 })          // needs a newer image
	r.install("0.4.0", func(m *selfupdate.Manifest) { m.Compat.DataSchema = 2 }) // incompatible
	got, sk := r.b.Select(selfupdate.State{Current: "0.5.0", Previous: "0.4.0"})
	if !got.Base || len(sk) != 2 {
		t.Fatalf("got %+v skipped %v", got, sk)
	}
	if !strings.Contains(sk[0].Reason, "image") || !strings.Contains(sk[1].Reason, "not compatible") {
		t.Fatalf("reasons %v", sk)
	}
	// bad list, missing dir and a signature from a key that is not baked
	r.install("0.6.0", nil)
	_, sk = r.b.Select(selfupdate.State{Current: "0.6.0", Previous: "0.9.9", Bad: []string{"0.6.0"}})
	if len(sk) != 2 {
		t.Fatalf("skipped %v", sk)
	}
	r.b.Image.Keys = []ed25519.PublicKey{func() ed25519.PublicKey { p, _, _ := selfupdate.GenerateKey(rand.Reader); return p }()}
	r.install("0.7.0", nil)
	_, sk = r.b.Select(selfupdate.State{Current: "0.7.0"})
	if len(sk) != 1 || !strings.Contains(sk[0].Reason, "signature") {
		t.Fatalf("rotated key: %v", sk)
	}
}

func TestHealthyUpdateBecomesGood(t *testing.T) {
	r := newRig(t)
	r.pendingUpdate("0.4.0", "0.3.5")
	r.run.scripts["0.4.0"] = []script{{exitAfter: 30 * time.Minute, code: 0}}
	code := r.b.Run()
	if code != 0 || strings.Join(r.run.started, ",") != "0.4.0" {
		t.Fatalf("code %d started %v", code, r.run.started)
	}
	st, _ := r.layout.LoadState()
	if st.Pending != "" || st.Good != "0.4.0" || st.Boots != 0 || st.Current != "0.4.0" {
		t.Fatalf("state %+v", st)
	}
	if m, _ := r.layout.ReadMarker(); m != nil {
		t.Fatalf("marker written for a healthy update: %+v", m)
	}
	// good only after the stability window
	if el := r.clk.Now().Sub(r.probe.t0); el < 30*time.Minute {
		t.Fatalf("run ended early: %s", el)
	}
}

func TestNeverAnswersRollsBack(t *testing.T) {
	cases := map[string]func(*rig){
		"api":   func(r *rig) { r.probe.apiAfter = -1 },
		"tick":  func(r *rig) { r.probe.tickAfter = -1 },
		"watch": func(r *rig) { r.probe.watch = false },
	}
	for name, mut := range cases {
		r := newRig(t)
		mut(r)
		r.pendingUpdate("0.4.0", "0.3.5")
		r.run.scripts["0.4.0"] = []script{{}} // runs forever
		r.run.scripts["0.3.5"] = []script{{exitAfter: time.Hour}}
		code := r.b.Run()
		if code != 0 || strings.Join(r.run.started, ",") != "0.4.0,0.3.5" {
			t.Fatalf("%s: code %d started %v", name, code, r.run.started)
		}
		if r.layout.CurrentVersion() != "0.3.5" {
			t.Fatalf("%s: current %q", name, r.layout.CurrentVersion())
		}
		st, _ := r.layout.LoadState()
		if !st.IsBad("0.4.0") || st.Pending != "" || st.Current != "0.3.5" {
			t.Fatalf("%s: state %+v", name, st)
		}
		m, _ := r.layout.ReadMarker()
		if m == nil || m.From != "0.4.0" || m.To != "0.3.5" || m.Reason == "" || m.Seen {
			t.Fatalf("%s: marker %+v", name, m)
		}
	}
}

func TestFrozenTickRollsBack(t *testing.T) {
	r := newRig(t)
	r.probe.tickAfter = -1 // the counter answers but never moves
	r.pendingUpdate("0.4.0", "")
	r.run.scripts["0.4.0"] = []script{{}}
	r.run.scripts["0.3.0"] = []script{{exitAfter: time.Hour}}
	if code := r.b.Run(); code != 0 || strings.Join(r.run.started, ",") != "0.4.0,0.3.0" {
		t.Fatalf("code %d started %v", code, r.run.started)
	}
	// no previous: the baked base runs and the link is gone
	if r.layout.CurrentVersion() != "" {
		t.Fatalf("current %q", r.layout.CurrentVersion())
	}
}

func TestCrashLoopRollsBackOnThirdBoot(t *testing.T) {
	r := newRig(t)
	r.pendingUpdate("0.4.0", "0.3.5")
	// simulate the container restart policy: each Run is one boot; the child dies at once
	r.run.scripts["0.4.0"] = []script{{exitAfter: time.Second, code: 1}, {exitAfter: time.Second, code: 1}, {exitAfter: time.Second, code: 1}}
	r.run.scripts["0.3.5"] = []script{{exitAfter: time.Hour}}
	// the first start of a pending release that exits early is rolled back inside the same Run
	if code := r.b.Run(); code != 0 || strings.Join(r.run.started, ",") != "0.4.0,0.3.5" {
		t.Fatalf("code %d started %v", code, r.run.started)
	}

	// with the container doing the restarts (the shim itself was killed each time), the third boot rolls back unstarted
	r2 := newRig(t)
	r2.pendingUpdate("0.4.0", "0.3.5")
	st, _ := r2.layout.LoadState()
	st.Boots = 2 // two earlier boots never reached the confirmation
	r2.layout.SaveState(st)
	r2.run.scripts["0.3.5"] = []script{{exitAfter: time.Hour}}
	if code := r2.b.Run(); code != 0 || strings.Join(r2.run.started, ",") != "0.3.5" {
		t.Fatalf("code %d started %v (0.4.0 must not start on boot 3)", code, r2.run.started)
	}
	if m, _ := r2.layout.ReadMarker(); m == nil || !strings.Contains(m.Reason, "crash loop") {
		t.Fatalf("marker %+v", m)
	}
}

func TestStartFailureRollsBack(t *testing.T) {
	r := newRig(t)
	r.pendingUpdate("0.4.0", "0.3.5")
	r.run.failFor = "0.4.0"
	r.run.scripts["0.3.5"] = []script{{exitAfter: time.Hour}}
	if code := r.b.Run(); code != 0 || strings.Join(r.run.started, ",") != "0.4.0,0.3.5" {
		t.Fatalf("code %d started %v", code, r.run.started)
	}
	if m, _ := r.layout.ReadMarker(); m == nil || !strings.Contains(m.Reason, "start failed") {
		t.Fatalf("marker %+v", m)
	}
}

func TestMissingCurrentFallsBackToBase(t *testing.T) {
	r := newRig(t)
	r.layout.SaveState(selfupdate.State{Current: "0.9.0", Pending: "0.9.0"}) // directory does not exist
	r.run.scripts["0.3.0"] = []script{{exitAfter: time.Minute, code: 5}}
	if code := r.b.Run(); code != 5 || strings.Join(r.run.started, ",") != "0.3.0" {
		t.Fatalf("code %d started %v", code, r.run.started)
	}
	raw, _ := os.ReadFile(filepath.Join(r.layout.Root, "image.json"))
	if !strings.Contains(string(raw), `"rebuild_needed": true`) || !strings.Contains(string(raw), "0.9.0") {
		t.Fatalf("image.json: %s", raw)
	}
}

func TestNormalExitOfConfirmedReleasePassesThrough(t *testing.T) {
	// a release that is not pending (already good) exits: no judgement, the container restart policy decides
	r := newRig(t)
	r.install("0.4.0", nil)
	r.layout.SwapCurrent("0.4.0")
	r.layout.SaveState(selfupdate.State{Current: "0.4.0", Good: "0.4.0"})
	r.run.scripts["0.4.0"] = []script{{exitAfter: 2 * time.Minute, code: 3}}
	if code := r.b.Run(); code != 3 {
		t.Fatalf("code %d", code)
	}
	if m, _ := r.layout.ReadMarker(); m != nil {
		t.Fatal("rollback marker for a confirmed release")
	}
}

func TestApplyRestartIsNotAFailure(t *testing.T) {
	// the OLD release (base) is running while an update was applied (pending = another version): its exit is not judged
	r := newRig(t)
	r.pendingUpdate("0.4.0", "")
	r.b.Image.BaseVersion = "0.5.0" // the base is newer than the stored release: base runs, 0.4.0 is not started
	r.run.scripts["0.5.0"] = []script{{exitAfter: time.Minute, code: 0}}
	if code := r.b.Run(); code != 0 || strings.Join(r.run.started, ",") != "0.5.0" {
		t.Fatalf("code %d started %v", code, r.run.started)
	}
	st, _ := r.layout.LoadState()
	if st.IsBad("0.4.0") {
		t.Fatal("pending release marked bad although it never ran")
	}
}

func TestStopRequestTerminatesChildWithoutRollback(t *testing.T) {
	r := newRig(t)
	r.pendingUpdate("0.4.0", "0.3.5")
	r.run.scripts["0.4.0"] = []script{{}}
	n := 0
	r.b.Stop = func() bool { n++; return n > 3 }
	if code := r.b.Run(); code != 0 || strings.Join(r.run.started, ",") != "0.4.0" {
		t.Fatalf("code %d started %v", code, r.run.started)
	}
	st, _ := r.layout.LoadState()
	if st.IsBad("0.4.0") || st.Pending != "0.4.0" {
		t.Fatalf("a container stop judged the release: %+v", st)
	}
}

func TestStubbornChildIsKilledAfterGrace(t *testing.T) {
	r := newRig(t)
	r.install("0.4.0", nil)
	r.layout.SwapCurrent("0.4.0")
	r.layout.SaveState(selfupdate.State{Current: "0.4.0", Good: "0.4.0"})
	r.run.scripts["0.4.0"] = []script{{ignoreTerm: true}}
	n := 0
	r.b.Stop = func() bool { n++; return n > 2 }
	if code := r.b.Run(); code != 137 {
		t.Fatalf("code %d", code)
	}
	if el := r.clk.Now().Sub(r.probe.t0); el < r.b.Policy.TermGrace {
		t.Fatalf("killed after %s, before the grace %s", el, r.b.Policy.TermGrace)
	}
}

func TestImageStatusRecordsChanges(t *testing.T) {
	r := newRig(t)
	r.run.scripts["0.3.0"] = []script{{exitAfter: time.Second}, {exitAfter: time.Second}}
	r.b.Run()
	r.b.Image.ABI = 3
	r.b.Image.BaseVersion = "0.3.1"
	r.run.scripts["0.3.1"] = []script{{exitAfter: time.Second}}
	r.b.Run()
	raw, _ := os.ReadFile(filepath.Join(r.layout.Root, "image.json"))
	s := string(raw)
	if !strings.Contains(s, `"abi"`) || !strings.Contains(s, `"base"`) || strings.Contains(s, `"keys",`) {
		t.Fatalf("image.json: %s", s)
	}
}

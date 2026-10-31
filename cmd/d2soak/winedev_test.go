package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeWineProc writes one /proc entry. ppid 0 leaves out the stat file.
func fakeWineProc(t *testing.T, root string, pid int, comm string, ppid int, prefix string) {
	t.Helper()
	d := filepath.Join(root, fmt.Sprint(pid))
	os.MkdirAll(d, 0o755)
	os.WriteFile(filepath.Join(d, "comm"), []byte(comm+"\n"), 0o644)
	// comm with a space and a ')' in the stat line must not confuse the parser
	os.WriteFile(filepath.Join(d, "stat"), []byte(fmt.Sprintf("%d (%s x) S %d 1 1 0\n", pid, comm, ppid)), 0o644)
	if prefix != "" {
		os.WriteFile(filepath.Join(d, "environ"), []byte("A=b\x00WINEPREFIX="+prefix+"\x00"), 0o644)
	}
}

func TestCountWineDevices(t *testing.T) {
	root := t.TempDir()
	pfx, other := "/d/prefixes/i4/proton/pfx", "/d/prefixes/i1/proton/pfx"
	fakeWineProc(t, root, 100, "services.exe", 1, pfx)
	fakeWineProc(t, root, 101, "winedevice.exe", 100, pfx+"/") // live: parent in the prefix (trailing slash tolerated)
	fakeWineProc(t, root, 102, "winedevice.exe", 100, pfx)
	fakeWineProc(t, root, 110, "winedevice.exe", 1, pfx)   // orphan, adopted by init
	fakeWineProc(t, root, 111, "winedevice.exe", 777, pfx) // orphan, parent gone
	fakeWineProc(t, root, 112, "winedevice.exe", 50, pfx)  // orphan, adopted by a subreaper outside the prefix
	fakeWineProc(t, root, 50, "d2bot", 1, "")
	fakeWineProc(t, root, 120, "winedevice.exe", 1, other) // other instance
	fakeWineProc(t, root, 121, "winedevice.exe", 1, "")    // no environ
	fakeWineProc(t, root, 122, "services.exe", 1, pfx)     // other program
	os.WriteFile(filepath.Join(root, "self"), nil, 0o644)  // non-numeric entry
	got := countWineDevices(root, []string{pfx})
	if got != (WineCount{Total: 5, Orphans: 3}) {
		t.Fatalf("%+v", got)
	}
	if got := countWineDevices(root, nil); got != (WineCount{}) {
		t.Fatalf("no prefix: %+v", got)
	}
	if got := countWineDevices(filepath.Join(root, "missing"), []string{pfx}); got != (WineCount{}) {
		t.Fatalf("missing root: %+v", got)
	}
	if got := countWineDevices(root, []string{pfx, other}); got.Total != 6 {
		t.Fatalf("two prefixes: %+v", got)
	}
}

func TestWineSamplerAlertGrowthAndSteadyState(t *testing.T) {
	root, pfx := t.TempDir(), "/p/i4/pfx"
	fakeWineProc(t, root, 100, "services.exe", 1, pfx)
	fakeWineProc(t, root, 101, "winedevice.exe", 100, pfx)
	fakeWineProc(t, root, 102, "winedevice.exe", 100, pfx)
	c := &Collector{ProcRoot: root, WinePrefixes: []string{pfx}}
	var ss []Sample
	take := func() Sample {
		s := synth(1, nil)[0]
		w := countWineDevices(c.ProcRoot, c.WinePrefixes)
		s.Wine = &w
		s.T = t0.Add(time.Duration(len(ss)) * 30 * time.Second)
		ss = append(ss, s)
		return s
	}
	// steady state: two live devices, nothing orphaned, all quiet
	for i := 0; i < 5; i++ {
		take()
	}
	r := DefaultAlertRules()
	if a := EvalAlerts(ss, DefaultLimits(), r, ss[len(ss)-1].T); hasCode(a, "wine-orphans") {
		t.Fatalf("steady state alerted: %+v", a)
	}
	// each wineserver restart leaves one orphan behind
	for pid := 110; pid < 114; pid++ {
		fakeWineProc(t, root, pid, "winedevice.exe", 1, pfx)
		take()
	}
	a := EvalAlerts(ss, DefaultLimits(), r, ss[len(ss)-1].T)
	if !hasCode(a, "wine-orphans") || a.Alerts[len(a.Alerts)-1].Level != "warn" {
		t.Fatalf("growth must warn: %+v", a)
	}
	lim := DefaultLimits()
	lim.MaxWineOrphans = 2
	rep := Analyze(ss, lim)
	if !strings.Contains(rep.Text(), "winedevice   9 samples, total max 6 (last 6), orphans max 4 (last 4)") || !hasFinding(rep, "wine-orphans") {
		t.Fatalf("report:\n%s", rep.Text())
	}
	if !rep.Pass {
		t.Fatal("orphans are a warning, never a failure")
	}
	// at exactly the limit and with the rule off: quiet
	if a := EvalAlerts(ss, DefaultLimits(), AlertRules{WineOrphans: 4}, ss[len(ss)-1].T); hasCode(a, "wine-orphans") {
		t.Fatalf("at the limit: %+v", a)
	}
	// an unsampled run has no winedevice line
	if txt := Analyze(synth(20, nil), lim).Text(); strings.Contains(txt, "winedevice") {
		t.Fatalf("unsampled:\n%s", txt)
	}
}

func hasFinding(r Report, code string) bool {
	for _, f := range r.Findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

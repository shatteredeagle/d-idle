package d2dsoakstat

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// logB writes a synthetic bot log in the real slog JSON format: one minute line per virtual minute, 3 wall seconds apart.
type logB struct {
	sb                           strings.Builder
	t0                           time.Time
	min                          int
	kills, xp, gold, deaths, lvl int
	area                         int
	restarts                     int
}

func newLog() *logB {
	return &logB{t0: time.Date(2026, 10, 31, 8, 0, 0, 0, time.UTC), gold: 100, lvl: 1, area: 2}
}

func (b *logB) at() string {
	return b.t0.Add(time.Duration(b.min) * 3 * time.Second).Format(time.RFC3339Nano)
}

func (b *logB) raw(msg string, kv map[string]any) {
	m := map[string]any{"time": b.at(), "level": "INFO", "msg": msg, "inst": "t"}
	for k, v := range kv {
		m[k] = v
	}
	j, _ := json.Marshal(m)
	b.sb.Write(j)
	b.sb.WriteByte('\n')
}

func (b *logB) line(msg string) {
	if strings.HasPrefix(msg, "exit ") { // the wrapper script's marker is not a log record
		b.sb.WriteString(msg + "\n")
		return
	}
	b.raw(msg, nil)
}

// minute advances the clock one virtual minute and writes the minute line. kill adds a kill and xp.
func (b *logB) minute(kill bool) {
	b.min++
	if kill {
		b.kills++
		b.xp += 40
	}
	b.raw("minute", map[string]any{"area": b.area, "clvl": b.lvl, "xp": b.xp, "kills": b.kills, "deaths": b.deaths, "potions": 3,
		"potionsUsed": 0, "gold": b.gold, "restarts": b.restarts, "rung": 0, "goal": "campaign", "step": "x", "state": "farming", "rstate": "clearing"})
}

func (b *logB) run(n int, kill bool) {
	for i := 0; i < n; i++ {
		b.minute(kill)
	}
}

// pace writes a pace line; virtual is in virtual minutes.
func (b *logB) pace(virtMin float64) {
	b.raw("pace", map[string]any{"wall": b.min * 3e9, "virtual": virtMin * 60e9, "ratio": 20.0})
}

func (b *logB) text() string { return b.sb.String() }

func scan(t *testing.T, log string, x ExitInfo) Invariants {
	t.Helper()
	inv, err := ScanInvariants(strings.NewReader(log), x, DefaultInvLimits())
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func want(t *testing.T, inv Invariants, verdict string, codes ...string) {
	t.Helper()
	if inv.Verdict != verdict || strings.Join(inv.Violated, ",") != strings.Join(codes, ",") {
		t.Errorf("verdict %s %v, want %s %v (%+v)", inv.Verdict, inv.Violated, verdict, codes, inv)
	}
}

// cleanLog is a 120 virtual minute run with kills, area entries, a chicken exit, a fresh game and a transient wait.
func cleanLog() *logB {
	b := newLog()
	b.line("d2d backend up")
	for i := 0; i < 12; i++ {
		b.line("Entered Blood Moor")
		b.run(10, true)
		b.pace(float64(b.min))
	}
	b.restarts = 1
	b.line("d2d: Save and Exit requested (chicken / leave game): the process quits and its socket goes away")
	for i := 0; i < 20; i++ { // a short timer wait: storm lines for 2 minutes
		b.raw("area gate refused the walk", map[string]any{"err": "area not allowed now: no eligible reachable leveling area for level 8"})
	}
	b.run(2, false)
	b.line("exit 0")
	return b
}

func TestInvariantsCleanRun(t *testing.T) {
	inv := scan(t, cleanLog().text(), ExitInfo{WantHours: 2})
	want(t, inv, InvPass)
	if inv.Exit != "limit" || inv.Chickens != 1 || inv.FreshGames != 1 || inv.WedgeLines != 20 || inv.Kills != 119 || inv.Deaths != 0 {
		t.Errorf("counters %+v", inv)
	}
	if inv.NoProgressMin > 3 || inv.WedgeMin > 3 || inv.Engine != "d2d" || inv.VirtMin != 121 || len(inv.Acts) != 1 || inv.Acts[0] != 1 {
		t.Errorf("stretches %+v", inv)
	}
}

func TestInvariantsHang(t *testing.T) {
	b := newLog()
	b.line("Entered Blood Moor")
	b.run(10, true)
	b.pace(10)
	b.run(5, true)
	b.pace(10) // wall passed, the virtual clock did not
	inv := scan(t, b.text(), ExitInfo{})
	want(t, inv, InvViolated, InvHang)
	if inv.PaceFrozen != 1 {
		t.Errorf("frozen %d", inv.PaceFrozen)
	}
	b = newLog()
	b.run(5, true)
	b.line("combat no progress, holding")
	b.run(5, true)
	want(t, scan(t, b.text(), ExitInfo{}), InvViolated, InvHang)
}

func TestInvariantsWedge(t *testing.T) { // the navsoak signature: storms in town, no Entered, no kills
	b := newLog()
	b.line("Entered Blood Moor")
	b.run(20, true)
	b.raw("action failed", map[string]any{"what": "back to field", "err": "nav: no exit to that level: no route from level 1 to 8"})
	for i := 0; i < 60; i++ {
		b.raw("area gate refused the walk", map[string]any{"err": "area not allowed now: no eligible reachable leveling area for level 8"})
		b.minute(false)
	}
	inv := scan(t, b.text(), ExitInfo{})
	want(t, inv, InvViolated, InvWedge, InvProgress)
	if inv.WedgeMin < 55 || inv.NoProgressMin < 55 || inv.WedgeLines != 61 {
		t.Errorf("wedge %+v", inv)
	}
}

func TestInvariantsStallLadder(t *testing.T) {
	b := newLog()
	b.run(5, true)
	b.line("campaign stalled: no progress for 30 virtual min")
	b.run(5, true)
	inv := scan(t, b.text(), ExitInfo{})
	want(t, inv, InvViolated, InvStall)
	b.line("campaign stalled, exiting for a restart")
	if inv = scan(t, b.text(), ExitInfo{}); inv.StallWarns != 1 || inv.StallExits != 1 {
		t.Errorf("rungs %+v", inv)
	}
}

func TestInvariantsEarlyExit(t *testing.T) {
	b := newLog()
	b.run(60, true)
	// a run.json that says cmd exited after 1 of 3 requested hours
	inv := scan(t, b.text(), ExitInfo{WantHours: 3, End: "cmd exited"})
	want(t, inv, InvViolated, InvExit)
	if inv.Exit != "early" {
		t.Errorf("exit %s", inv.Exit)
	}
	// the wrapper's exit marker with a code
	b.line("exit 2")
	if inv = scan(t, b.text(), ExitInfo{}); inv.Exit != "crash" {
		t.Errorf("exit %s", inv.Exit)
	}
	// supervisor log of the hidden soaks
	var x ExitInfo
	if err := ScanSupervisor(strings.NewReader("2026-10-31 02:29:33 launch attempt 0 (logs/bot-0.log), 14400s left\n2026-10-31 03:00:00 bot ended (exit code 5, restart budget used), supervisor exit\n"), &x); err != nil {
		t.Fatal(err)
	}
	if x.SupCode != 5 || !x.SupEnded || x.Launches != 1 {
		t.Errorf("supervisor %+v", x)
	}
	want(t, scan(t, newLogRun(60).text(), x), InvViolated, InvExit)
	// clean supervisor end
	x = ExitInfo{}
	ScanSupervisor(strings.NewReader("launch attempt 0\nbot ended (exit code 0), time limit reached, supervisor exit\n"), &x)
	want(t, scan(t, newLogRun(60).text(), x), InvPass)
	// an unfinished run is open, not a violation
	if inv = scan(t, newLogRun(60).text(), ExitInfo{}); inv.Exit != "open" || inv.Verdict != InvPass {
		t.Errorf("open %+v", inv)
	}
}

func newLogRun(n int) *logB {
	b := newLog()
	b.line("Entered Blood Moor")
	b.run(n, true)
	return b
}

func TestInvariantsRestarts(t *testing.T) {
	b := newLog()
	b.line("d2bot start")
	b.run(10, true)
	b.line("d2bot start")
	b.run(10, true)
	inv := scan(t, b.text(), ExitInfo{})
	want(t, inv, InvViolated, InvRestarts)
	if inv.BotRestarts != 1 {
		t.Errorf("restarts %d", inv.BotRestarts)
	}
	// supervisor launches beyond the first count too
	b = newLogRun(10)
	want(t, scan(t, b.text(), ExitInfo{Launches: 3, SupPresent: true}), InvViolated, InvRestarts)
}

func TestInvariantsDeaths(t *testing.T) {
	// a single death with recovery: fine
	b := newLogRun(10)
	b.deaths++
	b.run(1, false)
	b.run(15, true)
	want(t, scan(t, b.text(), ExitInfo{}), InvPass)
	// rate: 4 deaths in 200 kills (20 per 1000), spread out, each recovered
	b = newLog()
	for i := 0; i < 5; i++ {
		b.line("Entered Blood Moor")
		for j := 0; j < 40; j++ {
			b.minute(true)
			b.minute(false)
		}
		if i < 4 {
			b.deaths++
		}
	}
	b.run(3, true)
	inv := scan(t, b.text(), ExitInfo{})
	if inv.Deaths != 4 || inv.DeathsPer1000 <= 15 || inv.DeathStreak || inv.Unrecovered != 0 || !contains(inv.Violated, InvDeaths) {
		t.Errorf("rate %+v", inv)
	}
	// the same rate under 150 kills is not judged
	b = newLogRun(20)
	b.deaths++
	b.run(20, true)
	if inv = scan(t, b.text(), ExitInfo{}); inv.Verdict != InvPass {
		t.Errorf("few kills %+v", inv)
	}
	// streak: 3 deaths within 30 minutes without a level
	b = newLogRun(10)
	for i := 0; i < 3; i++ {
		b.deaths++
		b.run(1, true)
		b.line("Entered Blood Moor")
		b.run(5, true)
	}
	inv = scan(t, b.text(), ExitInfo{})
	if !inv.DeathStreak || inv.Verdict != InvViolated {
		t.Errorf("streak %+v", inv)
	}
	// unrecovered: dead and nothing happens for the rest of the log
	b = newLogRun(10)
	b.deaths++
	b.run(1, false)
	b.run(30, false)
	inv = scan(t, b.text(), ExitInfo{})
	if inv.Unrecovered != 1 || !contains(inv.Violated, InvDeaths) {
		t.Errorf("unrecovered %+v", inv)
	}
	// a death at the very end of the log is not judged
	b = newLogRun(10)
	b.deaths++
	b.run(3, false)
	if inv = scan(t, b.text(), ExitInfo{}); inv.Unrecovered != 0 {
		t.Errorf("tail %+v", inv)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func TestInvariantsNoData(t *testing.T) {
	inv := scan(t, newLogRun(0).text(), ExitInfo{})
	want(t, inv, InvNoData)
	rep := BuildInvReport([]Run{{Name: "seed-1", Inv: &inv}, {Name: "seed-2"}}, nil)
	if rep.Verdict != InvNoData {
		t.Errorf("set verdict %s", rep.Verdict)
	}
}

func TestInvariantsTextLogs(t *testing.T) { // slog text format and a minute-only original-game log still scan
	var sb strings.Builder
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&sb, "time=%s level=INFO msg=minute area=4 clvl=5 xp=%d kills=%d deaths=0 gold=10 restarts=0\n",
			t0.Add(time.Duration(i)*time.Minute).Format(time.RFC3339Nano), 1000+i*50, i*3)
	}
	want(t, scan(t, sb.String(), ExitInfo{}), InvPass)
}

func TestInvariantsRealWorldFixture(t *testing.T) { // the a18 original-game minute log: no pace, no wedge lines
	f, err := os.Open("testdata/a18_minutes.jsonl")
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()
	inv, err := ScanInvariants(f, ExitInfo{}, DefaultInvLimits())
	if err != nil || inv.Minutes < 100 {
		t.Fatalf("%v %+v", err, inv)
	}
	if inv.PaceFrozen != 0 || inv.WedgeLines != 0 || inv.StallWarns != 0 {
		t.Errorf("%+v", inv)
	}
}

func TestSeedGrouping(t *testing.T) {
	hs, err := ParseSeedSet(DefaultHoldout)
	if err != nil || len(hs) != 8 || !hs[16] || !hs[23] || hs[24] {
		t.Fatalf("%v %v", hs, err)
	}
	for _, bad := range []string{"x", "5-2", "1-5000"} {
		if _, err := ParseSeedSet(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	good := scan(t, newLogRun(30).text(), ExitInfo{})
	bad := scan(t, wedgeLog().text(), ExitInfo{})
	runs := []Run{
		{Name: "seed-3", Seed: 3, Inv: &good, XP: 100}, {Name: "x/seed-4", Inv: &good, XP: 100},
		{Name: "seed-16", Seed: 16, Inv: &good, XP: 100}, {Name: "logs/bot-base17.log", Inv: &bad, XP: 100},
	}
	rep := BuildInvReport(runs, hs)
	if len(rep.Groups) != 2 {
		t.Fatalf("groups %+v", rep.Groups)
	}
	tu, ho := rep.Groups[0], rep.Groups[1]
	if tu.Name != GroupTuning || SeedList(tu.Seeds) != "3-4" || tu.Verdict != InvPass {
		t.Errorf("tuning %+v", tu)
	}
	if ho.Name != GroupHoldout || SeedList(ho.Seeds) != "16-17" || ho.Verdict != InvViolated || !contains(ho.Violated, InvWedge) {
		t.Errorf("holdout %+v", ho)
	}
	if rep.Verdict != InvViolated || len(rep.Violated) == 0 || !strings.HasPrefix(rep.Violated[0], "seed 17:") {
		t.Errorf("set %s %v", rep.Verdict, rep.Violated)
	}
	c := rep.Coverage
	if SeedList(c.TuningSeeds) != "3-4" || SeedList(c.HoldoutSeeds) != "16-17" {
		t.Errorf("coverage %+v", c)
	}
	// no holdout set: one group, nobody is holdout
	if rep := BuildInvReport(runs[:2], nil); len(rep.Groups) != 1 || !strings.Contains(rep.Text(), "group holdout: none ran") {
		t.Errorf("no holdout: %+v", rep.Groups)
	}
}

func wedgeLog() *logB {
	b := newLogRun(5)
	for i := 0; i < 40; i++ {
		b.raw("area gate refused the walk", map[string]any{"err": "area not allowed now: no eligible reachable leveling area for level 8"})
		b.minute(false)
	}
	return b
}

func TestInvReportGolden(t *testing.T) {
	hs, _ := ParseSeedSet("16-23")
	mk := func(name string, seed int, b *logB, x ExitInfo) Run {
		inv := scan(t, b.text(), x)
		return Run{Name: name, Seed: seed, Inv: &inv, XP: 1000, Cover: &Cover{Class: "sorceress", Difficulty: "normal"}}
	}
	hang := newLogRun(10)
	hang.pace(10)
	hang.run(3, true)
	hang.pace(10)
	st := newLogRun(5)
	st.line("campaign stalled: no progress for 30 virtual min")
	st.run(5, true)
	runs := []Run{
		mk("seed-1", 1, cleanLog(), ExitInfo{WantHours: 2}),
		mk("seed-2", 2, hang, ExitInfo{}),
		mk("seed-3", 3, st, ExitInfo{}),
		mk("seed-16", 16, newLogRun(60), ExitInfo{WantHours: 3, End: "cmd exited"}),
		mk("seed-17", 17, wedgeLog(), ExitInfo{}),
	}
	got := BuildInvReport(runs, hs).Text()
	golden := "testdata/invreport.golden"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		os.WriteFile(golden, []byte(got), 0o644)
	}
	b, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != got {
		t.Errorf("golden mismatch (UPDATE_GOLDEN=1 to refresh):\n%s", got)
	}
}

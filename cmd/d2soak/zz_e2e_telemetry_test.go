package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/logx"
)

// testdata/e2e_botrun.jsonl is the output of internal/bot TestE2ETelemetryChain (D2E2E_KEEP=file): a Runner on the
// virtual clock with fakes, JSON slog through the logx.Dedup wrapper as cmd/d2bot builds it. The bot test logs 4
// fights of kills=3 sec=7 potions=1 in a row, 500 identical "nav exit" lines, and one death.
func TestE2EBotRunFixture(t *testing.T) {
	path := "testdata/e2e_botrun.jsonl"
	read := func(f func(*os.File)) {
		fh, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer fh.Close()
		f(fh)
	}
	read(func(f *os.File) {
		hs, err := ParseFightLines(f)
		if err != nil || len(hs) != 1 {
			t.Fatalf("%v %+v", err, hs)
		}
		if h := hs[0]; h.Fights != 4 || h.Kills != 12 || h.Sec != 28 || h.Potions != 4 || h.MinLife != 100 {
			t.Errorf("fight sums %+v", h)
		}
	})
	read(func(f *os.File) {
		ds, err := ParseDeathLines(f)
		if err != nil || len(ds) != 1 {
			t.Fatalf("%v %+v", err, ds)
		}
		if d := ds[0]; d.Diff != "Normal" || d.Area != 2 || d.CLvl != 3 || d.Build == "" || d.Build == "-" {
			t.Errorf("death %+v", d)
		}
	})
	read(func(f *os.File) {
		ms, err := ParseMinuteLines(f)
		if err != nil || len(ms) != 4 {
			t.Fatalf("%v %d", err, len(ms))
		}
		if z := ms[len(ms)-1]; z.Kills != 4 || z.Deaths != 1 || z.Goal != "farm_area" || z.Gold != 500 {
			t.Errorf("last minute %+v", z)
		}
	})
	out := logSummary(path)
	for _, want := range []string{"fights per hour", "4 12", "deaths: build", "Normal 1", "minutes: 4 lines", "deaths 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
}

// Dedup folds a run of identical nav lines but never the fight, death and minute telemetry, even when those repeat
// exactly; what the parsers sum equals what was logged.
func TestE2EDedupKeepsTelemetry(t *testing.T) {
	var buf bytes.Buffer
	h := logx.NewDedup(slog.NewJSONHandler(&buf, nil), time.Hour)
	log := slog.New(h).With("inst", "i0")
	ft := combat.FightTelemetry{Area: 3, Diff: 1, Danger: 2, Kills: 5, Damage: 40, MinLifePct: 60, Potions: 2, Dodges: 1}
	di := combat.DeathInfo{Area: 3, Difficulty: d2.Nightmare, CharLevel: 30, MonsterName: "Bone Warrior", MonsterLevel: 35, Affixes: 0}
	const n = 50
	for i := 0; i < n; i++ {
		for j := 0; j < 20; j++ {
			log.Info("nav exit", "to", 4)
		}
		log.Info("combat " + ft.Line(9*time.Second))
		log.Info("combat " + ft.Line(9*time.Second))
		log.Warn(di.Line() + " build=sorc-fire")
		log.Warn(di.Line() + " build=sorc-fire")
		log.Info("minute", "area", 3, "clvl", 30, "xp", 1, "kills", 1, "deaths", 1, "potions", 0, "gold", 5, "restarts", 0, "rung", 0, "goal", "x", "step", "y")
		log.Info("minute", "area", 3, "clvl", 30, "xp", 1, "kills", 1, "deaths", 1, "potions", 0, "gold", 5, "restarts", 0, "rung", 0, "goal", "x", "step", "y")
	}
	h.Flush()
	hs, _ := ParseFightLines(bytes.NewReader(buf.Bytes()))
	ds, _ := ParseDeathLines(bytes.NewReader(buf.Bytes()))
	ms, _ := ParseMinuteLines(bytes.NewReader(buf.Bytes()))
	if len(hs) != 1 || hs[0].Fights != 2*n || hs[0].Kills != 10*n || hs[0].Sec != 18*n || hs[0].Dodges != n*2 {
		t.Errorf("fights %+v", hs)
	}
	if len(ds) != 2*n || ds[0].Build != "sorc-fire" || ds[0].Diff != "Nightmare" || ds[0].Killer != "Bone_Warrior" || ds[0].Gap != 5 {
		t.Errorf("deaths %d %+v", len(ds), ds[0])
	}
	if len(ms) != 2*n {
		t.Errorf("minutes %d", len(ms))
	}
	// An exempt line flushes the counter but keeps the run key, so a nav line after a fight line stays folded (logx
	// design, not a telemetry loss): written lines plus counted repeats must account for every nav line.
	written := strings.Count(buf.String(), `"msg":"nav exit"`)
	counted := 0
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.Contains(l, `"line":"nav exit"`) {
			var n int
			fmt.Sscanf(l[strings.Index(l, `"times":`):], `"times":%d`, &n)
			counted += n
		}
	}
	if written+counted != n*20 || written >= n*20/10 {
		t.Errorf("nav exit: %d written + %d counted, want %d total and folded", written, counted, n*20)
	}
}

// 24 h of minute lines, fights and deaths plus 100k repeated lines, unfolded (an old log or a different writer):
// the parsers stay fast, bounded in memory and exact.
func TestE2E24hSyntheticLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "24h.log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	t0 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	ts := func(d time.Duration) string { return t0.Add(d).Format(time.RFC3339Nano) }
	ft := combat.FightTelemetry{Area: 5, Diff: 2, Danger: 3, Kills: 4, Damage: 50, MinLifePct: 40, Potions: 1, Dodges: 2}
	di := combat.DeathInfo{Area: 5, Difficulty: d2.Hell, CharLevel: 80, MonsterName: "Doom Knight", MonsterLevel: 85}
	var fights, deaths int
	line := func(d time.Duration, lvl, msg string) {
		fmt.Fprintf(f, "{\"time\":%q,\"level\":%q,\"msg\":%q,\"inst\":\"i0\"}\n", ts(d), lvl, msg)
	}
	for m := 0; m < 24*60; m++ {
		d := time.Duration(m) * time.Minute
		fmt.Fprintf(f, "{\"time\":%q,\"level\":\"INFO\",\"msg\":\"minute\",\"inst\":\"i0\",\"area\":5,\"clvl\":%d,\"xp\":%d,\"kills\":%d,\"deaths\":%d,\"potions\":%d,\"gold\":%d,\"restarts\":0,\"rung\":0,\"goal\":\"farm_area\",\"step\":\"s\"}\n",
			ts(d), 60+m/100, m*1000, m*3, deaths, m/10, m*10)
		if m%2 == 0 {
			line(d+time.Second, "INFO", "combat "+ft.Line(8*time.Second))
			fights++
		}
		if m%120 == 7 {
			line(d+2*time.Second, "WARN", di.Line()+" build=sorc-fire")
			deaths++
		}
		for i := 0; i < 70; i++ { // 100,800 repeated lines in all
			line(d+3*time.Second, "INFO", "nav exit")
		}
	}
	f.Close()

	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	start := time.Now()
	out := logSummary(path)
	el := time.Since(start)
	runtime.ReadMemStats(&m1)
	if el > 20*time.Second {
		t.Errorf("parse took %v", el)
	}
	if alloc := (m1.TotalAlloc - m0.TotalAlloc) >> 20; alloc > 1500 {
		t.Errorf("allocated %d MiB", alloc)
	}
	rd, _ := os.Open(path)
	defer rd.Close()
	hs, err := ParseFightLines(rd)
	if err != nil || len(hs) != 24 {
		t.Fatalf("hours %d err %v", len(hs), err)
	}
	var sum int
	for _, h := range hs {
		sum += h.Fights
	}
	if sum != fights || fights != 720 {
		t.Errorf("fights %d want %d", sum, fights)
	}
	rd.Seek(0, 0)
	ds, _ := ParseDeathLines(rd)
	rd.Seek(0, 0)
	ms, _ := ParseMinuteLines(rd)
	if len(ds) != deaths || deaths != 12 || len(ms) != 1440 {
		t.Errorf("deaths %d/%d minutes %d", len(ds), deaths, len(ms))
	}
	if !strings.Contains(out, "minutes: 1440 lines") {
		t.Errorf("summary:\n%s", out)
	}
	t.Logf("24h log parsed in %v, %d MiB allocated", el, (m1.TotalAlloc-m0.TotalAlloc)>>20)
}

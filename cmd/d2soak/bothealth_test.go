package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func minuteLog(t *testing.T, start time.Time, n int, restarts func(i int) int) string {
	t.Helper()
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "time=%s level=INFO msg=minute area=1 clvl=5 xp=%d kills=%d deaths=0 potions=0 gold=10 restarts=%d rung=0 goal=campaign step=clear\n",
			start.Add(time.Duration(i)*time.Minute).Format(time.RFC3339Nano), i*100, i, restarts(i))
	}
	return b.String()
}

func writeLog(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "bot.log")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func codes(as []Alert) string {
	var c []string
	for _, a := range as {
		c = append(c, a.Code)
	}
	return strings.Join(c, ",")
}

func TestTickAge(t *testing.T) {
	log := writeLog(t, minuteLog(t, t0, 30, func(int) int { return 0 })) // last line at t0+29m
	r := AlertRules{BotLog: log, TickAge: 3 * time.Minute}
	if a := botLogAlerts(r, t0.Add(31*time.Minute)); len(a) != 0 {
		t.Fatalf("fresh log alerted: %v", a)
	}
	a := botLogAlerts(r, t0.Add(33*time.Minute))
	if codes(a) != "tick-age" || a[0].Level != "crit" || !strings.Contains(a[0].Msg, "4m") {
		t.Fatalf("stale log: %+v", a)
	}
	if a := botLogAlerts(AlertRules{BotLog: log}, t0.Add(5*time.Hour)); len(a) != 0 {
		t.Fatalf("TickAge 0 must be off: %v", a)
	}
	if a := botLogAlerts(AlertRules{TickAge: time.Minute}, t0.Add(5*time.Hour)); len(a) != 0 {
		t.Fatalf("no log path must be off: %v", a)
	}
}

func TestTickAgeEmptyAndMissingLog(t *testing.T) {
	r := AlertRules{BotLog: writeLog(t, "boot\n"), TickAge: 3 * time.Minute}
	if a := botLogAlerts(r, t0); len(a) != 0 {
		t.Fatalf("bot not started yet must not alert: %v", a)
	}
	r.BotLog = filepath.Join(t.TempDir(), "nope.log")
	if a := botLogAlerts(r, t0); codes(a) != "bot-log" || a[0].Level != "warn" {
		t.Fatalf("missing log: %v", a)
	}
}

func TestTickAgeLargeLogWithoutMinuteLines(t *testing.T) {
	log := writeLog(t, strings.Repeat("time=2026-10-29T08:00:00Z level=INFO msg=spam\n", minuteTailBytes/40))
	if a := botLogAlerts(AlertRules{BotLog: log, TickAge: time.Minute}, t0); codes(a) != "tick-age" {
		t.Fatalf("%v", a)
	}
}

func TestTickAgeUsesTailOfBigLog(t *testing.T) {
	spam := strings.Repeat("time=2026-10-29T07:00:00Z level=INFO msg=spam\n", minuteTailBytes/40)
	log := writeLog(t, spam+minuteLog(t, t0, 5, func(int) int { return 0 }))
	if a := botLogAlerts(AlertRules{BotLog: log, TickAge: 3 * time.Minute}, t0.Add(5*time.Minute)); len(a) != 0 {
		t.Fatalf("%v", a)
	}
}

func TestBotLogDirectoryPicksNewest(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "a-d2bot.log")
	cur := filepath.Join(dir, "b-d2bot.log")
	os.WriteFile(old, []byte(minuteLog(t, t0.Add(-5*time.Hour), 3, func(int) int { return 0 })), 0o644)
	os.WriteFile(cur, []byte(minuteLog(t, t0, 3, func(int) int { return 0 })), 0o644)
	os.Chtimes(old, t0.Add(-4*time.Hour), t0.Add(-4*time.Hour))
	os.Chtimes(cur, t0, t0)
	if a := botLogAlerts(AlertRules{BotLog: dir, TickAge: 3 * time.Minute}, t0.Add(3*time.Minute)); len(a) != 0 {
		t.Fatalf("newest log is fresh: %v", a)
	}
}

func TestGameRestartsPerHour(t *testing.T) {
	// a game restart at minutes 10, 20, 30, 40 (4 in the hour), then the bot restarts (total drops to 0, then 1)
	log := writeLog(t, minuteLog(t, t0, 60, func(i int) int {
		switch {
		case i < 10:
			return 0
		case i < 20:
			return 1
		case i < 30:
			return 2
		case i < 40:
			return 3
		case i < 50:
			return 4
		}
		return 0
	}))
	now := t0.Add(59 * time.Minute)
	if n := gameRestartsSince(mustMinutes(t, log), now.Add(-time.Hour)); n != 4 {
		t.Fatalf("restarts %d, want 4 (a counter reset adds nothing)", n)
	}
	a := botLogAlerts(AlertRules{BotLog: log, GameRestartsPerHour: 2}, now)
	if codes(a) != "game-restarts" || a[0].Level != "warn" {
		t.Fatalf("%v", a)
	}
	if a := botLogAlerts(AlertRules{BotLog: log, GameRestartsPerHour: 4}, now); len(a) != 0 {
		t.Fatalf("at the limit must not alert: %v", a)
	}
	// only the trailing hour counts
	if n := gameRestartsSince(mustMinutes(t, log), t0.Add(35*time.Minute)); n != 1 {
		t.Fatalf("trailing window: %d", n)
	}
}

func mustMinutes(t *testing.T, path string) []Minute {
	t.Helper()
	ms, _, err := tailMinutes(path)
	if err != nil {
		t.Fatal(err)
	}
	return ms
}

func TestEvalAlertsWiresBotLogAndWine(t *testing.T) {
	ss := synth(20, func(i int, s *Sample) { s.Wine = &WineCount{Total: 6, Orphans: 4}; s.XP = int64(i) * 100 })
	now := ss[len(ss)-1].T
	log := writeLog(t, minuteLog(t, now.Add(-20*time.Minute), 10, func(int) int { return 0 }))
	r := DefaultAlertRules()
	r.BotLog = log
	a := EvalAlerts(ss, DefaultLimits(), r, now)
	if !hasCode(a, "tick-age") || !hasCode(a, "wine-orphans") || a.Level != "crit" {
		t.Fatalf("%+v", a)
	}
	r.TickAge, r.WineOrphans = 0, 0
	if a := EvalAlerts(ss, DefaultLimits(), r, now); a.Level != "ok" {
		t.Fatalf("rules off must be quiet: %+v", a)
	}
	// without a log path the default rules stay quiet (existing runs)
	if a := EvalAlerts(synth(20, nil), DefaultLimits(), DefaultAlertRules(), now); a.Level != "ok" {
		t.Fatalf("%+v", a)
	}
}

func TestExtraStatusAndStatusFile(t *testing.T) {
	ss := synth(30, func(i int, s *Sample) { s.Wine = &WineCount{Total: 3, Orphans: 1} })
	now := ss[len(ss)-1].T
	log := writeLog(t, minuteLog(t, now.Add(-10*time.Minute), 9, func(i int) int { return i / 5 }))
	r := DefaultAlertRules()
	r.BotLog = log
	x := ExtraStatus(ss, r, now)
	for _, want := range []string{"last minute line 2m ago", "1 game restarts in the last hour", "orphans max 1"} {
		if !strings.Contains(x, want) {
			t.Errorf("missing %q in\n%s", want, x)
		}
	}
	dir := t.TempDir()
	if err := writeLive(dir, ss, DefaultLimits(), r, now); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "status.txt"))
	if !strings.Contains(string(b), "bot log:") || !strings.Contains(string(b), "winedevice") {
		t.Fatalf("status.txt:\n%s", b)
	}
}

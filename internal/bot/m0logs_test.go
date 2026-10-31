package bot

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

func beltSnap(hp int, area int) *d2.Snapshot {
	inv := &d2.Inventory{}
	for i := 0; i < hp; i++ {
		inv.Items = append(inv.Items, d2.Unit{ID: uint32(100 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
	}
	p := &d2.Player{Name: "Tester", Level: 9, Class: d2.Sorceress, Life: 100, MaxLife: 100, Area: area, Gold: 70, StashGold: 500}
	return &d2.Snapshot{State: d2.StateInGame, Player: p, Inventory: inv}
}

func lastMinute(buf *lockedBuf) string {
	var last string
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.Contains(l, "msg=minute") {
			last = l
		}
	}
	return last
}

// potions is the belt healing count read from the snapshot, potionsUsed the drunk counter; state is the step group.
func TestMinuteLineBeltAndState(t *testing.T) {
	g := newRig(t)
	buf := &lockedBuf{}
	g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	now := g.r.e.Clock.Now()
	g.r.mu.Lock()
	g.r.activity = "Quest: fighting (Kill Andariel)"
	g.r.mu.Unlock()
	s := beltSnap(3, 2)
	g.r.obMu.Lock()
	g.r.drank.Store(7)
	g.r.minuteLine(s, s.Player, now)
	g.r.obMu.Unlock()
	l := lastMinute(buf)
	for _, w := range []string{" potions=3", " mana=0", " potionsUsed=7", " state=\"quest fight\"", " rstate=idle", " kills=0", " killsGame=0"} {
		if !strings.Contains(l, w) {
			t.Fatalf("%q missing: %s", w, l)
		}
	}
}

// kills is a per-character total: a game restart (observer reset of per-game state) and a bot restart (new Runner on the
// same death log path) never lower it; a different character name starts at 0.
func TestMinuteKillsMonotonic(t *testing.T) {
	dir := t.TempDir()
	run := func(name string, kills int, gameRestart bool) string {
		g := newRig(t)
		buf := &lockedBuf{}
		g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
		g.r.cfg.DeathLogPath = filepath.Join(dir, "deaths.json")
		s := beltSnap(0, 2)
		s.Player.Name = name
		g.r.obMu.Lock()
		defer g.r.obMu.Unlock()
		g.r.ob.kills = kills
		g.r.minuteLine(s, s.Player, g.r.e.Clock.Now())
		if gameRestart { // a game restart re-baselines the per-game tracking, not the counters
			g.r.ob.mons, g.r.ob.newGames = map[uint32]monTrack{}, g.r.ob.newGames+1
			g.r.ob.kills += 4
			g.r.minuteLine(s, s.Player, g.r.e.Clock.Now())
		}
		return lastMinute(buf)
	}
	if l := run("A", 10, true); !strings.Contains(l, " kills=14 ") || !strings.Contains(l, " killsGame=14") {
		t.Fatal(l)
	}
	if l := run("A", 3, false); !strings.Contains(l, " kills=17 ") || !strings.Contains(l, " killsGame=3") { // bot restart: 14 + 3
		t.Fatal(l)
	}
	if l := run("B", 2, false); !strings.Contains(l, " kills=2 ") {
		t.Fatal(l)
	}
}

// New deaths.jsonl fields: belt from memory, bug flag, gold, route, and the follow-up (gold next minute, back in field).
func TestDeathAutopsyM0Fields(t *testing.T) {
	g := newRig(t)
	dir := t.TempDir()
	g.r.cfg.DeathLogPath = filepath.Join(dir, "deaths.json")
	t0 := g.r.e.Clock.Now()
	g.r.obMu.Lock()
	for i, a := range []int{1, 2, 2, 8} {
		g.r.ob.noteRoute(a)
		s := beltSnap(2, a)
		g.r.trackAlive(s, t0.Add(time.Duration(i)*time.Second))
	}
	dead := &d2.Player{Level: 9, Class: d2.Sorceress, Life: 0, MaxLife: 100, Area: 8}
	g.r.recordDeath(dead, t0.Add(5*time.Second))
	au := g.r.dlog.last().Autopsy
	g.r.obMu.Unlock()
	if au.BeltHP != 2 || au.PotionsDrunk60s != 0 || !au.Bug || au.GoldCarried != 70 || au.StashGold != 500 || au.RouteID != "1>2>8" || au.MercAlive != nil {
		t.Fatalf("%+v", au)
	}
	// respawn in town, minute line, then the field again
	town := beltSnap(2, 1)
	g.r.obMu.Lock()
	g.r.noteBackInField(town.Player, t0.Add(10*time.Second))
	g.r.noteGoldNextMinute(33, t0.Add(60*time.Second))
	field := beltSnap(2, 8)
	g.r.noteBackInField(field.Player, t0.Add(25*time.Second))
	rec := g.r.dlog.last()
	g.r.obMu.Unlock()
	a := rec.Autopsy
	if a.GoldNextMinute == nil || *a.GoldNextMinute != 33 || a.SecondsToField == nil || *a.SecondsToField != 20 || a.BackInFieldTs == nil {
		t.Fatalf("follow-up %+v", a)
	}
	f, err := os.Open(filepath.Join(dir, "deaths.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if len(lines) != 2 {
		t.Fatalf("%d lines", len(lines))
	}
	var fl followLine
	if json.Unmarshal([]byte(lines[1]), &fl) != nil || !fl.FollowUp || !fl.Time.Equal(rec.Time) || *fl.SecondsToField != 20 {
		t.Fatalf("follow line %s", lines[1])
	}
}

// dmgTaken sums life drops only (potion and regen gains ignored); hpMinPct is the lowest life percent with a hostile
// near, 100 without a fight, and resets after every minute line.
func TestMinuteLineHPFields(t *testing.T) {
	g := newRig(t)
	buf := &lockedBuf{}
	g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	o := &g.r.ob
	p := &d2.Player{Life: 100, MaxLife: 100}
	g.r.obMu.Lock()
	defer g.r.obMu.Unlock()
	for _, st := range []struct{ life, hostiles int }{{100, 0}, {70, 2}, {90, 2}, {40, 1}, {100, 0}, {95, 0}} {
		p.Life = st.life
		o.noteHP(p, st.hostiles)
	}
	s := beltSnap(0, 2)
	g.r.minuteLine(s, s.Player, g.r.e.Clock.Now())
	l := lastMinute(buf)
	for _, w := range []string{" dmgTaken=85", " hpMinPct=40"} { // 30 + 50 + 5 = 85 lost
		if !strings.Contains(l, w) {
			t.Fatalf("%q missing: %s", w, l)
		}
	}
	g.r.minuteLine(s, s.Player, g.r.e.Clock.Now())
	if l := lastMinute(buf); !strings.Contains(l, " dmgTaken=85") || !strings.Contains(l, " hpMinPct=100") {
		t.Fatal(l)
	}
}

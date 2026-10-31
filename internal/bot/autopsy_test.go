package bot

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
)

// A death while travelling at zero mana leaves a post-mortem: state, trail, damage per 5 s, kit, a plain-words summary,
// and one line in deaths.jsonl. The record survives a reload of deaths.json.
func TestDeathRecordCarriesAutopsy(t *testing.T) {
	g := newRig(t)
	dir := t.TempDir()
	g.r.cfg.DeathLogPath = filepath.Join(dir, "deaths.json")
	now := g.r.e.Clock.Now()
	g.r.mu.Lock()
	g.r.st, g.r.activity = StateTraveling, "Quest: travelling (Go to the Barracks)"
	g.r.mu.Unlock()
	life := 100
	for i := 0; i < 12; i++ {
		p := &d2.Player{Level: 11, Class: d2.Sorceress, Life: life, MaxLife: 100, Mana: 0, MaxMana: 50, Area: 26, Gold: 40}
		s := &d2.Snapshot{State: d2.StateInGame, Player: p}
		g.r.obMu.Lock()
		g.r.ob.lastGood.area = 26
		g.r.trackAlive(s, now.Add(time.Duration(i)*time.Second))
		g.r.obMu.Unlock()
		life -= 8
	}
	dead := &d2.Player{Level: 11, Class: d2.Sorceress, Life: 0, MaxLife: 100, Area: 26}
	g.r.obMu.Lock()
	g.r.recordDeath(dead, now.Add(12*time.Second))
	rec := g.r.dlog.last()
	g.r.obMu.Unlock()
	au := rec.Autopsy
	if au == nil {
		t.Fatal("no autopsy")
	}
	if au.State != "traveling" || len(au.Trail) < 10 || len(au.DamagePer5s) == 0 || au.Kit.Gold != 40 {
		t.Fatalf("autopsy %+v", au)
	}
	if au.Trail[len(au.Trail)-1].Mana != 0 || !strings.Contains(au.Summary, "walking through") || !strings.Contains(au.Summary, "mana 0%") || !strings.Contains(au.Summary, "no survival action") {
		t.Fatalf("summary %q", au.Summary)
	}
	f, err := os.Open(filepath.Join(dir, "deaths.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		var back DeathRecord
		if err := json.Unmarshal(sc.Bytes(), &back); err != nil || back.Autopsy == nil {
			t.Fatalf("bad line %q: %v", sc.Text(), err)
		}
		n++
	}
	if n != 1 {
		t.Fatalf("%d lines", n)
	}
	dl, err := loadDeathLog(g.r.cfg.DeathLogPath)
	if err != nil || len(dl.Records) != 1 || dl.Records[0].Autopsy == nil {
		t.Fatalf("reload: %v %+v", err, dl.Records)
	}
}

func TestTrimLinesKeepsTheNewest(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.jsonl")
	os.WriteFile(p, []byte("1\n2\n3\n4\n"), 0o644)
	trimLines(p, 2)
	b, _ := os.ReadFile(p)
	if string(b) != "3\n4\n" {
		t.Fatalf("%q", b)
	}
}

// diagFighter adds the optional autopsy hooks to the rig's fighter.
type diagFighter struct {
	Fighter
	pc combat.PortalCast
}

func (d diagFighter) LastPortalCast() combat.PortalCast { return d.pc }
func (d diagFighter) RecentCasts(time.Duration) (int, map[int]int) {
	return 3, map[int]int{36: 2, 53: 1}
}

// 02:30 live death: potions drunk by combat (EvPotion through emit, the observer's own diff is off live) must be counted,
// so the BUG flag does not fire; the autopsy carries the route guard facts.
func TestAutopsyCountsCombatPotionsAndRouteFacts(t *testing.T) {
	g := newRig(t)
	g.r.cfg.DeathLogPath = filepath.Join(t.TempDir(), "deaths.json")
	now := g.r.e.Clock.Now()
	g.r.d.Fighter = diagFighter{Fighter: g.r.d.Fighter, pc: combat.PortalCast{Started: true, Start: now.Add(8*time.Second + 500*time.Millisecond), Dur: 1500 * time.Millisecond, OK: false}}
	g.r.mu.Lock()
	g.r.st = StateTraveling
	g.r.mu.Unlock()
	g.r.obMu.Lock()
	g.r.ob.routeLine = "route guard: blocked monastery (no merc, clvl 12 < 20)"
	g.r.obMu.Unlock()
	for i := 0; i < 10; i++ {
		p := &d2.Player{Level: 12, Class: d2.Sorceress, Life: 100 - 9*i, MaxLife: 100, Mana: 40, MaxMana: 50, Area: 7}
		s := &d2.Snapshot{State: d2.StateInGame, Player: p}
		g.r.obMu.Lock()
		g.r.ob.lastGood.area = 7
		g.r.trackAlive(s, now.Add(time.Duration(i)*time.Second))
		g.r.obMu.Unlock()
		if i == 5 || i == 8 {
			g.r.emit(goals.Event{Type: goals.EvPotion, Potion: "healing"})
		}
	}
	dead := &d2.Player{Level: 12, Class: d2.Sorceress, Life: 0, MaxLife: 100, Area: 7}
	g.r.obMu.Lock()
	g.r.recordDeath(dead, now.Add(10*time.Second))
	au := g.r.dlog.last().Autopsy
	g.r.obMu.Unlock()
	if au.PotionsDrunk60s != 2 || au.Bug {
		t.Fatalf("potions %d bug %v: %s", au.PotionsDrunk60s, au.Bug, au.Summary)
	}
	if au.Mode != "moving" || au.MercState != "none" || au.Casts5s != 3 || au.CastSkills["36"] != 2 {
		t.Fatalf("facts %+v", au)
	}
	if au.PortalCast == nil || au.PortalCast.DurMs != 1500 || au.PortalCast.OK {
		t.Fatalf("portal %+v", au.PortalCast)
	}
	if n := len(au.LastLife); n == 0 || n > 6 || au.LastLife[n-1].MsBefore > au.LastLife[0].MsBefore {
		t.Fatalf("last life %+v", au.LastLife)
	}
	for _, want := range []string{"route guard: blocked monastery (no merc, clvl 12 < 20)", "town portal cast started", "merc none"} {
		if !strings.Contains(au.Summary, want) {
			t.Errorf("summary lacks %q: %s", want, au.Summary)
		}
	}
}

func TestAutopsySummaryBugShowsFieldTime(t *testing.T) {
	au := &Autopsy{BeltHP: 2, Bug: true, FieldSecs: 6, LowLifeSecs: 1, Kit: Kit{BeltHP: 2}}
	if s := au.summarize("Carver", "Dark Wood"); !strings.Contains(s, "6 s in the field, 1 s under the drink line") {
		t.Fatal(s)
	}
}

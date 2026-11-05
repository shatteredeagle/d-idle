package nav

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

func TestStuckFightLowLife(t *testing.T) {
	builds := []struct {
		name    string
		maxLife int
	}{{"Sorceress", 200}, {"Barbarian", 900}}
	cases := []struct {
		name      string
		rung      int
		lifePct   int
		hostiles  int
		farAway   bool // hostiles outside the fight radius
		wantFight bool
	}{
		{"rung2 4 hostiles life 45", 2, 45, 4, false, true},
		{"rung2 life 80", 2, 80, 4, false, false},
		{"rung1 life 45", 1, 45, 4, false, false},
		{"no hostiles", 2, 45, 0, false, false},
		{"hostiles beyond fight radius", 2, 45, 4, true, false},
	}
	for _, b := range builds {
		for _, c := range cases {
			t.Run(b.name+"/"+c.name, func(t *testing.T) {
				s := newSim(openField(1000, 1000, 100, 60), input.Point{X: 1010, Y: 1030})
				n := s.navigator()
				var buf bytes.Buffer
				n.log = slog.New(slog.NewTextHandler(&buf, nil))
				calls := 0
				n.Yield = func(ctx context.Context, near []d2.Unit) error { calls++; return nil }
				pos := input.Point{X: 1010, Y: 1030}
				var mons []d2.Unit
				for i := 0; i < c.hostiles; i++ {
					off := 5 + i
					if c.farAway {
						off = n.e.Tune.Combat.FightRadius + 10
					}
					mons = append(mons, d2.Unit{ID: uint32(i + 1), Type: d2.TypeMonster, X: pos.X + off, Y: pos.Y, Mode: d2.MonModeStand})
				}
				o := &obs{pos: pos, snap: &d2.Snapshot{Player: &d2.Player{Life: b.maxLife * c.lifePct / 100, MaxLife: b.maxLife}, Monsters: mons}}
				w := &walkState{rung: c.rung}
				done, err := n.stuckFight(context.Background(), o, w)
				if err != nil {
					t.Fatal(err)
				}
				if done != c.wantFight || (calls == 1) != c.wantFight {
					t.Fatalf("done=%v calls=%d want fight=%v", done, calls, c.wantFight)
				}
				if got := strings.Contains(buf.String(), "nav: stuck ladder -> fight"); got != c.wantFight {
					t.Fatalf("log %q", buf.String())
				}
				if c.wantFight && (w.rung != 0 || w.have) {
					t.Fatalf("walk not reset: %+v", w)
				}
			})
		}
	}
}

func TestStuckFightPortalWhenFightsUsed(t *testing.T) {
	s := newSim(openField(1000, 1000, 100, 60), input.Point{X: 1010, Y: 1030})
	n := s.navigator()
	var buf bytes.Buffer
	n.log = slog.New(slog.NewTextHandler(&buf, nil))
	n.Yield = func(context.Context, []d2.Unit) error { t.Fatal("fought after the budget"); return nil }
	escaped := 0
	n.Escape = func(context.Context) error { escaped++; return nil }
	o := &obs{pos: input.Point{X: 1010, Y: 1030}, snap: &d2.Snapshot{Player: &d2.Player{Life: 40, MaxLife: 100},
		Monsters: []d2.Unit{{ID: 1, Type: d2.TypeMonster, X: 1015, Y: 1030, Mode: d2.MonModeStand}}}}
	w := &walkState{rung: 3, fights: maxYields}
	o.level = n.levelNow()
	if _, err := n.stuckFight(context.Background(), o, w); err != nil {
		t.Fatal(err)
	}
	if escaped != 1 || !strings.Contains(buf.String(), "nav: stuck ladder -> portal") {
		t.Fatalf("escaped=%d log %q", escaped, buf.String())
	}
}

func TestBorderBackLogged(t *testing.T) {
	s := newSim(openField(1000, 1000, 100, 60), input.Point{X: 1010, Y: 1030})
	n := s.navigator()
	var buf bytes.Buffer
	n.log = slog.New(slog.NewTextHandler(&buf, nil))
	n.lastCross = crossNote{from: 5, to: 6, at: n.clk.Now().Add(-300 * time.Millisecond), reason: "sidestep"}
	n.noteBorderBack(5)
	if out := buf.String(); !strings.Contains(out, "nav: area border crossed back") || !strings.Contains(out, "reason=sidestep") || !strings.Contains(out, "area=5 from=6") {
		t.Fatalf("log %q", out)
	}
	buf.Reset()
	n.lastCross = crossNote{from: 5, to: 6, at: n.clk.Now().Add(-5 * time.Second), reason: "sidestep"}
	n.noteBorderBack(5)
	if buf.Len() != 0 {
		t.Fatalf("late return logged: %q", buf.String())
	}
}

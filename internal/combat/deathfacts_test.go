package combat

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

type fixRow struct {
	T       float64
	Phase   string
	Mode    int
	Life    int
	Max     int
	Area    int
	X, Y    int
	Gold    int
	Equip   int
	Esc     bool
	Players []struct {
		ID   uint32
		Mode int
		X, Y int
	}
}

func readFix(t *testing.T, name string) []fixRow {
	t.Helper()
	f, err := os.Open("testdata/death/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var rows []fixRow
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var r fixRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	return rows
}

func rowPlayer(r fixRow) *d2.Player {
	return &d2.Player{ID: 1, Mode: r.Mode, Life: r.Life, MaxLife: r.Max, Area: r.Area, X: r.X, Y: r.Y, Gold: r.Gold}
}

// The recorded real deaths must satisfy the constants and helpers: dying (mode 0) first, dead (mode 17) about
// DyingSeconds later, an Esc only works from mode 17, the corpse is a second type-0 unit, equipment and gold leave at death.
func TestRecordedDeaths(t *testing.T) {
	for _, name := range []string{"death1.jsonl", "death2.jsonl"} {
		rows := readFix(t, name)
		var first, dead fixRow
		var haveFirst, haveDead bool
		for _, r := range rows {
			p := rowPlayer(r)
			if !haveFirst && IsDead(p) {
				first, haveFirst = r, true
				if p.Mode != ModeDying || !IsDying(p) || CanRespawn(p) || r.Gold != 0 {
					t.Fatalf("%s: first dead sample %+v", name, r)
				}
			}
			if haveFirst && !haveDead && r.Mode == ModeDead {
				dead, haveDead = r, true
				if !CanRespawn(p) || IsDying(p) || r.Equip != 0 || len(r.Players) != 2 {
					t.Fatalf("%s: dead sample %+v", name, r)
				}
				c, ok := FindCorpse([]d2.Unit{{Type: 0, ID: r.Players[0].ID, Mode: r.Players[0].Mode, X: r.Players[0].X, Y: r.Players[0].Y},
					{Type: 0, ID: r.Players[1].ID, Mode: r.Players[1].Mode, X: r.Players[1].X, Y: r.Players[1].Y}}, 1, r.Area)
				if !ok || c.ID != 2 || c.Pos != (input.Point{X: r.X, Y: r.Y}) || !c.Touched(input.Point{X: r.X + 1, Y: r.Y + 1}) || c.Touched(input.Point{X: r.X + 5, Y: r.Y}) {
					t.Fatalf("%s: corpse %+v ok=%v", name, c, ok)
				}
			}
		}
		if !haveFirst {
			t.Fatalf("%s: no death in the fixture", name)
		}
		if name == "death1.jsonl" { // death2 pressed Esc every 400 ms and never sampled mode 17 (0.87 s dying, 1.11 s respawned)
			if !haveDead {
				t.Fatalf("%s: no mode 17 sample", name)
			}
			if d := dead.T - first.T; d < 0.8 || d > DyingSeconds+0.15 {
				t.Fatalf("%s: dying lasted %.2fs", name, d)
			}
		}
		last := rows[len(rows)-1]
		if IsDead(rowPlayer(last)) || last.Area != ActTown(0) || last.Mode != ModeTownNeutral || last.Life != last.Max {
			t.Fatalf("%s: end state %+v", name, last)
		}
	}
}

func TestIsDeadVariants(t *testing.T) {
	if IsDead(nil) || IsDying(nil) || CanRespawn(nil) {
		t.Fatal("nil player")
	}
	if IsDead(&d2.Player{Mode: 1, Life: 0, MaxLife: 40}) || !ZeroLife(&d2.Player{Mode: 1, Life: 0, MaxLife: 40}) {
		t.Fatal("life 0 in a living mode is ZeroLife, not dead")
	}
	if IsDead(&d2.Player{Mode: ModeTownNeutral, Life: 40, MaxLife: 40}) {
		t.Fatal("town neutral is alive")
	}
	if ActTown(3) != 103 || ActTown(9) != 1 {
		t.Fatal("ActTown")
	}
}

func TestGoldPile(t *testing.T) {
	ground := []d2.Unit{{Class: 100, X: 10, Y: 10}, {Class: 523, X: 40, Y: 40}, {Class: 523, X: 12, Y: 11}}
	p, ok := GoldPile(ground, input.Point{X: 10, Y: 10}, func(c int) bool { return c == 523 })
	if !ok || p != (input.Point{X: 12, Y: 11}) {
		t.Fatalf("%v %v", p, ok)
	}
}

// Respawn must wait through the dying animation (Esc is ignored there and an Esc while alive opens the menu) and
// press Esc only once the body lies.
func TestRespawn(t *testing.T) {
	h := newHarness(t)
	now := time.Unix(0, 0)
	var keys []string
	var modes []int
	h.f.snapFn = func() (*d2.Snapshot, error) {
		el := now.Sub(time.Unix(0, 0)).Seconds()
		p := &d2.Player{ID: 1, Area: 2, Life: 0, MaxLife: 40, Mode: ModeDying}
		switch {
		case el >= 1.5 && len(keys) > 0:
			p.Mode, p.Life, p.Area = ModeTownNeutral, 40, 1
		case el >= 0.95:
			p.Mode = ModeDead
		}
		modes = append(modes, p.Mode)
		return &d2.Snapshot{State: d2.StateInGame, Player: p}, nil
	}
	h.s.mu.Lock()
	h.s.t = now
	h.s.mu.Unlock()
	hs := &clockIn{h: h, now: &now, keys: &keys}
	h.f.e.In, h.f.e.Clock = hs, hs
	if err := h.f.Respawn(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if len(keys) == 0 || keys[0] != "esc" {
		t.Fatalf("keys %v", keys)
	}
	if now.Sub(time.Unix(0, 0)) < 950*time.Millisecond {
		t.Fatal("Esc pressed before the body fell")
	}
}

// clockIn is a virtual clock plus a key recorder on top of the sim's input.
type clockIn struct {
	h    *harness
	now  *time.Time
	keys *[]string
}

func (c *clockIn) Now() time.Time                                       { return *c.now }
func (c *clockIn) Sleep(d time.Duration)                                { *c.now = c.now.Add(d) }
func (c *clockIn) Key(n string) error                                   { *c.keys = append(*c.keys, n); return nil }
func (c *clockIn) MoveMouse(x, y int) error                             { return nil }
func (c *clockIn) Click(b input.Button, x, y int, m ...input.Mod) error { return nil }
func (c *clockIn) Hold(string) error                                    { return nil }
func (c *clockIn) Release(string) error                                 { return nil }
func (c *clockIn) Pos() (input.Point, error)                            { return input.Point{}, nil }
func (c *clockIn) ClickWorld(input.Point) error                         { return nil }
func (c *clockIn) MoveTo(input.Point, input.MoveOpts) error             { return nil }
func (c *clockIn) SelectSkill(string) error                             { return nil }
func (c *clockIn) CastAt(string, input.Point) error                     { return nil }
func (c *clockIn) Interact(input.Target) error                          { return nil }
func (c *clockIn) CloseAll() error                                      { return nil }

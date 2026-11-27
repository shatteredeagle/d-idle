package survival

import (
	"testing"
	"time"

	"d2lod-bot/internal/power"
)

// The dial moves the line: at the same hero a bolder dial never picks an easier zone than a more cautious one, and it
// differs somewhere along the levels. Both builds.
func TestDialChangesTheChoice(t *testing.T) {
	for _, tc := range []struct {
		name, arch string
		hero       func(int) power.Hero
	}{{"paladin", "bruiser", paladin}, {"sorceress", "caster", sorceress}} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t, tc.arch)
			differ := false
			for lvl := 2; lvl <= 40; lvl++ {
				areas := map[string]int{}
				for _, id := range []string{"cautious", "balanced", "bold"} {
					d, err := e.m.P.Dial(id)
					if err != nil {
						t.Fatal(err)
					}
					in := e.input(tc.hero(lvl), ZoneKey{}, t0)
					in.Dial = d
					in.Cands = in.Cands[:0]
					for _, z := range zones {
						zm, err := e.m.Zone(tc.hero(lvl), e.arch, z, id)
						if err != nil {
							t.Fatal(err)
						}
						in.Cands = append(in.Cands, Candidate{Key: ZoneKey{z.Area, z.Diff}, Name: z.Name, ZM: zm, Eco: Economy(z, zm, lvl, e.p), TravelMin: 2})
					}
					dec, err := NewLedger(e.p).Choose(in)
					if err != nil {
						t.Fatal(err)
					}
					areas[id] = dec.Area
				}
				if areas["cautious"] > areas["bold"] && areas["bold"] != 0 && !(areas["cautious"] == areas["balanced"]) {
					t.Errorf("lvl %d: cautious %d is further than bold %d", lvl, areas["cautious"], areas["bold"])
				}
				if areas["cautious"] != areas["bold"] {
					differ = true
				}
			}
			if !differ {
				t.Error("the dial never changed the choice")
			}
		})
	}
}

// No thrash: a hero sitting in a safe zone that pays only a little less than the best stays (the switch margin), for
// both builds, over a power change below the 10 percent rule.
func TestNoThrashBetweenNeighbours(t *testing.T) {
	for _, tc := range []struct {
		name, arch string
		hero       func(int) power.Hero
	}{{"paladin", "bruiser", paladin}, {"sorceress", "caster", sorceress}} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t, tc.arch)
			l := NewLedger(e.p)
			cur := ZoneKey{}
			switches := 0
			for lvl := 10; lvl < 14; lvl++ { // a few levels at one power plateau, re-asked every minute
				for i := 0; i < 20; i++ {
					d, err := l.Choose(e.input(tc.hero(lvl), cur, t0.Add(time.Duration(i)*time.Minute)))
					if err != nil {
						t.Fatal(err)
					}
					k := ZoneKey{d.Area, d.Diff}
					if cur.Area != 0 && k != cur {
						switches++
					}
					cur = k
				}
			}
			if switches > 3 {
				t.Errorf("%d switches over four levels: the zone flaps", switches)
			}
		})
	}
}

// Card zonepace, class G: a left-for-cost lock lapses by time, and a zone far below the hero leaves the candidates.
// Both builds; the fake clock is the Input.Now / Observe time.
func TestZonePaceLockLapsesAndBandDrops(t *testing.T) {
	for _, tc := range []struct {
		name, arch string
		hero       func(int) power.Hero
	}{{"paladin", "bruiser", paladin}, {"sorceress", "caster", sorceress}} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t, tc.arch)
			k := ZoneKey{4, "normal"}
			h := tc.hero(20)
			in := e.input(h, ZoneKey{}, t0)
			l := NewLedger(e.p)
			l.MarkLeft(k, t0, in.Power)
			if !l.LockedAt(k, in.Power, t0.Add(time.Hour)) {
				t.Fatal("lock must hold inside the hold time")
			}
			hold := time.Duration(e.p.LockHoldMinutes * float64(time.Minute))
			if l.LockedAt(k, in.Power, t0.Add(hold+time.Minute)) {
				t.Fatal("must-fail before the fix: the lock never lapsed by time")
			}
			// band: monster levels 5, 7, 9 against a level 14 hero (normal drop 6): only Black Marsh (9) stays
			in = e.input(h, ZoneKey{}, t0)
			in.Level = 14
			for i := range in.Cands {
				in.Cands[i].MLvl = []int{5, 7, 9}[i]
			}
			if got := l.inBand(in); len(got) != 1 || got[0].Key.Area != 4 {
				t.Fatalf("band kept %v", got)
			}
			in.Level = 40 // every zone below the band: the list is kept whole, never emptied
			if got := l.inBand(in); len(got) != len(in.Cands) {
				t.Fatalf("an empty band must keep every zone, got %d", len(got))
			}
			in.Level = 0
			if got := l.inBand(in); len(got) != len(in.Cands) {
				t.Fatal("unknown level: band rule off")
			}
		})
	}
}

package senses

import (
	"sync"
	"testing"

	"d2lod-bot/internal/d2map"
)

func obs(t *testing.T, p *Publisher, no int, hero d2map.Point) {
	t.Helper()
	lm := &d2map.LevelMap{No: no, Grid: d2map.ParseASCII(0, 0, "#########", "#.......#", "#########")}
	if err := p.Observe(lm, "T", hero); err != nil {
		t.Fatal(err)
	}
}

func TestSetUnitsRadiusOrderCap(t *testing.T) {
	p := &Publisher{}
	p.SetUnits([]Unit{{ID: 1}}, Point{}) // no snapshot yet: nothing happens, no panic
	obs(t, p, 1, d2map.Point{X: 2, Y: 1})
	var us []Unit
	for i := 0; i < MaxUnits+30; i++ {
		us = append(us, Unit{ID: uint32(i + 1), Kind: KindMonster, X: 2 + i%30, Y: 1})
	}
	us = append(us, Unit{ID: 9999, Kind: KindMonster, X: 2 + UnitRadius + 1, Y: 1}) // outside the radius
	p.SetUnits(us, Point{X: 2, Y: 1})
	s, _ := p.Snapshot()
	if len(s.Unit) != MaxUnits {
		t.Fatalf("%d units", len(s.Unit))
	}
	for i, u := range s.Unit {
		if u.ID == 9999 {
			t.Fatal("unit outside the radius published")
		}
		if i > 0 && dist2(u.X, u.Y, s.Hero) < dist2(s.Unit[i-1].X, s.Unit[i-1].Y, s.Hero) {
			t.Fatal("not nearest first")
		}
	}
}

func TestUnitsImmutableAndAreaChange(t *testing.T) {
	p := &Publisher{}
	obs(t, p, 1, d2map.Point{X: 2, Y: 1})
	in := []Unit{{ID: 1, Kind: KindMonster, X: 3, Y: 1, Mods: []string{"Cursed"}}}
	p.SetUnits(in, Point{X: 2, Y: 1})
	in[0].Mods[0] = "changed"
	in[0].X = 7
	s, _ := p.Snapshot()
	if s.Unit[0].X != 3 || s.Unit[0].Mods[0] != "Cursed" {
		t.Fatalf("snapshot aliases the caller's data: %+v", s.Unit[0])
	}
	obs(t, p, 1, d2map.Point{X: 3, Y: 1}) // same area, hero moved: the units stay
	if s, _ = p.Snapshot(); len(s.Unit) != 1 {
		t.Fatal("units lost inside one area")
	}
	obs(t, p, 2, d2map.Point{X: 3, Y: 1}) // new area: no stale units
	if s, _ = p.Snapshot(); len(s.Unit) != 0 {
		t.Fatal("units carried into the next area")
	}
}

func TestUnitsConcurrent(t *testing.T) {
	p := &Publisher{}
	obs(t, p, 1, d2map.Point{X: 2, Y: 1})
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				p.SetUnits([]Unit{{ID: uint32(i), Kind: KindMonster, X: 3, Y: 1}}, Point{X: 2, Y: 1})
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				s, _ := p.Snapshot()
				_ = len(s.Unit)
			}
		}()
	}
	wg.Wait()
}

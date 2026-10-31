package goals

import (
	"testing"
	"time"
)

func TestSeriesAndRates(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 30, 20, 0, time.UTC)
	s := newStats(now)
	rec := func(d time.Duration, ev Event) { ev.Time = now.Add(-d); s.record(&ev, "") }
	rec(0, Event{Type: EvKill, XP: 100})
	rec(time.Minute, Event{Type: EvKill, XP: 50})
	rec(10*time.Minute, Event{Type: EvKill, XP: 300, Boss: true})
	rec(10*time.Minute, Event{Type: EvDeath})
	rec(2*time.Hour, Event{Type: EvPotion})
	if s.Totals.Kills != 3 || s.Totals.BossKills != 1 || s.Totals.XP != 450 || s.Totals.Deaths != 1 {
		t.Fatalf("%+v", s.Totals)
	}
	pts := s.Series(now, 15*time.Minute, 5*time.Minute)
	if len(pts) != 3 {
		t.Fatalf("%d points", len(pts))
	}
	var sum int64
	for _, p := range pts {
		sum += p.XP
	}
	if sum != 450 {
		t.Fatalf("series xp %d", sum)
	}
	if pts[2].XP != 100 || pts[2].XPPerHour != 100*12 {
		t.Fatalf("%+v", pts[2])
	}
	if hp := s.Series(now, 6*time.Hour, time.Hour); len(hp) != 6 || hp[5].XP != 450 || hp[3].Potions != 1 {
		t.Fatalf("%+v", hp)
	}
	if r := s.XPPerHour(now, 15*time.Minute); r < 1000 {
		t.Fatalf("rate %f", r)
	}
	if r := newStats(now).XPPerHour(now, time.Hour); r != 0 {
		t.Fatal("empty stats must be 0")
	}
}

func TestStatsBounded(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := newStats(now)
	for i := 0; i < 5000; i++ { // 5000 minutes of kills, and out-of-order stragglers
		ev := Event{Type: EvKill, XP: 1, Time: now.Add(time.Duration(i) * time.Minute)}
		s.record(&ev, "")
		d := Event{Type: EvDrop, Item: &Item{Name: "x", Quality: "magic"}, Time: ev.Time}
		s.record(&d, "")
		a := Event{Type: EvAreaChange, Area: "a", Time: ev.Time}
		s.record(&a, "")
	}
	if len(s.Minutes) != minuteBuckets || len(s.Hours) > hourBuckets || len(s.Drops) != maxDrops || len(s.Areas) != maxAreas {
		t.Fatalf("%d %d %d %d", len(s.Minutes), len(s.Hours), len(s.Drops), len(s.Areas))
	}
	for i := 1; i < len(s.Minutes); i++ {
		if s.Minutes[i].T <= s.Minutes[i-1].T {
			t.Fatal("buckets must stay sorted")
		}
	}
	late := Event{Type: EvKill, XP: 5, Time: now.Add(4990 * time.Minute)}
	s.record(&late, "")
	if s.Totals.XP != 5005 {
		t.Fatal("late event lost")
	}
}

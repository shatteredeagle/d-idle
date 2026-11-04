package drill

import (
	"context"
	"testing"

	"d2lod-bot/internal/env"
)

func TestRetreatDrinkLifeBetweenKnobs(t *testing.T) {
	wide := env.DefaultCombatTuning()
	wide.ChickenPct, wide.DrinkLifePct = 15, 80
	tight := env.DefaultCombatTuning()
	tight.ChickenPct, tight.DrinkLifePct = 44, 46
	for name, c := range map[string]env.CombatTuning{"default": env.DefaultCombatTuning(), "wide": wide, "tight": tight} {
		st := &retreatStub{info: RetreatInfo{Err: "stop"}}
		for _, d := range rt(st, c) {
			d.Do(context.Background(), nil)
		}
		if got := st.got[0].LifePct; got <= c.ChickenPct || got >= c.DrinkLifePct {
			t.Errorf("%s: drink life %d not in (%d,%d)", name, got, c.ChickenPct, c.DrinkLifePct)
		}
		if st.got[1].MaxTicks != 10 {
			t.Errorf("%s: portal bound %d, want 10", name, st.got[1].MaxTicks)
		}
	}
}

type retreatStub struct {
	FieldLab
	got  []RetreatSpec
	info RetreatInfo
}

func (s *retreatStub) Retreat(_ context.Context, _ *Run, sp RetreatSpec) RetreatInfo {
	s.got = append(s.got, sp)
	return s.info
}

func TestRetreatVerdicts(t *testing.T) {
	good := map[string]RetreatInfo{
		"RT.drink":  {Adjacent: 1, Life: 20, MaxLife: 100, LifeFloor: 15, LifeEnd: 60, Decision: "drink-rejuv", Acted: true, TicksToEffect: 3, BeltBefore: 2, BeltAfter: 1},
		"RT.portal": {Adjacent: 1, Decision: "town-portal", Acted: true, LifeFloor: 20, TicksToEffect: 1, ScrollsBefore: 1, PortalOpen: true},
		"RT.leave":  {Adjacent: 1, Decision: "chicken", Acted: true, LifeFloor: 5, TicksToEffect: 40, LeftGame: true},
	}
	mods := []struct {
		name string
		mod  func(*RetreatInfo)
		want Verdict
	}{
		{"ok", func(*RetreatInfo) {}, Pass},
		{"setup", func(i *RetreatInfo) { i.Adjacent = 0 }, Blocked},
		{"read error", func(i *RetreatInfo) { i.Err = "snapshot" }, Blocked},
		{"wrong decision", func(i *RetreatInfo) { i.Decision = "none" }, Fail},
		{"not executed", func(i *RetreatInfo) { i.Acted = false }, Fail},
		{"act error", func(i *RetreatInfo) { i.ActErr = "no potion" }, Fail},
		{"too slow", func(i *RetreatInfo) { i.TicksToEffect = 9999 }, Fail},
		{"no effect", func(i *RetreatInfo) { i.TicksToEffect = -1 }, Fail},
		{"dead", func(i *RetreatInfo) { i.Dead = true }, Fail},
	}
	for _, id := range []string{"RT.drink", "RT.portal", "RT.leave"} {
		for _, m := range mods {
			i := good[id]
			m.mod(&i)
			for _, d := range RT(&retreatStub{info: i}) {
				if d.ID != id {
					continue
				}
				if o := d.Do(context.Background(), &Run{}); o.Verdict != m.want {
					t.Errorf("%s/%s: %v (%s), want %v", id, m.name, o.Verdict, o.Reason, m.want)
				}
			}
		}
	}
}

func TestRetreatEffectNotSeenFails(t *testing.T) {
	i := RetreatInfo{Adjacent: 1, Decision: "chicken", Acted: true, LifeFloor: 5, TicksToEffect: 10}
	for _, d := range RT(&retreatStub{info: i}) {
		if d.ID != "RT.leave" {
			continue
		}
		if o := d.Do(context.Background(), &Run{}); o.Verdict != Fail {
			t.Fatalf("%v: a left game that was not seen must fail", o.Verdict)
		}
	}
}

func TestRetreatSpecsAreData(t *testing.T) {
	st := &retreatStub{info: RetreatInfo{Adjacent: 1}}
	for _, d := range RT(st) {
		d.Do(context.Background(), &Run{})
	}
	if len(st.got) != 3 || st.got[0].Variant != "drink" || !st.got[0].Belt || !st.got[1].Scroll || st.got[2].Belt || st.got[2].Scroll {
		t.Fatalf("%+v", st.got)
	}
}

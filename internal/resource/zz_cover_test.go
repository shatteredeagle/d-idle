package resource

import "testing"

func TestCoverAbortNeverPortalsWithoutAScroll(t *testing.T) {
	tu := DefaultTuning()
	for life := 1; life <= 60; life++ {
		for _, belt := range []Belt{{}, {HP: 3}, {MP: 2}, {RV: 1}} {
			for _, tp := range []bool{false, true} {
				for _, town := range []bool{false, true} {
					for _, sk := range [][]Skill{{bolt}, {bolt, staff}, nil} {
						in := with(func(i *Input) {
							i.Hostiles, i.Life, i.Belt, i.HasTP, i.NearTown, i.Skills = 3, life, belt, tp, town, sk
							i.Mana = 0
							i.MercKnown, i.MercLifePct = true, 0
						})
						d := AbortFight(in, tu)
						if d.Kind == PortalOut && !tp {
							t.Fatalf("portal without a scroll: %+v -> %v", in, d)
						}
						if d.Kind == RetreatOut && !town {
							t.Fatalf("retreat toward a town that is not near: %+v", in)
						}
						if d.Reason == "" {
							t.Fatalf("decision without a reason: %+v", in)
						}
					}
				}
			}
		}
	}
}

func TestCoverEscapeLadder(t *testing.T) {
	cases := []struct {
		in   Input
		want AbortKind
	}{
		{Input{HasTP: true}, PortalOut},
		{Input{NearTown: true}, RetreatOut},
		{Input{}, LeaveGame},
	}
	for _, c := range cases {
		if got := escape(c.in, "x"); got.Kind != c.want {
			t.Errorf("%+v: %v want %v", c.in, got.Kind, c.want)
		}
	}
}

func TestCoverAbortNoopInTownOrDead(t *testing.T) {
	tu := DefaultTuning()
	for _, in := range []Input{
		with(func(i *Input) { i.InTown = true; i.Hostiles = 5; i.Life = 1 }),
		with(func(i *Input) { i.Hostiles = 5; i.Life = 0 }),
	} {
		if d := AbortFight(in, tu); d.Kind != Continue {
			t.Fatalf("%+v: %v", in, d)
		}
	}
}

func TestCoverKindStrings(t *testing.T) {
	for k := Continue; k <= Flee; k++ {
		if k.String() == "" {
			t.Fatal(int(k))
		}
	}
}

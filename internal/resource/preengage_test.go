package resource

import "testing"

func TestPreEngageTrashPackNeedsOnlyCastBudget(t *testing.T) {
	tn := DefaultTuning()
	in := Input{Life: 100, MaxLife: 100, Mana: 12, MaxMana: 60, PackSeen: 1, Skills: []Skill{bolt}}
	if d := PreEngage(in, tn); d.Action != Proceed {
		t.Fatalf("one rat with mana 12 (budget 9): %v %s", d.Action, d.Reason)
	}
	in.PackSeen = 3
	if d := PreEngage(in, tn); d.Action == Proceed {
		t.Fatalf("pack of 3 must still rest to the floor")
	}
	in.PackSeen, in.PackDanger = 1, 1
	if d := PreEngage(in, tn); d.Action == Proceed {
		t.Fatalf("dangerous pack must still rest to the floor")
	}
}

func TestPreEngagePoorSkipsDangerAbovePoverty(t *testing.T) {
	tn := DefaultTuning()
	in := Input{Life: 100, MaxLife: 100, Mana: 60, MaxMana: 60, PackSeen: 3, PackDanger: 1, Poor: true, MaxPackDanger: 0}
	if d := PreEngage(in, tn); d.Action != Avoid {
		t.Fatalf("poor, no potions, champion pack: %v %s", d.Action, d.Reason)
	}
	in.PackDanger = 0
	if d := PreEngage(in, tn); d.Action == Avoid {
		t.Fatalf("trash pack must be engaged: %s", d.Reason)
	}
	in.PackDanger, in.MaxPackDanger = 1, 1
	if d := PreEngage(in, tn); d.Action == Avoid {
		t.Fatalf("danger within the limit: %s", d.Reason)
	}
	in.MaxPackDanger, in.Belt.HP = 0, 2
	if d := PreEngage(in, tn); d.Action == Avoid {
		t.Fatalf("with a healing potion the limit does not apply: %s", d.Reason)
	}
	in.Belt.HP, in.Poor = 0, false
	if d := PreEngage(in, tn); d.Action == Avoid {
		t.Fatalf("not poor: no limit: %s", d.Reason)
	}
}

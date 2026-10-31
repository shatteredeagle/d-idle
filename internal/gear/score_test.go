package gear

import "testing"

func TestResistValueCap(t *testing.T) {
	needTables(t)
	// Normal: cap 75. 60 + 20 -> 15 useful, 5 over.
	got := ResistValue(60, 20, Normal, 1)
	if want := 15 + 5*overflow; got != want {
		t.Errorf("normal %v want %v", got, want)
	}
	// Hell penalty -100: the raw cap is 175, the same 20 points are all useful.
	if got := ResistValue(60, 20, Hell, 1); got != 20 {
		t.Errorf("hell %v", got)
	}
	// Already capped: only overflow counts.
	if got := ResistValue(80, 10, Normal, 2); got != 10*2*overflow {
		t.Errorf("capped %v", got)
	}
	if got := ResistValue(0, -10, Normal, 1); got != -10 {
		t.Errorf("negative %v", got)
	}
}

func TestFCRValueBreakpoints(t *testing.T) {
	needTables(t)
	// Sorceress 10 + 10 crosses 20; 10 + 5 does not.
	if got := FCRValue("Sorceress", 10, 10, 15, 0.3); got != 15+3 {
		t.Errorf("cross %v", got)
	}
	if got := FCRValue("sor", 10, 5, 15, 0.3); got != 1.5 {
		t.Errorf("no cross %v", got)
	}
	if got := FCRValue("sor", 0, 20, 15, 0); got != 30 {
		t.Errorf("two breakpoints %v", got)
	}
	if FCRValue("xyz", 0, 10, 15, 0) != 0 {
		t.Error("unknown class")
	}
}

func TestEtherealAndRuneword(t *testing.T) {
	needTables(t)
	if EtherealFactor() >= 1.25 {
		t.Error("default must not reward ethereal")
	}
	EtherealRepairs = true
	defer func() { EtherealRepairs = false }()
	if EtherealFactor() != 1.25 {
		t.Error("repairable ethereal")
	}
	if RunewordScore("Spirit") <= RunewordScore("Stealth") || RunewordScore("nope") != 0 {
		t.Error("runeword scores")
	}
}

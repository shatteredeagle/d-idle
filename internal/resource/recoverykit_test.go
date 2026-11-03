package resource

import "testing"

func TestRecoveryKitTrip(t *testing.T) {
	// Builds differ in tuning, not in code: the default price and a pricier one (a build whose kit is bought at a dearer vendor tier).
	for _, tc := range []struct {
		name  string
		price int
		in    RecoveryKitInput
		want  bool
	}{
		{"default below price", 30, RecoveryKitInput{Recovering: true, Gold: 29, WayHome: true}, false},
		{"default at price", 30, RecoveryKitInput{Recovering: true, Gold: 30, WayHome: true}, true},
		{"gate1 seed19 purse", 30, RecoveryKitInput{Recovering: true, Gold: 54, WayHome: true}, true},
		{"has potion", 30, RecoveryKitInput{Recovering: true, Gold: 500, Healing: 1, WayHome: true}, false},
		{"not recovering", 30, RecoveryKitInput{Gold: 500, WayHome: true}, false},
		{"in town (after death in the trip)", 30, RecoveryKitInput{Recovering: true, InTown: true, Gold: 500, WayHome: true}, false},
		{"capped after failing trips", 30, RecoveryKitInput{Recovering: true, Gold: 500, WayHome: true, Capped: true}, false},
		{"no way home", 30, RecoveryKitInput{Recovering: true, Gold: 500}, false},
		{"dearer potion below", 50, RecoveryKitInput{Recovering: true, Gold: 49, WayHome: true}, false},
		{"dearer potion at", 50, RecoveryKitInput{Recovering: true, Gold: 50, WayHome: true}, true},
	} {
		tu := Tuning{RecoveryOn: 1, KitPotionGold: tc.price}
		if got, why := RecoveryKitTrip(tc.in, tu); got != tc.want || got == (why == "") {
			t.Errorf("%s: got %v %q, want %v", tc.name, got, why, tc.want)
		}
	}
	if got, _ := RecoveryKitTrip(RecoveryKitInput{Recovering: true, Gold: 99, WayHome: true}, Tuning{KitPotionGold: 30}); got {
		t.Error("recovery off must not trip")
	}
}

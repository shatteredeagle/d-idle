package resource

import (
	"testing"
	"time"
)

func TestRecoveryCheck(t *testing.T) {
	tu := DefaultTuning()
	t0 := time.Unix(1000, 0)
	in := RecoveryInput{Gold: 37, Shortfall: 250, SellMin: 2, Now: t0}
	if v := RecoveryCheck(in, tu); !v.Start {
		t.Fatalf("broke with a kit shortfall must start: %+v", v)
	}
	in.Gold = 600
	if v := RecoveryCheck(in, tu); v.On {
		t.Fatalf("gold above the kit gold: %+v", v)
	}
	in.Gold, in.Shortfall = 300, 30
	if v := RecoveryCheck(in, tu); v.On {
		t.Fatalf("affordable kit: %+v", v)
	}
	in = RecoveryInput{Gold: 37, Shortfall: 250, Now: t0.Add(5 * time.Minute), LastEnd: t0}
	if v := RecoveryCheck(in, tu); v.On {
		t.Fatalf("cooldown: %+v", v)
	}
	in.LowKit = true
	if v := RecoveryCheck(in, tu); !v.Start {
		t.Fatalf("low kit overrides the cooldown: %+v", v)
	}
	in.Shortfall, in.Gold = 30, 40
	if v := RecoveryCheck(in, tu); v.On {
		t.Fatalf("low kit but affordable: %+v", v)
	}
	in.LowKit, in.Shortfall, in.Gold = false, 250, 37
	run := RecoveryInput{Was: true, Gold: 950, Sellable: 3, SellMin: 2, Since: t0, Now: t0.Add(10 * time.Minute)}
	if v := RecoveryCheck(run, tu); !v.On || v.End {
		t.Fatalf("loot still unsold: %+v", v)
	}
	run.Sellable = 0
	if v := RecoveryCheck(run, tu); !v.End {
		t.Fatalf("target reached: %+v", v)
	}
	run.Gold, run.Now = 100, t0.Add(46*time.Minute)
	if v := RecoveryCheck(run, tu); !v.End {
		t.Fatalf("time cap: %+v", v)
	}
	tu.RecoveryOn = 0
	if v := RecoveryCheck(in, tu); v.On {
		t.Fatalf("off: %+v", v)
	}
}

func TestRecoveryForceBypassesCooldown(t *testing.T) {
	tu := DefaultTuning()
	t0 := time.Unix(1000, 0)
	in := RecoveryInput{Gold: 0, Shortfall: 250, Now: t0.Add(5 * time.Minute), LastEnd: t0, Force: true}
	if v := RecoveryCheck(in, tu); !v.Start {
		t.Fatalf("forced start: %+v", v)
	}
	in.Gold = 600
	if v := RecoveryCheck(in, tu); v.On {
		t.Fatalf("force must not start with the kit paid: %+v", v)
	}
}

func TestKitLow(t *testing.T) {
	tu := DefaultTuning()
	tu.RecoveryPotionFloor = 2
	if !KitLow(1, tu) || KitLow(2, tu) {
		t.Fatal("floor 2: 1 is low, 2 is not")
	}
	tu.RecoveryPotionFloor = 0
	if KitLow(0, tu) {
		t.Fatal("floor 0 disables the override")
	}
}

// The tester realtime hour (TASK-138): a clvl 1 to 4 hero earns a few gold per kill, so the 500 bar is out of reach and
// the recovery held the hero in Blood Moor for good (a low kit restarted it at once after the time cap).
func TestRecoveryStallsAtLowIncome(t *testing.T) {
	t0 := time.Unix(1000, 0)
	for name, edit := range map[string]func(*Tuning){
		"default":     func(*Tuning) {},
		"slow-income": func(tu *Tuning) { tu.RecoveryStallMin, tu.RecoveryStallGold = 30, 200 },
	} {
		tu := DefaultTuning()
		edit(&tu)
		min := time.Duration(tu.RecoveryStallMin) * time.Minute
		run := RecoveryInput{Was: true, Gold: 28, StartGold: 0, Since: t0, Now: t0.Add(min - time.Minute)}
		if v := RecoveryCheck(run, tu); !v.On || v.End {
			t.Fatalf("%s: before the check minute: %+v", name, v)
		}
		run.Now = t0.Add(min)
		if v := RecoveryCheck(run, tu); !v.End || v.Reason != StallReason {
			t.Fatalf("%s: low income must end as stalled: %+v", name, v)
		}
		run.Gold = tu.RecoveryStallGold
		if v := RecoveryCheck(run, tu); v.End {
			t.Fatalf("%s: enough gold gained goes on: %+v", name, v)
		}
		// after the stall a low kit does not restart it inside the hold, a death streak does, and the hold ends
		in := RecoveryInput{Gold: 28, Shortfall: 250, LowKit: true, Stalled: true, LastEnd: t0, Now: t0.Add(30 * time.Minute)}
		if v := RecoveryCheck(in, tu); v.On {
			t.Fatalf("%s: stalled hold: %+v", name, v)
		}
		in.Force = true
		if v := RecoveryCheck(in, tu); !v.Start {
			t.Fatalf("%s: death streak overrides the hold: %+v", name, v)
		}
		in.Force, in.Now = false, t0.Add(time.Duration(tu.RecoveryStallHoldMin+1)*time.Minute)
		if v := RecoveryCheck(in, tu); !v.Start {
			t.Fatalf("%s: hold over: %+v", name, v)
		}
		tu.RecoveryStallMin = 0
		run.Gold = 0
		if v := RecoveryCheck(run, tu); v.End {
			t.Fatalf("%s: stall rule off: %+v", name, v)
		}
	}
}

// TASK-183: a running recovery ends when the gold covers the missing kit (recoveryEndOnKit), for a caster kit (mana
// potions asked) and a melee kit (none). Off keeps the old end rules, so the gate defaults run as before.
func TestRecoveryEndsWhenKitAffordable(t *testing.T) {
	t0 := time.Unix(1000, 0)
	for _, c := range []struct {
		name      string
		mpFirst   int
		shortfall func(gold int) int
	}{
		{"caster", 2, func(int) int { return 190 }},
		{"melee", 0, func(int) int { return 130 }},
	} {
		tu := DefaultTuning()
		tu.MPFirst = c.mpFirst
		run := RecoveryInput{Was: true, Gold: 150, Shortfall: c.shortfall(150), SellMin: 2, Since: t0, Now: t0.Add(6 * time.Minute)}
		tu.RecoveryEndOnKit = 1
		if v := RecoveryCheck(run, tu); c.shortfall(0) == 190 && (!v.On || v.End) {
			t.Fatalf("%s: 150 gold does not cover a 190 kit: %+v", c.name, v)
		}
		run.Gold = 200
		run.Shortfall = c.shortfall(200)
		if v := RecoveryCheck(run, tu); !v.End || v.Reason != KitReason {
			t.Fatalf("%s: affordable kit must end the recovery: %+v", c.name, v)
		}
		tu.RecoveryEndOnKit = 0
		if v := RecoveryCheck(run, tu); !v.On || v.End {
			t.Fatalf("%s: with the knob off the recovery goes on (old rules): %+v", c.name, v)
		}
	}
}

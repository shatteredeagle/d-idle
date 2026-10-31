package bot

import "testing"

func TestPovertySellWorth(t *testing.T) {
	for _, c := range []struct {
		sell int
		tp   bool
		want bool
	}{{1, true, false}, {4, true, false}, {6, true, true}, {3, false, false}, {4, false, true}} {
		if got := povertySellWorth(c.sell, c.tp); got != c.want {
			t.Errorf("sell %d tp %v: got %v want %v", c.sell, c.tp, got, c.want)
		}
	}
}

func TestPovertySellMinKnob(t *testing.T) {
	for _, c := range []struct {
		sell, min int
		tp, want  bool
	}{{4, 0, false, true}, {4, 10, false, false}, {9, 10, false, false}, {10, 10, false, true}, {10, 10, true, true}, {7, 10, true, false}} {
		if got := povertySellOK(c.sell, c.tp, c.min); got != c.want {
			t.Errorf("sell %d min %d tp %v: got %v want %v", c.sell, c.min, c.tp, got, c.want)
		}
	}
}

// The low-gold sell door at gold 120 (under GearMinGold) with a scroll and a few junk items.
func TestLowGoldSellOK(t *testing.T) {
	for _, c := range []struct {
		sell, minItems, min int
		tp, want            bool
	}{
		{4, 3, 0, true, true},   // knob off: old rule, SellMinItems only
		{5, 3, 0, true, true},   // knob off
		{4, 3, 10, true, false}, // knob on: below the knob
		{5, 3, 10, true, false},
		{10, 3, 10, true, true},
		{4, 3, 4, true, false}, // knob on and a scroll held: also under the built-in poor-character minimum
		{6, 3, 4, true, true},
		{2, 3, 0, true, false},
	} {
		if got := lowGoldSellOK(c.sell, c.tp, c.minItems, c.min); got != c.want {
			t.Errorf("sell %d minItems %d min %d tp %v: got %v want %v", c.sell, c.minItems, c.min, c.tp, got, c.want)
		}
	}
}

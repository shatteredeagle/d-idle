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

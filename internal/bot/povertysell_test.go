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

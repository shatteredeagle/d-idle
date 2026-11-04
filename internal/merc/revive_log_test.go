package merc

import (
	"strconv"
	"strings"
	"testing"
)

// The skip line and the wait line carry the real bar (price x mult + kit reserve), so a stall is not a "near miss".
func TestReviveSkipLogCarriesNeed(t *testing.T) {
	dead := MercState{Known: true, Owned: true, Level: 10}
	for _, kit := range []int{310, 620} {
		c := ctx()
		c.Kit = kit
		c.Gold = 100
		d, price := ShouldRevive(dead, c, 0)
		if d.Do {
			t.Fatal("poor purse revived")
		}
		need := ReviveNeed(c, price)
		line := LogReviveSkipped(d.Reason, price, c.Gold, need)
		if !strings.Contains(line, "need="+strconv.Itoa(price+kit)) {
			t.Errorf("kit %d: %q lacks need=%d", kit, line, price+kit)
		}
		if w := LogGoldWait(need, c.Gold); w != "merc: gold wait need="+strconv.Itoa(need)+" gold=100" {
			t.Errorf("wait line %q", w)
		}
	}
}

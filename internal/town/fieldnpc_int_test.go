package town

import "testing"

func TestFieldNPCTable(t *testing.T) {
	for _, c := range []struct {
		classes []int
		area    int
		want    bool
	}{
		{[]int{251}, 73, true},      // Tyrael in Duriel's Lair
		{[]int{251}, 3, false},      // the same NPC in the field elsewhere
		{[]int{154}, 73, false},     // a town vendor never opens in the lair
		{[]int{154, 251}, 73, true}, // any listed class
		{nil, 73, false},
	} {
		if got := fieldNPC(c.classes, c.area); got != c.want {
			t.Errorf("fieldNPC(%v, %d) = %v, want %v", c.classes, c.area, got, c.want)
		}
	}
}

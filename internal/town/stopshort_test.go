package town

import "testing"

func TestStopShort(t *testing.T) {
	if p := stopShort(0, 0, 40, 0, 12); p.X != 28 || p.Y != 0 {
		t.Errorf("got %+v", p)
	}
	if p := stopShort(5, 5, 8, 5, 12); p.X != 5 || p.Y != 5 {
		t.Errorf("close target must not move: %+v", p)
	}
}

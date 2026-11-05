package nav

import "testing"

func TestTakePeakRungResets(t *testing.T) {
	var n Navigator
	n.peakRung.Store(5)
	if n.TakePeakRung() != 5 || n.TakePeakRung() != 0 {
		t.Fatal("TakePeakRung must return the peak once and reset it")
	}
}

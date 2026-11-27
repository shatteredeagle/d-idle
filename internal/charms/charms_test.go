package charms

import "testing"

func TestLookup(t *testing.T) {
	for code, cells := range map[string]int{"cm1": 1, "cm2": 2, "cm3": 3, "scha": 1, "lcha": 3, "CM2": 2} {
		b, ok := Lookup(code)
		if !ok || b.Cells() != cells {
			t.Errorf("%s: %+v %v, want %d cells", code, b, ok, cells)
		}
	}
	if Is("rin") || Is("") {
		t.Error("a ring is not a charm")
	}
}

func TestLegalQuality(t *testing.T) {
	for q, want := range map[string]bool{"magic": true, "unique": true, "rare": false, "set": false, "crafted": false} {
		if LegalQuality(q) != want {
			t.Errorf("%s: want %v", q, want)
		}
	}
}

func TestLegalQualityNum(t *testing.T) {
	for q, want := range map[int]bool{4: true, 7: true, 5: false, 6: false, 8: false, 2: false} {
		if LegalQualityNum(q) != want {
			t.Errorf("%d: want %v", q, want)
		}
	}
}

package town

import (
	"strings"
	"testing"
)

// The money rules of the player's shopping errand (TASK-298), without a game: planIDBuy and idBuySummary are pure.
func TestPlanIDBuy(t *testing.T) {
	const scroll, tome = 50, 450
	cases := []struct {
		name string
		in   idBuyIn
		want idBuyPlan
	}{
		{"fill the tome", idBuyIn{Gold: 2000, Reserve: 300, HaveTome: true, TomeQty: 8, TomeCap: 20, ScrollPrice: scroll, TomePrice: tome},
			idBuyPlan{Scrolls: 12, Wanted: 12, Cost: 600}},
		{"explicit count capped by the tome room", idBuyIn{Gold: 2000, Reserve: 300, HaveTome: true, TomeQty: 15, TomeCap: 20, Want: 12, ScrollPrice: scroll, TomePrice: tome},
			idBuyPlan{Scrolls: 5, Wanted: 5, Cost: 250}},
		{"no tome, gold for a tome: buy it and fill it", idBuyIn{Gold: 2000, Reserve: 300, TomeCap: 20, ScrollPrice: scroll, TomePrice: tome},
			idBuyPlan{Tome: true, Scrolls: 20, Wanted: 20, Cost: 450 + 1000}},
		{"no tome, no gold for one: a few scrolls", idBuyIn{Gold: 500, Reserve: 300, TomeCap: 20, ScrollPrice: scroll, TomePrice: tome},
			idBuyPlan{Scrolls: 4, Wanted: 5, Short: true, Cost: 200}},
		{"explicit count without a tome buys loose scrolls", idBuyIn{Gold: 2000, Reserve: 300, TomeCap: 20, Want: 3, ScrollPrice: scroll, TomePrice: tome},
			idBuyPlan{Scrolls: 3, Wanted: 3, Cost: 150}},
		{"short of gold for the rest", idBuyIn{Gold: 500, Reserve: 300, HaveTome: true, TomeQty: 0, TomeCap: 20, ScrollPrice: scroll, TomePrice: tome},
			idBuyPlan{Scrolls: 4, Wanted: 20, Short: true, Cost: 200}},
		{"at or below the reserve nothing is bought", idBuyIn{Gold: 300, Reserve: 300, HaveTome: true, TomeQty: 0, TomeCap: 20, ScrollPrice: scroll},
			idBuyPlan{Wanted: 20, Short: true}},
		{"below the reserve", idBuyIn{Gold: 100, Reserve: 300, TomeCap: 20, ScrollPrice: scroll},
			idBuyPlan{Wanted: 5, Short: true}},
		{"the vendor sells no scroll", idBuyIn{Gold: 2000, Reserve: 300, HaveTome: true, TomeQty: 0, TomeCap: 20},
			idBuyPlan{Wanted: 20, Short: true}},
	}
	for _, c := range cases {
		if got := planIDBuy(c.in); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestIDBuySummary(t *testing.T) {
	s, err := idBuySummary(12, false, 600, 0, false)
	if err != nil || s != "bought 12 for 600 gold" {
		t.Errorf("full: %q %v", s, err)
	}
	s, err = idBuySummary(12, false, 600, 8, true)
	if err != nil || !strings.Contains(s, "bought 12") || !strings.Contains(s, "short of gold for the other 8") {
		t.Errorf("short: %q %v", s, err)
	}
	s, err = idBuySummary(5, true, 1050, 0, false)
	if err != nil || !strings.Contains(s, "Tome of Identify and 5") {
		t.Errorf("tome: %q %v", s, err)
	}
	if _, err = idBuySummary(0, false, 0, 20, true); err == nil || !strings.Contains(err.Error(), "short of gold") {
		t.Errorf("nothing bought and short must fail with the reason: %v", err)
	}
}

// Cain's dialog has Talk, Identify, Cancel for every class of his; the other classes keep their own menus.
func TestCainMenuLines(t *testing.T) {
	for _, c := range cain {
		idx, n, ok := menuLine(c, EntryIdentify)
		if !ok || idx != 1 || n != 3 {
			t.Errorf("class %d: identify line %d of %d (%v), want 1 of 3", c, idx, n, ok)
		}
		if i, n, _ := menuLine(c, EntryTalk); i != 0 || n != 3 {
			t.Errorf("class %d: talk line moved to %d of %d", c, i, n)
		}
	}
	if _, _, ok := menuLine(148, EntryIdentify); ok {
		t.Error("Akara has no Identify line")
	}
}

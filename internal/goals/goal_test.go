package goals

import (
	"strings"
	"testing"
	"time"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		g       Goal
		wantErr string
		title   string
	}{
		{"level ok", Goal{Type: ReachLevel, Params: Params{Level: 30}}, "", "Reach level 30"},
		{"level 0", Goal{Type: ReachLevel}, "level", ""},
		{"level 100", Goal{Type: ReachLevel, Params: Params{Level: 100}}, "level", ""},
		{"farm needs area", Goal{Type: FarmArea}, "area", ""},
		{"farm endless", Goal{Type: FarmArea, Params: Params{Area: " Cold Plains "}}, "", "Farm Cold Plains"},
		{"farm limits", Goal{Type: FarmArea, Params: Params{Area: "Pit", UntilKills: 500, UntilMins: 60}}, "", "Farm Pit until 500 kills or 60 min"},
		{"boss default count", Goal{Type: KillBoss, Params: Params{Boss: "Mephisto"}}, "", "Kill Mephisto"},
		{"boss count", Goal{Type: KillBoss, Params: Params{Boss: "Baal", Count: 10}}, "", "Kill Baal x10"},
		{"boss missing", Goal{Type: KillBoss}, "boss", ""},
		{"collect empty filter", Goal{Type: CollectItem}, "filter", ""},
		{"collect quality any is empty", Goal{Type: CollectItem, Params: Params{Item: ItemFilter{Quality: "any"}}}, "filter", ""},
		{"collect ok", Goal{Type: CollectItem, Params: Params{Count: 3, Item: ItemFilter{Quality: "Unique"}}}, "", "Collect 3 x unique"},
		{"idle minutes", Goal{Type: Idle, Params: Params{Minutes: 20}}, "", "Rest for 20 min"},
		{"idle nothing", Goal{Type: Idle}, "idle", ""},
		{"idle window", Goal{Type: Idle, Schedule: []Window{{Start: "02:00", End: "06:00"}}}, "", "Rest during scheduled windows"},
		{"bad window", Goal{Type: Idle, Schedule: []Window{{Start: "2:00", End: "06:00"}}}, "bad time", ""},
		{"empty window", Goal{Type: Idle, Schedule: []Window{{Start: "02:00", End: "02:00"}}}, "empty", ""},
		{"bad day", Goal{Type: Idle, Schedule: []Window{{Days: []int{7}, Start: "01:00", End: "02:00"}}}, "weekday", ""},
		{"bad difficulty", Goal{Type: ReachLevel, Params: Params{Level: 5, Difficulty: "easy"}}, "difficulty", ""},
		{"bad priority", Goal{Type: ReachLevel, Priority: 11, Params: Params{Level: 5}}, "priority", ""},
		{"negative", Goal{Type: FarmArea, Params: Params{Area: "x", UntilKills: -1}}, "negative", ""},
		{"unknown type", Goal{Type: "dance"}, "unknown", ""},
		{"custom title kept", Goal{Type: ReachLevel, Title: "  Go go  ", Params: Params{Level: 5}}, "", "Go go"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.g.Normalize()
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.g.Title != tc.title {
				t.Errorf("title = %q, want %q", tc.g.Title, tc.title)
			}
		})
	}
}

func TestWindowContains(t *testing.T) {
	// 2026-10-28 is a Wednesday (3).
	at := func(day, h, m int) time.Time { return time.Date(2026, 10, 25+day, h, m, 0, 0, time.UTC) } // day 0 = Sunday
	tests := []struct {
		name string
		w    Window
		t    time.Time
		want bool
	}{
		{"inside", Window{Start: "09:00", End: "17:00"}, at(3, 12, 0), true},
		{"start inclusive", Window{Start: "09:00", End: "17:00"}, at(3, 9, 0), true},
		{"end exclusive", Window{Start: "09:00", End: "17:00"}, at(3, 17, 0), false},
		{"before", Window{Start: "09:00", End: "17:00"}, at(3, 8, 59), false},
		{"day filter hit", Window{Days: []int{3}, Start: "09:00", End: "17:00"}, at(3, 10, 0), true},
		{"day filter miss", Window{Days: []int{4}, Start: "09:00", End: "17:00"}, at(3, 10, 0), false},
		{"wrap evening", Window{Start: "22:00", End: "06:00"}, at(3, 23, 0), true},
		{"wrap morning", Window{Start: "22:00", End: "06:00"}, at(3, 5, 59), true},
		{"wrap gap", Window{Start: "22:00", End: "06:00"}, at(3, 12, 0), false},
		{"wrap morning belongs to previous day (hit)", Window{Days: []int{2}, Start: "22:00", End: "06:00"}, at(3, 3, 0), true},
		{"wrap morning belongs to previous day (miss)", Window{Days: []int{3}, Start: "22:00", End: "06:00"}, at(3, 3, 0), false},
		{"wrap evening day filter", Window{Days: []int{3}, Start: "22:00", End: "06:00"}, at(3, 23, 0), true},
		{"sunday wrap from saturday", Window{Days: []int{6}, Start: "22:00", End: "06:00"}, at(0, 1, 0), true},
	}
	for _, tc := range tests {
		if got := tc.w.contains(tc.t); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestItemFilter(t *testing.T) {
	it := Item{Name: "Tir Rune", Type: "rune", Quality: "normal"}
	tests := []struct {
		f    ItemFilter
		want bool
	}{
		{ItemFilter{Name: "tir"}, true}, {ItemFilter{Type: "RUNE"}, true}, {ItemFilter{Quality: "unique"}, false},
		{ItemFilter{Name: "tir", Quality: "normal", Type: "rune"}, true}, {ItemFilter{Name: "eld"}, false}, {ItemFilter{Quality: "any"}, true},
	}
	for _, tc := range tests {
		if got := tc.f.Match(it); got != tc.want {
			t.Errorf("%+v: got %v", tc.f, got)
		}
	}
}

func TestNormalTemplatesPinDifficulty(t *testing.T) {
	for _, tpl := range []string{CampaignNormalAct1, CampaignNormalAll} {
		if d := templateDifficulty(tpl); d != "normal" {
			t.Errorf("%s: %q", tpl, d)
		}
	}
}

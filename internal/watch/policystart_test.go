package watch

import (
	"strings"
	"testing"

	"d2lod-bot/internal/plandata"
)

// Card classchores: the sheet starts on the class's rows (internal/plandata chores-defaults.json).
func TestClassStartAppliesOnceAndIsNotAChange(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newPolRig(t, class, true)
		r.sheet.starts = func(c string) (map[string]string, error) {
			if c == "paladin" {
				return map[string]string{"potionmix": "h3m1", "repair": "default"}, nil
			}
			return map[string]string{"potionmix": "default"}, nil
		}
		v, err := r.sheet.View()
		if err != nil {
			t.Fatal(err)
		}
		want := "default"
		if class == "paladin" {
			want = "h3m1"
		}
		for _, row := range v.Rows {
			if row.ID == "potionmix" && row.Current != want {
				t.Errorf("%s: potionmix starts on %q, want %q", class, row.Current, want)
			}
			if row.ID == "repair" && row.Current != "default" {
				t.Errorf("%s: repair %q", class, row.Current)
			}
		}
		if class == "paladin" && (len(r.calls) != 1 || r.calls[0]["resource.beltPlan"] == nil || *r.calls[0]["resource.beltPlan"] != 4) {
			t.Errorf("start did not set the belt plan knob: %v", r.calls)
		}
		if len(r.sheet.Changes()) != 0 {
			t.Errorf("%s: a start is not a recorded change: %v", class, r.sheet.Changes())
		}
		r.sheet.View() // once only
		if class == "paladin" && len(r.calls) != 1 {
			t.Errorf("start applied twice: %d knob calls", len(r.calls))
		}
	}
}

func TestClassStartLeavesAPlayerChoice(t *testing.T) {
	r := newPolRig(t, "paladin", true)
	r.sheet.starts = func(string) (map[string]string, error) { return map[string]string{"potionmix": "h3m1"}, nil }
	r.store.SetPolicy("potionmix", "balanced", nil)
	if _, err := r.sheet.View(); err != nil {
		t.Fatal(err)
	}
	if got := r.store.PolicyOption("potionmix"); got != "balanced" {
		t.Errorf("player's choice replaced: %q", got)
	}
}

func TestClassStartErrorsNameTheCause(t *testing.T) {
	for _, c := range []struct {
		name string
		m    map[string]string
		want string
	}{
		{"row", map[string]string{"nonesuch": "x"}, `"nonesuch"`},
		{"option", map[string]string{"repair": "weekly"}, `"weekly"`},
	} {
		r := newPolRig(t, "sorceress", true)
		r.sheet.starts = func(string) (map[string]string, error) { return c.m, nil }
		if _, err := r.sheet.View(); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}

// The shipped data names rows and options the shipped table has, for every class that has chore data.
func TestShippedStartsMatchTheTable(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newPolRig(t, class, true)
		if _, err := r.sheet.View(); err != nil {
			t.Errorf("%s: %v", class, err)
		}
		ch, err := plandata.Chores(class)
		if err != nil {
			t.Fatal(err)
		}
		for row, opt := range ch.Start {
			rw, err := r.tbl.Row(row)
			if err != nil {
				t.Errorf("%s: %v", class, err)
				continue
			}
			if _, err := rw.Option(opt); err != nil {
				t.Errorf("%s: %v", class, err)
			}
		}
	}
	for _, class := range []string{"amazon", "assassin", "barbarian", "druid", "necromancer"} {
		ch, err := plandata.Chores(class)
		if err != nil {
			t.Fatal(err)
		}
		r := newPolRig(t, "sorceress", true)
		for row, opt := range ch.Start {
			rw, err := r.tbl.Row(row)
			if err == nil {
				_, err = rw.Option(opt)
			}
			if err != nil {
				t.Errorf("%s: %v", class, err)
			}
		}
	}
}

package watch

import (
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/decide"
)

// sampleFacts holds every fact name a headline may use, with a value as a call site gives it.
var sampleFacts = decide.Situation{"life": "28", "area": "Cold Plains", "potions": "1", "hostiles": "2", "pack": "cursing champion pack",
	"dest": "the Forgotten Tower", "floor": "14", "herolevel": "9", "retreats": "2", "packkind": "cursing champion pack", "monster": "Bishibosh",
	"distance": "14", "tripgold": "45", "have": "2", "want": "6", "kit": "THIN", "item": "a rune", "packused": "85", "gold": "300",
	"durability": "22", "cost": "120", "run": "quest", "deaths": "3", "boss": "Andariel", "merc": "dead"}

// TestEveryPointHasACard: every point of the shipped catalogue has a headline and a generic line, a plain phrase for
// each of its answers (and none for answers it does not have), and every placeholder resolves to a known fact.
func TestEveryPointHasACard(t *testing.T) {
	cat, err := decide.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	jargon := []string{"drinklife", "packdanger", "lifeengage", "champbonus", "areaclass", "toohard", "levelgap", "notsure", "t0.", "t1.", "t2."}
	for _, p := range cat.Points {
		c, ok := cardTexts[p.ID]
		if !ok {
			t.Errorf("point %s has no card text", p.ID)
			continue
		}
		if c.Headline == "" || c.Generic == "" {
			t.Errorf("%s: headline and generic line are required", p.ID)
		}
		for _, a := range p.Answers {
			if c.Does[a.ID] == "" {
				t.Errorf("%s: answer %s has no plain phrase", p.ID, a.ID)
			}
		}
		if len(c.Does) != len(p.Answers) {
			t.Errorf("%s: %d phrases for %d answers", p.ID, len(c.Does), len(p.Answers))
		}
		for _, n := range placeholders(c.Headline) {
			if _, ok := sampleFacts[n]; !ok {
				t.Errorf("%s: placeholder {%s} is not a known fact", p.ID, n)
			}
		}
		got, ok := renderHeadline(c.Headline, sampleFacts)
		if !ok || strings.ContainsAny(got, "{}#|") {
			t.Errorf("%s: headline does not resolve: %q", p.ID, got)
		}
		if _, ok := renderHeadline(c.Headline, nil); ok && len(placeholders(c.Headline)) > 0 {
			t.Errorf("%s: a headline with no facts must fall back to the generic line", p.ID)
		}
		for _, s := range append([]string{got, c.Generic}, valuesOf(c.Does)...) {
			for _, j := range jargon {
				if strings.Contains(s, j) {
					t.Errorf("%s: jargon %q in %q", p.ID, j, s)
				}
			}
		}
	}
	for id := range cardTexts {
		if _, err := cat.Point(id); err != nil {
			t.Errorf("card text for unknown point %s", id)
		}
	}
}

func valuesOf(m map[string]string) []string {
	var out []string
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func TestRenderHeadlineCounts(t *testing.T) {
	tm := "{n#enemy|enemies} close in {area}"
	for n, want := range map[string]string{"0": "no enemies close in X", "1": "1 enemy close in X", "3": "3 enemies close in X"} {
		if got, ok := renderHeadline(tm, decide.Situation{"n": n, "area": "X"}); !ok || got != want {
			t.Errorf("n=%s: %q %v", n, got, ok)
		}
	}
	if _, ok := renderHeadline(tm, decide.Situation{"n": "x", "area": "X"}); ok {
		t.Error("a non-number count must not render")
	}
	if _, ok := renderHeadline(tm, decide.Situation{"n": "2", "area": ""}); ok {
		t.Error("an empty fact must not render")
	}
}

// TestCardsReadWell renders the nine wired points with the facts their call sites report (run with -v for the text).
func TestCardsReadWell(t *testing.T) {
	r := newRig(t, "sorceress")
	cases := []struct {
		point, want string
		sit         decide.Situation
	}{
		{"t0.retreat.drink", "Life fell to 28% in Cold Plains, with 1 healing potion in the belt and 2 enemies close.", sampleFacts},
		{"t0.unique.fight", "A cursing champion pack led by Bishibosh is 14 squares away in Cold Plains, and life is 28%.", sampleFacts},
		{"t0.area.toohard", "Heading to the Forgotten Tower: the bot wants hero level 14 there and the hero is level 9.", sampleFacts},
		{"t1.restock", "In town after a trip with 45 gold to spend; the belt holds 2 of 6 healing potions.", sampleFacts},
		{"t1.return", "The town chores are done: the belt holds 2 of 6 healing potions and supplies are thin (potions or a portal are missing).", sampleFacts},
		{"t1.town.trigger", "The pack is 85% full and the belt holds 2 of 6 healing potions in Cold Plains.", sampleFacts},
		{"t1.tp.lost", "No town portal scroll is left, and the hero has 300 gold.", sampleFacts},
		{"t1.repair", "In town with gear worn down to 22% and 300 gold: repairs would cost about 120 gold.", sampleFacts},
		{"t2.mercenary", "The mercenary is dead and the hero holds 300 gold in Cold Plains.", sampleFacts},
	}
	for _, c := range cases {
		info := decide.ThoughtInfo{Point: c.point, Key: c.point + "|act=1", Sit: c.sit}
		card := r.sim.Hub.BuildCard(info, "")
		if card.Headline != c.want || !card.Specific {
			t.Errorf("%s: %q", c.point, card.Headline)
		}
		t.Logf("%s\n  %s\n  %s", c.point, card.Headline, card.IfNothing)
	}
}

// TestCardNamesTheEventInTheFeed: the live question of both builds carries the event's values, the plain phrase of every
// answer, the default and the fade time; a faded or answered thought keeps its situation in the journal.
func TestCardNamesTheEventInTheFeed(t *testing.T) {
	for _, b := range builds {
		t.Run(b, func(t *testing.T) {
			r := newRig(t, b)
			r.fight()
			th := r.nextThought()
			if !th.Specific || !strings.Contains(th.Headline, "Life fell to ") || !strings.Contains(th.Headline, "in Blood Moor") {
				t.Errorf("headline: %q", th.Headline)
			}
			if !strings.HasPrefix(th.IfNothing, "If you do nothing: the hero will ") || !strings.Contains(th.IfNothing, "45 s") {
				t.Errorf("if nothing: %q", th.IfNothing)
			}
			if !strings.HasPrefix(th.Rule, "Rule applies when: ") {
				t.Errorf("rule note: %q", th.Rule)
			}
			for _, c := range th.Choices {
				if c.Does == "" {
					t.Errorf("answer %s has no plain phrase", c.ID)
				}
			}
			r.sim.Step(50 * time.Second) // fades
			var sawFade bool
			for _, l := range r.journal("").Lines {
				if l.Kind == "missed" && strings.Contains(l.Text, "When: Life fell to ") {
					sawFade = true
				}
			}
			if !sawFade {
				t.Errorf("no fade line with the situation: %+v", r.journal("").Lines)
			}
		})
	}
}

func TestMissingFactsFallBackToTheGenericLine(t *testing.T) {
	r := newRig(t, "sorceress")
	card := r.sim.Hub.BuildCard(decide.ThoughtInfo{Point: "t1.repair", Key: "t1.repair|act=1", Sit: decide.Situation{"gold": "5"}}, "skip")
	if card.Specific || card.Headline != cardTexts["t1.repair"].Generic || strings.Contains(card.Headline, "{") {
		t.Errorf("%+v", card)
	}
	if !strings.Contains(card.IfNothing, "leave the gear as it is") {
		t.Errorf("default phrase: %q", card.IfNothing)
	}
}

package watch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	questdata "d2lod-bot/data/quests"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
)

type hintRig struct {
	t     *testing.T
	q     *QuestHints
	dir   string
	mode  string
	class string
	level int
	said  []string
	asked []string
	errs  []error
	now   float64
}

func newHintRig(t *testing.T, class string, level int) *hintRig {
	t.Helper()
	r := &hintRig{t: t, dir: t.TempDir(), mode: HintsOn, class: class, level: level}
	r.q = r.open()
	return r
}

// open builds a desk over the rig's state directory (a second call is a restart).
func (r *hintRig) open() *QuestHints {
	r.t.Helper()
	q, err := NewQuestHints(QuestHintsConfig{
		Mode: func() string { return r.mode }, Class: func() string { return r.class }, Level: func() int { return r.level },
		Now: func() float64 { r.now++; return r.now }, Say: func(s string) { r.said = append(r.said, s) },
		Ask:  func(p, quest string) error { r.asked = append(r.asked, p+"/"+quest); return nil },
		Fail: func(err error) { r.errs = append(r.errs, err) }, Dir: r.dir, Profile: "t",
	})
	if err != nil {
		r.t.Fatal(err)
	}
	return q
}

func (r *hintRig) enterTown() {
	r.q.OnGoalEvent(goals.Event{Type: goals.EvAreaChange, Area: "Blood Moor", AreaID: 2})
	r.q.OnGoalEvent(goals.Event{Type: goals.EvAreaChange, Area: "Rogue Encampment", AreaID: 1})
}

func (r *hintRig) visit(area string, id int) {
	r.q.OnGoalEvent(goals.Event{Type: goals.EvAreaChange, Area: area, AreaID: id})
}

func (r *hintRig) started(quest string) {
	r.q.OnGoalEvent(goals.Event{Type: goals.EvQuest, Quest: quest, Stage: 1})
}

func TestShippedHintTableMatchesQuestLog(t *testing.T) {
	ht, err := ParseHintTable(questdata.HintsJSON)
	if err != nil {
		t.Fatal(err)
	}
	qt, err := ParseQuestTable(questdata.JSON)
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]QuestDef{}
	for _, d := range qt.Quests {
		have[d.ID] = d
	}
	for _, h := range ht.Quests {
		d, ok := have[h.ID]
		if !ok {
			t.Errorf("hint quest %q is not in the quest log table", h.ID)
			continue
		}
		if h.Giver != "" && d.Giver != h.Giver && h.ID != "sisters_to_slaughter" {
			t.Errorf("quest %q: hint giver %q, quest log giver %q", h.ID, h.Giver, d.Giver)
		}
	}
	for _, c := range []string{"paladin", "sorceress", "necromancer", "amazon", "assassin", "barbarian", "druid"} {
		if _, err := ht.Family(c); err != nil {
			t.Error(err)
		}
	}
}

func TestHintTableRejects(t *testing.T) {
	base := string(questdata.HintsJSON)
	for name, mut := range map[string]func(string) string{
		"bad family": func(s string) string { return strings.Replace(s, `"family": "melee"`, `"family": "tank"`, 1) },
		"no tag":     func(s string) string { return strings.Replace(s, `"lines": "EXP"`, `"lines": "MAYBE"`, 1) },
		"empty line": func(s string) string {
			return strings.Replace(s, `"Warriv keeps looking east. I think I am ready."`, `""`, 1)
		},
		"missing lvl": func(s string) string { return strings.Replace(s, `"summoner": 3`, `"sommoner": 3`, 1) },
	} {
		if _, err := ParseHintTable([]byte(mut(base))); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	ht, _ := ParseHintTable(questdata.HintsJSON)
	if _, err := ht.Family("bard"); err == nil || !strings.Contains(err.Error(), `"bard"`) {
		t.Errorf("unknown class error must name it: %v", err)
	}
}

// The Den threshold differs by family: Paladin 1 (melee), Sorceress 2 (caster), Necromancer 3 (summoner).
func TestHintThresholdByFamily(t *testing.T) {
	cases := []struct {
		class string
		level int
		kind  string
	}{
		{"paladin", 1, HintGo}, {"sorceress", 1, HintWeak}, {"sorceress", 2, HintGo},
		{"necromancer", 2, HintWeak}, {"necromancer", 3, HintGo},
	}
	for _, c := range cases {
		r := newHintRig(t, c.class, c.level)
		r.enterTown()
		ls := r.q.Lines()
		if len(ls) != 1 || ls[0].Kind != c.kind || ls[0].Quest != "den_of_evil" {
			t.Errorf("%s clvl %d: lines %+v, want one %s for the Den", c.class, c.level, ls, c.kind)
		}
		if len(r.said) != 1 {
			t.Errorf("%s clvl %d: said %d lines", c.class, c.level, len(r.said))
		}
		if c.kind == HintGo && r.q.Errand() != "den_of_evil" {
			t.Errorf("%s: errand %q", c.class, r.q.Errand())
		}
		if c.kind == HintWeak && r.q.Errand() != "" {
			t.Errorf("%s: a weak hint set the errand %q", c.class, r.q.Errand())
		}
	}
}

// A started quest is never hinted; a "go" is said once; an area gate holds Kashya back until the Cold Plains were seen.
func TestHintGoOnceAndAvailability(t *testing.T) {
	r := newHintRig(t, "paladin", 8)
	r.started("den_of_evil")
	r.enterTown()
	if len(r.said) != 0 {
		t.Fatalf("Cold Plains not seen, Den started: %v", r.said)
	}
	r.visit("Cold Plains", 3)
	r.visit("Rogue Encampment", 1)
	if got := r.q.Lines(); len(got) != 1 || got[0].Quest != "sisters_burial" || got[0].Kind != HintGo {
		t.Fatalf("lines %+v, want Kashya go", got)
	}
	r.visit("Rogue Encampment", 1)
	r.visit("Blood Moor", 2)
	r.visit("Rogue Encampment", 1)
	if len(r.said) != 1 {
		t.Errorf("go said again: %v", r.said)
	}
	r.started("sisters_burial")
	r.visit("Rogue Encampment", 1)
	if len(r.said) != 1 {
		t.Errorf("a given quest was hinted: %v", r.said)
	}
}

// leave walks out of town and back in: a new town visit.
func (r *hintRig) newVisit() {
	r.visit("Blood Moor", 2)
	r.visit("Rogue Encampment", 1)
}

// Too weak: one line per town visit overall, and a quest speaks again only after the level rose by 2.
func TestHintWeakRearmsAfterTwoLevels(t *testing.T) {
	r := newHintRig(t, "necromancer", 9) // summoner: Cain waits until 12, Charsi until 15
	r.started("den_of_evil")
	r.started("sisters_burial")
	r.visit("Outer Cloister", 27)
	r.visit("Stony Field", 4)
	r.visit("Rogue Encampment", 1)
	want := func(n int, quests ...string) {
		t.Helper()
		ls := r.q.Lines()
		if len(ls) != n || len(r.said) != n {
			t.Fatalf("clvl %d: %d lines said %d, want %d: %+v", r.level, len(ls), len(r.said), n, ls)
		}
		for i, q := range quests {
			if got := ls[n-len(quests)+i]; got.Quest != q || got.Kind != HintWeak {
				t.Fatalf("clvl %d: line %d is %+v, want a weak line for %s", r.level, n-len(quests)+i, got, q)
			}
		}
	}
	want(1, "search_for_cain") // one weak line for the visit, the first quest of the table
	r.newVisit()
	want(2, "tools_of_the_trade") // the other waiting quest has not spoken yet
	r.newVisit()
	want(2) // both armed until clvl 11
	r.level = 10
	r.newVisit()
	want(2)
	r.level = 11
	r.newVisit()
	want(3, "search_for_cain") // +2: Cain speaks again, only one line per visit
	r.newVisit()
	want(4, "tools_of_the_trade")
	r.newVisit()
	want(4)
}

func TestHintRestartDoesNotRepeat(t *testing.T) {
	r := newHintRig(t, "sorceress", 1)
	r.enterTown() // Den weak (needs 2)
	r.level = 8
	r.visit("Cold Plains", 3)
	r.visit("Rogue Encampment", 1) // Den go, Kashya go
	n := len(r.said)
	if n != 3 { // the Den weak line, then the Den and Kashya go lines
		t.Fatalf("said %v", r.said)
	}
	r.q = r.open() // restart
	r.enterTown()
	if len(r.said) != n {
		t.Errorf("a restart repeated hints: %v", r.said[n:])
	}
	if r.q.Errand() == "" || len(r.q.Lines()) != 3 {
		t.Errorf("errand %q lines %d after reload", r.q.Errand(), len(r.q.Lines()))
	}
	// the visited Cold Plains survived too: Kashya is still available, not re-announced
	r.level = 20
	r.started("den_of_evil")
	r.enterTown()
	if len(r.said) != n {
		t.Errorf("repeat after restart: %v", r.said[n:])
	}
}

func TestHintQuietIsSilentAndKeepsNothing(t *testing.T) {
	r := newHintRig(t, "paladin", 5)
	r.mode = HintsQuiet
	r.enterTown()
	r.mode = "unknown-option"
	r.enterTown()
	if len(r.said) != 0 || len(r.q.Lines()) != 0 {
		t.Errorf("quiet spoke: %v", r.said)
	}
	if _, err := os.Stat(filepath.Join(r.dir, "questhints-t.json")); err == nil {
		// areas and stages are kept, hints are not
		q := r.open()
		if len(q.Lines()) != 0 || q.Errand() != "" {
			t.Errorf("quiet kept hint state")
		}
	}
	r.mode = HintsOn
	r.enterTown()
	if len(r.said) != 1 {
		t.Errorf("switching Hints on later: %v", r.said)
	}
}

func TestHintUnknownClassNamesIt(t *testing.T) {
	r := newHintRig(t, "bard", 5)
	r.enterTown()
	r.enterTown()
	if len(r.said) != 0 || len(r.errs) != 1 || !strings.Contains(r.errs[0].Error(), `"bard"`) {
		t.Errorf("said %v errs %v", r.said, r.errs)
	}
}

func TestHintWaitsForTheClass(t *testing.T) {
	r := newHintRig(t, "", 0)
	r.enterTown()
	if len(r.said) != 0 {
		t.Fatal("spoke before the class was known")
	}
	r.class, r.level = "paladin", 3
	r.q.Tick()
	if len(r.said) != 1 {
		t.Errorf("Tick after the class: %v", r.said)
	}
}

func TestHintAskOnlyInAskMode(t *testing.T) {
	for _, mode := range []string{HintsOn, HintsAsk} {
		r := newHintRig(t, "paladin", 13) // melee Charsi 12; Cain asks at 12
		r.mode = mode
		r.started("den_of_evil")
		r.started("sisters_burial")
		r.visit("Outer Cloister", 27)
		r.visit("Stony Field", 4)
		r.visit("Rogue Encampment", 1)
		r.visit("Rogue Encampment", 1)
		want := 0
		if mode == HintsAsk {
			want = 2
		}
		if len(r.asked) != want {
			t.Errorf("%s: asked %v", mode, r.asked)
		}
	}
	r := newHintRig(t, "paladin", 11) // below the Cain ask level (12): Cain go at 9 speaks, asks nothing
	r.mode = HintsAsk
	r.started("den_of_evil")
	r.started("sisters_burial")
	r.visit("Stony Field", 4)
	r.visit("Rogue Encampment", 1)
	if len(r.asked) != 0 || len(r.said) != 1 {
		t.Errorf("clvl 11: asked %v said %v", r.asked, r.said)
	}
}

// Golden text: the seeded pick is stable, so a replay says the same thing.
func TestHintTextGolden(t *testing.T) {
	r := newHintRig(t, "sorceress", 1)
	r.enterTown()
	r.level = 4
	r.visit("Rogue Encampment", 1)
	want := []string{
		"Akara's worry can wait one more level. So can I.",
		"Akara keeps glancing toward the Blood Moor. I think I can help her now.",
	}
	if strings.Join(r.said, "|") != strings.Join(want, "|") {
		t.Errorf("lines:\n got %q\nwant %q", r.said, want)
	}
}

// The question card has a stable key (TASK-137): asking twice opens one thought.
func TestHintQuestionKeyIsStable(t *testing.T) {
	cat, err := decide.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	st, err := decide.Open(t.TempDir(), "qh", cat, &decide.ManualClock{})
	if err != nil {
		t.Fatal(err)
	}
	ask := func(point, quest string, class string) error {
		ctx, err := cat.Build(point, decide.Obs{Class: class, Cat: map[string]string{"act": "1", "diff": "normal", "hc": "normal", "quest": quest}}, 1, nil)
		if err != nil {
			return err
		}
		_, err = st.Ask(ctx)
		return err
	}
	for i := 0; i < 3; i++ {
		for _, c := range []string{"paladin", "sorceress"} {
			if err := ask("t1.quest.imbue", "tools", c); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := ask("t1.quest.cain", "cain", "paladin"); err != nil {
		t.Fatal(err)
	}
	open := st.OpenThoughts()
	keys := map[string]bool{}
	for _, th := range open {
		info, _ := st.Thought(th.ID)
		keys[info.Key] = true
	}
	if len(open) != 2 || !keys["t1.quest.imbue|act=1,diff=normal,hc=normal,quest=tools"] || !keys["t1.quest.cain|act=1,diff=normal,hc=normal,quest=cain"] {
		t.Errorf("open thoughts %d keys %v", len(open), keys)
	}
}

// Through the Fun wiring: the policy row switches the hints; the feed and the return section carry them.
func TestFunQuestHintsWiring(t *testing.T) {
	r := newFunRigWith(t, "paladin", 2, func(r *funRig, fc *FunConfig) {
		tbl, err := decide.ShippedPolicy(r.hub.cfg.Cat)
		if err != nil {
			t.Fatal(err)
		}
		fc.Policy = &PolicyConfig{Table: tbl, Knobs: func(map[string]*float64) error { return nil }}
	})
	if r.fun.Hints == nil {
		t.Fatal("no hint desk with the policy sheet on")
	}
	if got, def := r.fun.Policy.CurrentOption(QuestHintsRow), r.fun.Policy.DefaultOption(QuestHintsRow); got != HintsOn || def != HintsQuiet {
		t.Fatalf("a Paladin starts on %q (row default %q), want hints and quiet", got, def)
	}
	if _, _, err := r.fun.Policy.Set(QuestHintsRow, HintsQuiet); err != nil {
		t.Fatal(err)
	}
	r.feed(goals.Event{Type: goals.EvAreaChange, Area: "Blood Moor", AreaID: 2}, goals.Event{Type: goals.EvAreaChange, Area: "Rogue Encampment", AreaID: 1})
	if n := len(r.fun.Hints.Lines()); n != 0 {
		t.Fatalf("Quiet spoke %d lines", n)
	}
	if _, _, err := r.fun.Policy.Set(QuestHintsRow, HintsOn); err != nil {
		t.Fatal(err)
	}
	r.feed(goals.Event{Type: goals.EvAreaChange, Area: "Blood Moor", AreaID: 2}, goals.Event{Type: goals.EvAreaChange, Area: "Rogue Encampment", AreaID: 1})
	ls := r.fun.Hints.Lines()
	if len(ls) != 1 || ls[0].Kind != HintGo {
		t.Fatalf("lines %+v", ls)
	}
	var feed []string
	for _, f := range r.hub.Thoughts().Feed {
		feed = append(feed, f.Text)
	}
	if !strings.Contains(strings.Join(feed, "\n"), ls[0].Text) {
		t.Errorf("the hint is not in the feed: %q", feed)
	}
	sec := r.fun.Hints.Section().(SectionFunc)(&ReturnData{Since: 0})
	if sec == nil || sec.Title != "What I noticed in town" || len(sec.Lines) != 1 || sec.Lines[0] != ls[0].Text {
		t.Errorf("return section %+v", sec)
	}
	if hide := r.fun.Hints.Section().(SectionFunc)(&ReturnData{Since: 1e9}); hide != nil {
		t.Errorf("section shows old lines: %+v", hide)
	}
}

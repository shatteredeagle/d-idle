package decide

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// Section A, store behaviours accepted by the user (2026-11-05).

func TestSkipBankedPerPointOnlyOnNonRisky(t *testing.T) {
	r := newRig(t)
	a := r.ctx("t0.retreat.drink", map[string]string{"kit": "NONE"}, nil)
	b := r.ctx("t0.retreat.drink", map[string]string{"kit": "THIN"}, nil)
	r.say(a, "run")
	for i := 0; i < 3; i++ { // at most one banked skip, however often the player sacrifices
		if err := r.s.Sacrifice("t0.retreat.drink"); err != nil {
			t.Fatal(err)
		}
	}
	// the next non-risky answer at the point consumes the skip, even at another key and even when it cannot use it
	r.say(b, "drink")
	if rec := r.rec(b); rec.Streak != 1 {
		t.Fatalf("a first answer cannot gain from a skip: %+v", rec)
	}
	r.say(a, "run")
	if rec := r.rec(a); rec.Streak != 2 || r.learned(a) {
		t.Fatalf("the skip was banked once and already consumed: %+v", rec)
	}
	r.say(a, "run")
	if !r.learned(a) {
		t.Fatal("3 plain answers learn a normal-class rule")
	}
	// banked skip + the next answer at the same key: counts once more, exactly once
	c := r.ctx("t0.retreat.drink", map[string]string{"kit": "NONE", "held": "hp"}, nil)
	r.say(c, "drink")
	r.s.Sacrifice("t0.retreat.drink")
	r.say(c, "drink")
	if rec := r.rec(c); rec.Streak != 3 || !r.learned(c) {
		t.Fatalf("skip should add one: %+v", rec)
	}
	if got := countOp(r.s, OpSacrifice); got != 4 {
		t.Fatalf("sacrifice lines = %d", got)
	}
}

func TestSkipNeverUsedOnRiskyClass(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t0.retreat.lethal", nil, nil) // risky: N is 5, then a confirm
	r.say(c, "fight")
	r.s.Sacrifice("t0.retreat.lethal")
	r.say(c, "fight")
	if rec := r.rec(c); rec.Streak != 2 {
		t.Fatalf("skip used on a risky key: %+v", rec)
	}
	for i := 0; i < 2; i++ {
		r.say(c, "fight")
	}
	if r.pendingConfirm(c) != 0 {
		t.Fatal("4 answers must not reach the confirm: the skip did not count")
	}
	r.say(c, "fight")
	if r.pendingConfirm(c) == 0 {
		t.Fatal("5 answers open the confirm")
	}
	// hardcore retreat is risky too, and a non-risky answer at the same point still consumes the banked skip
	hc := r.ctx("t0.retreat.drink", map[string]string{"hc": "hardcore"}, nil)
	soft := r.ctx("t0.retreat.drink", nil, nil)
	r.say(soft, "drink")
	r.s.Sacrifice("t0.retreat.drink")
	r.say(hc, "drink")
	r.say(hc, "drink")
	if rec := r.rec(hc); rec.Streak != 2 {
		t.Fatalf("hardcore key used the skip: %+v", rec)
	}
	r.say(soft, "drink") // 1 + 1 + the skip the hardcore answers left alone
	if rec := r.rec(soft); rec.Streak != 3 || !r.learned(soft) {
		t.Fatalf("the softcore key should have used the banked skip: %+v", rec)
	}
}

func countOp(s *Store, op string) int {
	n := 0
	for _, e := range s.Log() {
		if e.Op == op {
			n++
		}
	}
	return n
}

func TestLateAnswerIsAFadeAndEmitsAnEvent(t *testing.T) {
	r := newRig(t)
	var evs []Event
	r.s.Subscribe(func(e Event) { evs = append(evs, e) })
	c := r.ctx("t1.return", nil, nil)
	r.say(c, "no")
	before := r.rec(c)
	d := r.ask(c)
	r.clk.Advance(fade)
	err := r.s.Answer(d.Thought.ID, "yes")
	if !errors.Is(err, ErrFaded) {
		t.Fatalf("late answer: %v", err)
	}
	if after := r.rec(c); after.Streak != before.Streak || after.Last != before.Last {
		t.Fatalf("a late answer counted or reset: %+v -> %+v", before, after)
	}
	var late *Event
	for i := range evs {
		if evs[i].Kind == EvLateAnswer {
			late = &evs[i]
		}
	}
	if late == nil || late.Thought != d.Thought.ID || late.Key != c.Key || late.Answer != "yes" {
		t.Fatalf("no LateAnswer event: %+v", evs)
	}
	if _, _, faded := r.s.Stats(); faded != 1 {
		t.Fatalf("faded = %d", faded)
	}
	// a tick that fades a thought is a fade, not a late answer
	n := len(evs)
	d = r.ask(c)
	r.clk.Advance(fade)
	r.s.Tick()
	for _, e := range evs[n:] {
		if e.Kind == EvLateAnswer {
			t.Fatal("tick emitted a late answer")
		}
	}
}

func TestPendingConfirmBlocksNewAsks(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t0.retreat.lethal", nil, nil)
	for i := 0; i < 5; i++ {
		r.say(c, "fight")
	}
	id := r.pendingConfirm(c)
	if id == 0 {
		t.Fatal("no confirm pending")
	}
	opens := countOp(r.s, OpOpen)
	for i := 0; i < 3; i++ {
		d := r.ask(c)
		if d.Thought != nil || d.Source != "default" {
			t.Fatalf("a new ask thought opened under a pending confirm: %+v", d)
		}
	}
	if countOp(r.s, OpOpen) != opens {
		t.Fatal("open lines were written under a pending confirm")
	}
	r.clk.Advance(fade)
	r.s.Tick() // the confirm fades: asking works again
	if d := r.ask(c); d.Thought == nil || d.Thought.Confirm {
		t.Fatalf("after the fade a new ask thought should open: %+v", d)
	}
	// and a confirm resolved by the player also frees the key
	r2 := newRig(t)
	c2 := r2.ctx("t0.retreat.lethal", nil, nil)
	for i := 0; i < 5; i++ {
		r2.say(c2, "fight")
	}
	if err := r2.s.Confirm(r2.pendingConfirm(c2), false); err != nil {
		t.Fatal(err)
	}
	if d := r2.ask(c2); d.Thought == nil {
		t.Fatal("a resolved confirm must free the key")
	}
}

// API additions (notes: "API changes since stage 1").

func TestThoughtAccessorAndEventInfo(t *testing.T) {
	r := newRig(t)
	var opened []Event
	r.s.Subscribe(func(e Event) {
		if e.Kind == EvThoughtOpened || e.Kind == EvConfirmOpened {
			opened = append(opened, e)
		}
	})
	c := r.ctx("t0.retreat.drink", nil, nil)
	d := r.ask(c)
	if len(opened) != 1 || opened[0].Info == nil {
		t.Fatalf("events: %+v", opened)
	}
	info := opened[0].Info
	got, ok := r.s.Thought(d.Thought.ID)
	if !ok || got.ID != info.ID || got.Point != c.Point || got.Key != c.Key || got.Expires != d.Thought.FadesAt ||
		len(got.Choices) != 5 || got.Question == "" || got.Confirm {
		t.Fatalf("thought info: %+v", got)
	}
	if _, ok := r.s.Thought(999); ok {
		t.Fatal("unknown thought found")
	}
	r.clk.Advance(fade)
	r.s.Tick()
	if _, ok := r.s.Thought(d.Thought.ID); ok {
		t.Fatal("a faded thought is still readable")
	}
	// confirm thoughts carry the answer to learn and no choices
	lc := r.ctx("t0.retreat.lethal", nil, nil)
	for i := 0; i < 5; i++ {
		r.say(lc, "fight")
	}
	last := opened[len(opened)-1]
	if last.Kind != EvConfirmOpened || last.Info == nil || !last.Info.Confirm || last.Info.Answer != "fight" || len(last.Info.Choices) != 0 {
		t.Fatalf("confirm event: %+v", last)
	}
}

func TestAskWithDefault(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t1.restock", nil, nil) // catalogue default here is "no"
	d, err := r.s.AskWithDefault(c, "yes")
	if err != nil || d.Answer != "yes" || d.Source != "legacy" || d.Thought == nil {
		t.Fatalf("%+v %v", d, err)
	}
	if _, err := r.s.AskWithDefault(c, "maybe"); !errors.Is(err, ErrBadAnswer) {
		t.Fatalf("a legacy answer that is not an answer: %v", err)
	}
	r.say(c, "no")
	r.say(c, "no")
	if d, _ := r.s.AskWithDefault(c, "yes"); d.Answer != "no" || d.Source != "rule" {
		t.Fatalf("a learned rule must win over the legacy answer: %+v", d)
	}
	r.s.SetUnlockedTier(0)
	if d, _ := r.s.AskWithDefault(r.ctx("t1.return", nil, nil), "no"); d.Answer != "no" || d.Source != "default-locked" || d.Thought != nil {
		t.Fatalf("locked tier: %+v", d)
	}
	if _, err := r.s.AskWithDefault(Context{Point: "t9.zz"}, "x"); err == nil || !strings.Contains(err.Error(), "t9.zz") {
		t.Fatalf("unknown point: %v", err)
	}
}

func TestExportedProfiles(t *testing.T) {
	if len(KnownProfiles()) != 3 || len(ProfileNames) != 3 {
		t.Fatal("three known profiles")
	}
	for _, n := range []string{"cautious", "bold", "inconsistent", "holdout"} {
		if p, err := NewProfile(n, 7); err != nil || p.Name != n || p.Reply == nil || p.Confirm == nil {
			t.Errorf("%s: %v", n, err)
		}
	}
	if _, err := NewProfile("zany", 1); err == nil || !strings.Contains(err.Error(), "zany") {
		t.Fatalf("bad profile: %v", err)
	}
}

// Section B: catalogue version 2 and the migration of a version 1 file.

func v1Entry(seq int, op string, thought int, key, answer string) string {
	s := fmt.Sprintf(`{"seq":%d,"t":%d,"hero":"hero1","op":%q,"thought":%d`, seq, seq*1000000000, op, thought)
	if key != "" {
		s += fmt.Sprintf(`,"key":%q,"fadesAt":%d`, key, seq*1000000000+45000000000)
	}
	if answer != "" {
		s += fmt.Sprintf(`,"answer":%q`, answer)
	}
	return s + "}"
}

// A profile file written by catalogue version 1 still loads; its rules move to the new keys.
func TestVersion1FileMigrates(t *testing.T) {
	restock := "t1.restock|act=1,belt=full,diff=normal,gold=100-500,hc=normal"
	drink := "t0.retreat.drink|act=1,diff=normal,hc=normal,kit=OK,life=25-35,packdanger=1"
	trigger := "t1.town.trigger|act=1,diff=normal,freerows=0,hc=normal,kit=OK"
	fail := "t2.failstreak|act=1,deaths=3+,diff=normal,hc=normal,runkind=farm"
	var lines []string
	seq, th := 1, 1
	for _, k := range []struct{ key, ans string }{{restock, "yes"}, {restock, "yes"}, {trigger, "go"}, {trigger, "go"},
		{drink, "portal"}, {drink, "portal"}, {drink, "portal"}, {fail, "stop"}} {
		lines = append(lines, v1Entry(seq, "open", th, k.key, ""), v1Entry(seq+1, "answer", th, "", k.ans))
		seq, th = seq+2, th+1
	}
	body := `{"version":1,"profile":"old","hero":"hero1","log":[` + strings.Join(lines, ",") + `],"rules":[]}`
	dir := t.TempDir()
	if err := os.WriteFile(Path(dir, "old"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cat := mustCat(t)
	s, err := Open(dir, "old", cat, &ManualClock{})
	if err != nil {
		t.Fatalf("a version 1 file must load: %v", err)
	}
	want := map[string]string{
		"t1.restock|act=1,belt=full,diff=normal,hc=normal,tripgold=60-300":                            "yes",
		"t1.town.trigger|act=1,belt=full,diff=normal,hc=normal,packused=ge80":                         "go",
		"t0.retreat.drink|act=1,champbonus=none,diff=normal,drinklife=26-49,hc=normal,held=hp,kit=OK": "portal",
	}
	rules := map[string]string{}
	for _, v := range s.Rules() {
		rules[v.Key] = v.Answer
	}
	for k, a := range want {
		if rules[k] != a {
			t.Errorf("rule %s = %q, want %q (rules: %v)", k, rules[k], a, rules)
		}
	}
	// failstreak: one answer only, so no rule, but its streak moved to the new key
	if rec, ok := s.Record("t2.failstreak|act=1,deaths=4+,diff=normal,hc=normal,runkind=farm"); !ok || rec.Streak != 1 {
		t.Errorf("failstreak record: %+v %v", rec, ok)
	}
	ctx, _ := cat.Build("t1.restock", baseObs(cat, "sorceress").with(map[string]string{"belt": "full"}, map[string]float64{"tripgold": 100}), 1, nil)
	if d, _ := s.Ask(ctx); d.Source != "rule" || d.Answer != "yes" {
		t.Fatalf("the migrated rule must answer: %+v", d)
	}
	// the next save writes the current version and a reload sees the same rules
	s.SetQuality("x", 1)
	if b, _ := os.ReadFile(Path(dir, "old")); !strings.Contains(string(b), fmt.Sprintf(`"version": %d`, FileVersion)) {
		t.Fatal("the saved file is not the current version")
	}
	s2, err := Open(dir, "old", cat, &ManualClock{})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := s.RulesJSON()
	b, _ := s2.RulesJSON()
	if string(a) != string(b) {
		t.Fatal("reload of the migrated file differs")
	}
}

func TestMigrateKeyIsIdempotentAndLeavesOtherPoints(t *testing.T) {
	cat := mustCat(t)
	for _, k := range []string{
		"t1.repair|act=1,diff=normal,durability=lt20,gold=lt100,hc=normal",
		"t0.retreat.drink|act=1,champbonus=early,diff=normal,drinklife=ge65,hc=normal,held=rv,kit=OK",
		"t1.restock|act=1,belt=full,diff=normal,hc=normal,tripgold=ge300",
		"t0.unique.fight|act=1,diff=normal,hc=normal,kit=OK,lifeengage=40-55,packdanger=2",
		"t2.mercenary|act=1,diff=normal,hc=normal,kit=OK,merc=none,mercgold=ge400",
	} {
		if got := cat.MigrateKey(k); got != k {
			t.Errorf("%s changed to %s", k, got)
		}
	}
	old := "t1.town.trigger|act=1,diff=normal,freerows=2+,hc=normal,kit=THIN"
	once := cat.MigrateKey(old)
	if once == old || cat.MigrateKey(once) != once || !strings.Contains(once, "packused=lt80") || strings.Contains(once, "kit") {
		t.Fatalf("migrated %s", once)
	}
	if cat.MigrateKey("garbage") != "garbage" {
		t.Fatal("garbage key changed")
	}
}

// A profile file written by catalogue version 2 (file version 2) still loads; its rules for the three re-banded
// points move to the new keys and keep answering.
func TestVersion2FileMigrates(t *testing.T) {
	drink := "t0.retreat.drink|act=1,diff=normal,hc=normal,held=none,kit=OK,life=lt25,packdanger=3"
	uniq := "t0.unique.fight|act=1,diff=normal,hc=normal,kit=OK,life=35-55,packdanger=2"
	merc := "t2.mercenary|act=1,diff=normal,gold=500-1500,hc=normal,kit=OK,merc=dead"
	toohard := "t0.area.toohard|act=1,areaclass=field,clvl=14,diff=normal,hc=normal,kit=OK,levelgap=1-3"
	var lines []string
	seq, th := 1, 1
	for _, k := range []struct{ key, ans string }{{drink, "portal"}, {drink, "portal"}, {drink, "portal"},
		{uniq, "avoid"}, {uniq, "avoid"}, {uniq, "avoid"}, {merc, "hire"}, {merc, "hire"}, {merc, "hire"}, {toohard, "go"}} {
		lines = append(lines, v1Entry(seq, "open", th, k.key, ""), v1Entry(seq+1, "answer", th, "", k.ans))
		seq, th = seq+2, th+1
	}
	body := `{"version":2,"profile":"v2","hero":"hero1","log":[` + strings.Join(lines, ",") + `],"rules":[]}`
	dir := t.TempDir()
	if err := os.WriteFile(Path(dir, "v2"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cat := mustCat(t)
	s, err := Open(dir, "v2", cat, &ManualClock{})
	if err != nil {
		t.Fatalf("a version 2 file must load: %v", err)
	}
	rules := map[string]string{}
	for _, v := range s.Rules() {
		rules[v.Key] = v.Answer
	}
	for k, a := range map[string]string{
		"t0.retreat.drink|act=1,champbonus=none,diff=normal,drinklife=18-25,hc=normal,held=none,kit=OK": "portal",
		"t0.unique.fight|act=1,diff=normal,hc=normal,kit=OK,lifeengage=40-55,packdanger=2":              "avoid",
		"t2.mercenary|act=1,diff=normal,hc=normal,kit=OK,merc=dead,mercgold=ge400":                      "hire",
	} {
		if rules[k] != a {
			t.Errorf("rule %s = %q, want %q (rules: %v)", k, rules[k], a, rules)
		}
	}
	// a point that did not change keeps its key as written
	if got := cat.MigrateKey(toohard); got != toohard {
		t.Errorf("toohard key changed: %s", got)
	}
	// the migrated rule answers the context a call site builds today
	ctx, err := cat.Build("t0.retreat.drink", baseObs(cat, "sorceress").with(map[string]string{"kit": "OK", "held": "none"}, map[string]float64{"life": 20}), 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := s.Ask(ctx); d.Source != "rule" || d.Answer != "portal" {
		t.Fatalf("the migrated rule must answer: %+v", d)
	}
}

// The optional fields: a call site written before champbonus existed still builds a context (champbonus none).
func TestOptionalFieldAndFromBands(t *testing.T) {
	cat := mustCat(t)
	o := baseObs(cat, "sorceress").with(map[string]string{"kit": "OK", "held": "hp"}, map[string]float64{"life": 60})
	delete(o.Cat, "champbonus")
	ctx, err := cat.Build("t0.retreat.drink", o, 1, nil)
	if err != nil || ctx.Fields["champbonus"] != "none" || ctx.Fields["drinklife"] != "55-64" {
		t.Fatalf("%v %v", ctx, err)
	}
	// one reported life number feeds every band that reads it
	for _, tc := range []struct {
		point, field string
		life         float64
		want         string
	}{{"t0.unique.fight", "lifeengage", 39, "35-40"}, {"t0.unique.fight", "lifeengage", 40, "40-55"}} {
		o := baseObs(cat, "paladin").with(nil, map[string]float64{"life": tc.life})
		c, err := cat.Build(tc.point, o, 1, nil)
		if err != nil || c.Fields[tc.field] != tc.want {
			t.Errorf("%s life %v: %v %v", tc.point, tc.life, c.Fields, err)
		}
	}
}

// B1, B2: the retreat point can say "keep fighting" and "leave the game", and knows what potion is held.
func TestRetreatAnswersAndHeldField(t *testing.T) {
	cat := mustCat(t)
	p, _ := cat.Point("t0.retreat.drink")
	for _, a := range []string{"fight", "leave", "drink", "run", "portal"} {
		if _, ok := p.answer(a); !ok {
			t.Errorf("missing answer %q", a)
		}
	}
	if !contains(p.ContextKeys, "held") {
		t.Fatal("held-potion field missing")
	}
	def := func(life float64, kit, held string) string {
		o := baseObs(cat, "paladin").with(map[string]string{"kit": kit, "held": held}, map[string]float64{"life": life})
		ctx, err := cat.Build(p.ID, o, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		a, _ := cat.Default(ctx)
		return a
	}
	for _, tc := range []struct {
		life      float64
		kit, held string
		want      string
	}{
		{90, "OK", "hp", "fight"}, {60, "NONE", "none", "fight"}, {30, "OK", "hp", "drink"}, {30, "OK", "rv", "drink"},
		{30, "OK", "none", "fight"}, {30, "NONE", "none", "run"}, {20, "OK", "none", "portal"}, {20, "NONE", "none", "leave"},
		{20, "OK", "rv", "drink"},
	} {
		if got := def(tc.life, tc.kit, tc.held); got != tc.want {
			t.Errorf("life %v kit %s held %s: %q, want %q", tc.life, tc.kit, tc.held, got, tc.want)
		}
	}
}

package questspec

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeBE is a world of facts: act sets what the script says; Holds reads the facts.
type fakeBE struct {
	facts    map[string]bool
	approach []error
	acts     []string
	actErr   map[string][]error // per verb, popped in order
	onAct    func(verb string, n int)
	slept    []time.Duration
	nAct     int
}

func (f *fakeBE) Approach(context.Context, Target, int) error {
	if len(f.approach) == 0 {
		return nil
	}
	e := f.approach[0]
	f.approach = f.approach[1:]
	return e
}
func (f *fakeBE) Act(_ context.Context, verb string, t Target, args ActArgs) error {
	if args.Pick != "" {
		verb += "[" + args.Pick + "]"
	}
	if len(args.Inputs) > 0 {
		verb += "(" + strings.Join(args.Inputs, "+") + ")"
	}
	f.acts = append(f.acts, verb+":"+t.Name+t.Code)
	f.nAct++
	if f.onAct != nil {
		f.onAct(verb, f.nAct)
	}
	if q := f.actErr[verb]; len(q) > 0 {
		f.actErr[verb] = q[1:]
		return q[0]
	}
	return nil
}
func (f *fakeBE) Holds(c Clause) bool {
	for _, p := range c {
		if f.facts[p.String()] {
			return true
		}
	}
	return false
}
func (f *fakeBE) Sleep(d time.Duration) { f.slept = append(f.slept, d) }

func treeStep() Step {
	return Step{ID: "take_scroll", Act: "pickup", Target: Target{Kind: "item", Code: "bks"}, Tries: 3,
		Post: []string{"has_item(bks)|has_item(bkd)"},
		Exec: &Exec{Verb: "interact", On: &Target{Kind: "object", Name: "tree_of_inifuss", Class: 30}, Then: "pickup",
			WaitPolls: 4, WaitMS: 1000, BackoffMS: 500, BackoffCapMS: 1000}}
}

// talkStep is a second class of step: an npc dialog whose post is a quest word, no pick-up.
func talkStep() Step {
	return Step{ID: "read", Act: "talk", Target: Target{Kind: "npc", Name: "akara", Class: 148}, Tries: 2,
		Post: []string{"quest_state(search_for_cain,reward_pending)"},
		Exec: &Exec{Verb: "talk", WaitPolls: 2, WaitMS: 500}}
}

func TestPostAlreadyHoldsTouchesNothing(t *testing.T) {
	f := &fakeBE{facts: map[string]bool{"has_item(bkd)": true}}
	if o := RunStep(context.Background(), treeStep(), f); o.Kind != Done || len(f.acts) != 0 {
		t.Fatalf("%v acts %v", o, f.acts)
	}
}

func TestInteractThenPickupUntilThePostHolds(t *testing.T) {
	f := &fakeBE{facts: map[string]bool{}, actErr: map[string][]error{"pickup": {ErrNothing, ErrNothing}}}
	f.onAct = func(v string, n int) {
		if v == "pickup" && n == 4 { // click, nothing, nothing, then the third pick-up finds it
			f.facts["has_item(bks)"] = true
		}
	}
	o := RunStep(context.Background(), treeStep(), f)
	if o.Kind != Done || o.Attempts != 1 {
		t.Fatalf("%+v acts %v", o, f.acts)
	}
	if f.acts[0] != "interact:tree_of_inifuss" || f.acts[1] != "pickup:bks" {
		t.Fatalf("order %v", f.acts)
	}
}

// Must fail: a step whose act "succeeds" but whose fact never appears is Failed after its budget, never Done.
func TestActWithoutThePostIsFailedAfterTheBudget(t *testing.T) {
	for _, s := range []Step{treeStep(), talkStep()} {
		f := &fakeBE{facts: map[string]bool{}}
		o := RunStep(context.Background(), s, f)
		if o.Kind != Failed || o.Err == nil || o.Attempts != s.Tries {
			t.Fatalf("%s: %+v", s.ID, o)
		}
	}
}

func TestBackoffDoublesAndIsCapped(t *testing.T) {
	f := &fakeBE{facts: map[string]bool{}}
	st := treeStep()
	st.Exec.WaitMS = 10
	RunStep(context.Background(), st, f)
	var between []time.Duration
	for _, d := range f.slept {
		if d != 10*time.Millisecond { // the wait polls
			between = append(between, d)
		}
	}
	if len(between) != 2 || between[0] != 500*time.Millisecond || between[1] != time.Second {
		t.Fatalf("backoff %v", between)
	}
}

func TestRefusalIsDataNotFailure(t *testing.T) {
	for _, s := range []Step{treeStep(), talkStep()} {
		s.Exec.Wake = []string{"level rises"}
		f := &fakeBE{facts: map[string]bool{}, approach: []error{&NeedError{Need: "too_weak", Why: "level 4"}}}
		o := RunStep(context.Background(), s, f)
		if o.Kind != Refused || o.Need != "too_weak" || o.Wake != "level rises" || len(f.acts) != 0 {
			t.Fatalf("%s: %+v", s.ID, o)
		}
	}
}

func TestProgressDoesNotCountAsATry(t *testing.T) {
	s := talkStep()
	f := &fakeBE{facts: map[string]bool{}, approach: []error{ErrProgress}}
	if o := RunStep(context.Background(), s, f); o.Kind != Progress || o.Attempts != 0 {
		t.Fatalf("%+v", o)
	}
}

func TestCancelledIsInterruptedAndFatalFailsAtOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if o := RunStep(ctx, treeStep(), &fakeBE{facts: map[string]bool{}}); o.Kind != Interrupted {
		t.Fatalf("%+v", o)
	}
	boom := errors.New("no such object")
	f := &fakeBE{facts: map[string]bool{}, actErr: map[string][]error{"interact": {&FatalError{boom}}}}
	if o := RunStep(context.Background(), treeStep(), f); o.Kind != Failed || !errors.Is(o.Err, boom) || o.Attempts != 1 {
		t.Fatalf("%+v", o)
	}
}

func TestAPlainActErrorIsRetried(t *testing.T) {
	f := &fakeBE{facts: map[string]bool{}, actErr: map[string][]error{"talk": {errors.New("menu not open")}}}
	f.onAct = func(v string, n int) {
		if n == 2 {
			f.facts["quest_state(search_for_cain,reward_pending)"] = true
		}
	}
	if o := RunStep(context.Background(), talkStep(), f); o.Kind != Done {
		t.Fatalf("%+v", o)
	}
}

func TestExecRowIsValidated(t *testing.T) {
	c := &Chain{Schema: Schema, ID: "x", Initial: []string{}, Steps: []Step{{ID: "a", Act: "interact", OnFail: "retry",
		Pre: []string{}, Post: []string{}, Exec: &Exec{Verb: "dance"}}}}
	if err := c.Validate(); err == nil {
		t.Fatal("a bad exec row passed")
	}
}

// A talk step carries its dialogue option to the backend, on the act and on a Then act; a bad pick or a verb on the wrong
// kind of target is a row error.
func TestTalkStepPicksTheLine(t *testing.T) {
	s := talkStep()
	s.Exec.Pick = "bark"
	f := &fakeBE{facts: map[string]bool{}}
	f.onAct = func(string, int) { f.facts["quest_state(search_for_cain,reward_pending)"] = true }
	if o := RunStep(context.Background(), s, f); o.Kind != Done || len(f.acts) != 1 || f.acts[0] != "talk[bark]:akara" {
		t.Fatalf("%v acts %v", o, f.acts)
	}
	bad := func(mut func(*Step)) bool {
		x := talkStep()
		mut(&x)
		return x.Exec != nil && validateExecForTest(x)
	}
	if !bad(func(x *Step) {
		x.Exec.Pick = "reward"
		x.Exec.Verb = "interact"
		x.Exec.On = &Target{Kind: "object", Class: 30}
	}) {
		t.Error("pick on a non-talk verb accepted")
	}
	if !bad(func(x *Step) { x.Target = Target{Kind: "area", Name: "x"} }) {
		t.Error("talk to an area accepted")
	}
	if !bad(func(x *Step) { x.Exec.Verb = "use" }) {
		t.Error("use on an npc accepted")
	}
	if bad(func(x *Step) {}) {
		t.Error("a good talk step rejected")
	}
}

// validateExecForTest reports whether the step's exec row has a validation error.
func validateExecForTest(s Step) (hasErr bool) {
	s.Exec.validate(s, func(string, ...any) { hasErr = true })
	return
}

func cubeStep() Step {
	return Step{ID: "transmute", Act: "transmute", Target: Target{Kind: "item", Code: "hst"}, Tries: 2,
		Post: []string{"has_item(hst)"}, Exec: &Exec{Verb: "transmute", Inputs: []string{"msf", "vip"}, WaitPolls: 3, WaitMS: 500}}
}

// The transmute verb passes the row's Cube inputs and waits for the product in the pack; a Cube that answers without the
// product uses both tries and fails (the post is the pack, not the Cube's answer).
func TestRunStepTransmute(t *testing.T) {
	f := &fakeBE{facts: map[string]bool{}}
	f.onAct = func(string, int) { f.facts["has_item(hst)"] = true }
	if o := RunStep(context.Background(), cubeStep(), f); o.Kind != Done || len(f.acts) != 1 || f.acts[0] != "transmute(msf+vip):hst" {
		t.Fatalf("%v acts %v", o, f.acts)
	}
	f = &fakeBE{facts: map[string]bool{}}
	if o := RunStep(context.Background(), cubeStep(), f); o.Kind != Failed || len(f.acts) != 2 {
		t.Fatalf("must fail: %v acts %v", o, f.acts)
	}
	bad := func(mut func(*Step)) bool {
		x := cubeStep()
		mut(&x)
		return validateExecForTest(x)
	}
	if !bad(func(x *Step) { x.Exec.Inputs = []string{"msf"} }) {
		t.Error("one input accepted")
	}
	if !bad(func(x *Step) { x.Exec.Inputs = []string{"msf", "Amulet of the Viper"} }) {
		t.Error("a display name as input accepted")
	}
	if !bad(func(x *Step) { x.Exec.Verb = "interact"; x.Exec.On = &Target{Kind: "object", Class: 152} }) {
		t.Error("inputs on an interact accepted")
	}
	if bad(func(x *Step) {}) {
		t.Error("a good transmute step rejected")
	}
}

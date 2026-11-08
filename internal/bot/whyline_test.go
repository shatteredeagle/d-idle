package bot

import (
	"slices"
	"testing"

	"d2lod-bot/internal/watch"
)

// Every code the bot can set has a template and every template has a code (the sentences live in watch/whyline.json).
func TestWhyCodesMatchTemplates(t *testing.T) {
	got := slices.Clone(whyCodes)
	slices.Sort(got)
	want := slices.DeleteFunc(watch.WhyCodes(), func(c string) bool { return c == "hold" || c == "activity" }) // watch-only codes
	if !slices.Equal(got, want) {
		t.Errorf("bot codes %v\ntemplate codes %v", got, want)
	}
}

// A wounded hero in the field with no way to town carries the reason of its wait; a healthy one walking to an area
// names where and at which level. The reason goes with the activity and is not left behind when the activity changes.
func TestStatusCarriesTheReason(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.town.toTownErr = errBoom
	g.w.set(func(w *world) { w.level = 3; w.area = 4; w.life = 20 })
	g.w.belt()
	g.run()
	defer g.finish()
	g.r.Start(farm("Stony Field"))
	g.waitFor("a wait with a reason", func() bool { return g.r.Status().Why.Code == WhyWaitHeal })
	if st := g.r.Status(); st.Why.Params["why"] == "" {
		t.Errorf("no parameter: %+v", st.Why)
	}
}

package campaign_test

import (
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/campaign"
	"d2lod-bot/internal/task"
)

func TestControllerDecidedApplyView(t *testing.T) {
	s := shipped(t)
	path := filepath.Join(t.TempDir(), campaign.StateFile)
	hero := "Palli"
	now := t0
	jc, err := campaign.NewController(s, path, func() string { return hero }, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	var doneSeen []string
	jc.OnDone = func(_ string, ids []string) { doneSeen = append(doneSeen, ids...) }
	jc.Pace = func(int) (float64, time.Duration, bool) { return 0.62, 25 * time.Minute, true }

	p := prog("paladin", 7, false, task.QDen)
	o := jc.Opts()
	d, err := s.Next(p, jc.State(), o)
	if err != nil {
		t.Fatal(err)
	}
	jc.Decided(d, p, o)
	if len(doneSeen) != 3 { // start, Cold Plains waypoint, Den
		t.Fatalf("done %v", doneSeen)
	}
	v := jc.View()
	if v.Decision == nil || v.Decision.Chapter != "blood_raven" || v.Meter == nil || v.Meter.To != 9 || v.Meter.ETASeconds != 1500 {
		t.Fatalf("view %+v", v)
	}
	states := map[string]string{}
	for _, c := range v.Chapters {
		states[c.ID] = c.State
	}
	if states["den_of_evil"] != campaign.StDone || states["blood_raven"] != campaign.StNext || states["andariel"] != campaign.StWaiting {
		t.Errorf("states %v", states)
	}
	yes := true
	for _, r := range []campaign.Request{{Chapter: "countess", Intent: campaign.Skip}, {Chapter: "andariel", GoNow: &yes}, {GoAt: campaign.GoFloor}} {
		if err := jc.Apply(r); err != nil {
			t.Fatal(err)
		}
	}
	for _, bad := range []campaign.Request{{}, {Chapter: "nope", Intent: campaign.Skip}, {Mode: "auto"}, {Chapter: "countess"}} {
		if jc.Apply(bad) == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
	// A second controller on the same file sees the player's changes; another hero starts clean.
	jc2, err := campaign.NewController(s, path, func() string { return hero }, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	st := jc2.State()
	if st.Intents["countess"] != campaign.Skip || st.GoNow != "andariel" || st.GoAt != campaign.GoFloor || st.Done["den_of_evil"].IsZero() {
		t.Errorf("persisted %+v", st)
	}
	hero = "Sorci"
	if st := jc2.State(); st.GoNow != "" || len(st.Intents) != 0 {
		t.Errorf("other hero %+v", st)
	}
	// The state copy is detached: changing it does not change the controller.
	st = jc2.State()
	st.Intents = map[string]campaign.Intent{"countess": campaign.Hold}
	if jc2.State().Intents != nil {
		t.Error("state copy leaks")
	}
}

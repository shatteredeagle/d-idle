package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"fmt"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// Charsi's menu with the fourth line (Imbue after the quest, U live): Talk, Trade, Imbue, Cancel. The default
// Talk/Trade/Cancel geometry is one line off, and the pickEntry adjustments must still reach Trade.
func TestTradeWithAnExtraMenuLine(t *testing.T) {
	r := newRig(t)
	r.W.Lines = map[int]int{154: 4}
	openCharsi(t, r)
	if r.W.UI[d2.UIShop] == 0 {
		t.Fatal("trade window not open")
	}
}

// The vendor does not open on the first clicks (the NPC speaks first): pickEntry reopens and succeeds.
func TestTradeAfterASpeech(t *testing.T) {
	r := newRig(t)
	r.W.Speech = 1
	openCharsi(t, r)
	if r.W.UI[d2.UIShop] == 0 {
		t.Fatal("trade window not open")
	}
}

// A click that keeps closing the dialog (wrong line count it can never fix) stops after a few closes with a reason,
// instead of clicking the same line forever.
func TestMenuClosesAreBounded(t *testing.T) {
	r := newRig(t)
	r.W.TradeIndex = map[int]int{154: 7} // no such line: every click lands on Cancel or Talk
	npc, err := r.T.OpenForTest(context.Background(), 154)
	if err != nil {
		t.Fatal(err)
	}
	err = r.T.PickEntryForTest(context.Background(), npc, town.EntryTrade, d2.UIShop)
	if err == nil {
		t.Fatal("expected a failure")
	}
	if !strings.Contains(err.Error(), "not reached") && !strings.Contains(err.Error(), "closed") {
		t.Fatalf("unclear reason: %v", err)
	}
}

// The NPC wanders between the open and the click: the click follows the current position.
func TestTradeFollowsAWanderingNPC(t *testing.T) {
	r := newRig(t)
	npc, err := r.T.OpenForTest(context.Background(), 154)
	if err != nil {
		t.Fatal(err)
	}
	for i := range r.W.Mons {
		if r.W.Mons[i].ID == npc.ID {
			r.W.Mons[i].X += 3
			r.W.Mons[i].Y += 3
		}
	}
	if err := r.T.PickEntryForTest(context.Background(), npc, town.EntryTrade, d2.UIShop); err != nil {
		t.Fatal(err)
	}
}

// interactsWith counts the clicks that opened (or tried to open) the NPC's dialog.
func interactsWith(r *rig, npc d2.Unit) int {
	n := 0
	want := fmt.Sprintf("interact %d/%d", d2.TypeMonster, npc.ID)
	for _, c := range r.In.Calls {
		if c == want {
			n++
		}
	}
	return n
}

// One town trip opens each NPC once: sell + belt + repair at Charsi share one window, shop + heal + identify at
// Akara share another (live G2: Charsi 4 and Akara 3 visits in 13 minutes, one per chore).
func TestOneVisitPerNPCPerTrip(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	dressed(w)
	w.Player.Level = 14
	w.Player.Life = 12
	sword := w.Add("ssd", d2.ContInventory, 1, 0)
	sword.Item.Quality = 4
	w.Items[len(w.Items)-1] = sword
	w.Add("tsc", d2.ContInventory, 9, 3)
	rep, err := r.T.Chores(context.Background(), town.FullPlan())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Errors != 0 {
		t.Fatalf("steps %+v", rep.Steps)
	}
	if n := interactsWith(r, r.Charsi); n != 1 {
		t.Errorf("Charsi opened %d times, want 1 (%+v)", n, rep.Steps)
	}
	if n := interactsWith(r, r.Akara); n != 1 {
		t.Errorf("Akara opened %d times, want 1 (%+v)", n, rep.Steps)
	}
	if !rep.Healed {
		t.Error("not healed")
	}
}

// A walk that stops far from the NPC is repeated before the click: nobody is clicked from more than 4 subtiles away,
// and the first click opens the dialog.
func TestInteractOnlyWhenClose(t *testing.T) {
	r := newRig(t)
	r.Mv.Short = 1
	npc, err := r.T.OpenForTest(context.Background(), 148)
	if err != nil {
		t.Fatal(err)
	}
	if n := interactsWith(r, npc); n != 1 {
		t.Fatalf("%d clicks, want 1", n)
	}
	d := (r.W.Player.X-npc.X)*(r.W.Player.X-npc.X) + (r.W.Player.Y-npc.Y)*(r.W.Player.Y-npc.Y)
	if d > 16 {
		t.Fatalf("clicked from dist^2 %d", d)
	}
}

// Akara wanders while the greeting plays and Esc dismisses it: the re-click must use the position read now. The stale
// position of the first click misses the sprite (V live i5: the greeting loop re-clicked a stale spot and the visit failed).
func TestGreetingReclickUsesTheFreshPosition(t *testing.T) {
	r := newRig(t)
	r.W.Speech = 1
	r.W.StrictPos = true
	r.W.WanderOnClose = [2]int{0, -7}
	start := r.Env.Clock.Now()
	npc, err := r.T.OpenForTest(context.Background(), 148)
	if err != nil {
		t.Fatal(err)
	}
	if d := r.Env.Clock.Now().Sub(start); d > 15*time.Second {
		t.Fatalf("visit took %v of game time", d)
	}
	if r.W.UI[d2.UINPCMenu] == 0 || npc.Class != 148 {
		t.Fatal("menu not open")
	}
}

// Hover dead for a few clicks during the greeting (the speech) must not fail the visit: it keeps waiting.
func TestDeadHoverDuringGreetingDoesNotFailTheVisit(t *testing.T) {
	r := newRig(t)
	r.W.Speech = 1
	r.W.NoHoverAfterSpeech = 2
	if _, err := r.T.OpenForTest(context.Background(), 148); err != nil {
		t.Fatal(err)
	}
	if r.W.UI[d2.UINPCMenu] == 0 {
		t.Fatal("menu not open")
	}
}

// The dialog ignores every click (V live: 18 failed visits, flag set, nothing changes): the visit ends after a few
// clicks and well under 10 s of game time, with a reason, instead of 8 clicks and 35 s.
func TestIgnoredClicksEndTheVisitFast(t *testing.T) {
	r := newRig(t)
	r.W.MenuInert = 1 << 30
	npc, err := r.T.OpenForTest(context.Background(), 148)
	if err != nil {
		t.Fatal(err)
	}
	start := r.Env.Clock.Now()
	err = r.T.PickEntryForTest(context.Background(), npc, town.EntryTrade, d2.UIShop)
	if err == nil || !strings.Contains(err.Error(), "ignored") {
		t.Fatalf("want an 'ignored' failure, got %v", err)
	}
	if d := r.Env.Clock.Now().Sub(start); d > 10*time.Second {
		t.Fatalf("failure cost %v of game time", d)
	}
	if r.W.UI[d2.UINPCMenu] != 0 {
		t.Error("the dialog was left open")
	}
}

// One ignored click (hover not registered) is recovered by the slow-hover retry.
func TestOneIgnoredClickRecovers(t *testing.T) {
	r := newRig(t)
	r.W.MenuInert = 1
	openCharsi(t, r)
	if r.W.UI[d2.UIShop] == 0 {
		t.Fatal("trade window not open")
	}
}

// Gheed's box needs +5 px (V live): the first click already lands.
func TestGheedTradeFirstClick(t *testing.T) {
	r := newRig(t)
	npc, err := r.T.OpenForTest(context.Background(), 147)
	if err != nil {
		t.Skip("no Gheed in the rig")
	}
	clicks := len(r.In.Calls)
	if err := r.T.PickEntryForTest(context.Background(), npc, town.EntryTrade, d2.UIShop); err != nil {
		t.Fatal(err)
	}
	if n := len(r.In.Calls) - clicks; n > 2 {
		t.Errorf("%d input calls for one click", n)
	}
}

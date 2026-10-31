package town

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town/hand"
	"d2lod-bot/internal/town/panel"
)

// StashGoldButton is the coin icon left of the gold amount under the backpack (800x600). With the stash open it
// opens "How much gold would you like to deposit?" with the full amount typed in. VERIFIED LIVE (tooltip
// "Deposit", dialog on screen). The hit area is small (x 484..490); the round buttons at (369,451) and (433,452)
// are CLOSE buttons of the stash and backpack panels, not gold buttons, and the stash's own coin at (165,91)
// opens the withdraw dialog.
var StashGoldButton = panel.Pt{X: 487, Y: 453}

// typer is the optional part of an input.Controller that posts WM_CHAR (the gold amount edit box).
type typer interface{ TypeChar(ch rune) error }

// openStash walks to the stash chest, clicks it with hover confirmation and waits for the stash flag.
// Opening the stash also shows the backpack; the backpack is opened by key if it did not appear.
func (t *Town) openStash(ctx context.Context, a *Act) error {
	var last error = ErrNoNPC
	for try := 0; try < t.tune().NPCOpenTries; try++ {
		s, err := t.current()
		if err != nil {
			return err
		}
		if hand.PanelOpen(s.UI) {
			if err := t.h.CloseAll(ctx); err != nil {
				return err
			}
			continue
		}
		obj, ok := findObject(s, ObjStash)
		if err := t.approach(ctx, obj, ok, t.objPreset(ObjStash)); err != nil {
			last = err
			continue
		}
		if s, err = t.current(); err != nil {
			return err
		}
		if obj, ok = findObject(s, ObjStash); !ok {
			last = ErrNoNPC
			continue
		}
		err = t.e.In.Interact(input.Target{Type: TypeObject, ID: obj.ID, Pos: input.Point{X: obj.X, Y: obj.Y}})
		if err != nil {
			last = fmt.Errorf("click stash: %w", err)
			continue
		}
		// The stash opens the backpack beside it without setting the inventory flag (VERIFIED LIVE: flags
		// 0x19 only), so do not press the inventory key.
		if _, err = t.h.WaitFlag(ctx, d2.UIStash, true); err != nil {
			last = err
			continue
		}
		return nil
	}
	return last
}

// stashItems moves every pack item the pickit wants stashed into the stash. The stash must be open.
func (t *Town) stashItems(ctx context.Context, rep *ChoreReport, insure bool) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	var todo []d2.Unit
	seen := map[uint32]bool{}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if v := t.judge.EvaluateUnit(u); t.wantsStash(s, u, v) {
			todo = append(todo, u)
			seen[u.ID] = true
		}
	}
	if vj, ok := t.judge.(interface {
		Valuables(*d2.Snapshot) loot.ValuablesReport
	}); ok && insure {
		for _, u := range vj.Valuables(s).Stash {
			if !seen[u.ID] {
				todo = append(todo, u)
			}
		}
	}
	// Best first, so a stash that fills up keeps the most valuable items; an item that does not fit stays in the
	// pack (smaller ones may still fit) and is handed to relieveStash instead of failing the chore every visit.
	prio := func(u d2.Unit) int { return t.judge.EvaluateUnit(u).Prio }
	sort.SliceStable(todo, func(i, j int) bool { return prio(todo[i]) > prio(todo[j]) })
	t.stashLeft = nil
	for _, u := range todo {
		err := t.h.Transfer(ctx, u, lay.Inventory, lay.Stash, d2.ContStash)
		switch {
		case err == nil:
			rep.Stashed++
		case errors.Is(err, hand.ErrNoSpace):
			t.stashLeft = append(t.stashLeft, u)
		default:
			return err
		}
	}
	return nil
}

// goldKeep is the gold the pack keeps: StashGoldBase plus level*StashGoldPerLvl.
func (t *Town) goldKeep(level int) int {
	tu := t.tune()
	keep := tu.StashGoldBase + level*tu.StashGoldPerLvl
	if tu.BankFirst != 0 {
		keep = max(keep, t.kitGold()) // BASICSFIX: banking never cuts into the minimal kit
	}
	return keep
}

// StashGoldCap is the most gold the stash holds (classic LoD 1.14d: 2,500,000). UNVERIFIED on the lab: the value
// is from game knowledge and card m1-liveproof measures it. A refused deposit into a stash holding at least
// stashCapFloor gold lowers the cap the bot assumes to the stash gold seen then (stashCapSeen), so a wrong constant
// costs one failed deposit, not one per visit.
const (
	StashGoldCap  = 2_500_000
	stashCapFloor = 100_000
)

// bankAmount clamps a deposit to the room left in the stash and logs once while the stash is full. 0 = nothing to do.
func (t *Town) bankAmount(s *d2.Snapshot, want int) int {
	limit := StashGoldCap
	if t.stashCapSeen > 0 {
		limit = min(limit, t.stashCapSeen)
	}
	room := limit - s.Player.StashGold
	if room <= 0 {
		if !t.stashFull {
			t.stashFull = true
			t.e.Log.Info("town: stash gold is full, deposit skipped", "stash", s.Player.StashGold, "cap", limit, "pack", s.Player.Gold)
		}
		return 0
	}
	t.stashFull = false
	return min(want, room)
}

// stashGold deposits the inventory gold above goldKeep (all of it when that is 0). The stash must
// be open. The dialog opens with the full amount; a smaller amount is typed: backspaces clear the field, digits
// go in as WM_CHAR, Enter confirms.
func (t *Town) stashGold(ctx context.Context, rep *ChoreReport) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	keep := t.goldKeep(s.Player.Level)
	if s.Player.Gold <= keep || s.Player.Gold == 0 {
		return nil
	}
	return t.depositCapped(ctx, s, s.Player.Gold-keep)
}

// depositCapped deposits want gold limited by the stash cap. A deposit the game refuses while the stash holds
// a lot of gold is taken as the cap: it is remembered and logged once and is not an error.
func (t *Town) depositCapped(ctx context.Context, s *d2.Snapshot, want int) error {
	amount := t.bankAmount(s, want)
	if amount <= 0 {
		return nil
	}
	err := t.depositGold(ctx, amount)
	if err != nil && s.Player.StashGold >= stashCapFloor {
		if cur, cerr := t.current(); cerr == nil && cur.Player.StashGold == s.Player.StashGold && cur.Player.Gold == s.Player.Gold {
			t.stashCapSeen = cur.Player.StashGold
			t.stashFull = true
			t.e.Log.Info("town: stash refused the deposit, taking its gold as the cap", "stash", t.stashCapSeen, "wanted", amount)
			return nil
		}
	}
	return err
}

// depositGold moves amount gold from the pack into the stash and confirms it in memory.
func (t *Town) depositGold(ctx context.Context, amount int) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	g0, bank0 := s.Player.Gold, s.Player.StashGold
	if amount > g0 {
		amount = g0
	}
	if err := t.e.In.Click(input.Left, StashGoldButton.X, StashGoldButton.Y); err != nil {
		return err
	}
	t.sleep(t.tune().ClickSettleMs + 250) // the dialog has no UI flag; give it a frame or two to appear
	if amount < g0 {
		ty, ok := t.e.In.(typer)
		if !ok {
			_ = t.e.In.Key("esc")
			return fmt.Errorf("town: partial gold deposit needs a controller that can type")
		}
		for i := 0; i < len(strconv.Itoa(g0))+1; i++ {
			// WM_CHAR 8 is a backspace in the edit box and works with any deployed helper; the helper's
			// "backspace" key name (VK_BACK) exists as well but needs the new d2input.exe.
			if err := ty.TypeChar('\b'); err != nil {
				return err
			}
		}
		for _, ch := range strconv.Itoa(amount) {
			if err := ty.TypeChar(ch); err != nil {
				return err
			}
		}
	}
	if err := t.e.In.Key("enter"); err != nil {
		return err
	}
	_, ok, err := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool { return s.Player.StashGold >= bank0+amount && s.Player.Gold <= g0-amount })
	if err != nil {
		return err
	}
	if !ok {
		_ = t.e.In.Key("esc") // leave a stuck dialog
		return fmt.Errorf("town: gold deposit of %d not confirmed (gold %d, bank %d)", amount, g0, bank0)
	}
	return nil
}

// DepositForTest deposits exactly amount (the stash must be open).
func (t *Town) DepositForTest(ctx context.Context, amount int) error {
	return t.depositGold(ctx, amount)
}

// doStash runs the whole stash visit.
func (t *Town) doStash(ctx context.Context, a *Act, plan ChorePlan, rep *ChoreReport) error {
	if err := t.openStash(ctx, a); err != nil {
		return err
	}
	var first error
	if plan.Stash {
		first = t.stashItems(ctx, rep, plan.Insure)
		if first == nil && len(t.stashLeft) > 0 {
			first = t.relieveStash(ctx, a, rep)
		}
	}
	if plan.Gold {
		if err := t.stashGold(ctx, rep); err != nil && first == nil {
			first = err
		}
	}
	if err := t.h.CloseAll(ctx); err != nil && first == nil {
		first = err
	}
	return first
}

// wantsStash reports whether the pickit wants u stashed now. An unidentified item that needs identifying waits for
// the identify step, except in a full pack: unidentified jewelry is sold then (sellableIf), a charm (valuable) goes to
// the stash as it is, there is nothing else to do with it.
func (t *Town) wantsStash(s *d2.Snapshot, u d2.Unit, v loot.Verdict) bool {
	if !v.NeedID || u.Item.Identified() {
		return v.Action == loot.Stash
	}
	if v.Prio < t.e.Tune.Loot.ValuableMinPrio {
		return false
	}
	lay, err := t.layout(s)
	return err == nil && t.h.Occupancy(s.Inventory, d2.ContInventory, lay.Inventory).UsedPct() >= t.tune().TripPackUsedPct
}

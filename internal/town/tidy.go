package town

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/tidy"
	"d2lod-bot/internal/town/hand"
)

// The Tidy chore (docs/tidy.md): the pure policy of internal/tidy decides, this file builds its facts from the
// snapshot and runs the stash and drop actions through the same hand primitives as the stash chore. Sales stay with the
// sell chore. Equipment, the belt, the cube, the stash and anything the player marked are never touched (the policy
// answers Ignore for them, and only backpack units are ever offered).

var (
	tidyOnce sync.Once
	tidyPol  *tidy.Policy
	tidyErr  error
)

func tidyPolicy() (*tidy.Policy, error) {
	tidyOnce.Do(func() { tidyPol, tidyErr = tidy.Default() })
	return tidyPol, tidyErr
}

var qualityNames = map[int]string{d2.QualityLow: "low", d2.QualityNormal: "normal", d2.QualitySuperior: "superior",
	d2.QualityMagic: "magic", d2.QualitySet: "set", d2.QualityRare: "rare", d2.QualityUnique: "unique", d2.QualityCrafted: "crafted"}

var placeNames = map[d2.Container]string{d2.ContInventory: "pack", d2.ContStash: "stash", d2.ContEquipment: "equipment",
	d2.ContBelt: "belt", d2.ContCube: "cube", d2.ContCursor: "cursor", d2.ContGround: "ground"}

// TidyFacts builds the policy facts of one unit. maxBuy caps the sell estimate (loot.MaxBuyOf); mark is the player's
// mark, passed in so the inventory page (which holds the desk lock) never calls back into the desk.
func (t *Town) TidyFacts(u d2.Unit, maxBuy int, mark string) tidy.Facts {
	f := tidy.Facts{Place: "unknown"}
	if u.Item == nil {
		return f
	}
	f.Place = placeNames[u.Item.Where]
	f.Quality = qualityNames[u.Item.Quality]
	f.Identified, f.Ethereal = u.Item.Identified(), u.Item.Ethereal()
	f.StartItem = u.Item.Flags&d2.FlagStartItem != 0
	f.Mark = mark
	b := t.base(u)
	if b == nil {
		f.Place = "unknown" // an unknown base is never touched
		return f
	}
	f.Code, f.Type, f.Stackable, f.Quest = b.Code, b.Type, b.Stackable, b.Quest || neverSell[b.Code] || b.Type == "key"
	f.SellValue = loot.SellValue(t.e.Data, b, u.Item, maxBuy)
	if b.Sockets >= 4 && (f.Quality == "normal" || f.Quality == "superior") {
		if st, err := t.unitStats(u.Addr); err == nil {
			f.Sockets = int(st.Get(statSockets))
		}
	}
	return f
}

// tidyCtx reads the hero class and how full pack and stash are.
func (t *Town) tidyCtx(s *d2.Snapshot) tidy.Ctx {
	c := tidy.Ctx{Class: s.Player.Class.String()}
	if lay, err := t.layout(s); err == nil {
		c.PackFullPct = t.h.Occupancy(s.Inventory, d2.ContInventory, lay.Inventory).UsedPct()
		c.StashFullPct = t.h.Occupancy(s.Inventory, d2.ContStash, lay.Stash).UsedPct()
	}
	return c
}

// TidyWhyFor binds the tooltip line to one snapshot: the returned func answers "Will stash: rune" for a backpack unit
// and its mark, "" for everything else or when tidying is off. The grid fill is read once, not per item.
func (t *Town) TidyWhyFor(s *d2.Snapshot) func(u d2.Unit, mark string) string {
	if t.tune().TidyEnabled == 0 || s == nil || s.Player == nil || s.Inventory == nil {
		return nil
	}
	p, err := tidyPolicy()
	if err != nil {
		return nil
	}
	c, maxBuy := t.tidyCtx(s), loot.MaxBuyOf(s.Player)
	return func(u d2.Unit, mark string) string {
		if u.Item == nil || u.Item.Where != d2.ContInventory {
			return ""
		}
		return p.Decide(t.TidyFacts(u, maxBuy, mark), c).Why
	}
}

// tidyPlan is the stash and drop actions of the pack now, capped at TidyMaxActions.
func (t *Town) tidyPlan(s *d2.Snapshot) ([]tidy.Entry, map[uint32]d2.Unit, tidy.Ctx, error) {
	p, err := tidyPolicy()
	if err != nil {
		return nil, nil, tidy.Ctx{}, err
	}
	c := t.tidyCtx(s)
	t.ReconcileMarks(s)
	var items []tidy.Item
	byID := map[uint32]d2.Unit{}
	maxBuy := loot.MaxBuyOf(s.Player)
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if u.Item == nil || t.stashRefused[u.ID] {
			continue
		}
		items = append(items, tidy.Item{ID: u.ID, Facts: t.TidyFacts(u, maxBuy, t.markOf(u))})
		byID[u.ID] = u
	}
	return p.Plan(items, c, t.tune().TidyMaxActions), byID, c, nil
}

func (t *Town) needTidy(s *d2.Snapshot) bool {
	if t.tune().TidyEnabled == 0 {
		return false
	}
	plan, _, c, err := t.tidyPlan(s)
	if err != nil || len(plan) == 0 {
		return false
	}
	for _, e := range plan {
		if e.Action == tidy.Stash {
			return true
		}
	}
	return c.PackFullPct >= t.tune().TidyMinPackPct
}

// tidyPass runs the plan: stash entries through the open stash, drop entries on the ground. It stops at the first
// hostile or low-life reading, at the action cap and on a failed drop (a drop that fell back into the pack).
func (t *Town) tidyPass(ctx context.Context, a *Act, rep *ChoreReport) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	plan, byID, c, err := t.tidyPlan(s)
	if err != nil {
		return err
	}
	var nStash, nDrop, nSell int
	for _, e := range plan {
		if e.Action == tidy.Stash {
			nStash++
		} else {
			nDrop++
		}
	}
	if p, perr := tidyPolicy(); perr == nil {
		for _, u := range s.Inventory.Container(d2.ContInventory) {
			if u.Item != nil && p.Decide(t.TidyFacts(u, loot.MaxBuyOf(s.Player), t.markOf(u)), c).Action == tidy.Sell {
				nSell++
			}
		}
	}
	t.e.Log.Info("tidy: plan", "stash", nStash, "drop", nDrop, "sell_left_to_vendor", nSell, "pack_pct", c.PackFullPct, "stash_pct", c.StashFullPct)
	done := map[string]int{}
	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, tidyPassCap)
	defer cancel()
	defer func() { _ = t.h.CloseAll(parent) }()
	var rerr error
	if nStash > 0 {
		rerr = t.tidyStash(ctx, a, plan, byID, done)
	}
	if nDrop > 0 && rerr == nil {
		rerr = t.tidyDrop(ctx, plan, byID, done)
	}
	rep.Tidied += done["stash"] + done["drop"]
	// the 120 s cap is a normal end of the pass, not a failed chore; a cancelled parent or a real error is returned
	if errors.Is(rerr, context.DeadlineExceeded) && parent.Err() == nil {
		t.e.Log.Info("tidy: pass cap reached", "cap", tidyPassCap.String())
		rerr = nil
	}
	if rerr != nil {
		return rerr
	}
	t.e.Log.Info("tidy: done", "stashed", done["stash"], "dropped", done["drop"], "refused", done["refused"], "stopped", done["stopped"])
	return nil
}

// tidyUnsafe is the abort rule: a fight close by, or life under half.
func (t *Town) tidyUnsafe(s *d2.Snapshot) bool {
	return t.fightNear(s) || s.Player.MaxLife > 0 && s.Player.Life*2 < s.Player.MaxLife
}

func (t *Town) tidyStash(ctx context.Context, a *Act, plan []tidy.Entry, byID map[uint32]d2.Unit, done map[string]int) error {
	if err := t.openStash(ctx, a); err != nil {
		return err
	}
	for _, e := range plan {
		if e.Action != tidy.Stash {
			continue
		}
		s, err := t.current()
		if err != nil {
			return err
		}
		if t.tidyUnsafe(s) || ctx.Err() != nil {
			done["stopped"]++
			return ctx.Err()
		}
		lay, err := t.layout(s)
		if err != nil {
			return err
		}
		u, ok := findIn(s.Inventory, e.ID)
		if !ok || u.Item == nil || u.Item.Where != d2.ContInventory || t.markOf(u) != "" {
			done["refused"]++
			continue
		}
		switch err := t.h.Transfer(ctx, u, lay.Inventory, lay.Stash, d2.ContStash); {
		case err == nil:
			done["stash"]++
			t.e.Log.Info("tidy: stashed", "item", t.nameOf(u), "why", e.Why)
		case errors.Is(err, hand.ErrNoSpace):
			t.markRefused([]d2.Unit{byID[e.ID]})
			done["refused"]++
		default:
			return err
		}
	}
	return nil
}

func (t *Town) tidyDrop(ctx context.Context, plan []tidy.Entry, byID map[uint32]d2.Unit, done map[string]int) error {
	if err := t.h.CloseAll(ctx); err != nil {
		return err
	}
	failures := 0
	for _, e := range plan {
		if e.Action != tidy.Drop {
			continue
		}
		s, err := t.current()
		if err != nil {
			return err
		}
		if t.tidyUnsafe(s) || ctx.Err() != nil {
			done["stopped"]++
			return ctx.Err()
		}
		u, ok := findIn(s.Inventory, e.ID)
		if !ok || u.Item == nil || u.Item.Where != d2.ContInventory || t.markOf(u) != "" {
			done["refused"]++
			continue
		}
		if err := t.dropOne(ctx, s, u); err != nil {
			t.e.Log.Warn("tidy: drop failed", "item", t.nameOf(u), "err", err)
			done["refused"]++
			if failures++; failures >= 2 {
				return nil
			}
			continue
		}
		done["drop"]++
		t.tidyDrops.Note(u.ID, tidyPrint(u), s.Player.X, s.Player.Y)
		t.e.Log.Info("tidy: dropped", "item", t.nameOf(u), "why", e.Why)
	}
	return nil
}

// dropOne takes u from the open pack and drops it beside the hero, then proves from memory that it left the pack.
func (t *Town) dropOne(ctx context.Context, s *d2.Snapshot, u d2.Unit) error {
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	if !s.UI.Open(d2.UIInventory) {
		if _, err := t.h.OpenInventory(ctx); err != nil {
			return err
		}
	}
	if err := t.h.Take(ctx, u, lay.Inventory); err != nil {
		_ = t.h.ClearCursor(ctx)
		return err
	}
	if err := t.h.DropCursor(ctx); err != nil {
		return err
	}
	ns, err := t.current()
	if err != nil {
		return err
	}
	if cur, ok := findIn(ns.Inventory, u.ID); ok && cur.Item != nil && cur.Item.Where == d2.ContInventory {
		return fmt.Errorf("item %d is still in the pack", u.ID)
	}
	return nil
}

// tidyPassCap is the SME's hard cap for one tidy pass.
const tidyPassCap = 120 * time.Second

func tidyPrint(u d2.Unit) string {
	if u.Item == nil {
		return ""
	}
	return fmt.Sprintf("%d/%d/%d", u.Class, u.Item.Quality, u.Item.ILvl)
}

// TidyDropped is the looter's skip test for items this chore threw away (loot.Looter.SetIgnore, together with the
// desk's player-trash test). Without it the pickup would fetch the junk again on the next pass.
func (t *Town) TidyDropped(u d2.Unit) bool {
	if t == nil || u.Item == nil {
		return false
	}
	return t.tidyDrops.Ignored(u.ID, tidyPrint(u), u.X, u.Y)
}

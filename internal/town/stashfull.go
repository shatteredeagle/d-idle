package town

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town/hand"
)

// ErrCubeNoRoom: the cube chore needs pack room to pull its inputs from the stash and there is none.
var ErrCubeNoRoom = errors.New("town: no pack room for the cube inputs")

// neverSell lists base codes that are never sold to make room: quest items, the Cube, keys and tomes.
var neverSell = func() map[string]bool {
	m := map[string]bool{"box": true, "msf": true, "vip": true, "hst": true, "tr1": true, "ass": true,
		"key": true, "tbk": true, "ibk": true, "tsc": true, "isc": true}
	for _, c := range loot.QuestItemCodes {
		m[c] = true
	}
	return m
}()

// sellProtected reports whether u must not be sold or dropped to make stash or pack room: quest items, keys, tomes,
// anything the Cube chore consumes (gems, runes), keep verdicts, and everything at or above ValuableMinPrio
// (runes, gems, uniques, good charms), which is what a wanted runeword's runes and the pickit's best finds are.
func (t *Town) sellProtected(u d2.Unit, v loot.Verdict) bool {
	if v.Action == loot.Keep || v.Tier == "S" || v.Prio >= t.e.Tune.Loot.ValuableMinPrio {
		return true
	}
	b := t.h.Base(u)
	if b == nil {
		return true // unknown: keep
	}
	return neverSell[b.Code] || b.Type == "rune" || b.Type == "key" || isCubeItem(b.Code) || b.Code == "" ||
		u.Item.Flags&d2.FlagStartItem != 0
}

type rated struct {
	u    d2.Unit
	prio int
	val  int
	prot bool
}

func (t *Town) rate(u d2.Unit) rated {
	v := t.judge.EvaluateUnit(u)
	return rated{u, v.Prio, loot.SellValue(t.h.Base(u), u.Item.Quality), t.sellProtected(u, v)}
}

func lowestFirst(l []rated) {
	sort.SliceStable(l, func(i, j int) bool {
		if l[i].prio != l[j].prio {
			return l[i].prio < l[j].prio
		}
		return l[i].val < l[j].val
	})
}

// relieveStash runs after stashItems left items in the pack because the stash is full. Pack items that are not
// protected are sold; for each protected one that did not fit, the lowest-priority unprotected stash item of lower
// priority is moved to the pack and sold, then the stash is filled again. What cannot be placed stays in the pack
// and is not retried until the pack changes (stashRefused). The stash is open on entry and on return.
func (t *Town) relieveStash(ctx context.Context, a *Act, rep *ChoreReport) error {
	left := t.stashLeft
	t.stashLeft = nil
	s, err := t.current()
	if err != nil {
		return err
	}
	var sell, prot, pool []rated
	for _, u := range left {
		if r := t.rate(u); r.prot {
			prot = append(prot, r)
		} else {
			sell = append(sell, r)
		}
	}
	for _, u := range s.Inventory.Container(d2.ContStash) {
		if r := t.rate(u); !r.prot {
			pool = append(pool, r)
		}
	}
	lowestFirst(pool)
	sort.SliceStable(prot, func(i, j int) bool { return prot[i].prio > prot[j].prio })
	var pull []rated
	for i, p := range prot {
		if i < len(pool) && pool[i].prio < p.prio {
			pull = append(pull, pool[i])
		}
	}
	t.e.Log.Info("town: stash full", "left", len(left), "sell_from_pack", len(sell), "swap_from_stash", len(pull), "protected_unplaced", len(prot)-len(pull))
	if len(sell) == 0 && len(pull) == 0 {
		t.markRefused(left)
		return nil
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	for _, r := range pull {
		cur, ok := findIn(s.Inventory, r.u.ID)
		if !ok {
			cur = r.u
		}
		if err := t.h.Transfer(ctx, cur, lay.Stash, lay.Inventory, d2.ContInventory); err != nil {
			if errors.Is(err, hand.ErrNoSpace) {
				t.e.Log.Info("town: stash swap stopped, pack has no room")
				break
			}
			return err
		}
		sell = append(sell, r)
	}
	if err := t.h.CloseAll(ctx); err != nil {
		return err
	}
	list := make([]d2.Unit, 0, len(sell))
	for _, r := range sell {
		list = append(list, r.u)
	}
	n0 := rep.Sold
	refused, serr := t.sellAtVendors(ctx, a, list, rep)
	t.e.Log.Info("town: stash full sale", "sold", rep.Sold-n0, "refused", len(refused))
	if serr != nil {
		return serr
	}
	if rep.Sold == n0 && len(refused) == 0 {
		t.markRefused(left)
		return nil
	}
	if err := t.openStash(ctx, a); err != nil {
		return err
	}
	if err := t.stashItems(ctx, rep, false); err != nil {
		return err
	}
	t.markRefused(t.stashLeft) // one round only: whatever still does not fit waits for the pack to change
	t.markRefused(refused)
	t.stashLeft = nil
	return nil
}

func (t *Town) markRefused(l []d2.Unit) {
	if t.stashRefused == nil {
		t.stashRefused = map[uint32]bool{}
	}
	for _, u := range l {
		t.stashRefused[u.ID] = true
	}
}

// sellAtVendors offers list to the act's vendors in turn and returns what nobody bought.
func (t *Town) sellAtVendors(ctx context.Context, a *Act, list []d2.Unit, rep *ChoreReport) ([]d2.Unit, error) {
	tried := map[int]bool{}
	for _, class := range a.Sell {
		if len(list) == 0 {
			break
		}
		if tried[class] {
			continue
		}
		tried[class] = true
		if _, err := t.openShop(ctx, []int{class}, t.npcPreset(class)); err != nil {
			continue
		}
		refused, err := t.sellTo(ctx, list, rep)
		if cerr := t.h.CloseAll(ctx); cerr != nil && err == nil {
			err = cerr
		}
		if err != nil {
			return refused, err
		}
		list = refused
	}
	return list, nil
}

// cubeSig summarises what decides whether the cube chore can run: the item counts and the free pack cells.
func (t *Town) cubeSig(s *d2.Snapshot, counts map[string]int) string {
	free := -1
	if lay, err := t.layout(s); err == nil {
		free = t.h.Occupancy(s.Inventory, d2.ContInventory, lay.Inventory).FreeCells()
	}
	return fmt.Sprint(counts, free)
}

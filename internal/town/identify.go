package town

import (
	"context"
	"errors"
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/hand"
	"d2lod-bot/internal/town/panel"
)

// idTool finds an identify scroll or tome in the pack.
func (t *Town) idTool(s *d2.Snapshot) (d2.Unit, bool) {
	var tome d2.Unit
	haveTome := false
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		switch b := t.base(u); {
		case b == nil:
		case b.Code == "isc":
			return u, true
		case b.Code == "ibk" && t.qty(u) > 0:
			tome, haveTome = u, true
		}
	}
	return tome, haveTome
}

// identify identifies the pack items that need it, with identify scrolls or a tome (right-click the tool,
// then left-click the item). Scrolls are bought first when none are held. Cain is not used: his dialog is
// not mapped (docs/town.md). UNVERIFIED live: the whole flow.
func (t *Town) identify(ctx context.Context, a *Act, rep *ChoreReport) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	todo := t.needIdentify(s)
	if _, ok := t.idTool(s); !ok {
		if err := t.buyIDScrolls(ctx, a, len(todo), rep); err != nil {
			return err
		}
	}
	var first error
	for _, u := range todo {
		if err := t.identifyOne(ctx, u); err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		rep.Identified++
	}
	if err := t.h.CloseAll(ctx); err != nil && first == nil {
		first = err
	}
	return first
}

func (t *Town) buyIDScrolls(ctx context.Context, a *Act, n int, rep *ChoreReport) error {
	npc, err := t.openShop(ctx, a.Shop, t.npcPreset(a.Shop...))
	if err != nil {
		return err
	}
	defer func() { _ = t.h.CloseAll(ctx) }()
	s, err := t.current()
	if err != nil {
		return err
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	stock, err := t.Stock(npc)
	if err != nil {
		return err
	}
	for _, u := range stock {
		b := t.base(u)
		if b == nil || b.Code != "isc" {
			continue
		}
		for i := 0; i < n; i++ {
			price, err := t.buyUnit(ctx, lay, u)
			if errors.Is(err, ErrNoMoney) {
				rep.Broke = true
				return nil
			}
			if err != nil {
				return err
			}
			rep.Spent += price
			rep.Bought[t.nameOf(u)]++
		}
		return nil
	}
	return fmt.Errorf("town: npc %d does not sell identify scrolls", npc.Class)
}

// identifyPoint is where the identify cursor clicks: the item's cell in the pack, or the centre of its doll slot when it
// is worn (U live: a worn item is read through the slot rectangle of the inventory panel).
func identifyPoint(h *hand.Hand, lay *panel.Layout, u d2.Unit) (panel.Pt, error) {
	if u.Item != nil && u.Item.Where == d2.ContEquipment {
		r, ok := lay.Slots[hand.SlotName(u.Item.Body)]
		if !ok {
			return panel.Pt{}, fmt.Errorf("town: no slot rectangle for body location %d", u.Item.Body)
		}
		return r.Center(), nil
	}
	w, hh := h.Size(u)
	return lay.Inventory.ItemCenter(u.X, u.Y, w, hh), nil
}

func (t *Town) identifyOne(ctx context.Context, target d2.Unit) error {
	for try := 0; try < t.tune().MoveTries; try++ {
		s, err := t.h.OpenInventory(ctx)
		if err != nil {
			return err
		}
		tool, ok := t.idTool(s)
		if !ok {
			return fmt.Errorf("town: out of identify scrolls")
		}
		lay, err := t.layout(s)
		if err != nil {
			return err
		}
		cur, ok := findIn(s.Inventory, target.ID)
		if !ok {
			return fmt.Errorf("town: item to identify vanished")
		}
		if err := t.h.RightClick(tool, lay.Inventory); err != nil {
			return err
		}
		t.sleep(t.tune().ClickSettleMs)
		click, err := identifyPoint(t.h, lay, cur)
		if err != nil {
			return err
		}
		if err := t.e.In.Click(input.Left, click.X, click.Y); err != nil {
			return err
		}
		_, done, err := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool {
			u, ok := findIn(s.Inventory, target.ID)
			return ok && u.Item.Identified()
		})
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		_ = t.e.In.Key("esc") // leave identify-cursor mode
		t.sleep(t.tune().ClickSettleMs)
	}
	return fmt.Errorf("town: identify of item %d not confirmed", target.ID)
}

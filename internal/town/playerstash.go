package town

import (
	"context"
	"errors"
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/invdesk"
	"d2lod-bot/internal/town/hand"
)

// The player's stash actions (docs/inventory-actions.md, stash actions; API schema 6). One StashVisit opens the stash
// once and runs every queued job: moves between the backpack and the stash (hand.Transfer, the path the stash chore
// uses), a gold move (the chore's own deposit and withdraw) and the identify of stash items in place. The moves and the
// identify click paths are U live: nothing here ran against a game.

var _ invdesk.StashServices = (*Town)(nil)

// ErrNoRoomPack and ErrNoRoomStash are the refusals of a move without a free spot of the item's size.
var (
	ErrNoRoomPack  = errors.New("the backpack has no free spot for it: make room first")
	ErrNoRoomStash = errors.New("the stash has no free spot for it")
)

func (t *Town) findUnit(s *d2.Snapshot, id uint32) (d2.Unit, bool) {
	if s == nil || s.Inventory == nil {
		return d2.Unit{}, false
	}
	return findIn(s.Inventory, id)
}

// CheckStashMove is invdesk.StashServices. Reads the snapshot only.
func (t *Town) CheckStashMove(s *d2.Snapshot, id uint32, toStash bool) (string, error) {
	if s == nil || s.Player == nil || s.Inventory == nil {
		return "", errNoHero
	}
	u, ok := t.findUnit(s, id)
	if !ok || u.Item == nil {
		return "", errors.New("the item is not in the inventory")
	}
	from, to, no := d2.ContStash, d2.ContInventory, ErrNoRoomPack
	if toStash {
		from, to, no = d2.ContInventory, d2.ContStash, ErrNoRoomStash
	}
	if u.Item.Where != from {
		return "", errors.New("the item is not where the move starts")
	}
	lay, err := t.layout(s)
	if err != nil {
		return "", err
	}
	grid := lay.Inventory
	if toStash {
		grid = lay.Stash
	}
	w, hh := t.h.Size(u)
	if _, _, found := t.h.Occupancy(s.Inventory, to, grid).FindSpot(w, hh); !found {
		return "", no
	}
	return t.nameOf(u), nil
}

// CheckStashIdentify is invdesk.StashServices.
func (t *Town) CheckStashIdentify(s *d2.Snapshot, id uint32) (string, error) {
	if s == nil || s.Player == nil || s.Inventory == nil {
		return "", errNoHero
	}
	u, ok := t.findUnit(s, id)
	switch {
	case !ok || u.Item == nil || u.Item.Where != d2.ContStash:
		return "", errors.New("the item is not in the stash")
	case u.Item.Identified():
		return "", errors.New("it is identified already")
	}
	if _, ok := t.idTool(s); !ok {
		return "", ErrNoIDTool
	}
	return t.nameOf(u), nil
}

// StashVisit is invdesk.StashServices: open the stash, run the plan, close.
func (t *Town) StashVisit(ctx context.Context, p invdesk.StashPlan) invdesk.StashResult {
	res := invdesk.StashResult{Errs: map[uint32]error{}}
	ids := append(append(append([]uint32{}, p.Take...), p.Stow...), p.Identify...)
	failAll := func(err error) invdesk.StashResult {
		for _, id := range ids {
			res.Errs[id] = err
		}
		if p.Gold > 0 {
			res.GoldErr = err
		}
		return res
	}
	s, err := t.current()
	if err == nil && (s.Player == nil || s.Inventory == nil) {
		err = errNoHero
	}
	if err != nil {
		return failAll(err)
	}
	a := ActOfLevel(s.Player.Area)
	if a == nil {
		return failAll(ErrNotInTown)
	}
	if t.fightNear(s) {
		return failAll(ErrFight)
	}
	// A held item would block every click: put it into the pack first (a refusal fails the visit with that reason).
	if s.Inventory.Cursor != nil {
		if err := t.h.ClearCursor(ctx); err != nil {
			return failAll(fmt.Errorf("an item is on the cursor and the pack cannot take it: %w", err))
		}
	}
	if err := t.h.CloseAll(ctx); err != nil {
		return failAll(err)
	}
	if err := t.openStash(ctx, a); err != nil {
		return failAll(err)
	}
	// Close on a context that survives a cancel: the panels must close and the cursor be released.
	defer func() {
		if cerr := t.h.CloseAll(context.WithoutCancel(ctx)); cerr != nil {
			t.e.Log.Warn("town: closing the stash after the player's stash visit failed", "err", cerr)
		}
	}()
	// Takes first (they free stash cells and need pack room), then stows, then gold, then the in-place identifies.
	for _, id := range p.Take {
		res.Errs[id] = t.moveOne(ctx, id, false)
	}
	for _, id := range p.Stow {
		res.Errs[id] = t.moveOne(ctx, id, true)
	}
	if p.Gold > 0 {
		res.Gold, res.GoldErr = t.moveGold(ctx, p.Gold, p.Deposit)
	}
	for _, id := range p.Identify {
		res.Errs[id] = t.identifyInStash(ctx, id)
	}
	return res
}

func (t *Town) moveOne(ctx context.Context, id uint32, toStash bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s, err := t.current()
	if err != nil {
		return err
	}
	if t.fightNear(s) {
		return ErrFight
	}
	u, ok := t.findUnit(s, id)
	if !ok {
		return errors.New("the item is not in the inventory any more")
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	src, dst, cont, no := lay.Stash, lay.Inventory, d2.ContInventory, ErrNoRoomPack
	if toStash {
		src, dst, cont, no = lay.Inventory, lay.Stash, d2.ContStash, ErrNoRoomStash
	}
	switch err := t.h.Transfer(ctx, u, src, dst, cont); {
	case err == nil:
		t.e.Log.Info("player stash move", "item", t.nameOf(u), "toStash", toStash)
		return nil
	case errors.Is(err, hand.ErrNoSpace):
		return no
	default:
		return err
	}
}

// moveGold moves amount gold and proves it from memory; the deposit is clamped to the stash cap by depositCapped, so the
// sentence reports what actually moved.
func (t *Town) moveGold(ctx context.Context, amount int, deposit bool) (string, error) {
	s, err := t.current()
	if err != nil {
		return "", err
	}
	g0, b0 := s.Player.Gold, s.Player.StashGold
	if deposit {
		err = t.depositCapped(ctx, s, amount)
	} else {
		err = t.withdrawOpen(ctx, s, min(amount, b0, max(0, s.Player.Level*invdesk.WithdrawPerLevel-g0)))
	}
	if err != nil {
		return "", err
	}
	e, err := t.current()
	if err != nil {
		return "", err
	}
	moved := e.Player.StashGold - b0
	verb := "deposited"
	if !deposit {
		moved, verb = e.Player.Gold-g0, "withdrew"
	}
	if moved <= 0 {
		return "", fmt.Errorf("no gold moved (the game kept the amounts: pack %d, stash %d)", e.Player.Gold, e.Player.StashGold)
	}
	return fmt.Sprintf("%s %d gold", verb, moved), nil
}

// withdrawOpen is withdrawAmount for an open stash: the game's own gold command, confirmed in memory.
func (t *Town) withdrawOpen(ctx context.Context, s *d2.Snapshot, amount int) error {
	if amount <= 0 {
		return nil
	}
	gm, ok := t.e.In.(goldMover)
	if !ok {
		return ErrNoGoldMover
	}
	if err := gm.MoveGold(amount, false); err != nil {
		_ = t.e.In.Key("esc")
		return fmt.Errorf("town: gold withdrawal of %d: %w", amount, err)
	}
	g0 := s.Player.Gold
	_, ok, err := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool { return s.Player.Gold >= g0+amount })
	if err != nil {
		return err
	}
	if !ok {
		_ = t.e.In.Key("esc")
		return fmt.Errorf("town: gold withdrawal of %d not confirmed", amount)
	}
	return nil
}

// identifyInStash reads a Scroll or the Tome of Identify on a stash item where it lies. The stash is open and shows the
// backpack beside it (the tool is right-clicked there, the identify cursor then clicks the item's stash cell).
// U live: that the game accepts the identify cursor on stash cells (S: it does in the single-player game).
func (t *Town) identifyInStash(ctx context.Context, id uint32) error {
	for try := 0; try < t.tune().MoveTries; try++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		s, err := t.current()
		if err != nil {
			return err
		}
		if t.fightNear(s) {
			return ErrFight
		}
		cur, ok := t.findUnit(s, id)
		switch {
		case !ok:
			return errors.New("the item is not in the stash any more")
		case cur.Item.Identified():
			return nil
		case cur.Item.Where != d2.ContStash:
			return errors.New("the item is not in the stash")
		}
		tool, ok := t.idTool(s)
		if !ok {
			return ErrNoIDTool
		}
		lay, err := t.layout(s)
		if err != nil {
			return err
		}
		if err := t.h.RightClick(tool, lay.Inventory); err != nil {
			return err
		}
		t.sleep(t.tune().ClickSettleMs)
		w, hh := t.h.Size(cur)
		c := lay.Stash.ItemCenter(cur.X, cur.Y, w, hh)
		if err := t.e.In.Click(input.Left, c.X, c.Y); err != nil {
			return err
		}
		_, done, err := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool {
			u, ok := findIn(s.Inventory, id)
			return ok && u.Item.Identified()
		})
		if err != nil {
			return err
		}
		if done {
			t.e.Log.Info("player identify in the stash", "item", t.nameOf(cur))
			return nil
		}
		_ = t.e.In.Key("esc") // leave identify-cursor mode
		t.sleep(t.tune().ClickSettleMs)
	}
	return fmt.Errorf("town: identify of stash item %d not confirmed", id)
}

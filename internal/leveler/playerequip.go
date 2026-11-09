package leveler

import (
	"context"
	"errors"
	"fmt"

	"d2lod-bot/internal/d2"
)

// Player-requested equips (docs/inventory-ui.md). The watch screen queues "wear this item in that slot"; the bot calls
// PlayerEquip at a calm point. The click path is the one Equip uses (doSwapRetry: pick up, click the slot, prove the
// item is worn); nothing here writes game memory.

// pinnedScore makes a pinned slot unbeatable for ChooseGear, so Equip and Shop never plan a swap away from what the
// player put there. A finite value: scores are added and compared.
const pinnedScore = 1e9

var (
	ErrNoItem   = errors.New("the item is not in the backpack")
	ErrNoSlot   = errors.New("the item does not fit that slot")
	ErrOffHand  = errors.New("a two-handed weapon and a shield cannot be worn together; take the other one off first")
	ErrNoPlayer = errors.New("no character data yet")
)

// CheckPlayerEquip says why the pack item id cannot be worn in body slot body now; nil means it can. Requirements are
// the ones the bot's own gear logic uses (class, level, strength, dexterity, identified), except that a low quality
// weapon is allowed: wearing it is the player's call.
func (l *Leveler) CheckPlayerEquip(s *d2.Snapshot, id uint32, body int) (name string, err error) {
	if s == nil || s.Player == nil || s.Inventory == nil {
		return "", ErrNoPlayer
	}
	var u d2.Unit
	found := false
	for _, x := range s.Inventory.Container(d2.ContInventory) {
		if x.ID == id {
			u, found = x, true
		}
	}
	if !found || u.Item == nil {
		return "", ErrNoItem
	}
	c := l.cand(s.Player, u)
	name = c.Name
	if c.Why == "low quality" {
		c.Why, c.Usable = "", true
	}
	if !c.Usable {
		return name, fmt.Errorf("%s: %s", name, c.Why)
	}
	ok := false
	for _, sl := range c.Slots {
		if sl == body {
			ok = true
		}
	}
	if !ok {
		return name, fmt.Errorf("%s: %w", name, ErrNoSlot)
	}
	eq, _ := l.gearState(s)
	other := map[int]int{d2.BodyRHand: d2.BodyLHand, d2.BodyLHand: d2.BodyRHand}[body]
	if o, has := eq[other]; has && other != 0 && (c.TwoHanded || o.TwoHanded) {
		return name, ErrOffHand
	}
	return name, nil
}

// PlayerEquip wears pack item id in body slot body through the normal click path, after the same checks. The caller
// picks the moment (town, no hostiles); a fight nearby returns ErrBusy.
func (l *Leveler) PlayerEquip(ctx context.Context, id uint32, body int) error {
	h := l.newHand()
	if !l.o.Dry {
		if _, err := h.EnsureMainSet(ctx); err != nil {
			return err
		}
	}
	s, err := h.Fresh()
	if err != nil {
		return err
	}
	if l.busy(s) {
		return ErrBusy
	}
	if _, err := l.CheckPlayerEquip(s, id, body); err != nil {
		return err
	}
	var u d2.Unit
	for _, x := range s.Inventory.Container(d2.ContInventory) {
		if x.ID == id {
			u = x
		}
	}
	c := l.cand(s.Player, u)
	c.Usable = true
	sw := Swap{Item: c, Slot: body}
	eq, _ := l.gearState(s)
	if o, ok := eq[body]; ok {
		sw.Old = &o
	}
	if l.o.Dry {
		return nil
	}
	err = l.doSwapRetry(ctx, h, s, sw)
	if cerr := h.CloseAll(ctx); cerr != nil {
		l.e.Log.Warn("leveler: closing panels after the player's equip failed", "err", cerr)
	}
	if err == nil {
		l.logf("player equip", "swap", sw.String())
	}
	return err
}

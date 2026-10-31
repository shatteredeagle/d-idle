package town

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear"
	"d2lod-bot/internal/input"
)

// Ammo and ranged gear (docs/gear.md). Charsi's Misc tab sells Arrows and Bolts (V live, smoke-amazon finding 3); the
// quiver is worn in the left hand, body slot 5, replacing the Buckler (V live). Buying and the slot click are U until
// a live Amazon run (checklist in docs/gear.md).

// AmmoPath is "bow" or "javelin" for an Amazon (set by cmd/d2bot from the leveler build name), "" for every other
// class and for an Amazon that is not told: then only a worn launcher or thrown stack is restocked.
func (t *Town) SetAmmoPath(path string) { t.ammoPath = path }

// gearOf reads the worn main and left hand as the gear package wants them.
func (t *Town) gearOf(s *d2.Snapshot) (gear.Equipped, bool) {
	var e gear.Equipped
	if s == nil || s.Inventory == nil || t.e.Data == nil {
		return e, false
	}
	one := func(body int) (string, int) {
		u := s.Inventory.Equipped(body)
		if u == nil {
			return "", 0
		}
		b := t.base(*u)
		if b == nil {
			return "", 0
		}
		if _, ok := gear.BaseByCode(b.Code); !ok {
			return b.Code, 0
		}
		return b.Code, t.qty(*u)
	}
	e.Weapon, e.WeaponQty = one(d2.BodyRHand)
	e.Offhand, e.OffhandQty = one(d2.BodyLHand)
	return e, true
}

// packCodes lists the base codes of the pack items (a bought bow or quiver that is not worn yet).
func (t *Town) packCodes(s *d2.Snapshot) map[string]d2.Unit {
	out := map[string]d2.Unit{}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if b := t.base(u); b != nil {
			out[b.Code] = u
		}
	}
	return out
}

// AmmoPlan is the pure decision: what to buy for a character wearing e with pack codes in pack. A bow-path Amazon
// without any launcher gets gear.StartPurchases; otherwise gear.Restock. Lines whose item is already in the pack are
// dropped (equip it instead). Needs no game data beyond gear's tables.
func AmmoPlan(class, path string, e gear.Equipped, pack map[string]bool) []gear.Purchase {
	var want []gear.Purchase
	if gear.ClassCode(class) == "ama" && path == "bow" && !e.Launcher() {
		// The bow-path Amazon wears a launcher: one already bought (still in the pack) only needs a quiver.
		launcher := ""
		for code := range pack {
			if (gear.Equipped{Weapon: code}).Launcher() {
				launcher = code
			}
		}
		if launcher == "" {
			want = gear.StartPurchases(class, path)
		} else {
			want = gear.Restock(gear.Equipped{Weapon: launcher, Offhand: e.Offhand, OffhandQty: e.OffhandQty})
		}
	} else {
		want = gear.Restock(e)
	}
	var out []gear.Purchase
	for _, p := range want {
		if !pack[p.Code] {
			out = append(out, p)
		}
	}
	return out
}

// ammoTodo is the chore's need check: something to buy or a bow/quiver in the pack that should be worn.
func (t *Town) ammoTodo(s *d2.Snapshot) bool {
	if s == nil || s.Player == nil || s.Inventory == nil {
		return false
	}
	e, ok := t.gearOf(s)
	if !ok {
		return false
	}
	pk := t.packCodes(s)
	codes := map[string]bool{}
	for c := range pk {
		codes[c] = true
	}
	class := s.Player.Class.String()
	if len(AmmoPlan(class, t.ammoPath, e, codes)) > 0 {
		return true
	}
	return len(t.equipTodo(class, e, pk)) > 0
}

type equipJob struct {
	unit d2.Unit
	slot string // layout slot name
	body int
}

// equipTodo lists pack items to wear: a launcher into the right hand when none is worn (bow path only) and a
// matching quiver into the left hand when the launcher has none.
func (t *Town) equipTodo(class string, e gear.Equipped, pack map[string]d2.Unit) []equipJob {
	var jobs []equipJob
	if gear.ClassCode(class) != "ama" {
		return nil
	}
	launcher := e.Weapon
	if t.ammoPath == "bow" && !e.Launcher() {
		for code, u := range pack {
			if (gear.Equipped{Weapon: code}).Launcher() {
				jobs = append(jobs, equipJob{u, "rArm", d2.BodyRHand})
				launcher = code
				break
			}
		}
	}
	if (gear.Equipped{Weapon: launcher}).Launcher() {
		if b, code, ok := gear.AmmoFor(launcher); ok {
			quiverWorn := e.Offhand != "" && gear.IsA(gear.TypeOf(e.Offhand), b.Type)
			if u, have := pack[code]; have && !quiverWorn {
				jobs = append(jobs, equipJob{u, "lArm", d2.BodyLHand})
			}
		}
	}
	return jobs
}

// buyAmmo visits the act's ammo vendor and buys the AmmoPlan lines, then wears bow and quiver.
func (t *Town) buyAmmo(ctx context.Context, a *Act, rep *ChoreReport) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	e, _ := t.gearOf(s)
	pk := t.packCodes(s)
	codes := map[string]bool{}
	for c := range pk {
		codes[c] = true
	}
	class := s.Player.Class.String()
	buys := AmmoPlan(class, t.ammoPath, e, codes)
	var first error
	if len(buys) > 0 {
		first = t.buyFrom(ctx, a, buys, rep)
	}
	if err := t.wearRanged(ctx, class); err != nil && first == nil {
		first = err
	}
	return first
}

func (t *Town) buyFrom(ctx context.Context, a *Act, buys []gear.Purchase, rep *ChoreReport) error {
	v, ok := gear.VendorForAct(a.No)
	if !ok {
		return fmt.Errorf("town: no ammo vendor for act %d", a.No)
	}
	npc, err := t.openShop(ctx, []int{v.NPC}, t.npcPreset(v.NPC))
	if err != nil {
		return err
	}
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
	for _, p := range buys {
		var it d2.Unit
		found := false
		for _, u := range stock {
			if b := t.base(u); b != nil && b.Code == p.Code {
				it, found = u, true
				break
			}
		}
		if !found {
			t.e.Log.Warn("town: ammo vendor does not stock", "code", p.Code, "vendor", v.Name)
			continue
		}
		price, err := t.buyUnit(ctx, lay, it)
		if errors.Is(err, ErrNoMoney) {
			rep.Broke = true
			return nil
		}
		if err != nil {
			return err
		}
		rep.Spent += price
		rep.Bought[t.nameOf(it)]++
	}
	return nil
}

// wearRanged puts a bought launcher and quiver on, one verified click each (same proof as leveler.doSwap).
func (t *Town) wearRanged(ctx context.Context, class string) error {
	for i := 0; i < 2; i++ { // launcher first, then the quiver that fits it: plan again after each click
		s, err := t.current()
		if err != nil {
			return err
		}
		e, _ := t.gearOf(s)
		jobs := t.equipTodo(class, e, t.packCodes(s))
		if len(jobs) == 0 {
			return nil
		}
		if err := t.wear(ctx, jobs[0]); err != nil {
			return err
		}
	}
	return nil
}

func (t *Town) wear(ctx context.Context, j equipJob) error {
	s, err := t.h.OpenInventory(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = t.h.CloseAll(ctx) }()
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	r, ok := lay.Slots[j.slot]
	if !ok {
		return fmt.Errorf("town: no screen position for slot %s", j.slot)
	}
	if s.Inventory.Cursor != nil {
		if err := t.h.ClearCursor(ctx); err != nil {
			return fmt.Errorf("cursor not empty: %w", err)
		}
	}
	if err := t.h.Take(ctx, j.unit, lay.Inventory); err != nil {
		return err
	}
	c := r.Center()
	if err := t.e.In.MoveMouse(c.X, c.Y); err != nil {
		return err
	}
	t.e.Clock.Sleep(450 * time.Millisecond)
	if err := t.e.In.Click(input.Left, c.X, c.Y); err != nil {
		return err
	}
	id := j.unit.ID
	s2, done, err := t.h.Wait(ctx, 2*time.Second, func(s *d2.Snapshot) bool {
		it, found := findIn(s.Inventory, id)
		return found && it.Item != nil && it.Item.Where == d2.ContEquipment && it.Item.Body == j.body
	})
	if err != nil {
		return err
	}
	if !done {
		_ = t.h.ClearCursor(ctx)
		return fmt.Errorf("town: item %d did not reach slot %s", id, j.slot)
	}
	if s2.Inventory.Cursor != nil { // the displaced Buckler or Javelin
		if err := t.h.ClearCursor(ctx); err != nil {
			return fmt.Errorf("displaced item stays on the cursor: %w", err)
		}
	}
	return nil
}

// bowTripGold is the gold at which a bow-path Amazon still fighting with the starting Javelin makes a town trip for a
// Short Bow (about 100) and arrows (BUILDV2, SME: before any potion top-up beyond the 4 starting ones; U price of the quiver).
const bowTripGold = 150

// wantsBow says whether the character is a bow-path Amazon with no launcher worn or in the pack and the gold to buy one.
func (t *Town) wantsBow(s *d2.Snapshot) bool {
	if t.ammoPath != "bow" || s.Player.Class != d2.Amazon || s.Player.Gold < bowTripGold {
		return false
	}
	e, ok := t.gearOf(s)
	if !ok || e.Launcher() {
		return false
	}
	for code := range t.packCodes(s) {
		if (gear.Equipped{Weapon: code}).Launcher() {
			return false
		}
	}
	return true
}

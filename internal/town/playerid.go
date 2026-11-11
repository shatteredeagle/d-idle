package town

import (
	"context"
	"errors"
	"fmt"

	"d2lod-bot/internal/d2"
)

// The player's identify commands (docs/inventory-actions.md, TASK-298). Identifying is a player-commanded action, not an
// equipment decision (docs/gear-policy.md). Three paths, all real input, nothing writes game memory:
//   - PlayerIdentify: right-click a Scroll or Tome of Identify, left-click the item (the proven chore path, identifyOne);
//   - BuyIdentify: a shopping errand at the act's scroll vendor (Act.Shop, so Akara in Act 1, Drognan in Act 2 and so on:
//     the table decides, nothing is named here), limited to the gold above the repair and kit reserve;
//   - CainIdentify: open Deckard Cain's dialog and pick Identify, which identifies every carried item for free (S: the
//     game's Cain menu; the line index is U live).

var (
	// ErrFight postpones a field identify while a monster is close (the click path needs a quiet hero).
	ErrFight = errors.New("town: a fight is on")
	// ErrNoIDTool is the failure of an identify without a scroll or tome; the text names the other two commands.
	ErrNoIDTool = errors.New("no Scroll or Tome of Identify in the backpack: use Buy identify scrolls or Identify all (Cain)")

	errNoHero = errors.New("town: no character data yet")
)

const (
	idTomeCap        = tomeCap // a Tome of Identify holds 20 scrolls (S: D2 rules, same as the portal tome)
	idScrollsDefault = 5       // scrolls an unplanned errand buys when no tome is held and none is affordable
	idFightRadius    = 25      // subtiles: a live monster this close postpones a field identify (leveler.busy uses the same)
)

// cainMenu is Cain's dialog after the rescue: Talk, Identify, Cancel. S: the game's Cain menu has an Identify line (the
// quest tables list the same rows, quest/dialog.go, all U). The default menu has the same Talk and Cancel positions.
var cainMenu = []Entry{EntryTalk, EntryIdentify, EntryCancel}

func init() {
	for _, c := range cain {
		if _, ok := MenuOrder[c]; !ok {
			MenuOrder[c] = cainMenu
		}
	}
}

// IsCainClass reports whether a monster class is one of Deckard Cain's (acts.go lists them per act).
func IsCainClass(class int) bool { return hasClass(cain, class) }

func (t *Town) fightNear(s *d2.Snapshot) bool {
	if s.Player.InTown() {
		return false
	}
	for i := range s.Monsters {
		m := &s.Monsters[i]
		if m.Alive() && max(abs(m.X-s.Player.X), abs(m.Y-s.Player.Y)) <= idFightRadius {
			return true
		}
	}
	return false
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// unidentified lists the carried and worn items that still need identifying.
func unidentified(s *d2.Snapshot) []d2.Unit {
	var out []d2.Unit
	for _, c := range []d2.Container{d2.ContInventory, d2.ContEquipment} {
		for _, u := range s.Inventory.Container(c) {
			if u.Item != nil && !u.Item.Identified() {
				out = append(out, u)
			}
		}
	}
	return out
}

// CheckPlayerIdentify says why item id cannot be identified now; nil means it can. It returns the item's name.
func (t *Town) CheckPlayerIdentify(s *d2.Snapshot, id uint32) (string, error) {
	if s == nil || s.Player == nil || s.Inventory == nil {
		return "", errNoHero
	}
	for _, c := range []d2.Container{d2.ContInventory, d2.ContEquipment} {
		for _, u := range s.Inventory.Container(c) {
			if u.ID != id || u.Item == nil {
				continue
			}
			if u.Item.Identified() {
				return "", errors.New("it is identified already")
			}
			if _, ok := t.idTool(s); !ok {
				return "", ErrNoIDTool
			}
			return t.nameOf(u), nil
		}
	}
	return "", errors.New("the item is not in the backpack or worn")
}

// PlayerIdentifyBatch identifies the items in one inventory-open: the panel is opened once, every item is read in turn
// and the panels close at the end. One error per id (nil = identified). Without a scroll or tome every entry fails at
// once with ErrNoIDTool (fail fast, no clicking). A fight close by returns ErrFight for the items not yet done, so the
// caller puts them back to queued. The caller picks the moment (town, or the field with nothing hostile near).
func (t *Town) PlayerIdentifyBatch(ctx context.Context, ids []uint32) []error {
	errs := make([]error, len(ids))
	fail := func(from int, err error) []error {
		for i := from; i < len(ids); i++ {
			errs[i] = err
		}
		return errs
	}
	s, err := t.h.Fresh()
	if err == nil && (s.Player == nil || s.Inventory == nil) {
		err = errNoHero
	}
	if err != nil {
		return fail(0, err)
	}
	if _, ok := t.idTool(s); !ok {
		return fail(0, ErrNoIDTool)
	}
	for i, id := range ids {
		if err := ctx.Err(); err != nil {
			fail(i, err)
			break
		}
		if s, err = t.h.Fresh(); err != nil {
			fail(i, err)
			break
		}
		if t.fightNear(s) {
			fail(i, ErrFight)
			break
		}
		u, ok := findIn(s.Inventory, id)
		switch {
		case !ok:
			errs[i] = errors.New("the item is not in the backpack or worn")
		case u.Item.Identified():
			errs[i] = nil // identified meanwhile: the request's goal is met
		default:
			if _, ok := t.idTool(s); !ok {
				fail(i, ErrNoIDTool)
				i = len(ids)
				break
			}
			if errs[i] = t.identifyOne(ctx, u); errs[i] == nil {
				t.e.Log.Info("player identify", "item", t.nameOf(u))
			}
		}
		if i >= len(ids) {
			break
		}
	}
	// Close on a context that survives a cancel: a safety interrupt or a fight cancels ctx mid-action, and the panels
	// must still close and the identify cursor and any held item must be released (the requests go back to queued).
	if cerr := t.h.CloseAll(context.WithoutCancel(ctx)); cerr != nil {
		t.e.Log.Warn("town: closing panels after the player's identify failed", "err", cerr)
	}
	return errs
}

// PlayerIdentify identifies one item (a batch of one).
func (t *Town) PlayerIdentify(ctx context.Context, id uint32) error {
	return t.PlayerIdentifyBatch(ctx, []uint32{id})[0]
}

// IdentifyTools reports the Tome of Identify (scrolls in it, -1 without a tome), its capacity and the loose scrolls.
func (t *Town) IdentifyTools(s *d2.Snapshot) (tome, capacity, scrolls int) {
	tome, capacity = -1, idTomeCap
	if s == nil || s.Inventory == nil {
		return tome, capacity, 0
	}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		switch b := t.base(u); {
		case b == nil:
		case b.Code == "isc":
			scrolls++
		case b.Code == "ibk":
			tome = t.qty(u)
		}
	}
	return tome, capacity, scrolls
}

// idBuyIn and idBuyPlan are the pure part of the shopping errand, so the money rules are tested without a game.
type idBuyIn struct {
	Gold, Reserve          int  // the purse and the gold that must remain (repair and kit)
	HaveTome               bool // a Tome of Identify is in the pack
	TomeQty, TomeCap       int
	Want                   int // 0: fill the tome (default), else the count the player asked for
	ScrollPrice, TomePrice int // 0 = the vendor does not sell it
}

type idBuyPlan struct {
	Tome    bool // buy a tome first
	Scrolls int  // scrolls to buy
	Wanted  int  // scrolls the plan wanted before the gold limited it
	Short   bool // the gold above the reserve does not cover Wanted
	Cost    int
}

// planIDBuy decides what the errand buys. Default (Want 0): fill the tome; without a tome buy one when the gold above
// the reserve covers it, then fill it; else a few loose scrolls. An explicit count is honoured up to the tome's room (or
// the tome capacity without a tome). Nothing is bought below the reserve.
func planIDBuy(in idBuyIn) idBuyPlan {
	var p idBuyPlan
	avail := in.Gold - in.Reserve
	want := in.Want
	switch {
	case in.HaveTome:
		room := max(in.TomeCap-in.TomeQty, 0)
		if want == 0 || want > room {
			want = room
		}
	case want == 0:
		if in.TomePrice > 0 && avail >= in.TomePrice {
			p.Tome, avail = true, avail-in.TomePrice
			p.Cost += in.TomePrice
			want = in.TomeCap
		} else {
			want = idScrollsDefault
		}
	default:
		want = min(want, in.TomeCap)
	}
	p.Wanted = want
	if in.ScrollPrice > 0 && avail > 0 {
		p.Scrolls = min(want, avail/in.ScrollPrice)
	}
	p.Cost += p.Scrolls * in.ScrollPrice
	p.Short = p.Scrolls < want
	return p
}

// idBuyReserve is the gold the errand never spends: the optional-buy reserve plus the minimal kit (portal scroll and
// first potions, beltBuyReserve) plus the repair the worn gear needs (repairEstimate).
func (t *Town) idBuyReserve(s *d2.Snapshot) int { return t.beltBuyReserve() + t.repairEstimate(s) }

// CheckBuyIdentify validates the count (0 = fill the tome) and names the errand.
func (t *Town) CheckBuyIdentify(s *d2.Snapshot, n int) (string, error) {
	if s == nil || s.Player == nil || s.Inventory == nil {
		return "", errNoHero
	}
	if n < 0 || n > idTomeCap {
		return "", fmt.Errorf("pick 1 to %d scrolls (0 fills the tome)", idTomeCap)
	}
	tome, _, _ := t.IdentifyTools(s)
	if tome >= idTomeCap {
		return "", errors.New("the Tome of Identify is full")
	}
	switch {
	case n > 0:
		return fmt.Sprintf("Buy %d identify scrolls", n), nil
	case tome >= 0:
		return "Fill the Tome of Identify", nil
	}
	return "Buy a Tome of Identify and fill it", nil
}

// BuyIdentify runs the shopping errand in the current town and returns what it did in a sentence. It spends only the
// gold above idBuyReserve; running out of gold is a clean "short of gold" result, not an error, unless nothing at all
// could be bought.
func (t *Town) BuyIdentify(ctx context.Context, n int) (string, error) {
	s, err := t.current()
	if err != nil {
		return "", err
	}
	a := ActOfLevel(s.Player.Area)
	if a == nil {
		return "", ErrNotInTown
	}
	if _, err := t.CheckBuyIdentify(s, n); err != nil {
		return "", err
	}
	reserve := t.idBuyReserve(s)
	if err := t.h.CloseAll(ctx); err != nil {
		return "", err
	}
	npc, err := t.openShop(ctx, a.Shop, t.npcPreset(a.Shop...))
	if err != nil {
		return "", err
	}
	defer func() { _ = t.h.CloseAll(ctx) }()
	if s, err = t.current(); err != nil {
		return "", err
	}
	lay, err := t.layout(s)
	if err != nil {
		return "", err
	}
	stock, err := t.Stock(npc)
	if err != nil {
		return "", err
	}
	var scroll, tomeU d2.Unit
	var haveScroll, haveTomeU bool
	for _, u := range stock {
		switch b := t.base(u); {
		case b == nil:
		case b.Code == "isc" && !haveScroll:
			scroll, haveScroll = u, true
		case b.Code == "ibk" && !haveTomeU:
			tomeU, haveTomeU = u, true
		}
	}
	if !haveScroll {
		return "", fmt.Errorf("town: npc %d does not sell identify scrolls", npc.Class)
	}
	tome, _, _ := t.IdentifyTools(s)
	in := idBuyIn{Gold: s.Player.Gold, Reserve: reserve, HaveTome: tome >= 0, TomeQty: max(tome, 0), TomeCap: idTomeCap, Want: n,
		ScrollPrice: t.base(scroll).Cost}
	if haveTomeU {
		in.TomePrice = t.base(tomeU).Cost
	}
	plan := planIDBuy(in)
	t.e.Log.Info("player buy identify scrolls", "gold", in.Gold, "reserve", reserve, "tome", tome, "want", n, "plan", fmt.Sprintf("%+v", plan))

	var spent, bought int
	boughtTome, short := false, plan.Short
	if plan.Tome {
		price, err := t.buyUnit(ctx, lay, tomeU)
		switch {
		case errors.Is(err, ErrNoMoney):
			short = true
		case err != nil:
			return "", err
		default:
			spent += price
			boughtTome = true
		}
	}
	for i := 0; i < plan.Scrolls && (boughtTome || !plan.Tome); i++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		cur, err := t.current()
		if err != nil {
			return "", err
		}
		if cur.Player.Gold-t.base(scroll).Cost < reserve {
			short = true
			break
		}
		price, err := t.buyUnit(ctx, lay, scroll)
		if errors.Is(err, ErrNoMoney) {
			short = true
			break
		}
		if err != nil {
			return "", err
		}
		spent += price
		bought++
	}
	return idBuySummary(bought, boughtTome, spent, plan.Wanted-bought, short)
}

// idBuySummary words the result; nothing bought at all is a failure with the reason, so the queue card shows red.
func idBuySummary(bought int, tome bool, spent, missing int, short bool) (string, error) {
	if bought == 0 && !tome {
		if short {
			return "", errors.New("short of gold: nothing could be bought above the repair and kit reserve")
		}
		return "nothing to buy", nil
	}
	what := fmt.Sprintf("bought %d", bought)
	if tome {
		what = fmt.Sprintf("bought a Tome of Identify and %d", bought)
	}
	out := fmt.Sprintf("%s for %d gold", what, spent)
	if short && missing > 0 {
		out += fmt.Sprintf(", short of gold for the other %d", missing)
	}
	return out, nil
}

// CheckCainIdentify says whether the Cain errand has anything to do. Whether Cain is rescued is the desk's check.
func (t *Town) CheckCainIdentify(s *d2.Snapshot) error {
	if s == nil || s.Player == nil || s.Inventory == nil {
		return errNoHero
	}
	if len(unidentified(s)) == 0 {
		return errors.New("nothing to identify")
	}
	return nil
}

// CainIdentify opens Deckard Cain in this town, picks his Identify line and proves in memory that every carried and
// worn item is identified. Free: no scroll is used. It needs no new primitive: the dialog click is PickLine, the one the
// quest dialogs use (V for Akara's Talk menu, U for Cain's lines). Cain stands in the town of every act after the rescue
// (S: acts.go lists his classes per act); in a town where he is not seen the errand fails with that reason.
func (t *Town) CainIdentify(ctx context.Context) (string, error) {
	s, err := t.current()
	if err != nil {
		return "", err
	}
	a := ActOfLevel(s.Player.Area)
	if a == nil {
		return "", ErrNotInTown
	}
	before := len(unidentified(s))
	if before == 0 {
		return "nothing to identify", nil
	}
	if err := t.h.CloseAll(ctx); err != nil {
		return "", err
	}
	npc, err := t.openNPC(ctx, a.Cain, t.npcPreset(a.Cain...))
	if err != nil {
		if errors.Is(err, ErrNoNPC) {
			return "", fmt.Errorf("Deckard Cain is not in this town (go to the Rogue Encampment): %w", err)
		}
		return "", err
	}
	defer func() { _ = t.h.CloseAll(ctx) }()
	idx, n, ok := menuLine(npc.Class, EntryIdentify)
	if !ok {
		return "", fmt.Errorf("town: npc %d has no Identify line", npc.Class)
	}
	if err := t.PickLine(ctx, npc, idx, n); err != nil {
		return "", err
	}
	end, done, err := t.h.Wait(ctx, 3*t.uiTimeout(), func(s *d2.Snapshot) bool { return len(unidentified(s)) == 0 })
	if err != nil {
		return "", err
	}
	if !done {
		left := before
		if end != nil {
			left = len(unidentified(end))
		}
		return "", fmt.Errorf("Cain left %d of %d items unidentified (his Identify line was not reached)", left, before)
	}
	return fmt.Sprintf("Deckard Cain identified %d items for free", before), nil
}

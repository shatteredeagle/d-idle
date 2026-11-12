package d2dctl

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Item verbs of the private fork (docs/design/item-verbs.md): the fork plans an equip or unequip on a copy of the
// character with the server's own rule functions, then sends the commands a client would (ToCursor, ToBody, ToGrid).
// Stock d2d and every older fork answer "caps" with an error: Caps then returns nothing and the caller keeps the click
// path. V (design) / U (live): the verbs are built in the fork branch fork-only/itemverbs; no live run yet.

// Capability names a fork build advertises in "caps".
const (
	CapItemEquip   = "item.equip"
	CapItemUnequip = "item.unequip"
)

// Item verb refusal codes (the reasons the click path reports too).
const (
	ItemErrItem         = "item"         // no such item
	ItemErrCursor       = "cursor"       // the hand is not empty
	ItemErrPlace        = "place"        // not a backpack item (equip) / not worn (unequip)
	ItemErrSlot         = "slot"         // the item does not fit that slot
	ItemErrIdentified   = "identified"   // identify it first
	ItemErrClass        = "class"        // another class's item
	ItemErrRequirements = "requirements" // level, strength or dexterity too low
	ItemErrTwoHand      = "twohand"      // a second item would have to come off
	ItemErrNoSpace      = "nospace"      // no room in the backpack for the item that comes off
)

// ItemRefused is a rule refusal of an item verb: the verb understood the request and the game's rules say no.
// Repeating it through the click path would fail the same way.
type ItemRefused struct {
	Code, Why string
}

func (e *ItemRefused) Error() string { return "item verb refused (" + e.Code + "): " + e.Why }

// ItemResult is a successful (queued) item verb. Noop: nothing to do, the item is already there.
type ItemResult struct {
	Noop      bool
	Slot      int // body slot used / taken off
	Displaced int // equip: unit id of the item that came off, -1 none
	AtCol     int // where the item that came off (or the unequipped one) lands in the backpack, -1 none
	AtRow     int
}

// Caps lists the capabilities of the connected d2d. A build without the verb (stock d2d, an older fork) gives an empty
// list and no error: only transport errors are returned.
func (c *Client) Caps(ctx context.Context) ([]string, error) {
	r, err := c.Do(ctx, "caps")
	if err != nil {
		var ce *CmdError
		if errors.As(err, &ce) && !errors.Is(err, ErrBusy) {
			return nil, nil
		}
		return nil, err
	}
	for _, l := range append(append([]string(nil), r.Lines...), r.Rest) {
		if v, ok := ParseKV(l).Get("caps"); ok {
			return strings.Split(v, ","), nil
		}
	}
	return nil, nil
}

// HasCap reports whether caps lists name.
func HasCap(caps []string, name string) bool {
	for _, c := range caps {
		if c == name {
			return true
		}
	}
	return false
}

// ItemEquip wears backpack item id on body slot (0: the one the fork picks). The change is applied at the next tick:
// the caller proves it from a fresh snapshot.
func (c *Client) ItemEquip(ctx context.Context, id uint32, slot int) (ItemResult, error) {
	return c.itemVerb(ctx, fmt.Sprintf("item equip %d %d", id, slot))
}

// ItemUnequip takes worn item id off into the backpack.
func (c *Client) ItemUnequip(ctx context.Context, id uint32) (ItemResult, error) {
	return c.itemVerb(ctx, fmt.Sprintf("item unequip %d", id))
}

func (c *Client) itemVerb(ctx context.Context, line string) (ItemResult, error) {
	r, err := c.Do(ctx, line)
	if err != nil {
		var ce *CmdError
		if errors.As(err, &ce) && !errors.Is(err, ErrBusy) && !errors.Is(err, ErrLineTooLong) {
			code, why, _ := strings.Cut(ce.Msg, " ")
			if strings.HasPrefix(ce.Msg, "lab verbs off") || strings.HasPrefix(ce.Msg, "unknown") || strings.HasPrefix(ce.Msg, "not in a game") {
				return ItemResult{}, err // not a rule refusal: the caller falls back
			}
			return ItemResult{}, &ItemRefused{Code: code, Why: why}
		}
		return ItemResult{}, err
	}
	res := ItemResult{Displaced: -1, AtCol: -1, AtRow: -1}
	first := ""
	if len(r.Lines) > 0 {
		first = r.Lines[0]
	}
	res.Noop = strings.HasPrefix(first, "noop")
	kv := ParseKV(first)
	res.Slot, _ = kv.Int("slot")
	if v, ok := kv.Int("displaced"); ok {
		res.Displaced = v
	}
	if p, ok := kv.Get("at"); ok {
		if a, b, ok := strings.Cut(p, ","); ok {
			res.AtCol, _ = strconv.Atoi(a)
			res.AtRow, _ = strconv.Atoi(b)
		}
	}
	return res, nil
}

// RuleRefusal marks the error as a game-rule refusal (invdesk does not fall back to the click path for these).
func (e *ItemRefused) RuleRefusal() bool { return true }

// ItemVerbs adapts a Client to the shape internal/invdesk uses (no import of this package there).
type ItemVerbs struct{ C *Client }

func (v ItemVerbs) Caps(ctx context.Context) ([]string, error) { return v.C.Caps(ctx) }

// ItemEquip wears backpack item id on body slot; noop reports "already worn there".
func (v ItemVerbs) ItemEquip(ctx context.Context, id uint32, slot int) (bool, error) {
	r, err := v.C.ItemEquip(ctx, id, slot)
	return r.Noop, err
}

// Capability names of the stage 2 verbs.
const (
	CapItemMove     = "item.move"
	CapItemIdentify = "item.identify"
	CapItemDrop     = "item.drop"
	CapGoldStash    = "gold.stash"
)

// Further refusal codes: the stash and its gold only work at the chest.
const (
	ItemErrFar     = "far"    // not at the chest
	ItemErrBounds  = "bounds" // outside the grid
	ItemErrOverlap = "overlap"
	ItemErrNoTool  = "notool"
	ItemErrDead    = "dead"
	ItemErrGold    = "gold"  // not that much gold
	ItemErrLimit   = "limit" // would carry more than level x 10,000
	ItemErrFull    = "full"  // the stash holds 2,500,000
)

// ItemMove puts item id at (col, row) of panel ("inv", "cube", "stash"); col and row -1 pick the first free spot.
// The stash needs the hero at the chest (the fork answers "far" otherwise).
func (c *Client) ItemMove(ctx context.Context, id uint32, panel string, col, row int) (ItemResult, error) {
	return c.itemVerb(ctx, fmt.Sprintf("item move %d %s %d %d", id, panel, col, row))
}

// ItemIdentify identifies item id with tool (0: the fork picks a tome with charges, else a scroll).
func (c *Client) ItemIdentify(ctx context.Context, id, tool uint32) (ItemResult, error) {
	if tool == 0 {
		return c.itemVerb(ctx, fmt.Sprintf("item identify %d", id))
	}
	return c.itemVerb(ctx, fmt.Sprintf("item identify %d %d", id, tool))
}

// ItemDrop drops backpack item id at the hero's feet.
func (c *Client) ItemDrop(ctx context.Context, id uint32) (ItemResult, error) {
	return c.itemVerb(ctx, fmt.Sprintf("item drop %d", id))
}

// GoldStash moves gold between the purse and the stash (at the chest). It returns the balances the fork predicts.
func (c *Client) GoldStash(ctx context.Context, amount int, deposit bool) (bank, carried int, err error) {
	dir := "withdraw"
	if deposit {
		dir = "deposit"
	}
	r, err := c.Do(ctx, fmt.Sprintf("gold %s %d", dir, amount))
	if err != nil {
		var ce *CmdError
		if errors.As(err, &ce) && !errors.Is(err, ErrBusy) && !errors.Is(err, ErrLineTooLong) &&
			!strings.HasPrefix(ce.Msg, "lab verbs off") && !strings.HasPrefix(ce.Msg, "unknown") && !strings.HasPrefix(ce.Msg, "not in a game") {
			code, why, _ := strings.Cut(ce.Msg, " ")
			return 0, 0, &ItemRefused{Code: code, Why: why}
		}
		return 0, 0, err
	}
	kv := ParseKV(strings.Join(r.Lines, " "))
	bank, _ = kv.Int("bank")
	carried, _ = kv.Int("carried")
	return bank, carried, nil
}

func (v ItemVerbs) ItemMove(ctx context.Context, id uint32, panel string, col, row int) (bool, error) {
	r, err := v.C.ItemMove(ctx, id, panel, col, row)
	return r.Noop, err
}

func (v ItemVerbs) ItemIdentify(ctx context.Context, id uint32) (bool, error) {
	r, err := v.C.ItemIdentify(ctx, id, 0)
	return r.Noop, err
}

func (v ItemVerbs) ItemDrop(ctx context.Context, id uint32) error {
	_, err := v.C.ItemDrop(ctx, id)
	return err
}

func (v ItemVerbs) GoldStash(ctx context.Context, amount int, deposit bool) (int, int, error) {
	return v.C.GoldStash(ctx, amount, deposit)
}

package sim

import (
	"fmt"
	"math"
	"strconv"

	"d2lod-bot/internal/cubejobs"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/panel"
)

// Input is the simulated controller. It embeds input.Fake for call recording and overrides the calls that
// change the world.
type Input struct {
	*input.Fake
	W *World
}

// NewInput returns the controller for w.
func NewInput(w *World) *Input {
	return &Input{Fake: input.NewFake(input.Point{X: w.Player.X, Y: w.Player.Y}), W: w}
}

var _ input.Controller = (*Input)(nil)

func (c *Input) Pos() (input.Point, error) { return input.Point{X: c.W.Player.X, Y: c.W.Player.Y}, nil }

func (c *Input) Key(name string) error {
	_ = c.Fake.Key(name)
	w := c.W
	switch name {
	case "i":
		if w.UI[d2.UIInventory] != 0 {
			w.UI[d2.UIInventory] = 0
		} else {
			w.UI[d2.UIInventory] = 1
		}
	case "esc":
		if w.idMode || w.pick != "" {
			w.idMode, w.pick = false, ""
			return nil
		}
		w.closeTop()
	case "backspace":
		if w.goldDialog {
			w.goldTyped = true
			if n := len(w.goldBuf); n > 0 {
				w.goldBuf = w.goldBuf[:n-1]
			}
		}
	case "enter":
		if w.goldDialog {
			w.goldDialog = false
			amt := w.Player.Gold
			if w.goldTyped {
				n, _ := strconv.Atoi(w.goldBuf)
				amt = min(n, w.Player.Gold)
			}
			w.Player.StashGold += amt
			w.Player.Gold -= amt
		}
	}
	return nil
}

// TypeChar types a digit into the open deposit dialog.
func (c *Input) TypeChar(ch rune) error {
	_ = c.Fake.TypeChar(ch)
	if w := c.W; w.goldDialog && ch >= '0' && ch <= '9' {
		w.goldTyped = true
		w.goldBuf += string(ch)
	} else if w.goldDialog && ch == '\b' {
		w.goldTyped = true
		if n := len(w.goldBuf); n > 0 {
			w.goldBuf = w.goldBuf[:n-1]
		}
	}
	return nil
}

func (w *World) closeTop() {
	for _, i := range []int{d2.UIShop, d2.UINPCMenu, d2.UICube, d2.UIStash, d2.UIInventory} {
		if w.UI[i] != 0 {
			w.UI[i] = 0
			if i == d2.UIStash {
				w.UI[d2.UIInventory] = 0
			}
			return
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (c *Input) CloseAll() error {
	_ = c.Fake.CloseAll()
	for i := range c.W.Mons {
		c.W.Mons[i].X += c.W.WanderOnClose[0]
		c.W.Mons[i].Y += c.W.WanderOnClose[1]
	}
	for i := range c.W.UI {
		c.W.UI[i] = 0
	}
	c.W.idMode, c.W.pick = false, ""
	return nil
}

func (c *Input) MoveTo(p input.Point, _ input.MoveOpts) error {
	c.W.Player.X, c.W.Player.Y = p.X, p.Y
	return c.Fake.MoveTo(p, input.MoveOpts{})
}

func (c *Input) ClickWorld(p input.Point) error {
	_ = c.Fake.ClickWorld(p)
	w := c.W
	for _, n := range w.Mons {
		if abs(n.X-p.X) <= 2 && abs(n.Y-p.Y) <= 2 {
			if w.speaking && w.SpeechSkipByClick {
				w.Speech, w.deadHover, w.speaking = 0, 0, false
				w.UI[d2.UINPCMenu] = 1
				w.talk = n.ID
			}
			return nil
		}
	}
	w.WalkedAway++
	w.Player.X, w.Player.Y = p.X, p.Y
	return nil
}

func (c *Input) Interact(t input.Target) error {
	_ = c.Fake.Interact(t)
	w := c.W
	if w.NoHover > 0 {
		w.NoHover--
		return input.ErrNotHovered
	}
	d := func(x, y int) float64 { return float64((x-w.Player.X)*(x-w.Player.X) + (y-w.Player.Y)*(y-w.Player.Y)) }
	switch t.Type {
	case d2.TypeMonster:
		n, ok := w.npcAt(t.ID)
		if !ok || d(n.X, n.Y) > 20*20 {
			return input.ErrOffscreen
		}
		if w.deadHover > 0 {
			w.deadHover--
			return input.ErrNotHovered
		}
		if w.StrictPos && (abs(n.X-t.Pos.X) > 2 || abs(n.Y-t.Pos.Y) > 2) {
			return input.ErrNotHovered
		}
		if w.Speech > 0 {
			w.Speech--
			w.speaking = true
			w.deadHover = w.NoHoverAfterSpeech
			return nil
		}
		w.UI[d2.UINPCMenu] = 1
		w.talk = n.ID
		if w.HealSkip > 0 {
			w.HealSkip-- // the healer ignores this click
		} else if w.Heal[n.Class] {
			w.Player.Life, w.Player.Mana = w.Player.MaxLife, w.Player.MaxMana
		}
	case d2.TypeObject:
		for _, o := range w.Objs {
			if o.ID != t.ID {
				continue
			}
			if d(o.X, o.Y) > 20*20 {
				return input.ErrOffscreen
			}
			switch o.Class {
			case 267:
				w.UI[d2.UIStash] = 1
			case 59:
				w.Player.Area = w.PortalLevel
			}
		}
	case d2.TypeItem:
		for i, g := range w.Ground {
			if g.ID != t.ID {
				continue
			}
			if d(g.X, g.Y) > 6*6 {
				return input.ErrOffscreen // the real game walks; the sim wants the caller to walk first
			}
			w.Ground = append(w.Ground[:i], w.Ground[i+1:]...)
			if w.base(g).Type == "gold" {
				w.Player.Gold += 50
				return nil
			}
			g.Mode = d2.ItemModeStored
			if body, ok := map[string]int{"msf": d2.BodyRHand, "vip": d2.BodyNeck}[w.base(g).Code]; ok && w.AutoEquipQuest {
				g.Item.Where, g.Item.Body, g.Mode = d2.ContEquipment, body, d2.ItemModeEquipped
				w.Items = append(w.Items, g)
				return nil
			}
			if w.PickupToCursor {
				w.Cursor = &g
				return nil
			}
			w.autoPlace(g)
			return nil
		}
		return input.ErrNotHovered
	}
	return nil
}

func (c *Input) Click(b input.Button, x, y int, mods ...input.Mod) error {
	_ = c.Fake.Click(b, x, y, mods...)
	w := c.W
	p := panel.Pt{X: x, Y: y}
	w.Clicks = append(w.Clicks, fmt.Sprintf("%v %d,%d", b, x, y))
	if b == input.Left {
		if len(mods) > 0 && mods[0] == input.ModShift && c.shiftPotion(p) {
			return nil
		}
		return c.left(p)
	}
	return c.right(p)
}

func (c *Input) left(p panel.Pt) error {
	w := c.W
	if w.UI[d2.UINPCMenu] != 0 {
		if w.MenuInert > 0 {
			w.MenuInert--
			return nil
		}
		n, _ := w.npcAt(w.talk)
		f := d2.WorldToScreen(n.X, n.Y, d2.Point{})
		lines := 3
		if w.talkClass() == 147 {
			lines = 4
		}
		if n, ok := w.Lines[w.talkClass()]; ok {
			lines = n
		}
		// line index counted from the top: the last line is Cancel at feet+Cancel.
		idx := lines - 1 - int(math.Round(float64(f.Y+8+w.Menu.Cancel-p.Y)/(float64(w.Menu.Step2)/2)))
		trade, ok := w.TradeIndex[w.talkClass()]
		if !ok {
			trade = 1
		}
		if kind, ok := w.Services[w.talkClass()][idx]; ok {
			w.UI[d2.UINPCMenu] = 0
			w.UI[d2.UIInventory] = 1
			w.pick = kind
			return nil
		}
		if idx == lines-1 {
			idx = 2
		}
		switch {
		case idx == trade:
			w.UI[d2.UINPCMenu] = 0
			w.UI[d2.UIShop] = 1
		case idx == 2:
			w.UI[d2.UINPCMenu] = 0
		}
		return nil
	}
	if w.UI[d2.UIShop] != 0 && p == w.RepairBtn {
		for _, u := range w.Items {
			if u.Item.Where == d2.ContEquipment {
				if st, ok := w.Stats[u.Addr]; ok {
					st[72] = st[73]
				}
			}
		}
		return nil
	}
	if w.UI[d2.UIStash] != 0 && p == w.GoldBtn && w.Player.Gold > 0 {
		w.goldDialog, w.goldBuf, w.goldTyped = true, "", false
		return nil
	}
	if w.UI[d2.UICube] != 0 && p == (panel.Pt{X: 160 + w.Lay.Offset.X, Y: 268 + w.Lay.Offset.Y}) {
		w.transmute()
		return nil
	}
	if w.pick != "" {
		if _, i := w.hitPack(p); i >= 0 && w.applyService(&w.Items[i]) {
			w.pick = ""
		}
		return nil
	}
	if w.idMode {
		if g, i := w.hitPack(p); i >= 0 {
			w.Items[i].Item.Flags |= d2.FlagIdentified
			w.idMode = false
			_ = g
		}
		return nil
	}
	if w.Cursor == nil && w.packVisible() {
		body := map[string]int{"rArm": d2.BodyRHand, "lArm": d2.BodyLHand, "neck": d2.BodyNeck, "torso": d2.BodyTorso, "head": d2.BodyHead,
			"rHand": d2.BodyRRing, "lHand": d2.BodyLRing, "belt": d2.BodyBelt, "feet": d2.BodyFeet, "gloves": d2.BodyGloves}
		for name, b := range body {
			if r, ok := w.Lay.Slots[name]; ok && r.Contains(p) {
				for i, u := range w.Items {
					if u.Item.Where == d2.ContEquipment && u.Item.Body == b {
						cu := u
						w.Cursor = &cu
						w.Items = append(w.Items[:i], w.Items[i+1:]...)
						return nil
					}
				}
			}
		}
	}
	if w.Cursor != nil && w.packVisible() && w.base(*w.Cursor).AutoBelt {
		if r, ok := w.Lay.Slots["belt"]; ok && r.Contains(p) { // a potion dropped on the belt slot goes into the belt (U live)
			if w.DragFails > 0 {
				w.DragFails--
				return nil
			}
			u := *w.Cursor
			if w.autoPlace(u) {
				w.Cursor = nil
			}
			return nil
		}
	}
	if w.Cursor != nil && w.packVisible() {
		slotLoc := map[string]struct {
			loc  string
			body int
		}{"rArm": {"rarm", d2.BodyRHand}, "lArm": {"larm", d2.BodyLHand}, "torso": {"tors", d2.BodyTorso}, "head": {"head", d2.BodyHead},
			"feet": {"feet", d2.BodyFeet}, "gloves": {"glov", d2.BodyGloves}, "belt": {"belt", d2.BodyBelt},
			"neck": {"neck", d2.BodyNeck}, "rHand": {"rrin", d2.BodyRRing}, "lHand": {"lrin", d2.BodyLRing}}
		for name, r := range w.Lay.Slots {
			info, ok := slotLoc[name]
			if !ok || !r.Contains(p) {
				continue
			}
			if w.SlotIgnore > 0 {
				w.SlotIgnore--
				return nil
			}
			if w.EquipRefuse[w.Cursor.ID] {
				return nil // the game refuses the item (requirement): it stays on the cursor
			}
			ty := w.DB.ItemType(w.base(*w.Cursor).Type)
			if ty == nil || ty.BodyLocs[0] != info.loc && ty.BodyLocs[1] != info.loc {
				return nil
			}
			var out *d2.Unit // the worn item swaps to the cursor, as in the game
			for i, u := range w.Items {
				if u.Item.Where == d2.ContEquipment && u.Item.Body == info.body {
					cu := u
					out = &cu
					w.Items = append(w.Items[:i], w.Items[i+1:]...)
					break
				}
			}
			u := *w.Cursor
			u.Item = &d2.Item{}
			*u.Item = *w.Cursor.Item
			u.Item.Where, u.Item.Body, u.Mode = d2.ContEquipment, info.body, d2.ItemModeEquipped
			w.Items = append(w.Items, u)
			w.Cursor = out
			return nil
		}
	}
	if w.Cursor != nil && w.UI[d2.UIShop] != 0 {
		if _, _, ok := w.Lay.Vendor.CellAt(p); ok {
			b := w.base(*w.Cursor)
			if w.Refuse[w.talk][b.Code] {
				return nil // the vendor does not take it; the cursor keeps the item
			}
			w.Player.Gold += max(1, b.Cost/4)
			w.Cursor = nil
			return nil
		}
	}
	type zone struct {
		g     panel.Grid
		where d2.Container
		open  bool
	}
	zones := []zone{
		{w.Lay.Inventory, d2.ContInventory, w.packVisible()},
		{w.Lay.Stash, d2.ContStash, w.UI[d2.UIStash] != 0},
		{w.Lay.Cube, d2.ContCube, w.UI[d2.UICube] != 0},
	}
	for _, z := range zones {
		if !z.open {
			continue
		}
		col, row, ok := z.g.CellAt(p)
		if !ok {
			continue
		}
		if w.Cursor == nil { // pick up
			for i, u := range w.Items {
				ww, hh := w.size(u)
				if u.Item.Where == z.where && col >= u.X && col < u.X+ww && row >= u.Y && row < u.Y+hh {
					cu := u
					w.Cursor = &cu
					w.Items = append(w.Items[:i], w.Items[i+1:]...)
					return nil
				}
			}
			return nil
		}
		ww, hh := w.size(*w.Cursor)
		cc, rr := topLeft(z.g, p, ww, hh)
		if fits(w.occupied(z.where, z.g), z.g, cc, rr, ww, hh) {
			u := *w.Cursor
			u.Item = &d2.Item{}
			*u.Item = *w.Cursor.Item
			u.Item.Where, u.Mode, u.X, u.Y = z.where, d2.ItemModeStored, cc, rr
			w.Items = append(w.Items, u)
			w.Cursor = nil
		}
		return nil
	}
	return nil
}

func (w *World) talkClass() int {
	if n, ok := w.npcAt(w.talk); ok {
		return n.Class
	}
	return -1
}

func (w *World) hitPack(p panel.Pt) (panel.Grid, int) {
	col, row, ok := w.Lay.Inventory.CellAt(p)
	if !ok || !w.packVisible() {
		return w.Lay.Inventory, -1
	}
	for i, u := range w.Items {
		ww, hh := w.size(u)
		if u.Item.Where == d2.ContInventory && col >= u.X && col < u.X+ww && row >= u.Y && row < u.Y+hh {
			return w.Lay.Inventory, i
		}
	}
	return w.Lay.Inventory, -1
}

func (c *Input) right(p panel.Pt) error {
	w := c.W
	if w.UI[d2.UIShop] != 0 {
		if col, row, ok := w.Lay.Vendor.CellAt(p); ok {
			for _, u := range w.Vendors[w.talk] {
				ww, hh := w.size(u)
				if col >= u.X && col < u.X+ww && row >= u.Y && row < u.Y+hh {
					b := w.base(u)
					if w.Player.Gold < b.Cost {
						return nil
					}
					nu := w.NewItem(b.Code, 2)
					if w.intoTome(b.Code) {
						w.Player.Gold -= b.Cost
						return nil
					}
					if b.Code == "tbk" || b.Code == "ibk" {
						w.Stats[nu.Addr] = d2.Stats{70: 0}
					}
					if w.autoPlace(nu) {
						w.Player.Gold -= b.Cost
					}
					return nil
				}
			}
			return nil
		}
		return nil
	}
	if _, i := w.hitPack(p); i >= 0 {
		u := w.Items[i]
		if w.RightClickEquip && w.rightEquip(i) {
			return nil
		}
		switch w.base(u).Code {
		case "box":
			w.UI[d2.UICube] = 1
		case "tsc":
			w.Items = append(w.Items[:i], w.Items[i+1:]...)
			w.AddObject(59, w.Player.X+2, w.Player.Y)
		case "isc":
			w.Items = append(w.Items[:i], w.Items[i+1:]...)
			w.idMode = true
		}
	}
	return nil
}

// intoTome adds a bought scroll to the matching tome's quantity when one is in the pack.
func (w *World) intoTome(scroll string) bool {
	tome := map[string]string{"tsc": "tbk", "isc": "ibk"}[scroll]
	if tome == "" {
		return false
	}
	for _, u := range w.Items {
		if u.Item.Where == d2.ContInventory && w.base(u).Code == tome {
			w.Stats[u.Addr][70]++
			return true
		}
	}
	return false
}

// packVisible: the backpack is drawn with the inventory flag, and beside a vendor or the stash without it
// (VERIFIED LIVE).
func (w *World) packVisible() bool {
	return w.UI[d2.UIInventory] != 0 || w.UI[d2.UIShop] != 0 || w.UI[d2.UIStash] != 0
}

// transmute implements the one recipe the tests need: Shaft (msf) + Top of the Staff (vip) -> Horadric Staff (hst).
// Anything else leaves the cube unchanged.
func (w *World) transmute() {
	var msf, vip = -1, -1
	n := 0
	for i, u := range w.Items {
		if u.Item.Where != d2.ContCube {
			continue
		}
		n++
		switch w.base(u).Code {
		case "msf":
			msf = i
		case "vip":
			vip = i
		}
	}
	if n != 2 || msf < 0 || vip < 0 {
		w.transmuteUpgrade()
		return
	}
	hi, lo := max(msf, vip), min(msf, vip)
	w.Items = append(w.Items[:hi], w.Items[hi+1:]...)
	w.Items = append(w.Items[:lo], w.Items[lo+1:]...)
	w.Add("hst", d2.ContCube, 0, 0)
}

// transmuteUpgrade runs a gem or rune upgrade when the cube holds exactly the inputs of one cubejobs recipe.
func (w *World) transmuteUpgrade() {
	have := map[string]int{}
	for _, u := range w.Items {
		if u.Item.Where == d2.ContCube {
			have[w.base(u).Code]++
		}
	}
	jobs := cubejobs.Plan(have, cubejobs.Opts{MaxJobs: 1, RuneCap: 33, GemCap: 4})
	if len(jobs) != 1 {
		return
	}
	used := 0
	for _, c := range jobs[0].Inputs {
		used++
		have[c]--
	}
	for _, n := range have {
		if n != 0 {
			return // extra items: no recipe
		}
	}
	if used == 0 {
		return
	}
	for _, c := range jobs[0].Inputs {
		for i, u := range w.Items {
			if u.Item.Where == d2.ContCube && w.base(u).Code == c {
				w.Items = append(w.Items[:i], w.Items[i+1:]...)
				break
			}
		}
	}
	w.Add(jobs[0].Out, d2.ContCube, 0, 0)
}

// applyService changes a plain (normal or superior, unsocketed) pack item: imbue makes it rare and unidentified,
// socket gives it the base's socket count. False leaves the picker waiting (the game refuses other items, S).
func (w *World) applyService(u *d2.Unit) bool {
	it := u.Item
	if it.Quality != d2.QualityNormal && it.Quality != d2.QualitySuperior || it.Socketed() {
		return false
	}
	switch w.pick {
	case "imbue":
		it.Quality, it.Flags = d2.QualityRare, it.Flags&^d2.FlagIdentified
	case "socket":
		it.Flags |= d2.FlagSocketed
		st := w.Stats[u.Addr]
		if st == nil {
			st = d2.Stats{}
			w.Stats[u.Addr] = st
		}
		st[194] = int32(max(1, w.base(*u).Sockets))
	}
	return true
}

// shiftPotion models D2's shift-click on a pack potion: it goes to the first free belt slot. W.ShiftFails > 0 makes
// that many shift-clicks do nothing (the click is swallowed, U on the real game).
func (c *Input) shiftPotion(p panel.Pt) bool {
	w := c.W
	_, i := w.hitPack(p)
	if i < 0 || w.UI[d2.UIShop] != 0 || !w.base(w.Items[i]).AutoBelt {
		return false
	}
	if w.ShiftFails > 0 {
		w.ShiftFails--
		return true
	}
	u := w.Items[i]
	w.Items = append(w.Items[:i], w.Items[i+1:]...)
	if !w.autoPlace(u) {
		w.Items = append(w.Items, u)
	}
	return true
}

// rightEquip wears pack item i into its empty body slot, as a right click on it does (U live).
func (w *World) rightEquip(i int) bool {
	u := w.Items[i]
	ty := w.DB.ItemType(w.base(u).Type)
	if ty == nil {
		return false
	}
	locs := map[string]int{"rarm": d2.BodyRHand, "larm": d2.BodyLHand, "tors": d2.BodyTorso, "head": d2.BodyHead,
		"feet": d2.BodyFeet, "glov": d2.BodyGloves, "belt": d2.BodyBelt}
	for _, l := range ty.BodyLocs {
		body, ok := locs[l]
		if !ok {
			continue
		}
		for _, o := range w.Items {
			if o.Item.Where == d2.ContEquipment && o.Item.Body == body {
				return false
			}
		}
		it := *u.Item
		it.Where, it.Body = d2.ContEquipment, body
		u.Item, u.Mode = &it, d2.ItemModeEquipped
		w.Items[i] = u
		return true
	}
	return false
}

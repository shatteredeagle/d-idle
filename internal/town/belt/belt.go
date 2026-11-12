// Package belt models the potion belt: how many slots the equipped belt has, which potions sit in it, and
// how many potions of each kind a level-appropriate plan wants. Pure functions over d2.Inventory and data.DB.
package belt

import (
	_ "embed"
	"encoding/json"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/resource"
)

// Kind is a potion family.
type Kind int

const (
	None Kind = iota
	HP
	MP
	RV
)

func (k Kind) String() string { return [...]string{"none", "hp", "mp", "rv"}[k] }

// Columns is the belt width in every belt type.
const Columns = 4

// Band is one level range of the plan.
type Band struct {
	ClvlMin, ClvlMax int
	Cols             [Columns]Kind
	MaxHPTier        int
	MaxMPTier        int
}

type planFile struct {
	Bands []struct {
		ClvlMin   int      `json:"clvlMin"`
		ClvlMax   int      `json:"clvlMax"`
		Cols      []string `json:"cols"`
		MaxHPTier int      `json:"maxHPTier"`
		MaxMPTier int      `json:"maxMPTier"`
	} `json:"bands"`
}

//go:embed plan.json
var planJSON []byte

// Bands returns the embedded plan.
func Bands() []Band {
	var f planFile
	if err := json.Unmarshal(planJSON, &f); err != nil {
		panic("belt: bad embedded plan.json: " + err.Error())
	}
	var out []Band
	for _, b := range f.Bands {
		nb := Band{ClvlMin: b.ClvlMin, ClvlMax: b.ClvlMax, MaxHPTier: b.MaxHPTier, MaxMPTier: b.MaxMPTier}
		for i, c := range b.Cols {
			if i >= Columns {
				break
			}
			switch strings.ToLower(c) {
			case "hp":
				nb.Cols[i] = HP
			case "mp":
				nb.Cols[i] = MP
			case "rv":
				nb.Cols[i] = RV
			}
		}
		out = append(out, nb)
	}
	return out
}

// PlanFor returns the band covering clvl (the first band for clvl below the table, the last above it).
func PlanFor(clvl int) Band {
	bs := Bands()
	for _, b := range bs {
		if clvl >= b.ClvlMin && clvl <= b.ClvlMax {
			return b
		}
	}
	if clvl < bs[0].ClvlMin {
		return bs[0]
	}
	return bs[len(bs)-1]
}

// PotionKind classifies an item base: HP, MP, RV potion, or None.
func PotionKind(b *data.ItemBase) Kind {
	if b == nil {
		return None
	}
	switch b.Type {
	case "hpot":
		return HP
	case "mpot":
		return MP
	case "rpot":
		return RV
	}
	return None
}

// Tier is 1..5 for healing/mana potions (the digit of hp1..hp5/mp1..mp5) and 1/2 for rejuvenation (rvs, rvl).
func Tier(b *data.ItemBase) int {
	if b == nil {
		return 0
	}
	switch PotionKind(b) {
	case HP, MP:
		if n := len(b.Code); n > 0 && b.Code[n-1] >= '1' && b.Code[n-1] <= '5' {
			return int(b.Code[n-1] - '0')
		}
	case RV:
		if b.Code == "rvl" {
			return 2
		}
		return 1
	}
	return 0
}

// MaxTier is the best tier the band allows for a kind.
func (b Band) MaxTier(k Kind) int {
	switch k {
	case HP:
		return b.MaxHPTier
	case MP:
		return b.MaxMPTier
	case RV:
		return 2
	}
	return 0
}

// State is what the belt holds right now.
type State struct {
	Slots int          // total potion slots of the equipped belt (4 without a belt)
	Count map[Kind]int // potions per kind in the belt
	Other int          // non-plan items in the belt (antidotes, stamina, ...)
}

// Rows is the number of potion rows.
func (s State) Rows() int { return s.Slots / Columns }

// Total potions of any kind.
func (s State) Total() int {
	n := s.Other
	for _, c := range s.Count {
		n += c
	}
	return n
}

// Free slots.
func (s State) Free() int { return s.Slots - s.Total() }

// Slots returns the number of belt slots granted by the equipped belt (4 when none is worn), never fewer than the boxes
// the belt items already occupy. V (sandbox log 2026-11-12, docs/beltfix.md): a hero whose belt read as 4 slots held six
// potions in boxes 0,1,2,4,5,6; the negative free count (4-6) made the fill and the pickup rule see "no room" for good, so
// healing potions never reached the empty column 3. Where the 8 boxes came from is U (the equipment read showed no belt).
func Slots(db *data.DB, inv *d2.Inventory) int {
	n := Columns
	if inv != nil {
		if u := inv.Equipped(d2.BodyBelt); u != nil && u.Class >= 0 && u.Class < len(db.Items) {
			if b := db.BeltByIndex(db.Items[u.Class].BeltIndex); b != nil && b.Slots >= Columns {
				n = b.Slots
			}
		}
		for _, u := range inv.Container(d2.ContBelt) {
			if u.Item == nil {
				continue
			}
			if box := u.Item.GY*Columns + u.Item.GX; box >= n && box < 4*Columns {
				n = (box/Columns + 1) * Columns
			}
		}
	}
	return n
}

// Read summarises the belt of inv.
func Read(db *data.DB, inv *d2.Inventory) State {
	st := State{Slots: Slots(db, inv), Count: map[Kind]int{}}
	if inv == nil {
		return st
	}
	for _, u := range inv.Container(d2.ContBelt) {
		var base *data.ItemBase
		if u.Class >= 0 && u.Class < len(db.Items) {
			base = db.Items[u.Class]
		}
		if k := PotionKind(base); k != None {
			st.Count[k]++
		} else {
			st.Other++
		}
	}
	return st
}

// Capacity is how many potions of kind k the plan band reserves room for in a belt with the given slots:
// the columns assigned to k times the belt's rows.
func (b Band) Capacity(k Kind, slots int) int {
	n := 0
	for _, c := range b.Cols {
		if c == k {
			n++
		}
	}
	return n * (slots / Columns)
}

// Want returns how many potions of each kind to buy so the belt reaches fillPct (0..100) of the planned
// capacity. Rejuvenation columns the shop cannot fill are converted to healing by the caller (see Convert).
func (b Band) Want(st State, fillPct int) map[Kind]int {
	out := map[Kind]int{}
	free := st.Free()
	for _, k := range []Kind{HP, MP, RV} {
		target := (b.Capacity(k, st.Slots)*fillPct + 99) / 100
		if need := target - st.Count[k]; need > 0 {
			out[k] = need
		}
	}
	// The belt cannot hold more than its free slots; clip in plan priority order.
	for _, k := range []Kind{HP, MP, RV} {
		if out[k] > free {
			out[k] = free
		}
		free -= out[k]
	}
	return out
}

// FillPct returns, for kind k, how full the planned columns are (0..100); 100 when the plan reserves none.
func (b Band) FillPct(st State, k Kind) int {
	c := b.Capacity(k, st.Slots)
	if c == 0 {
		return 100
	}
	n := st.Count[k]
	if n > c {
		n = c
	}
	return n * 100 / c
}

// PackCount counts potions of kind k in the backpack (not the belt).
func PackCount(db *data.DB, inv *d2.Inventory, k Kind) int {
	n := 0
	if inv == nil {
		return 0
	}
	for _, u := range inv.Container(d2.ContInventory) {
		if u.Class >= 0 && u.Class < len(db.Items) && PotionKind(db.Items[u.Class]) == k {
			n++
		}
	}
	return n
}

// PlanForClass is PlanFor with the column kinds of resource.PlanFor, the single plan for mana-hungry classes (two
// mana columns below clvl 31, the Combi starvation) and for the Barbarian (no mana column). The tier caps stay the
// band's from plan.json (they equal resource.PlanFor's MaxTier).
func PlanForClass(class d2.Class, clvl int) Band {
	return PlanForBuild(class, clvl, d2.Normal, "")
}

// PlanForBuild is PlanForClass with resource.PlanForBuild's columns: the mana-heavy build columns in Nightmare and
// Hell (and the Hell rejuvenation columns of resource.PlanForDiff). In Normal it is bit-for-bit the resource.PlanFor columns.
func PlanForBuild(class d2.Class, clvl int, diff d2.Difficulty, build string) Band {
	b := PlanFor(clvl)
	rp := resource.PlanFor(class, clvl)
	if diff != d2.Normal {
		rp = resource.PlanForBuild(class, clvl, diff, build)
	}
	for i, k := range rp.Cols {
		switch k {
		case resource.PotionLife:
			b.Cols[i] = HP
		case resource.PotionMana:
			b.Cols[i] = MP
		case resource.PotionRejuv:
			b.Cols[i] = RV
		}
	}
	return b
}

// Placed is one item sitting in a belt box (box = row*Columns + column, row 0 at the bottom). Kind is None for a
// non-potion item (antidote, stamina), which still occupies its box.
type Placed struct {
	Box  int
	Kind Kind
}

// PlacedItems lists what sits in the belt of inv.
func PlacedItems(db *data.DB, inv *d2.Inventory) []Placed {
	if inv == nil {
		return nil
	}
	var out []Placed
	for _, u := range inv.Container(d2.ContBelt) {
		if u.Item == nil {
			continue
		}
		var base *data.ItemBase
		if u.Class >= 0 && u.Class < len(db.Items) {
			base = db.Items[u.Class]
		}
		out = append(out, Placed{Box: u.Item.GY*Columns + u.Item.GX, Kind: PotionKind(base)})
	}
	return out
}

// Place is the belt box game.exe gives a potion of kind k that is picked up or shift-clicked into the belt (V by the
// emulator, docs/research/belt-pickup-trace.md): the lowest free box of the first column whose bottom item is of kind k
// (hp1-5 mix, mp1-5 mix, rvs/rvl mix); else the bottom box of the first EMPTY column; else it does not fit (the potion
// goes to the pack). A column's kind is its bottom (lowest-row) item, so a column holding a non-potion item never
// matches. slots is the belt's box count (4, 8, 12 or 16). Pure: shared by the pickit and the belt fill.
func Place(slots int, items []Placed, k Kind) (int, bool) {
	if k == None || slots < Columns {
		return 0, false
	}
	used := map[int]bool{}
	bottom := [Columns]int{-1, -1, -1, -1}
	var colKind [Columns]Kind
	for _, p := range items {
		if p.Box < 0 || p.Box >= slots {
			continue
		}
		used[p.Box] = true
		if c := p.Box % Columns; bottom[c] < 0 || p.Box < bottom[c] {
			bottom[c], colKind[c] = p.Box, p.Kind
		}
	}
	lowest := func(c int) (int, bool) {
		for b := c; b < slots; b += Columns {
			if !used[b] {
				return b, true
			}
		}
		return 0, false
	}
	for c := 0; c < Columns; c++ {
		if bottom[c] >= 0 && colKind[c] == k {
			if b, ok := lowest(c); ok {
				return b, true
			}
		}
	}
	for c := 0; c < Columns; c++ {
		if bottom[c] < 0 {
			return c, true
		}
	}
	return 0, false
}

// WillBelt reports whether a potion of kind k picked up from the ground now lands on the belt of inv (a same-kind column
// with room, or an empty column) instead of the pack.
func WillBelt(db *data.DB, inv *d2.Inventory, k Kind) bool {
	_, ok := Place(Slots(db, inv), PlacedItems(db, inv), k)
	return ok
}

// PlaceFor is Place under the layout rule: a potion may open an EMPTY column only while enough empty columns stay for
// the planned columns the other kinds are still missing (band columns minus the columns they hold), and a kind that
// already holds all its planned columns opens no more. The game puts a potion into the first empty column whatever the
// plan says, so unguarded pickups let mana take every column and healing never gets one (post-fix soak 1: 42 of 47 failed
// fills were mana potions on an 8-box belt whose planned mana column healing had taken; sandbox 2026-11-12: six mana
// potions in three columns, no healing box, docs/beltfix.md). A kind with no column of its own may always open one.
// Joining an existing column is never refused. The columns per kind are the band's data (class and build), not a constant.
func PlaceFor(slots int, items []Placed, k Kind, b Band) (int, bool) {
	box, ok := Place(slots, items, k)
	if !ok {
		return 0, false
	}
	held, empty := heldColumns(slots, items)
	if box%Columns < Columns && columnHeld(slots, items, box%Columns) {
		return box, true // joins a column that already exists
	}
	if held[k] == 0 {
		return box, true
	}
	if pc := b.planCols(k); pc > 0 && held[k] >= pc {
		return 0, false // the plan's columns for this kind are full
	}
	reserve := 0
	for _, j := range []Kind{HP, MP, RV} {
		if j != k {
			reserve += max(0, b.planCols(j)-held[j])
		}
	}
	if empty > reserve {
		return box, true
	}
	return 0, false
}

// planCols is the number of band columns that hold kind k.
func (b Band) planCols(k Kind) int {
	n := 0
	for _, c := range b.Cols {
		if c == k {
			n++
		}
	}
	return n
}

// heldColumns counts the belt columns by the kind of their bottom item and the empty columns.
func heldColumns(slots int, items []Placed) (held map[Kind]int, empty int) {
	held = map[Kind]int{}
	var bottom [Columns]int
	var colKind [Columns]Kind
	for c := range bottom {
		bottom[c] = -1
	}
	for _, p := range items {
		if p.Box < 0 || p.Box >= slots {
			continue
		}
		if c := p.Box % Columns; bottom[c] < 0 || p.Box < bottom[c] {
			bottom[c], colKind[c] = p.Box, p.Kind
		}
	}
	for c := range bottom {
		if bottom[c] < 0 {
			empty++
		} else {
			held[colKind[c]]++
		}
	}
	return held, empty
}

func columnHeld(slots int, items []Placed, c int) bool {
	for _, p := range items {
		if p.Box >= 0 && p.Box < slots && p.Box%Columns == c {
			return true
		}
	}
	return false
}

// Surplus finds the belt column to empty so that a potion of the starved kind k gets a column: among the columns of the
// OTHER kinds that hold more columns than the band plans for them, the one with the fewest potions (ties: the right-most).
// It returns that column's boxes top first (a column is emptied from the top) and its kind. ok is false when k already
// fits (PlaceFor says yes) or no kind has a column to spare, so a plan the player set on purpose (all mana) is never
// "repaired" and a belt with nothing to give is left alone. Hysteresis: only columns above the plan are offered, and
// PlaceFor refuses to refill them, so a repaired belt does not flip back.
func Surplus(slots int, items []Placed, k Kind, b Band) (boxes []int, from Kind, ok bool) {
	if _, fits := PlaceFor(slots, items, k, b); fits {
		return nil, None, false
	}
	held, _ := heldColumns(slots, items)
	bestCol, bestN := -1, 1<<30
	for c := Columns - 1; c >= 0; c-- {
		var kind Kind
		bottom, n := -1, 0
		for _, p := range items {
			if p.Box < 0 || p.Box >= slots || p.Box%Columns != c {
				continue
			}
			n++
			if bottom < 0 || p.Box < bottom {
				bottom, kind = p.Box, p.Kind
			}
		}
		if n == 0 || kind == k || kind == None || held[kind] <= b.planCols(kind) {
			continue
		}
		if n < bestN {
			bestCol, bestN, from = c, n, kind
		}
	}
	if bestCol < 0 {
		return nil, None, false
	}
	for box := bestCol + (slots/Columns-1)*Columns; box >= bestCol; box -= Columns {
		for _, p := range items {
			if p.Box == box {
				boxes = append(boxes, box)
			}
		}
	}
	return boxes, from, len(boxes) > 0
}

// WillBeltFor is WillBelt with PlaceFor's reserve: a ground potion the game would put in a column the plan keeps for
// another kind is not welcome (clicking it cannot be steered, so it is left on the ground).
func WillBeltFor(db *data.DB, inv *d2.Inventory, k Kind, b Band) bool {
	_, ok := PlaceFor(Slots(db, inv), PlacedItems(db, inv), k, b)
	return ok
}

// WithSpec lays the player's belt plan choice (resource.beltPlan, beltCol1..4) over the band's columns. The tier caps stay.
func (b Band) WithSpec(s resource.PlanSpec) (Band, error) {
	if s.IsDefault() {
		return b, nil
	}
	var rp resource.BeltPlan
	for i, k := range b.Cols {
		switch k {
		case HP:
			rp.Cols[i] = resource.PotionLife
		case MP:
			rp.Cols[i] = resource.PotionMana
		case RV:
			rp.Cols[i] = resource.PotionRejuv
		}
	}
	rp, err := s.Apply(rp)
	if err != nil {
		return b, err
	}
	for i, k := range rp.Cols {
		b.Cols[i] = None
		switch k {
		case resource.PotionLife:
			b.Cols[i] = HP
		case resource.PotionMana:
			b.Cols[i] = MP
		case resource.PotionRejuv:
			b.Cols[i] = RV
		}
	}
	return b, nil
}

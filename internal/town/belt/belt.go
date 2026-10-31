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

// Slots returns the number of belt slots granted by the equipped belt (4 when none is worn).
func Slots(db *data.DB, inv *d2.Inventory) int {
	if inv != nil {
		if u := inv.Equipped(d2.BodyBelt); u != nil && u.Class >= 0 && u.Class < len(db.Items) {
			if b := db.BeltByIndex(db.Items[u.Class].BeltIndex); b != nil && b.Slots >= Columns {
				return b.Slots
			}
		}
	}
	return Columns
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

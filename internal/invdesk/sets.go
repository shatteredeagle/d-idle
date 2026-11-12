package invdesk

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/status"
)

// Gear sets (API schema 7, docs/gearsets.md): named sets of equipment the PLAYER saves and applies. The bot never
// chooses, saves or applies a set (docs/gear-policy.md): Apply is a player command that queues ordinary equip, unequip and
// take requests, which run at a safe moment like every other player command. Nothing is substituted: an item the set
// names that cannot be found is reported and skipped.
const (
	TargetHero = "hero"
	TargetMerc = "merc"
	// SetsSchema is the version of the sets block in the state file and in the view.
	SetsSchema = 1
	// DefaultMaxSets is the number of sets a hero may keep when the knob invdesk.maxSets is not wired.
	DefaultMaxSets = 8
	// MaxApplySteps bounds the requests one Apply queues (they do not count against MaxQueue).
	MaxApplySteps = 24
	MaxSetName    = 24
	// SetsLadderID is the row of the unlock ladder (data/unlocks/ladder.json) that opens the feature.
	SetsLadderID = "gearsets"
)

// SetItem is the identity of one remembered item: the same family as the sell marks (docs/inventory-ui.md). The id is
// tried first, because it is exact inside one game session; after a restart (new ids) or an identify (new name) the
// base code, quality and item level find it again, preferring the slot and the place it was saved from.
type SetItem struct {
	ID      uint32 `json:"id"`
	Code    string `json:"code"`
	Quality string `json:"quality,omitempty"`
	Level   int    `json:"level,omitempty"`
	Name    string `json:"name,omitempty"`
	Ident   bool   `json:"ident,omitempty"` // identified when saved: its name is part of what it is
	Where   string `json:"where,omitempty"` // worn, pack or stash when saved: a tie-break only
	X       int    `json:"x,omitempty"`
	Y       int    `json:"y,omitempty"`
}

// GearSet is one saved set: what the hero (or the merc) wore in each slot when the player saved it.
type GearSet struct {
	ID     string             `json:"id"`
	Name   string             `json:"name"`
	Target string             `json:"target"` // hero or merc
	Saved  int64              `json:"saved"`
	Slots  map[string]SetItem `json:"slots"`
}

// SetsGate says whether the Gear Sets feature is unlocked (the unlock ladder row `gearsets`) and, while it is not, the
// teaser. A nil gate means open.
type SetsGate func() (open bool, teaser string)

// SetSlotView is one remembered slot as the page lists it.
type SetSlotView struct {
	Slot    string `json:"slot"`
	Name    string `json:"name"`
	Code    string `json:"code"`
	Quality string `json:"quality,omitempty"`
}

// SetView is one saved set in the view.
type SetView struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Target   string        `json:"target"`
	Saved    int64         `json:"saved"`
	Slots    []SetSlotView `json:"slots"`
	Applying string        `json:"applying,omitempty"` // group id while steps of an apply of this set wait or run
}

// SetsView is the "sets" block of the view; absent when the desk has no inventory source.
type SetsView struct {
	Schema int       `json:"schema"`
	Open   bool      `json:"open"`
	Why    string    `json:"why,omitempty"` // the unlock teaser while locked
	Max    int       `json:"max"`
	Sets   []SetView `json:"sets"`
}

// SetSlotPreview says where the item a set names for one slot is now.
type SetSlotPreview struct {
	Slot   string `json:"slot"`
	Name   string `json:"name"`           // what the set names
	Now    string `json:"now,omitempty"`  // what is worn in the slot now
	Status string `json:"status"`         // worn, pack, stash, elsewhere, missing, blocked
	Why    string `json:"why,omitempty"`  // the reason of blocked, or a note
	ID     uint32 `json:"id,omitempty"`   // the item found
	From   string `json:"from,omitempty"` // the slot it is worn in now (status elsewhere)
}

// SetStep is one request an Apply would queue.
type SetStep struct {
	Op   string `json:"op"` // unequip, take, equip
	ID   uint32 `json:"id"`
	Slot string `json:"slot,omitempty"`
	Name string `json:"name"`
	Note string `json:"note,omitempty"`
}

// SetPreview is what Apply would do, shown before the player confirms.
type SetPreview struct {
	SetID   string           `json:"setId"`
	Name    string           `json:"name"`
	Target  string           `json:"target"`
	Slots   []SetSlotPreview `json:"slots"`
	Steps   []SetStep        `json:"steps"`
	Missing int              `json:"missing"`
	Ready   bool             `json:"ready"`         // Apply would queue the steps
	Why     string           `json:"why,omitempty"` // why it would not (nothing to do, no room, locked, merc)
}

// maxSets is the knob invdesk.maxSets (Config.MaxSets).
func (d *Desk) maxSets() int {
	d.mu.Lock()
	f := d.c.MaxSets
	d.mu.Unlock()
	if f != nil {
		if n := f(); n > 0 {
			return n
		}
	}
	return DefaultMaxSets
}

// SetMaxSets installs the knob invdesk.maxSets.
func (d *Desk) SetMaxSets(f func() int) { d.mu.Lock(); d.c.MaxSets = f; d.mu.Unlock() }

// SetSetsGate installs the unlock gate after the desk exists (the ladder is opened later than the desk).
func (d *Desk) SetSetsGate(g func() (bool, string)) { d.mu.Lock(); d.c.SetsGate = g; d.mu.Unlock() }

func (d *Desk) setsOpen() (bool, string) {
	d.mu.Lock()
	g := d.c.SetsGate
	d.mu.Unlock()
	if g == nil {
		return true, ""
	}
	return g()
}

func (d *Desk) setsGuard() error {
	if open, why := d.setsOpen(); !open {
		if why == "" {
			why = "Gear sets are not unlocked yet"
		}
		return refuse("locked", "%s", why)
	}
	return nil
}

func cleanName(n string) (string, error) {
	n = strings.Join(strings.Fields(n), " ")
	switch {
	case n == "":
		return "", refuse("name", "give the set a name")
	case utf8.RuneCountInString(n) > MaxSetName:
		return "", refuse("name", "a set name has at most %d characters", MaxSetName)
	}
	for _, r := range n {
		if unicode.IsControl(r) {
			return "", refuse("name", "the name has a character that cannot be shown")
		}
	}
	return n, nil
}

func (d *Desk) nameTakenLocked(name, except string) bool {
	for _, s := range d.st.Sets {
		if s.ID != except && strings.EqualFold(s.Name, name) {
			return true
		}
	}
	return false
}

func setItemOf(it status.Item, where string) SetItem {
	return SetItem{ID: it.ID, Code: it.Code, Quality: it.Quality, Level: it.Level, Name: itemName(it), Ident: it.Identified, Where: where, X: it.X, Y: it.Y}
}

// SaveSet snapshots what the hero (target "hero" or "") or the merc wears right now under a name.
func (d *Desk) SaveSet(name, target string) (GearSet, error) {
	if err := d.setsGuard(); err != nil {
		return GearSet{}, err
	}
	if target == "" {
		target = TargetHero
	}
	if target != TargetHero && target != TargetMerc {
		return GearSet{}, refuse("target", "a set is for the hero or the merc")
	}
	name, err := cleanName(name)
	if err != nil {
		return GearSet{}, err
	}
	if d.c.Inventory == nil {
		return GearSet{}, refuse("off", "inventory is not available")
	}
	data := d.c.Inventory()
	worn := data.Equipment
	if target == TargetMerc {
		if data.Merc == nil || !data.Merc.Hired {
			return GearSet{}, refuse("merc", "no mercenary is hired")
		}
		worn = data.Merc.Equipment
	}
	gs := GearSet{Target: target, Name: name, Slots: map[string]SetItem{}}
	for k, it := range worn {
		if it != nil {
			gs.Slots[k] = setItemOf(*it, "worn")
		}
	}
	if len(gs.Slots) == 0 {
		return GearSet{}, refuse("empty", "nothing is worn, so there is nothing to save")
	}
	limit := d.maxSets() // takes the lock itself
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.st.Sets) >= limit {
		return GearSet{}, refuse("full", "%d sets are saved: delete one first", limit)
	}
	if d.nameTakenLocked(name, "") {
		return GearSet{}, refuse("name", "a set called %q exists: pick another name", name)
	}
	n := 0
	for _, s := range d.st.Sets {
		var k int
		if _, e := fmt.Sscanf(s.ID, "gs-%d", &k); e == nil && k > n {
			n = k
		}
	}
	gs.ID, gs.Saved = fmt.Sprintf("gs-%d", n+1), d.c.Now().Unix()
	d.st.Sets, d.st.SetsSchema = append(d.st.Sets, gs), SetsSchema
	return gs, d.saveLocked()
}

// RenameSet changes the name of a saved set.
func (d *Desk) RenameSet(id, name string) error {
	if err := d.setsGuard(); err != nil {
		return err
	}
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.st.Sets {
		if d.st.Sets[i].ID == id {
			if d.nameTakenLocked(name, id) {
				return refuse("name", "a set called %q exists: pick another name", name)
			}
			d.st.Sets[i].Name = name
			return d.saveLocked()
		}
	}
	return refuse("set", "that set does not exist any more")
}

// DeleteSet removes a saved set. Steps of an apply already queued stay (cancel them from the queue).
func (d *Desk) DeleteSet(id string) error {
	if err := d.setsGuard(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.st.Sets {
		if d.st.Sets[i].ID == id {
			d.st.Sets = append(d.st.Sets[:i:i], d.st.Sets[i+1:]...)
			return d.saveLocked()
		}
	}
	return refuse("set", "that set does not exist any more")
}

func (d *Desk) findSet(id string) (GearSet, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, s := range d.st.Sets {
		if s.ID == id {
			return s, true
		}
	}
	return GearSet{}, false
}

// slotOrder lists the slots a plan walks: weapon hand first so a two-hander to one-hander swap frees the shield hand
// before the shield goes on.
var slotOrder = []string{"rhand", "lhand", "head", "neck", "torso", "belt", "feet", "gloves", "rring", "lring"}

func sortedSlots(m map[string]SetItem) []string {
	var out []string
	seen := map[string]bool{}
	for _, k := range slotOrder {
		if _, ok := m[k]; ok {
			out, seen[k] = append(out, k), true
		}
	}
	var rest []string
	for k := range m {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

type located struct {
	it    status.Item
	place string // worn, pack, stash
	slot  string // the worn slot
}

func poolOf(data status.InventoryData, target string) []located {
	var out []located
	worn := data.Equipment
	if target == TargetMerc && data.Merc != nil {
		worn = data.Merc.Equipment
	}
	keys := make([]string, 0, len(worn))
	for k := range worn {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if worn[k] != nil {
			out = append(out, located{*worn[k], "worn", k})
		}
	}
	for _, it := range data.Inventory {
		out = append(out, located{it, "pack", ""})
	}
	for _, it := range data.Stash {
		out = append(out, located{it, "stash", ""})
	}
	return out
}

// fitsRef is the sell marks' markFits on the status item: the same base, and the same quality or the same item level
// (an identify changes the name and may change what is known of the quality, never the base or the item level).
//
// An item that was identified when saved is also the same name: another item of the same base, quality and level is NOT
// that item (never substituted). An item saved unidentified may have been identified since (the name changed).
func fitsRef(ref SetItem, it status.Item) bool {
	if ref.Code != it.Code || !(ref.Quality == it.Quality || ref.Level != 0 && ref.Level == it.Level) {
		return false
	}
	return !ref.Ident || ref.Name == "" || ref.Name == itemName(it)
}

// resolve finds, for every slot of the set, the item it names. Pass one takes exact ids; pass two takes the best fitting
// unclaimed item (worn in that very slot first, then the place and cell it was saved from, then the same name). An item
// is claimed once, so two identical rings fill two slots. A slot with no fitting item stays out of the result.
func resolve(refs map[string]SetItem, pool []located) map[string]*located {
	res := map[string]*located{}
	claimed := map[int]bool{}
	slots := sortedSlots(refs)
	for _, k := range slots {
		ref := refs[k]
		if ref.ID == 0 {
			continue
		}
		for i := range pool {
			if !claimed[i] && pool[i].it.ID == ref.ID && fitsRef(ref, pool[i].it) {
				claimed[i], res[k] = true, &pool[i]
				break
			}
		}
	}
	for _, k := range slots {
		if res[k] != nil {
			continue
		}
		ref, best, rank := refs[k], -1, 99
		for i := range pool {
			if claimed[i] || !fitsRef(ref, pool[i].it) {
				continue
			}
			r := 3
			switch {
			case pool[i].place == "worn" && pool[i].slot == k:
				r = 0
			case pool[i].place == ref.Where && pool[i].it.X == ref.X && pool[i].it.Y == ref.Y:
				r = 1
			case ref.Name != "" && itemName(pool[i].it) == ref.Name:
				r = 2
			}
			if r < rank {
				best, rank = i, r
			}
		}
		if best >= 0 {
			claimed[best], res[k] = true, &pool[best]
		}
	}
	return res
}

// isOffHand matches the two-hander and shield refusal of the leveler (leveler.ErrOffHand) without importing it.
func isOffHand(err error) bool {
	return err != nil && strings.Contains(err.Error(), "cannot be worn together")
}

func wornName(data status.InventoryData, slot string) string {
	if it := data.Equipment[slot]; it != nil {
		return itemName(*it)
	}
	return ""
}

// mercWhy is why nothing can be done to the merc's gear yet: the executor (docs/research/merc-gear.md stage C) does
// not exist. Reading, saving and previewing his sets works.
func (d *Desk) mercWhy(data status.InventoryData) string {
	switch m := data.Merc; {
	case m == nil || !m.Hired:
		return "no mercenary is hired"
	case !m.Alive:
		return "the mercenary is dead: revive him first"
	}
	return "the bot cannot change the mercenary's gear yet: the sets you save for him are kept, and you equip him by hand"
}

// PreviewSet works out what Apply would do now, without changing anything.
func (d *Desk) PreviewSet(id string) (SetPreview, error) {
	pv, _, err := d.plan(id)
	return pv, err
}

type planStep = SetStep

// plan resolves the set against the current pack, stash and worn items and orders the steps: first take off what must
// go (the shield for a two-hander, an item worn in another slot), then fetch from the stash, then wear, weapon hand first.
func (d *Desk) plan(id string) (SetPreview, []planStep, error) {
	if err := d.setsGuard(); err != nil {
		return SetPreview{}, nil, err
	}
	gs, ok := d.findSet(id)
	if !ok {
		return SetPreview{}, nil, refuse("set", "that set does not exist any more")
	}
	if d.c.Inventory == nil {
		return SetPreview{}, nil, refuse("off", "inventory is not available")
	}
	d.mu.Lock()
	eq, un, hands, sv := d.eq, d.un, d.hands, d.sv
	d.mu.Unlock()
	data := d.c.Inventory()
	s := d.snap()
	pv := SetPreview{SetID: gs.ID, Name: gs.Name, Target: gs.Target, Slots: []SetSlotPreview{}, Steps: []SetStep{}}
	res := resolve(gs.Slots, poolOf(data, gs.Target))
	want := map[string]*located{} // slots that will receive an item
	for _, k := range sortedSlots(gs.Slots) {
		ref, loc := gs.Slots[k], res[k]
		sp := SetSlotPreview{Slot: k, Name: ref.Name, Now: wornName(data, k)}
		if gs.Target == TargetMerc {
			sp.Now = ""
			if data.Merc != nil && data.Merc.Equipment[k] != nil {
				sp.Now = itemName(*data.Merc.Equipment[k])
			}
		}
		switch {
		case loc == nil:
			sp.Status, sp.Why = "missing", "not found in the backpack, stash or worn: skipped, nothing is substituted"
			pv.Missing++
		case loc.place == "worn" && loc.slot == k:
			sp.Status, sp.ID = "worn", loc.it.ID
		case loc.place == "worn":
			sp.Status, sp.ID, sp.From = "elsewhere", loc.it.ID, loc.slot
			want[k] = loc
		default:
			sp.Status, sp.ID = loc.place, loc.it.ID
			if loc.place == "pack" && gs.Target == TargetHero && eq != nil && s != nil {
				if _, err := eq.CheckPlayerEquip(s, loc.it.ID, SlotBody[k]); err != nil && !isOffHand(err) {
					sp.Status, sp.Why = "blocked", err.Error()
					break
				}
			}
			want[k] = loc
		}
		pv.Slots = append(pv.Slots, sp)
	}
	if gs.Target == TargetMerc {
		pv.Why = d.mercWhy(data)
		return pv, nil, nil
	}
	// two-hander and shield: whatever is left in the hands after the swap must be legal
	twoH := func(id uint32) bool { return hands != nil && s != nil && hands.IsTwoHanded(s, id) }
	final := func(slot string) *status.Item {
		if l, ok := want[slot]; ok {
			return &l.it
		}
		return data.Equipment[slot] // not receiving anything (not named, missing or blocked): it keeps what it has
	}
	var steps []planStep
	blockSlot := func(slot, why string) {
		delete(want, slot)
		for i := range pv.Slots {
			if pv.Slots[i].Slot == slot {
				pv.Slots[i].Status, pv.Slots[i].Why = "blocked", why
			}
		}
	}
	r, l := final("rhand"), final("lhand")
	if r != nil && l != nil && (twoH(r.ID) || twoH(l.ID)) {
		_, rw := want["rhand"]
		_, lw := want["lhand"]
		switch {
		case !lw && data.Equipment["lhand"] != nil:
			it := *data.Equipment["lhand"]
			steps = append(steps, SetStep{Op: KindUnequip, ID: it.ID, Slot: "lhand", Name: itemName(it), Note: "a two-handed weapon and a shield cannot be worn together"})
		case !rw && data.Equipment["rhand"] != nil:
			it := *data.Equipment["rhand"]
			steps = append(steps, SetStep{Op: KindUnequip, ID: it.ID, Slot: "rhand", Name: itemName(it), Note: "a two-handed weapon and a shield cannot be worn together"})
		default:
			blockSlot("lhand", "a two-handed weapon and a shield cannot be worn together")
		}
	}
	for _, k := range sortedSlots(gs.Slots) {
		if loc := want[k]; loc != nil && loc.place == "worn" { // worn in another slot: off first, then on
			steps = append(steps, SetStep{Op: KindUnequip, ID: loc.it.ID, Slot: loc.slot, Name: itemName(loc.it), Note: "worn in another slot"})
		}
	}
	for _, k := range sortedSlots(gs.Slots) {
		if loc := want[k]; loc != nil && loc.place == "stash" {
			steps = append(steps, SetStep{Op: KindTake, ID: loc.it.ID, Name: itemName(loc.it), Note: "from the stash, in town"})
		}
	}
	for _, k := range sortedSlots(gs.Slots) {
		if loc := want[k]; loc != nil {
			steps = append(steps, SetStep{Op: KindEquip, ID: loc.it.ID, Slot: k, Name: itemName(loc.it)})
		}
	}
	// the backpack must take what comes off and what comes out of the stash (equips free cells: not counted, so this is
	// the safe side)
	g := newPackGrid(data.Inventory)
	for _, st := range steps {
		var it status.Item
		switch st.Op {
		case KindUnequip:
			it = findItem(poolOf(data, TargetHero), st.ID, "worn")
		case KindTake:
			it = findItem(poolOf(data, TargetHero), st.ID, "stash")
		default:
			continue
		}
		if !g.place(it.W, it.H) {
			pv.Why = fmt.Sprintf("the backpack has no free spot for %s: make room first", st.Name)
			break
		}
	}
	switch {
	case pv.Why != "":
	case len(steps) == 0 && pv.Missing == 0 && !anyBlocked(pv):
		pv.Why = "the set is already worn"
	case len(steps) == 0:
		pv.Why = "nothing in the set can be applied"
	case len(steps) > MaxApplySteps:
		pv.Why = fmt.Sprintf("more than %d steps", MaxApplySteps)
	case eq == nil:
		pv.Why = "equipping is not available on this bot"
	}
	for _, st := range steps {
		if pv.Why != "" {
			break
		}
		if st.Op == KindUnequip && un == nil {
			pv.Why = "taking items off is not available on this bot"
		}
		if st.Op == KindTake && stashOf(sv) == nil {
			pv.Why = "moving items out of the stash is not available on this bot"
		}
	}
	if steps == nil {
		steps = []planStep{}
	}
	pv.Steps, pv.Ready = steps, pv.Why == ""
	return pv, steps, nil
}

func stashOf(sv Services) StashServices { s, _ := sv.(StashServices); return s }

func findItem(pool []located, id uint32, place string) status.Item {
	for _, l := range pool {
		if l.it.ID == id && l.place == place {
			return l.it
		}
	}
	return status.Item{}
}

func anyBlocked(pv SetPreview) bool {
	for _, s := range pv.Slots {
		if s.Status == "blocked" {
			return true
		}
	}
	return false
}

// ApplySet queues the steps of the set's preview as one group. Player command only: nothing in the bot calls it. The
// steps run in order at a safe moment; a failed or cancelled step ends the rest of the group.
func (d *Desk) ApplySet(id string) (SetPreview, error) {
	d.mu.Lock()
	busy := false
	for _, x := range d.st.Queue {
		busy = busy || x.Group != ""
	}
	d.mu.Unlock()
	if busy {
		return SetPreview{}, refuse("queued", "a set is being applied already: cancel it first")
	}
	pv, steps, err := d.plan(id)
	if err != nil {
		return pv, err
	}
	if pv.Target == TargetMerc {
		return pv, refuse("merc", "%s", pv.Why)
	}
	if !pv.Ready {
		return pv, refuse("plan", "%s", pv.Why)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, x := range d.st.Queue {
		if x.Group != "" {
			return pv, refuse("queued", "a set is being applied already: cancel it first")
		}
	}
	grp := fmt.Sprintf("a%d", d.c.Now().UnixNano()/1e6)
	now := d.c.Now().Unix()
	for i, st := range steps {
		r := Request{ID: st.ID, Kind: st.Op, Slot: st.Slot, Name: st.Name, State: StateQueued, At: now, Group: grp, GroupName: pv.Name, Seq: i + 1}
		d.st.Queue = append(d.st.Queue, r)
	}
	return pv, d.saveLocked()
}

// CancelApply removes the queued steps of an apply group (the step already running finishes).
func (d *Desk) CancelApply(group string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := d.endGroupLocked(group, "cancelled")
	if n == 0 {
		return refuse("item", "that apply is not queued any more")
	}
	return d.saveLocked()
}

// endGroupLocked drops the queued steps of a group into Recent as cancelled with the reason; returns how many.
func (d *Desk) endGroupLocked(group, why string) int {
	q := d.st.Queue[:0:0]
	n := 0
	for _, x := range d.st.Queue {
		if x.Group == group && x.State == StateQueued {
			x.State, x.Reason = StateCancelled, why
			d.recentLocked(x)
			n++
			continue
		}
		q = append(q, x)
	}
	d.st.Queue = q
	return n
}

// groupBlockedLocked: a step waits while an earlier step of its group is still queued or running. Two takes do not
// wait for each other (they share one stash visit).
func (d *Desk) groupBlockedLocked(r Request) bool {
	if r.Group == "" {
		return false
	}
	for _, x := range d.st.Queue {
		if x.Group == r.Group && x.Seq < r.Seq && !(x.kind() == KindTake && r.kind() == KindTake) {
			return true
		}
	}
	return false
}

// singles counts the requests that count against MaxQueue (a set apply has its own bound).
func singles(q []Request) int {
	n := 0
	for _, r := range q {
		if r.Group == "" {
			n++
		}
	}
	return n
}

// performUnequip is the executor call of an unequip request; the item must still be worn in the slot it was queued for.
func (d *Desk) performUnequip(ctx context.Context, r Request) error {
	d.mu.Lock()
	un := d.un
	d.mu.Unlock()
	if un == nil {
		return refuse("off", "taking items off is not available on this bot")
	}
	if s := d.snap(); s != nil && s.Inventory != nil {
		worn := false
		for _, u := range s.Inventory.Container(d2.ContEquipment) {
			if u.ID == r.ID {
				worn = true
			}
		}
		if !worn {
			return fmt.Errorf("%s is not worn any more", r.Name)
		}
	}
	return un.PlayerUnequip(ctx, r.ID)
}

// setsView builds the sets block of the view.
func (d *Desk) setsView() *SetsView {
	open, why := d.setsOpen()
	v := &SetsView{Schema: SetsSchema, Open: open, Why: why, Max: d.maxSets(), Sets: []SetView{}}
	if !open {
		return v
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	applying := map[string]string{}
	for _, r := range d.st.Queue {
		if r.Group != "" {
			applying[r.GroupName] = r.Group
		}
	}
	for _, s := range d.st.Sets {
		sv := SetView{ID: s.ID, Name: s.Name, Target: s.Target, Saved: s.Saved, Slots: []SetSlotView{}, Applying: applying[s.Name]}
		for _, k := range sortedSlots(s.Slots) {
			it := s.Slots[k]
			sv.Slots = append(sv.Slots, SetSlotView{Slot: k, Name: it.Name, Code: it.Code, Quality: it.Quality})
		}
		v.Sets = append(v.Sets, sv)
	}
	return v
}

// RequestEquipFor is RequestEquip with a target. The merc has no executor yet: his drops are answered with the reason
// (code "merc") so the page can say why the slot is red.
func (d *Desk) RequestEquipFor(id uint32, slot, target string) (Request, error) {
	switch target {
	case "", TargetHero:
		return d.RequestEquip(id, slot)
	case TargetMerc:
		var data status.InventoryData
		if d.c.Inventory != nil {
			data = d.c.Inventory()
		}
		return Request{}, refuse("merc", "%s", d.mercWhy(data))
	}
	return Request{}, refuse("target", "a target is the hero or the merc")
}

// RequestUnequipFor is RequestUnequip with a target (see RequestEquipFor).
func (d *Desk) RequestUnequipFor(slot, target string) (Request, error) {
	switch target {
	case "", TargetHero:
		return d.RequestUnequip(slot)
	case TargetMerc:
		return d.RequestEquipFor(0, slot, target)
	}
	return Request{}, refuse("target", "a target is the hero or the merc")
}

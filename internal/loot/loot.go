// Package loot decides what to do with items (the pickit) and picks them up off the ground with hover
// confirmation. Contract: docs/archive/wave2.md "Looter". Details: docs/town.md.
package loot

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/town/belt"
	"d2lod-bot/internal/town/hand"
)

// Mover is the slice of the navigator the looter uses.
type Mover interface {
	GoTo(ctx context.Context, p input.Point) error
}

// Verdict is the pickit's answer for one item.
type Verdict struct {
	Action Action
	Tier   string
	Prio   int
	Rule   string // name of the matching rule, "" for the default
	Reason string
	NeedID bool // the verdict is provisional: identify the item and ask again
}

// Wanted reports whether the item is worth carrying (keep or stash).
func (v Verdict) Wanted() bool { return v.Action == Keep || v.Action == Stash }

func (v Verdict) String() string {
	s := fmt.Sprintf("%s [%s] %s", v.Action, v.Tier, v.Reason)
	if v.NeedID {
		s += " (identify first)"
	}
	return s
}

// Needs answers the "needed" conditions of rules.
type Needs interface {
	Want(kind string, b *data.ItemBase) bool
}

// Ctx is everything besides the item that rules may look at.
type Ctx struct {
	CLvl  int
	Gold  int
	Needs Needs    // nil: every need is satisfied (rules with "needed" match)
	Stats d2.Stats // the item's stat list, nil when unreadable

	// Naked slots: the character has no weapon / no body armor, so a usable one is kept to wear whatever it is worth
	// (G1 attempt 6: Soaky sold a found Leather Armor and stayed gear-gated). Str, Dex and Class (class code) are the
	// wearer's, for the requirement check.
	NeedWeapon, NeedBody bool
	// Worn is the WearScore of what each armor body slot holds now (absent key = empty slot). WearBetter switches on
	// the equip-if-better rule: a wearable armor piece that scores higher than the slot's item is kept to be worn.
	// TwoHander: the weapon in hand is two-handed, so no shield can be worn.
	Worn       map[int]int
	WearBetter bool
	TwoHander  bool
	Str, Dex   int
	Class      string
}

// Looter is the loot behaviour. Create with New; safe for one goroutine.
type Looter struct {
	potionSkipMu sync.Mutex
	potionSkipAt map[string]time.Time // logPotionSkip rate limit

	cursorFails atomic.Int32 // pack clears of a held item that failed in a row (clearHeld)

	e    *env.Env
	mv   Mover
	pk   *Pickit
	hand *hand.Hand

	// NoRunewords turns off the runeword-base rules (pickit "needed": "runeword"). The zero value allows them.
	// Q1 (docs/items.md section 9): whether ladder-flagged runewords work in offline 1.14d SP is UNVERIFIED, we
	// assume they do. TODO live probe (P10): try a ladder-flagged recipe offline, and set this from the result.
	NoRunewords bool

	// recovering: the bot's recovery phase (docs/recovery.md) is on. Every gold pile and every sellable magic or rare is
	// taken then, up to a higher gold line than the poor mode, and sell pickups may use the last cells of the pack.
	recovering atomic.Bool
	recTotals  [3]atomic.Int64 // picked items, gold gained, "no room" skips since the process started (RecoveryTotals)

	leftBehind atomic.Int64 // valuable items the last pickup left for lack of room (LeftBehind)

	ignore atomic.Pointer[func(d2.Unit) bool] // ground items the player trashed (SetIgnore)

	build string // leveler build name; with the snapshot's difficulty it picks the belt columns (SetBuild)

	// Seams for tests.
	snap  func() (*d2.Snapshot, error)
	stats func(addr uint32) (d2.Stats, error)
}

// GemKeep is how many gems of one base code the looter keeps for cube upgrades before surplus ones are sold.
const GemKeep = 6

// New builds a Looter with the built-in starter pickit.
func New(e *env.Env, mv Mover) *Looter {
	l := &Looter{e: e, mv: mv, pk: Starter(), hand: hand.New(e), snap: e.Snapshot}
	if e.Read != nil {
		l.stats = e.Read.UnitStats
	}
	return l
}

// SetRecovering switches the recovery pickup rules (see Looter.recovering).
func (l *Looter) SetRecovering(on bool) { l.recovering.Store(on) }

// RecoveryTotals are the running totals of picked items, gold gained by piles and items left for lack of room. The bot logs
// the deltas once a minute during recovery.
func (l *Looter) RecoveryTotals() (picked, gold, full int) {
	return int(l.recTotals[0].Load()), int(l.recTotals[1].Load()), int(l.recTotals[2].Load())
}

// goldBar is the smallest gold pile worth walking to in the snapshot's area: the flat GoldMinStack, lowered to the pile
// size that GoldPileShare percent of the area's piles reach (a pile is ilvl .. 6 ilvl - 1 coins and ilvl is the monster
// level, treasure-trace B3: the flat 10 skipped every Blood Moor pile). Unknown area data keeps the flat bar. In Normal
// the Levels.txt monster level is an upper bound of the real unit levels, so the bar errs a little high there.
func (l *Looter) goldBar(s *d2.Snapshot) int {
	t := l.e.Tune.Loot
	if t.GoldPileShare <= 0 || l.e.Data == nil {
		return t.GoldMinStack
	}
	a := l.e.Data.AreaByID(s.Player.Area)
	if a == nil {
		return t.GoldMinStack
	}
	ml := a.MonLvl[min(max(int(s.Player.Difficulty), 0), 2)]
	if ml <= 0 {
		return t.GoldMinStack
	}
	return min(t.GoldMinStack, data.PileMinWorth(ml, float64(t.GoldPileShare)/100))
}

// poorBelow is the gold under which the looter is in poor mode: the tuned line, or the recovery kit line while recovering.
func (l *Looter) poorBelow() int {
	n := l.e.Tune.Loot.PoorGoldBelow
	if l.recovering.Load() {
		n = max(n, l.e.Tune.Resource.RecoveryKitGold)
	}
	return n
}

// SetBuild tells the looter which build the character plays, so potion pickup in Nightmare and Hell follows the
// same mana-heavy belt columns the shop buys by (belt.PlanForBuild). Normal ignores it.
func (l *Looter) SetBuild(name string) { l.build = name }

// SetPickit replaces the rule set (for example one loaded from a user file).
func (l *Looter) SetPickit(p *Pickit) { l.pk = p }

// Pickit returns the active rule set.
func (l *Looter) Pickit() *Pickit { return l.pk }

func (l *Looter) base(class int) *data.ItemBase {
	if db := l.e.Data; db != nil && class >= 0 && class < len(db.Items) {
		return db.Items[class]
	}
	return nil
}

// Evaluate is the contract entry point. A bare d2.Item carries no base-item class, so only the quality,
// flag and item-level conditions can match; use EvaluateUnit when the unit is at hand.
func (l *Looter) Evaluate(it d2.Item) Verdict {
	return l.evaluate(nil, &it, Ctx{CLvl: 1}, nil)
}

// EvaluateUnit judges an item unit with its base row, and reads the stat list when a rule needs it.
// The player's level and needs are taken from a fresh snapshot when it can be read.
func (l *Looter) EvaluateUnit(u d2.Unit) Verdict {
	c := Ctx{CLvl: 1}
	if s, err := l.snap(); err == nil && s.State == d2.StateInGame && s.Player != nil {
		c = l.ctxFrom(s)
	}
	return l.EvaluateWith(u, c)
}

// EvaluatorFor is EvaluateUnit bound to the snapshot s: the context is built once and no game read happens per
// item, so a page that judges every carried item costs one pass over the data, not one snapshot read per item.
func (l *Looter) EvaluatorFor(s *d2.Snapshot) func(u d2.Unit) Verdict {
	c := Ctx{CLvl: 1}
	if s != nil && s.State == d2.StateInGame && s.Player != nil {
		c = l.ctxFrom(s)
	}
	return func(u d2.Unit) Verdict { return l.EvaluateWith(u, c) }
}

// EvaluateWith is EvaluateUnit with an explicit context (no snapshot read).
func (l *Looter) EvaluateWith(u d2.Unit, c Ctx) Verdict {
	if u.Item == nil {
		return Verdict{Action: Drop, Tier: "C", Reason: "not an item"}
	}
	var st d2.Stats
	if c.Stats == nil && l.stats != nil && u.Addr != 0 {
		if s, err := l.stats(u.Addr); err == nil {
			st = s
		}
	} else {
		st = c.Stats
	}
	c.Stats = st
	b := l.base(u.Class)
	v := l.evaluate(b, u.Item, c, &u)
	if v.Action != Keep && l.fillsEmptySlot(b, u.Item, c) {
		return Verdict{Action: Keep, Tier: "A", Prio: 75, Rule: nakedRule, Reason: "empty weapon or body armor slot"}
	}
	if v.Action == Drop && c.Gold < l.poorBelow() && l.poorSellable(b, u.Item) {
		return Verdict{Action: Sell, Tier: "B", Prio: 3, Rule: poorRule, Reason: "poor: junk gear sells for gold"}
	}
	return v
}

// nakedRule names the verdict of a weapon or body armor kept for an empty slot.
const nakedRule = "empty slot"

// fillsEmptySlot reports whether it is a weapon or body armor the naked character can wear now: base requirements
// (strength, dexterity, level, class) met. Any quality passes: an empty slot is worse than a gray item.
func (l *Looter) fillsEmptySlot(b *data.ItemBase, it *d2.Item, c Ctx) bool {
	db := l.e.Data
	if b == nil || db == nil || b.Quest {
		return false
	}
	ty := db.ItemType(b.Type)
	if ty == nil {
		return false
	}
	if c.WearBetter && b.Kind == data.KindArmor {
		return l.wearsBetter(b, it, c)
	}
	if !(c.NeedWeapon || c.NeedBody) {
		return false
	}
	switch {
	case c.NeedBody && b.Kind == data.KindArmor && (ty.BodyLocs[0] == "tors" || ty.BodyLocs[1] == "tors"):
	case c.NeedWeapon && b.Kind == data.KindWeapon && !db.IsA(b.Type, "miss") && !db.IsA(b.Type, "thro"):
	default:
		return false
	}
	if ty.Class != "" && ty.Class != c.Class {
		return false
	}
	return b.ReqStr <= c.Str && b.ReqDex <= c.Dex && b.LevelReq <= c.CLvl
}

// wearsBetter is the early-game rule for armor: the piece fits the wearer (class, strength, dexterity, level) and its
// slot is empty or holds something that scores lower.
func (l *Looter) wearsBetter(b *data.ItemBase, it *d2.Item, c Ctx) bool {
	body, ok := WearSlot(l.e.Data, b)
	if !ok || body == d2.BodyLHand && c.TwoHander {
		return false
	}
	ty := l.e.Data.ItemType(b.Type)
	if ty.Class != "" && ty.Class != c.Class || b.ReqStr > c.Str || b.ReqDex > c.Dex || b.LevelReq > c.CLvl {
		return false
	}
	if cur, filled := c.Worn[body]; filled {
		return WearScore(b, c.Stats) > cur
	}
	return true
}

// poorRule names the verdict of a junk weapon or armor taken while the character is broke.
const poorRule = "poor sell"

// poorSellable reports whether a plain (white or gray) weapon or armor sells for enough to be worth carrying home
// while the character is poor. Normal-quality gear is what a clvl 8 field drops, and the pickit calls it junk (LIVE
// G1 attempt 5: 167 kills in Blood Moor, no pickup, gold stuck at 0 with no weapon and no armor to rebuy).
func (l *Looter) poorSellable(b *data.ItemBase, it *d2.Item) bool {
	if b == nil || it.Quality < 1 || it.Quality > 3 || b.Quest {
		return false
	}
	if !l.isAny(b, []string{"weap", "armo", "tors", "helm", "shld", "boot", "glov", "belt"}) {
		return false
	}
	return SellValue(l.e.Data, b, it, 0) >= l.e.Tune.Loot.PoorMinSell
}

func (l *Looter) evaluate(b *data.ItemBase, it *d2.Item, c Ctx, u *d2.Unit) Verdict {
	for i := range l.pk.Rules {
		r := &l.pk.Rules[i]
		if l.matches(&r.Match, b, it, c) {
			return verdictOf(r)
		}
	}
	v := verdictOf(&l.pk.Default)
	v.Rule = ""
	return v
}

func verdictOf(r *Rule) Verdict {
	a, _ := parseAction(r.Action)
	return Verdict{Action: a, Tier: r.Tier, Prio: r.Prio, Rule: r.Name, Reason: r.Reason, NeedID: r.NeedID}
}

func (l *Looter) matches(m *Match, b *data.ItemBase, it *d2.Item, c Ctx) bool {
	if len(m.Codes) > 0 {
		if b == nil || !anyGlob(m.Codes, b.Code) {
			return false
		}
	}
	if len(m.Types) > 0 {
		if b == nil || !l.isAny(b, m.Types) {
			return false
		}
	}
	if len(m.NotTypes) > 0 && b != nil && l.isAny(b, m.NotTypes) {
		return false
	}
	if m.Quest && (b == nil || !b.Quest) {
		return false
	}
	if len(m.Qualities) > 0 {
		ok := false
		for _, q := range m.Qualities {
			if qualityNo(q) == it.Quality {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	if m.Identified != nil && it.Identified() != *m.Identified {
		return false
	}
	if m.Ethereal != nil && it.Ethereal() != *m.Ethereal {
		return false
	}
	if m.ClvlMin > 0 && c.CLvl < m.ClvlMin || m.ClvlMax > 0 && c.CLvl > m.ClvlMax {
		return false
	}
	if m.ILvlMin > 0 && it.ILvl < m.ILvlMin || m.ILvlMax > 0 && it.ILvl > m.ILvlMax {
		return false
	}
	if len(m.Stats) > 0 {
		if c.Stats == nil {
			return false
		}
		for _, sc := range m.Stats {
			if !sc.holds(c.Stats) {
				return false
			}
		}
	}
	if len(m.AnyStats) > 0 {
		if c.Stats == nil {
			return false
		}
		ok := false
		for _, sc := range m.AnyStats {
			if sc.holds(c.Stats) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if m.Needed != "" && c.Needs != nil && !c.Needs.Want(m.Needed, b) {
		return false
	}
	return true
}

func (sc StatCond) holds(st d2.Stats) bool {
	if !st.Has(sc.Stat) {
		return false
	}
	v := int(st.Get(sc.Stat)) >> uint(sc.Shift)
	return v >= sc.Min && (sc.Max == 0 || v <= sc.Max)
}

func anyGlob(pats []string, s string) bool {
	for _, p := range pats {
		if ok, _ := path.Match(p, s); ok {
			return true
		}
	}
	return false
}

func (l *Looter) isAny(b *data.ItemBase, types []string) bool {
	for _, t := range types {
		if b.Type == t || b.Type2 == t || l.e.Data != nil && (l.e.Data.IsA(b.Type, t) || b.Type2 != "" && l.e.Data.IsA(b.Type2, t)) {
			return true
		}
	}
	return false
}

// SellValue is what a vendor pays for the item, by game.exe's formula (VERIFIED, docs/research/price-trace.md): cost / 2,
// an ethereal item and a class item a quarter each, quality extras only when identified, capped at maxBuy (0: no
// cap), at least 1. Armor uses its base maximum defence (the item's own defence is not read here: an upper bound).
// The stat cost of identified magic, rare, set and unique items is not modelled, so for those it is a lower bound.
func SellValue(db *data.DB, b *data.ItemBase, it *d2.Item, maxBuy int) int {
	if b == nil || it == nil {
		return 0
	}
	f := db.PriceFactsOf(b)
	f.Identified, f.Ethereal, f.Low = it.Identified(), it.Ethereal(), it.Quality == d2.QualityLow
	return b.SellPrice(f, maxBuy)
}

// MaxBuyOf is the most the vendors of the player's current act pay for one item (npc.txt max buy).
func MaxBuyOf(p *d2.Player) int {
	if p == nil {
		return 0
	}
	return data.MaxBuy(data.ActOfArea(p.Area), int(p.Difficulty))
}

// ctxFrom builds a rule context (level, gold, needs) from a snapshot.
func (l *Looter) ctxFrom(s *d2.Snapshot) Ctx {
	c := Ctx{CLvl: s.Player.Level, Gold: s.Player.Gold}
	c.Needs = needsOf{l: l, s: s}
	if inv := s.Inventory; inv != nil {
		c.NeedWeapon = !hand.FilledEither(inv, d2.BodyRHand)
		c.NeedBody = !hand.FilledEither(inv, d2.BodyTorso)
		c.Str, c.Dex = s.Player.Strength, s.Player.Dexterity
		c.WearBetter = resource.Early(s.Player.Level, l.e.Tune.Resource.EarlyGearMax)
		if c.WearBetter {
			c.Worn = map[int]int{}
			for _, u := range inv.Container(d2.ContEquipment) {
				if u.Item == nil {
					continue
				}
				b := l.base(u.Class)
				if body, ok := WearSlot(l.e.Data, b); ok && body == u.Item.Body {
					var st d2.Stats
					if l.stats != nil && u.Addr != 0 {
						st, _ = l.stats(u.Addr)
					}
					c.Worn[body] = WearScore(b, st)
				}
				if u.Item.Body == d2.BodyRHand && b != nil && b.TwoHanded {
					c.TwoHander = true
				}
			}
		}
		if cls := l.e.Data.ClassByName(s.Player.Class.String()); cls != nil {
			c.Class = cls.Code
		}
	}
	return c
}

// wantRejuv is the emergency-heal carry rule (REJUVUSE, SME advice EXP): up to Loot.RejuvSpare rejuvenation potions in
// total, belt plus pack, with Full preferred: at the cap a Full is still wanted while a regular one sits in the pack
// (town sells the regular one).
func (n needsOf) wantRejuv(b *data.ItemBase, st belt.State) bool {
	db, inv := n.l.e.Data, n.s.Inventory
	limit := n.l.e.Tune.Loot.RejuvSpare
	have := st.Count[belt.RV] + belt.PackCount(db, inv, belt.RV)
	if have < limit {
		return true
	}
	if limit <= 0 || belt.Tier(b) < 2 {
		return false
	}
	for _, u := range inv.Container(d2.ContInventory) {
		if u.Class >= 0 && u.Class < len(db.Items) && belt.PotionKind(db.Items[u.Class]) == belt.RV && belt.Tier(db.Items[u.Class]) < 2 {
			return true
		}
	}
	return false
}

// lowGem reports a chipped or flawed gem or skull (codes gc*, gf*, skc, skf).
func lowGem(code string) bool {
	return strings.HasPrefix(code, "gc") || strings.HasPrefix(code, "gf") || code == "skc" || code == "skf"
}

// haveCube reports whether the Horadric Cube (code box) is in the pack or the stash: the Cube recipe is available from
// then on. UNVERIFIED that no earlier state (the quest chain) should count; the Cube item is the only existing helper.
func (n needsOf) haveCube() bool {
	for _, c := range []d2.Container{d2.ContInventory, d2.ContStash} {
		for _, u := range n.s.Inventory.Container(c) {
			if bb := n.l.base(u.Class); bb != nil && bb.Code == "box" {
				return true
			}
		}
	}
	return false
}

// freeCells counts the unused cells of the pack grid (items' width times height against the grid size).
func (n needsOf) freeCells() int {
	cells := 40 // the classic 10 x 4 pack
	for _, u := range n.s.Inventory.Container(d2.ContInventory) {
		if bb := n.l.base(u.Class); bb != nil {
			cells -= bb.Width * bb.Height
		}
	}
	return cells
}

// TomeCap is how many scrolls a tome of town portal or identify holds (S: D2 rules; town.tomeCap is the same number).
const TomeCap = 20

const statQuantity = 70 // the tome's scroll count; an empty tome has no stat list at all (town.qty, V live i4)

// tomeQty reads a tome's scroll count; an unreadable or missing stat counts as empty so the scroll is still taken.
func (n needsOf) tomeQty(u d2.Unit) int {
	if n.l.stats == nil || u.Addr == 0 {
		return 0
	}
	st, err := n.l.stats(u.Addr)
	if err != nil || !st.Has(statQuantity) {
		return 0
	}
	return int(st.Get(statQuantity))
}

type needsOf struct {
	l *Looter
	s *d2.Snapshot
}

// Want implements Needs from the player's belt and pack.
func (n needsOf) Want(kind string, b *data.ItemBase) bool {
	p, inv, db := n.s.Player, n.s.Inventory, n.l.e.Data
	switch kind {
	case "potion":
		k := belt.PotionKind(b)
		if k == belt.None {
			return false
		}
		if k != belt.RV {
			// SME EXP (docs/lootskip.md): a tier below the floor is not worth a belt box or a pack cell.
			if floor, _ := PotionTierFloor(p.Class, p.Level, p.Difficulty); belt.Tier(b) < floor {
				return false
			}
		}
		st := belt.Read(db, inv)
		band := belt.PlanForBuild(p.Class, p.Level, p.Difficulty, n.l.build) // same plan the shop buys by, else mana-hungry classes pick up the wrong kind
		if b, err := band.WithSpec(resource.SpecOfEnv(n.l.e.Tune.Resource)); err == nil {
			band = b // the player's belt plan (docs/belt-plan.md); a bad value is logged by the shop
		}
		// B1: game.exe puts a clicked ground potion on the belt first (same-kind column with room, else an empty column),
		// so a potion the plan carries and the belt will take is wanted whatever the pack holds; the pack buffer below
		// only limits potions that would land in the pack.
		if band.Capacity(k, st.Slots) > 0 && belt.WillBelt(db, inv, k) {
			// FIXBELT2: the click cannot be steered, so a potion that would open the empty column kept for another planned kind
			// (healing in the mana column: 42 of 47 failed fills in post-fix soak 1) is left on the ground.
			return belt.WillBeltFor(db, inv, k, band)
		}
		if k == belt.RV {
			return n.wantRejuv(b, st)
		}
		have := belt.PackCount(db, inv, k)
		want := have < n.l.e.Tune.Loot.PotionBuffer
		if !want {
			n.l.logPotionSkip(b, have, st.Free())
		}
		return want
	case "scroll":
		if b == nil {
			return false
		}
		tome := "tbk"
		if b.Code == "isc" {
			tome = "ibk"
		}
		have := 0
		var tomeU *d2.Unit
		for _, u := range inv.Container(d2.ContInventory) {
			if bb := n.l.base(u.Class); bb != nil {
				switch bb.Code {
				case b.Code:
					have++
				case tome:
					u := u
					tomeU = &u
				}
			}
		}
		if tomeU != nil {
			return n.tomeQty(*tomeU) < TomeCap // user rule: only until the tome is full, no loose scrolls beside it
		}
		// No tome: scrolls do not stack, each takes a cell, so carry only the shopping target (a pack of 20 loose
		// scrolls was half the backpack).
		target := n.l.e.Tune.Town.TPScrollTarget
		if b.Code == "isc" {
			target = max(n.l.e.Tune.Town.IDScrollTarget, 2)
		}
		return have < target
	case "tome":
		if b == nil {
			return false
		}
		for _, u := range inv.Container(d2.ContInventory) {
			if bb := n.l.base(u.Class); bb != nil && bb.Code == b.Code {
				return false
			}
		}
		return true
	case "key":
		return n.l.e.Tune.Town.KeyTarget > 0
	case "runeword":
		return !n.l.NoRunewords
	case "gem":
		// Keep gems of one grade and colour while we hold fewer than GemKeep: three of them make the next grade in the cube.
		// Counts pack, stash and cube as far as the snapshot shows them.
		if b == nil {
			return false
		}
		// D4: chipped and flawed gems are only worth keeping once the Cube can upgrade them; until then they sell.
		if n.l.e.Tune.Town.SellLowGems != 0 && lowGem(b.Code) && !n.haveCube() {
			return false
		}
		have := 0
		for _, c := range []d2.Container{d2.ContInventory, d2.ContStash, d2.ContCube} {
			for _, u := range inv.Container(c) {
				if bb := n.l.base(u.Class); bb != nil && bb.Code == b.Code {
					have++
				}
			}
		}
		return have < GemKeep
	case "charmroom":
		return n.freeCells() >= 3
	case "gold":
		return p.Gold < p.Level*10000
	}
	return true
}

// UseSource replaces the snapshot and stat readers (replay and tests). Nil arguments keep the current ones.
func (l *Looter) UseSource(snap func() (*d2.Snapshot, error), stats func(addr uint32) (d2.Stats, error)) {
	if snap != nil {
		l.snap = snap
		l.hand.Snap = snap
	}
	if stats != nil {
		l.stats = stats
	}
}

// logPotionSkip is the diagnostic line of a potion left on the ground because the pack already holds the buffer
// (primitives-review "what to log" 5). One line per potion code every 30 virtual seconds, so a long walk past the same
// potion does not flood the log.
func (l *Looter) logPotionSkip(b *data.ItemBase, packHave, beltFree int) {
	l.potionSkipMu.Lock()
	defer l.potionSkipMu.Unlock()
	now := l.e.Clock.Now()
	if last, ok := l.potionSkipAt[b.Code]; ok && now.Sub(last) < 30*time.Second {
		return
	}
	if l.potionSkipAt == nil {
		l.potionSkipAt = map[string]time.Time{}
	}
	l.potionSkipAt[b.Code] = now
	l.e.Log.Info(fmt.Sprintf("loot: potion skip code=%s beltWant=0 packHave=%d buffer=%d beltFree=%d", b.Code, packHave, l.e.Tune.Loot.PotionBuffer, beltFree))
}

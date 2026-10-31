package town

import (
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Req is what an item needs before the character may wear it.
type Req struct {
	Level, Str, Dex int
	Unidentified    bool // magic or better and not identified: the affixes (and so the requirement) are hidden
	Estimated       bool // Level is a conservative guess (affixes unknown), not read from the tables
}

// RequirementSource gives the effective requirement of an item unit. Seam so tests and backends can plug in what they
// know; the default is tableReq.
type RequirementSource interface {
	Requirement(u d2.Unit, b *data.ItemBase) Req
}

// SetRequirementSource replaces the default source (nil restores it).
func (t *Town) SetRequirementSource(r RequirementSource) { t.req = r }

// MagicLevelMargin is the level added to a base's levelreq for magic and better items whose affixes cannot be read
// (knob request gear.magicLevelMargin, default 6; U: the typical prefix/suffix levelreq above base is 0..30, so 6 is a
// judgement that clears most early affixes without freezing every drop). Game.exe path: the ItemData prefix/suffix
// offsets are U, so this estimate is what it gets.
const MagicLevelMargin = 6

// affixNamer is the optional reader capability that makes exact affix lookups possible: the item's display name
// ("Sturdy Cap of the Fox"). d2d's `dump items` gives it (name field, V in docs/d2d-fidelity.md for vendor stock); the
// d2denv reader does not expose it yet (request in the EQUIPREQ report).
type affixNamer interface{ ItemName(id uint32) string }

type tableReq struct {
	db     *data.DB
	name   func(id uint32) string
	margin int
}

// Requirement: Str and Dex come from the base only; the level is max(base levelreq, highest affix levelreq).
// Quality below magic (low, normal, superior) is the base row unchanged. Magic items are read from the affix tables
// when the name is known, else estimated. Rare, set and unique items are always estimated (a rare has up to 6 affixes,
// set and unique levelreq live in other tables: U).
func (r tableReq) Requirement(u d2.Unit, b *data.ItemBase) Req {
	if b == nil {
		return Req{}
	}
	req := Req{Level: b.LevelReq, Str: b.ReqStr, Dex: b.ReqDex}
	if u.Item == nil || u.Item.Quality < d2.QualityMagic {
		return req
	}
	if !u.Item.Identified() {
		req.Unidentified = true
		return req
	}
	if u.Item.Quality == d2.QualityMagic && r.name != nil {
		if lr, ok := r.db.MagicNameLevelReq(b, r.name(u.ID)); ok {
			req.Level = max(req.Level, lr)
			return req
		}
	}
	req.Level = b.LevelReq + r.margin
	req.Estimated = true
	return req
}

func (t *Town) requirements() RequirementSource {
	if t.req != nil {
		return t.req
	}
	r := tableReq{db: t.e.Data, margin: MagicLevelMargin}
	if n, ok := t.e.Read.(affixNamer); ok {
		r.name = n.ItemName
	}
	return r
}

// unitReject names the requirement that stops the character from wearing item u (rejUnid, rejLevel, rejStr, rejDex),
// "" when none does, with what the item needs and what the character has.
func (t *Town) unitReject(p *d2.Player, u d2.Unit) (why string, need, have int) {
	b := t.base(u)
	if b == nil || p == nil {
		return "", 0, 0
	}
	r := t.requirements().Requirement(u, b)
	switch {
	case r.Unidentified:
		return rejUnid, 0, 0
	case r.Level > p.Level:
		return rejLevel, r.Level, p.Level
	case r.Str > p.Strength:
		return rejStr, r.Str, p.Strength
	case r.Dex > p.Dexterity:
		return rejDex, r.Dex, p.Dexterity
	}
	return "", 0, 0
}

const rejUnid = "unidentified"

// reasonKey maps a reject class to the short word of the `equip: skip` log line.
func reasonKey(why string) string {
	switch why {
	case rejLevel:
		return "level"
	case rejStr:
		return "str"
	case rejDex:
		return "dex"
	case rejUnid:
		return "unidentified"
	}
	return why
}

// logSkip logs `equip: skip <item> reason=<...> need=<n> have=<n>` once per item, character level and reason, so the
// per-tick found-gear scan does not flood the log.
func (t *Town) logSkip(u d2.Unit, clvl int, reason string, need, have int) {
	key := fmt.Sprintf("%d/%d/%s", u.ID, clvl, reason)
	if t.skipLogged == nil {
		t.skipLogged = map[string]bool{}
	}
	if t.skipLogged[key] {
		return
	}
	t.skipLogged[key] = true
	t.e.Log.Info(fmt.Sprintf("equip: skip %s reason=%s need=%d have=%d", t.nameOf(u), reason, need, have), "id", u.ID)
}

// minEquipGain is the WearScore gain (about defense points) an armor swap must bring: smaller upgrades cost a pack
// round trip and a possible refusal for nothing (EXP, d2-sme answer).
const minEquipGain = 3

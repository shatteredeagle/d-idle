package bot

import "d2lod-bot/internal/d2"

// hpStat feeds the dmgTaken and hpMinPct fields of the minute line (gate v2 proxies, internal/d2dsoakstat).
type hpStat struct {
	prev    int  // life at the previous sample
	seen    bool // prev is valid
	dmg     int  // running counter since the bot started: raw life points lost, sum of negative life deltas
	minPct  int  // lowest life percent seen with a hostile in fight range since the last minute line
	fighted bool // minPct is valid
}

// noteHP is called on every alive sample (and once with the zero-life sample of a death). Life gains from potions
// and regeneration are ignored; only drops add to dmg. Caller holds obMu.
func (o *observer) noteHP(p *d2.Player, hostiles int) {
	if o.hp.seen && p.Life < o.hp.prev {
		o.hp.dmg += o.hp.prev - p.Life
	}
	o.hp.prev, o.hp.seen = p.Life, true
	if hostiles > 0 {
		if pct := p.LifePct(); !o.hp.fighted || pct < o.hp.minPct {
			o.hp.minPct, o.hp.fighted = pct, true
		}
	}
}

// takeHPMin returns the minute's lowest fight life percent (100 when no fight) and starts the next minute.
func (o *observer) takeHPMin() int {
	v := 100
	if o.hp.fighted {
		v = o.hp.minPct
	}
	o.hp.fighted = false
	return v
}

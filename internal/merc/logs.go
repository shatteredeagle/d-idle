package merc

import (
	"fmt"
	"strings"
)

// Log line formats. All start with Prefix so the harness and dashboard can grep `merc: <kind> `; fields are
// key=value pairs, values without spaces (spaces become '_') except the trailing free text after a "(".
const Prefix = "merc: "

// Kinds, the first words after the prefix (line kinds, not item Kind).
const (
	LineHired         = "hired"
	LineSeen          = "seen"
	LineRevived       = "revived"
	LineReviveSkipped = "revive skipped"
	LineEquip         = "equip"
	LineStrip         = "strip"
	LineReqcheck      = "reqcheck"
	LineHireWindow    = "hire window"
	LineGoldFarm      = "gold farm"
)

func tok(s string) string {
	if s == "" {
		return "-"
	}
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "_")
}

// LogHired: `merc: hired class=271 aura=Defiance price=360 gold_before=1500 gold_after=1140`.
func LogHired(class int, aura string, price, goldBefore, goldAfter int) string {
	return fmt.Sprintf("%shired class=%d aura=%s price=%d gold_before=%d gold_after=%d", Prefix, class, tok(aura), price, goldBefore, goldAfter)
}

// LogSeen: `merc: seen class=271 alive=true life=100 level=9 aura=-`.
func LogSeen(s MercState) string {
	return fmt.Sprintf("%sseen class=%d alive=%t life=%d level=%d aura=%s", Prefix, s.Class, s.Alive, s.LifePct, s.Level, tok(s.Aura))
}

// LogRevived: `merc: revived price=600 level=9 estimate=675`. price is the gold actually taken (the read price);
// estimate is the formula pre-check (S, never observed on 1.14d), logged so the next live run compares the two.
func LogRevived(price, level, estimate int) string {
	return fmt.Sprintf("%srevived price=%d level=%d estimate=%d", Prefix, price, level, estimate)
}

// LogHireWindow is the one-shot read-only line per hire window row: `merc: hire window row=0 name=Aliza class=338
// level=9 aura=Defiance price=360 gold=1500 clvl=12`. It answers what the SME has no formula for.
func LogHireWindow(o HireOffer, gold, clvl int) string {
	return fmt.Sprintf("%shire window row=%d name=%s class=%d level=%d aura=%s price=%d gold=%d clvl=%d", Prefix, o.Row, tok(o.Name), o.Class, o.Level, tok(o.Aura), o.Price, gold, clvl)
}

// LogGoldFarm: `merc: gold farm on need=1200 gold=300 clvl=11` (or off), once per change.
func LogGoldFarm(on bool, need, gold, clvl int) string {
	w := "off"
	if on {
		w = "on"
	}
	return fmt.Sprintf("%sgold farm %s need=%d gold=%d clvl=%d", Prefix, w, need, gold, clvl)
}

// LogReviveSkipped: `merc: revive skipped (gold) price=600 gold=900`.
func LogReviveSkipped(reason string, price, gold int) string {
	return fmt.Sprintf("%srevive skipped (%s) price=%d gold=%d", Prefix, reason, price, gold)
}

// LogEquip: `merc: equip slot=helm item=Skull_Cap reason=empty_slot`.
func LogEquip(slot Slot, item, reason string) string {
	return fmt.Sprintf("%sequip slot=%s item=%s reason=%s", Prefix, slot, tok(item), tok(reason))
}

// LogStrip: `merc: strip slot=weapon item=Partizan cells=8`.
func LogStrip(slot Slot, item string, cells int) string {
	return fmt.Sprintf("%sstrip slot=%s item=%s cells=%d", Prefix, slot, tok(item), cells)
}

// LogReqcheck: `merc: reqcheck item=Skull_Cap ok=true reason=ok rule=level_only str=0 dex=0 level=9`; rule is
// "str_dex" or "level_only", the probe question of docs/merc.md.
func LogReqcheck(item string, r ReqResult, strDex bool, m MercState) string {
	rule := "level_only"
	if strDex {
		rule = "str_dex"
	}
	return fmt.Sprintf("%sreqcheck item=%s ok=%t reason=%s rule=%s str=%d dex=%d level=%d", Prefix, tok(item), r.OK, tok(r.Reason), rule, m.Str, m.Dex, m.Level)
}

// KindOf returns the kind of a merc log line ("" when it is none): "revive skipped" for that two-word kind.
func KindOf(line string) string {
	rest, ok := strings.CutPrefix(line, Prefix)
	if !ok {
		return ""
	}
	for _, k := range []string{LineHired, LineSeen, LineRevived, LineReviveSkipped, LineEquip, LineStrip, LineReqcheck, LineHireWindow, LineGoldFarm} {
		if strings.HasPrefix(rest, k+" ") {
			return k
		}
	}
	return ""
}

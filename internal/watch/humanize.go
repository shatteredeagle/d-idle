package watch

import (
	"sort"
	"strings"

	"d2lod-bot/internal/decide"
)

// fieldText is the plain-language phrase for every context field value of the shipped catalogue (V: TestEveryFieldValueHasAPhrase
// walks the catalogue, so a new field or band fails the test until it gets its line here). Data, not class code.
var fieldText = map[string]map[string]string{
	"act":        {"1": "in act 1", "2": "in act 2", "3": "in act 3", "4": "in act 4", "5": "in act 5"},
	"diff":       {"normal": "on Normal", "nightmare": "on Nightmare", "hell": "on Hell"},
	"hc":         {"normal": "softcore hero", "hardcore": "hardcore hero"},
	"kit":        {"OK": "with potions and a portal", "THIN": "with only one of potions or portal", "NONE": "with neither potions nor a portal"},
	"urgency":    {"calm": "danger calm", "wary": "danger rising", "urgent": "danger urgent", "critical": "danger critical"},
	"areaclass":  {"town": "in town", "field": "in the field", "boss": "at a boss"},
	"itemkind":   {"rune": "item is a rune", "gem": "item is a gem", "jewel": "item is a jewel", "rare": "item is a rare", "charm": "item is a charm", "unique": "item is a unique"},
	"itemclass":  {"magic": "item is magic", "higher": "item is rare or better"},
	"scroll":     {"yes": "with an identify scroll", "no": "no identify scroll"},
	"belt":       {"full": "belt full", "half": "belt half empty", "low": "belt nearly empty"},
	"packdanger": {"1": "pack danger low", "2": "pack danger medium", "3": "pack danger high"},
	"packtype":   {"ordinary": "ordinary pack", "fast": "fast pack", "ranged": "ranged pack", "curse": "cursing pack", "immune": "immune pack"},
	"runkind":    {"farm": "farming run", "quest": "quest run", "level": "levelling run"},
	"boss":       {"minor": "minor boss", "major": "major boss"},
	"gear":       {"low": "gear weak", "mid": "gear fair", "high": "gear strong"},
	"merc":       {"alive": "mercenary alive", "dead": "mercenary dead", "none": "no mercenary"},
	"slot":       {"armor": "slot is armor", "weapon": "slot is weapon", "jewelry": "slot is jewelry"},
	"potiontier": {"minor": "minor potions", "light": "light potions", "normal": "normal potions", "greater": "greater potions"},
	"held":       {"none": "no potion in the belt", "hp": "a healing potion at hand", "rv": "a rejuvenation potion at hand"},
	"life":       {"lt25": "life under 25%", "25-35": "life 25 to 35%", "35-55": "life 35 to 55%", "55-70": "life 55 to 70%", "gt70": "life over 70%"},
	"gold":       {"lt100": "gold under 100", "100-500": "gold 100 to 500", "500-1500": "gold 500 to 1500", "gt1500": "gold over 1500"},
	"levelgap":   {"le0": "area is not above the hero", "1-3": "area 1 to 3 levels above", "4-6": "area 4 to 6 levels above", "ge7": "area 7+ levels above"},
	"freerows":   {"0": "no free pack row", "1": "1 free pack row", "2+": "2 or more free pack rows"},
	"stash":      {"lt50": "stash under half full", "50-85": "stash 50 to 85% full", "85-99": "stash nearly full", "full": "stash full"},
	"deaths":     {"1": "1 death", "2-3": "2 or 3 deaths", "4+": "4 or more deaths"},
	"clvl":       {"lt14": "level under 14", "14": "level 14", "ge15": "level 15 or more"},
	"retreats":   {"0-1": "0 or 1 retreats", "2": "2 retreats", "3+": "3 or more retreats"},
	"tripgold":   {"lt60": "trip gold under 60", "60-300": "trip gold 60 to 300", "ge300": "trip gold 300 or more"},
	"packused":   {"lt80": "pack under 80% full", "ge80": "pack 80% full or more"},
	"timeratio":  {"fast": "run faster than usual", "normal": "run on pace", "slow": "run slower than usual"},
	"durability": {"lt20": "gear nearly broken", "20-30": "gear worn", "gt30": "gear in good shape"},
	"mana":       {"low": "mana low", "mid": "mana medium", "high": "mana high"},
}

// FieldView is one context field in plain language.
type FieldView struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Text  string `json:"text"`
}

// Fields splits a context key into plain-language fields, sorted by name. An unknown field or value falls back to
// "name value" so nothing is hidden; the test keeps the shipped catalogue fully covered.
func Fields(key string) []FieldView {
	ctx, err := decide.ParseKey(key)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(ctx.Fields))
	for n := range ctx.Fields {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]FieldView, 0, len(names))
	for _, n := range names {
		v := ctx.Fields[n]
		t, ok := fieldText[n][v]
		if !ok {
			t = n + " " + v
		}
		out = append(out, FieldView{Name: n, Value: v, Text: t})
	}
	return out
}

// PlainContext is the one-line plain-language context of a key ("in act 1, on Normal, life 25 to 35%, ...").
func PlainContext(key string) string {
	fs := Fields(key)
	parts := make([]string, len(fs))
	for i, f := range fs {
		parts[i] = f.Text
	}
	return strings.Join(parts, ", ")
}

// UrgencyText names how urgent a situation is for the feed: the urgency band when the key has one, else the life band
// (the stage-1 stand-in, ADR 0016 amendment), else "". It is the "how bad is it" line under a fight-time question.
func UrgencyText(key string) string {
	var life string
	for _, f := range Fields(key) {
		switch f.Name {
		case "urgency":
			return f.Text
		case "life":
			life = f.Text
		}
	}
	return life
}

package shadowlog

import (
	"fmt"
	"strings"
)

// Admission: internal/bot/admission.go admitNote writes `admission shadow` with differ (new-only, old-only, words),
// area, purpose, old, new, row, need. Only disagreements are logged, once per case; the agree count is not in the log
// (it is in the ops status --why line).
func addAdmission(a *acc, _ string, rec map[string]any, n int) bool {
	cell, ok1 := str(rec, "differ")
	purpose, ok2 := str(rec, "purpose")
	old, ok3 := str(rec, "old")
	nw, ok4 := str(rec, "new")
	area, ok5 := str(rec, "area")
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		return false
	}
	row, _ := str(rec, "row")
	a.total("cell="+cell, n)
	a.total("purpose="+purpose, n)
	a.row(fmt.Sprintf("%s purpose=%s row=%s old=%q new=%q", cell, purpose, row, old, nw), "area "+area, n)
	return true
}

func admissionVerdict(a *acc) (string, string) {
	if a.lines == 0 {
		return NoLines, "no disagreement logged; this is agreement only if the shadow ran (the agree count is in ops status --why)"
	}
	return Review, tally(a, "cell=", "new-only", "old-only", "words") +
		" (old-only: the function admits where the old judge refused; new-only: it refuses where the old judge let through)"
}

// Planner: internal/agency/planner.go ShadowLine. Every line carries agree, so this family can be clean.
var plannerKeys = []string{"clvl", "class", "plan", "profile", "agree", "best", "aura", "auraProfile", "arch", "unplayed", "warnings", "via"}

func addPlanner(a *acc, msg string, _ map[string]any, n int) bool {
	f, ok := fields(strings.TrimPrefix(msg, "planner shadow: "), plannerKeys...)
	if !ok || f["class"] == "" || (f["agree"] != "true" && f["agree"] != "false") {
		return false
	}
	a.total("class="+f["class"]+" lines", n)
	if f["warnings"] != "-" {
		for _, w := range strings.Split(f["warnings"], ",") {
			a.total("warning="+w, n)
		}
	}
	if f["agree"] == "true" {
		a.total("class="+f["class"]+" agree", n)
		return true
	}
	// The plan's attack is skill/level: the level moves every few minutes, the skill is what a reviewer compares.
	plan, _, _ := strings.Cut(f["plan"], "/")
	a.row(fmt.Sprintf("class=%s plan=%s profile=%s aura=%s auraProfile=%s", f["class"], plan, f["profile"], f["aura"], f["auraProfile"]),
		"clvl "+f["clvl"]+" via "+f["via"], n)
	return true
}

func plannerVerdict(a *acc) (string, string) {
	switch {
	case a.lines == 0:
		return NoLines, "no planner line: the shadow did not run (agency.planner 0) or the log holds no planning pass"
	case a.differ == 0:
		return Clean, fmt.Sprintf("the plan agreed with the combat profile on all %d line(s)", a.lines)
	}
	return Review, fmt.Sprintf("the plan differed from the combat profile on %d of %d line(s)", a.differ, a.lines)
}

// Upkeep: internal/combat/upkeep.go upkeepShadow writes `combat upkeep shadow` with skill, what (due, mark, aura,
// aura recheck), old, driver. The bot logs a lasting disagreement once, so a repeat of the same case is a new episode.
func addUpkeep(a *acc, _ string, rec map[string]any, n int) bool {
	skill, ok1 := str(rec, "skill")
	what, ok2 := str(rec, "what")
	old, ok3 := str(rec, "old")
	drv, ok4 := str(rec, "driver")
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return false
	}
	a.total("what="+what, n)
	a.total("skill="+skill, n)
	a.row(fmt.Sprintf("skill=%s what=%s old=%s driver=%s", skill, what, old, drv), "", n)
	return true
}

func upkeepVerdict(a *acc) (string, string) {
	if a.lines == 0 {
		return NoLines, "no disagreement logged; this is agreement only if mode 1 ran (mode 2 writes no shadow line)"
	}
	once := 0
	for _, r := range a.rows {
		if r.Count == 1 {
			once++
		}
	}
	return Review, fmt.Sprintf("%d case(s), %d seen once and %d seen again; a case seen once per skill may be the driver learning the state id, a case seen again is a standing disagreement",
		len(a.rows), once, len(a.rows)-once)
}

// Loot: internal/loot/lootshadow.go writes `loot shadow: <scope> <item> today=<action> rules=<action> rule=<id>
// reason=<text>`, deduplicated per scope, area, item code and quality and capped per minute, so the counts here are a
// floor (the full counters are at GET /api/lootrules/shadow).
func addLoot(a *acc, msg string, _ map[string]any, n int) bool {
	rest := strings.TrimPrefix(msg, "loot shadow: ")
	scope, rest, ok := strings.Cut(rest, " ")
	if !ok {
		return false
	}
	i := strings.Index(rest, " today=")
	if i < 0 {
		return false
	}
	f, ok := fields(rest[i+1:], "today", "rules", "rule", "reason")
	if !ok || f["today"] == "" || f["rules"] == "" {
		return false
	}
	a.total("scope="+scope, n)
	if scope == lootGround && f["today"] == lootPickup && f["rules"] != lootPickup {
		a.total(lootLost, n)
	}
	a.row(fmt.Sprintf("%s today=%s rules=%s rule=%s", scope, f["today"], f["rules"], f["rule"]), rest[:i], n)
	return true
}

// The words of internal/loot (ShadowGround) and internal/lootrules (Pickup) as they stand in the line.
const (
	lootGround = "ground"
	lootPickup = "pickup"
	lootLost   = "ground pickups lost"
)

func lootVerdict(a *acc) (string, string) {
	if a.lines == 0 {
		return NoLines, "no disagreement logged; this is agreement only if the shadow ran (bot.lootShadow 1)"
	}
	if r := a.totals[lootLost]; r != nil {
		return Worse, fmt.Sprintf("%d ground line(s) where today's code picks the item up and the rules leave it: stage 1 would lose these pickups", r.Count)
	}
	return Review, fmt.Sprintf("%d disagreement line(s), none of them a lost ground pickup", a.differ)
}

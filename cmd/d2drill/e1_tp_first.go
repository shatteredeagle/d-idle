package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Group E1 shortdrill-tpfirst (task g2gfixscroll, Gate 2 g2g seed 17 S1): a hero in town with a full belt, gold for
// both the missing scroll and a gear pass, and fewer scrolls than town.tpReserve restocks the scroll BEFORE the gear
// shop spends the gold.
//
//	E1.tp-first-a  clvl 20, 1 scroll, 700 gold, full belt: `gear: scroll before gear` and a scroll bought before the
//	               first `leveler: buying`
//	E1.tp-first-b  control: 2 scrolls (the reserve is held): no `scroll before gear` line, the gear pass is unchanged
//
// A full belt keeps the town trip quiet (one scroll is not a trip reason: only none is), so the gear step is the
// first thing the bot does in town, as in the seed 17 minute 1035.

func init() { e1Extra = append(e1Extra, e1TPFirstScenarios()...) }

func e1TPFirstScenarios() []e1Scenario {
	mk := func(id, title string, scrolls int, eval func([]string) (string, string)) e1Scenario {
		return e1Scenario{ID: id, Group: "shortdrill-tpfirst", Title: title, Goals: "farm:Blood Moor",
			Setup: e1GearSetup(scrolls, 700, nil), Done: e1TPFirstDone,
			Eval: func(l []string, _ map[string]string) (string, string) { return eval(l) }}
	}
	return []e1Scenario{
		mk("E1.tp-first-a", "clvl 20, 1 scroll, 700 gold, full belt: the scroll is bought before the gear shop spends", 1, e1TPFirstEvalA),
		mk("E1.tp-first-b", "control: 2 scrolls, same purse: no scroll restock before the gear pass", 2, e1TPFirstEvalB),
	}
}

// e1GearSetup: town, clvl 20, a full belt (healing in all 4 boxes: the town chores trigger counts belt healing against 2 x bot.resupplyPotions), n separate scrolls, gold,
// and extra lab verbs (extra may be nil).
func e1GearSetup(scrolls, gold int, extra func(context.Context, *e1h) error) func(context.Context, *e1h) (map[string]string, error) {
	return func(ctx context.Context, h *e1h) (map[string]string, error) {
		f := map[string]string{"scrolls": strconv.Itoa(scrolls), "gold": strconv.Itoa(gold)}
		if err := e1Reset(ctx, h, 20); err != nil {
			return f, err
		}
		for box := 0; box < 4; box++ { // the starting belt has 4 boxes (d2d: "no belt box" above 3, V 2026-11-07)
			if _, err := h.cheat(ctx, fmt.Sprintf("debug give hp1 belt %d", box), "E1 setup: a full belt, so no potion trip competes with the gear pass"); err != nil {
				return f, err
			}
		}
		for _, code := range []string{"hp1", "hp1", "mp1", "mp1", "mp1"} {
			if _, err := h.cheat(ctx, "debug give "+code+" pack", "E1 setup: spare potions in the pack, so no potion trip competes with the gear pass"); err != nil {
				return f, err
			}
		}
		for i := 0; i < scrolls; i++ {
			if _, err := h.cheat(ctx, "debug give tsc pack", "E1 setup: a town portal scroll in the pack"); err != nil {
				return f, err
			}
		}
		if extra != nil {
			if err := extra(ctx, h); err != nil {
				return f, err
			}
		}
		if _, err := h.cheat(ctx, fmt.Sprintf("debug stat 14 %d", gold), fmt.Sprintf("E1 setup: purse %d", gold)); err != nil {
			return f, err
		}
		_ = h.step(ctx, 25)
		st, _, err := h.c.State(ctx)
		if err != nil {
			return f, err
		}
		f["state"] = fmt.Sprintf("area=%d level=%d gold=%d", h.area(ctx), st.Level, st.Gold)
		return f, nil
	}
}

func e1TPFirstDone(l []string) bool {
	return e1Find(l, `"gear shop"`) >= 0 || e1Find(l, "Chores in town") >= 0 || e1Find(l, `"to":"traveling"`) >= 0
}

// e1Scroll is the first portal scroll or tome purchase line of the town code (a purse from 450 gold buys the tome first, as the town code decides).
func e1Scroll(l []string) int {
	for i, s := range l {
		if strings.Contains(s, `"shop: bought"`) && (strings.Contains(s, "Scroll of Town Portal") || strings.Contains(s, "Tome of Town Portal")) {
			return i
		}
	}
	return -1
}

func e1TPFirstEvalA(l []string) (string, string) {
	mark, buy, gear := e1Find(l, "gear: scroll before gear"), e1Scroll(l), e1Find(l, "leveler: buying")
	switch {
	case mark < 0:
		return "FAIL", "no `gear: scroll before gear` line (the restock did not run before the gear step)"
	case buy < 0:
		return "FAIL", "the restock ran but no portal scroll or tome was bought"
	case gear >= 0 && gear < buy:
		return "FAIL", fmt.Sprintf("the gear shop bought (line %d) before the scroll (line %d)", gear, buy)
	}
	return "PASS", fmt.Sprintf("scroll bought (log line %d) before the first gear purchase (%s)", buy, e1LineOr(l, gear, "none within the window"))
}

func e1TPFirstEvalB(l []string) (string, string) {
	if i := e1Find(l, "gear: scroll before gear"); i >= 0 {
		return "FAIL", "the reserve was held but the scroll restock ran: " + strings.TrimSpace(l[i])
	}
	if e1Find(l, `"gear shop"`) < 0 {
		return "INCONCLUSIVE", "no gear pass seen in the window"
	}
	return "PASS", "the gear pass ran without a scroll restock"
}

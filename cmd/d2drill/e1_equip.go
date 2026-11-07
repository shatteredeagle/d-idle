package main

import (
	"context"
	"fmt"
	"strings"
)

// Group E1 shortdrill-equip (task g2gfixscroll, Gate 2 g2g seed 13 E1): a failed equip must neither leave the game nor
// abandon the area. Provoking a refused swap needs the game to refuse a wearable-looking item: the setup gives a magic
// Cap while another Cap is worn, the swap that failed live ("Cap 12.0 -> head (was Cap 4.0)"). If the d2d accepts the
// swap there is no failure to judge and the verdict is INCONCLUSIVE, never PASS.
//
//	E1.equip-fail  clvl 20, a worn Cap and a magic Cap in the pack, scrolls 2, 700 gold, full belt: after an
//	               `equip failed` line there is no `leaving the game`, `Save and Exit` or fresh-game line before the
//	               bot walks on, and each failed swap is logged at most twice

func init() { e1Extra = append(e1Extra, e1EquipScenarios()...) }

func e1EquipScenarios() []e1Scenario {
	caps := func(ctx context.Context, h *e1h) error {
		for _, v := range []string{"debug give cap pack q3", "debug give cap pack q4"} {
			if _, err := h.cheat(ctx, v, "E1 setup: Caps in the pack so that the equip pass swaps over a worn Cap"); err != nil {
				return err
			}
		}
		return nil
	}
	return []e1Scenario{{ID: "E1.equip-fail", Group: "shortdrill-equip", Goals: "farm:Blood Moor",
		Title: "clvl 20, Caps in the pack: a failed equip does not leave the game",
		Setup: e1GearSetup(2, 700, caps),
		Done:  func(l []string) bool { return e1Find(l, `"gear equip"`) >= 0 && len(e1After(l, "gear equip")) >= 25 },
		Eval:  func(l []string, _ map[string]string) (string, string) { return e1EquipEval(l) }}}
}

// e1After is the lines after the first line containing sub.
func e1After(l []string, sub string) []string {
	if i := e1Find(l, sub); i >= 0 {
		return l[i+1:]
	}
	return nil
}

func e1EquipEval(l []string) (string, string) {
	f := e1Find(l, "leveler: equip failed")
	if f < 0 {
		if e1Find(l, `"gear equip"`) < 0 {
			return "INCONCLUSIVE", "no gear pass seen in the window"
		}
		return "INCONCLUSIVE", "no equip failed: the d2d accepted every swap, so there is no failure to judge"
	}
	// The gear step ends with its own `gear equip` line; the leave of a save point or a fresh game would follow it. The
	// save point is out of scope here (it needs minutes in the game), a fresh game or a leave is the defect.
	for _, bad := range []string{"starting a fresh game", "leaving the game", "leave game"} {
		if i := e1Find(l[f:], bad); i >= 0 {
			return "FAIL", "after the failed equip: " + strings.TrimSpace(l[f+i])
		}
	}
	perSwap := map[string]int{}
	for _, s := range l {
		if strings.Contains(s, "leveler: equip failed") {
			perSwap[e1Swap(s)]++
		}
	}
	for sw, n := range perSwap {
		if n > 2 {
			return "FAIL", fmt.Sprintf("swap %q failed %d times: the retry is not bounded", sw, n)
		}
	}
	return "PASS", fmt.Sprintf("%d failed swap(s), no leave or fresh game after them (%s)", len(perSwap), e1LineOr(l, f, ""))
}

// e1Swap pulls the swap text out of an `equip failed` line.
func e1Swap(line string) string {
	const k = `"swap":"`
	if i := strings.Index(line, k); i >= 0 {
		r := line[i+len(k):]
		if j := strings.Index(r, `"`); j >= 0 {
			return r[:j]
		}
	}
	return line
}

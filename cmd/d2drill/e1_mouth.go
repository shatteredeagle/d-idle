package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// E1.mouth-pocket (card g2gfixmouth, TASK-67): the Black Marsh pocket of Gate 2 g2g seed 17 (hero at (87,287), area 6,
// V in the g2g logs: the walk to town failed there and only restarts freed the hero). Setup writes happen before the
// start: `debug goto 6`, then `debug warp 17 57` (tile units; the pocket drills of DRILLS25 map `warp 33 44` to the
// subtile (165,220), so 17 57 is (85..89, 285..289), S). Pass: the bot leaves the 20 subtile neighbourhood without a
// "Save and Exit" / failsafe restart line. INCONCLUSIVE when the mouth gate never spoke (the map of this slot's seed
// has no unwanted exit near the spot, the pocket is seed dependent, U). Run on the seed of the g2g seed 17 if the slot
// takes one; the walkable pocket itself is U.
const (
	mouthPocketArea = 6
	mouthPocketX    = 87
	mouthPocketY    = 287
)

var e1StatusRe = regexp.MustCompile(`"msg":"status".*"area":(\d+),"x":(\d+),"y":(\d+)`)

func e1MouthScenario() e1Scenario {
	return e1Scenario{ID: "E1.mouth-pocket", Group: "shortdrill-mouth", Goals: "farm:Blood Moor",
		Title: "Black Marsh (87,287) after debug warp: the bot leaves the spot without a game restart",
		Setup: func(ctx context.Context, h *e1h) (map[string]string, error) {
			f := map[string]string{}
			if err := e1Reset(ctx, h, 15); err != nil {
				return f, err
			}
			for _, v := range []struct{ verb, why string }{
				{fmt.Sprintf("debug goto %d", mouthPocketArea), "E1 setup: Black Marsh (g2g seed 17 pocket)"},
				{"debug warp 17 57", "E1 setup: the pocket cell (87,287) in tile units"},
			} {
				if _, err := h.cheat(ctx, v.verb, v.why); err != nil {
					return f, err
				}
			}
			_ = h.step(ctx, 50)
			f["state"] = fmt.Sprintf("area=%d", h.area(ctx))
			if a := h.area(ctx); a != mouthPocketArea {
				return f, fmt.Errorf("not in the Black Marsh: %s", f["state"])
			}
			return f, nil
		},
		Done: func(l []string) bool {
			return e1Find(l, "Save and Exit") >= 0 || e1MouthLeft(l) >= 0
		},
		Eval: e1MouthEval,
	}
}

// e1MouthLeft is the index of the first status line more than 20 subtiles from the pocket (same area), or -1.
func e1MouthLeft(lines []string) int {
	for i, l := range lines {
		m := e1StatusRe.FindStringSubmatch(l)
		if m == nil || m[1] != fmt.Sprint(mouthPocketArea) {
			continue
		}
		x, y := e1Atoi(m[2]), e1Atoi(m[3])
		if dx, dy := x-mouthPocketX, y-mouthPocketY; dx*dx+dy*dy > 400 {
			return i
		}
	}
	return -1
}

func e1Atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func e1MouthEval(lines []string, _ map[string]string) (string, string) {
	gate := 0
	for _, l := range lines {
		if strings.Contains(l, "nav mouth gate") {
			gate++
		}
	}
	left, restart := e1MouthLeft(lines), e1Find(lines, "Save and Exit")
	switch {
	case restart >= 0 && (left < 0 || restart < left):
		return "FAIL", "restart before the hero left the pocket: " + strings.TrimSpace(lines[restart])
	case left >= 0 && gate > 0:
		return "PASS", fmt.Sprintf("left the pocket (%s) with %d mouth gate lines and no restart", strings.TrimSpace(lines[left]), gate)
	case left >= 0:
		return "INCONCLUSIVE", "the hero walked off but the mouth gate never spoke: the spot is no pocket on this seed"
	}
	return "FAIL", fmt.Sprintf("the hero never left (87,287) within the run, %d mouth gate lines", gate)
}

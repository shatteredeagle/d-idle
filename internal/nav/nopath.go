package nav

import (
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// The failed-plan memo. A target the known map cannot reach was searched again on every fight approach and on each of the
// level's teleport pads, the whole 900x900 Arcane Sanctuary grid each time: half a second of wall time per approach and almost no
// game time. The boss's attack slice is wall time, so each Kill ended by its slice ("not an error"), the fail counter never
// grew and the runner watchdog ended the run after 614 s wall and 36 s of game time (V lab night 2026-11-28,
// drill.act2.summoner.stepexec: 251 x "{26 450} is unreachable from {86 516}" and 409 x "nav teleport pad: none usable" in
// those ten minutes). The same question on the same field, hard blocks and blacklist has the same answer: it is asked once.

// noPathTTL bounds a memo entry in navigator time, for whatever the key misses.
const noPathTTL = 2 * time.Second

type noPathKey struct {
	field       *Field
	start, goal input.Point
	tol         float64
	ignore      uint32
	hard, black int
	fullGrid    bool
}

type noPathMemo struct {
	res planResult
	err error
	at  time.Time
}

func (n *Navigator) noPathKeyFor(lm *d2map.LevelMap, start, goal input.Point, tol float64, ignore uint32) noPathKey {
	n.planMu.Lock()
	defer n.planMu.Unlock()
	return noPathKey{field: n.fieldFor(lm), start: start, goal: goal, tol: tol, ignore: ignore, hard: len(n.hard), black: len(n.black), fullGrid: n.tune().FullGrid}
}

func (n *Navigator) noPathHit(k noPathKey) (noPathMemo, bool) {
	n.planMu.Lock()
	defer n.planMu.Unlock()
	m, ok := n.noPath[k]
	if !ok || n.clk == nil || n.clk.Now().Sub(m.at) > noPathTTL {
		return noPathMemo{}, false
	}
	m.res.wps = append([]input.Point(nil), m.res.wps...) // the caller may trim its copy
	return m, true
}

func (n *Navigator) noPathPut(k noPathKey, res planResult, err error) {
	n.planMu.Lock()
	defer n.planMu.Unlock()
	if n.clk == nil {
		return
	}
	if len(n.noPath) >= 64 {
		n.noPath = nil
	}
	for old := range n.noPath {
		if old.field != k.field { // a new field (cells learned, a door opened): the old answers are void
			n.noPath = nil
		}
		break
	}
	if n.noPath == nil {
		n.noPath = map[noPathKey]noPathMemo{}
	}
	n.noPath[k] = noPathMemo{res: res, err: err, at: n.clk.Now()}
}

package resource

import (
	"fmt"
	"math"

	"d2lod-bot/internal/d2"
)

// SwingData is the plain-attack animation of a class with its starting weapon: Frames is the AnimData frame count
// of the A1 record, Start the game.exe start frame (docs/research/swing1-trace.md Q1: 2 with a melee weapon for the
// Amazon and the Sorceress, 0 for every other class).
type SwingData struct{ Frames, Start int }

// swingData holds the classes the stock scenarios use. Sorceress staff (PAA1STF) 18 frames, Paladin sword (PAA11HS)
// 15 frames: V run in swing1-trace.md (16 and 15 ticks at rate 256). Add a class here to give it a scenario.
var swingData = map[d2.Class]SwingData{
	d2.Sorceress: {Frames: 18, Start: 2},
	d2.Paladin:   {Frames: 15, Start: 0},
}

// tickSec is one game tick (25 fps).
const tickSec = 0.04

// SwingSeconds is the game.exe swing length of a class at rate 256 (no IAS, WSM 0): ceil(256 (F - s) / 256) ticks,
// the closed form of swing1-trace.md Q1. An unknown class is an error, never the Sorceress.
func SwingSeconds(c d2.Class) (float64, error) {
	d, ok := swingData[c]
	if !ok {
		return 0, fmt.Errorf("resource: no swing data for class %v", c)
	}
	return float64(swingTicks(d.Frames, d.Start, 256)) * tickSec, nil
}

func swingTicks(frames, start, rate int) int {
	return int(math.Max(1, math.Ceil(256*float64(frames-start)/float64(rate))))
}

func mustSwing(c d2.Class) float64 {
	s, err := SwingSeconds(c)
	if err != nil {
		panic(err)
	}
	return s
}

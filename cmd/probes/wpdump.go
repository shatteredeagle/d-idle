package main

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
)

// wpdump (docs/act3.md X4): which acts 3-5 waypoints the game holds as activated (d2.ReadProgress masks, V
// docs/questmem.md for the struct; the bit -> level table is S, saves.waypointTable order), whether the panel flag
// (UI 0x14) is up, and the waypoint objects in the unit table with their positions. The object positions are what
// nav.knownWaypointPos for levels 75/103/109 needs (per-seed, so record the seed with it). The panel's own text and
// the click geometry are not in memory: for the act 3-5 tab x and the icon rows take tools/shot.sh while the panel is
// open, and click with the existing navlive tooling.

type wpBit struct {
	Bit   int
	Level int
	Name  string
	Act   int
}

// wpBits are bits 18..38 of the waypoint mask (S, copied from internal/saves waypointTable, acts 3, 4, 5).
var wpBits = []wpBit{
	{18, 75, "Kurast Docks", 3}, {19, 76, "Spider Forest", 3}, {20, 77, "Great Marsh", 3}, {21, 78, "Flayer Jungle", 3},
	{22, 79, "Lower Kurast", 3}, {23, 80, "Kurast Bazaar", 3}, {24, 81, "Upper Kurast", 3}, {25, 83, "Travincal", 3},
	{26, 101, "Durance of Hate Level 2", 3},
	{27, 103, "The Pandemonium Fortress", 4}, {28, 106, "City of the Damned", 4}, {29, 107, "River of Flame", 4},
	{30, 109, "Harrogath", 5}, {31, 111, "Frigid Highlands", 5}, {32, 112, "Arreat Plateau", 5},
	{33, 113, "Crystalline Passage", 5}, {34, 123, "Halls of Pain", 5}, {35, 115, "Glacial Trail", 5},
	{36, 117, "Frozen Tundra", 5}, {37, 118, "The Ancients' Way", 5}, {38, 129, "Worldstone Keep Level 2", 5},
}

func parseWpdump(args []string) (func(context.Context, *Env) error, error) {
	fs, sec := newFlags("wpdump")
	if err := parseFlags(fs, args, sec); err != nil {
		return nil, err
	}
	n := *sec
	return func(ctx context.Context, e *Env) error { return runWpdump(e, n) }, nil
}

// wpActive decodes one difficulty's mask into the acts 3-5 rows.
func wpActive(mask uint64) []map[string]any {
	var out []map[string]any
	for _, b := range wpBits {
		out = append(out, map[string]any{"bit": b.Bit, "level": b.Level, "name": b.Name, "act": b.Act, "active": mask>>uint(b.Bit)&1 == 1})
	}
	return out
}

func runWpdump(e *Env, sec int) error {
	const p = "wpdump"
	diffNames := [3]string{"normal", "nightmare", "hell"}
	panelWas := false
	err := e.poll(time.Duration(sec)*time.Second, time.Second, func(tick int) error {
		s, err := e.Snap()
		if err != nil {
			return err
		}
		if s.Player == nil {
			e.emit(p, "skip", map[string]any{"reason": "no player"})
			return nil
		}
		if tick == 0 && e.Progress != nil {
			pr, perr := e.Progress(s.Player.Addr)
			if perr != nil {
				e.emit(p, "progress", map[string]any{"ok": false, "error": perr.Error()})
			} else {
				for d := 0; d < 3; d++ {
					e.emit(p, "progress", map[string]any{"ok": true, "difficulty": diffNames[d], "mask": fmt.Sprintf("0x%010x", pr.Waypoints[d]),
						"acts3to5": wpActive(pr.Waypoints[d])})
				}
			}
		}
		panel := s.UI.Open(d2.UIWaypoint)
		var wps []map[string]any
		for _, o := range s.Objects {
			if d2map.IsWaypointObject(o.Class) {
				wps = append(wps, map[string]any{"id": o.ID, "class": o.Class, "x": o.X, "y": o.Y, "mode": o.Mode})
			}
		}
		if tick == 0 || panel != panelWas || len(wps) > 0 && tick%5 == 0 {
			e.emit(p, "state", map[string]any{"area": s.Player.Area, "act": s.Player.Act + 1, "x": s.Player.X, "y": s.Player.Y,
				"panel_open": panel, "waypoint_objects": wps, "viewport": []int{s.Viewport.X, s.Viewport.Y}})
		}
		panelWas = panel
		return nil
	})
	if err != nil {
		return e.fail(p, err)
	}
	return nil
}

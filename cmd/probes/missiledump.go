package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"d2lod-bot/internal/d2"
)

// missiledump (docs/difficulty.md "Wiring (MISSILES)"): the client missile table (U live) and the player's curse
// states (S ids, V once resolved by name from states.txt). The summary lists every missile name seen and the kind
// MissileKindByName gave it, so the U substrings firewall/hydra/chaos/lightning can be fixed from real names, and
// the state ids that were on while a curse was cast (cast Amplify Damage etc. from a second character, or take a
// Mephisto-class curse, then read states_on).

func parseMissiledump(args []string) (func(context.Context, *Env) error, error) {
	fs, sec := newFlags("missiledump")
	ms := fs.Int("ms", 250, "sample interval in ms (50..5000)")
	if err := parseFlags(fs, args, sec); err != nil {
		return nil, err
	}
	if *ms < 50 || *ms > 5000 {
		return nil, fmt.Errorf("missiledump: -ms 50..5000")
	}
	n, step := *sec, *ms
	return func(ctx context.Context, e *Env) error { return runMissiledump(e, n, step) }, nil
}

func runMissiledump(e *Env, sec, ms int) error {
	const p = "missiledump"
	names := map[string]map[string]any{} // missile name -> {class, kind, count}
	statesSeen := map[int]int{}
	readFails, samples := 0, 0
	err := e.poll(time.Duration(sec)*time.Second, time.Duration(ms)*time.Millisecond, func(int) error {
		s, err := e.Snap()
		if err != nil {
			return err
		}
		samples++
		if s.Player != nil {
			c := s.Player.Curses(e.CurseIDs)
			var on []map[string]any
			for id := 0; id < 192; id++ {
				if s.Player.States.Has(id) {
					statesSeen[id]++
					on = append(on, map[string]any{"id": id, "name": e.stateName(id)})
				}
			}
			e.emit(p, "player", map[string]any{
				"curses": map[string]bool{"amplify": c.Amplify, "decrepify": c.Decrepify, "lower_resist": c.LowerResist,
					"weaken": c.Weaken, "iron_maiden": c.IronMaiden},
				"curse_ids": map[string]int{"amplify": e.CurseIDs.Amplify, "decrepify": e.CurseIDs.Decrepify,
					"lower_resist": e.CurseIDs.LowerResist, "weaken": e.CurseIDs.Weaken, "iron_maiden": e.CurseIDs.IronMaiden},
				"states_on": on, "x": s.Player.X, "y": s.Player.Y,
			})
		}
		units, ok := e.Missiles()
		if !ok {
			readFails++
			e.emit(p, "missiles", map[string]any{"ok": false})
			return nil
		}
		var list []map[string]any
		for _, m := range units {
			name := ""
			if e.MissileName != nil {
				name = e.MissileName(m.Class)
			}
			kind := d2.MissileKindByName(name)
			list = append(list, map[string]any{"id": m.ID, "class": m.Class, "name": name, "kind": kind.String(),
				"x": m.X, "y": m.Y, "mode": m.Mode, "owner_type": m.OwnerType, "owner_id": m.OwnerID})
			if name == "" {
				name = fmt.Sprintf("class%d", m.Class)
			}
			if names[name] == nil {
				names[name] = map[string]any{"class": m.Class, "kind": kind.String(), "count": 0}
			}
			names[name]["count"] = names[name]["count"].(int) + 1
		}
		if len(list) > 0 {
			e.emit(p, "missiles", map[string]any{"ok": true, "n": len(list), "list": list})
		}
		return nil
	})
	if err != nil {
		return e.fail(p, err)
	}
	var other []string
	for n, v := range names {
		if v["kind"] == "other" {
			other = append(other, n)
		}
	}
	sort.Strings(other)
	e.emit(p, "summary", map[string]any{"samples": samples, "table_read_failures": readFails, "missile_names": names,
		"classified_other": other, "states_seen": statesSeen,
		"verdict": missileVerdict(samples, readFails, len(names))})
	return nil
}

func (e *Env) stateName(id int) string {
	if e.StateName == nil {
		return ""
	}
	return e.StateName(id)
}

func missileVerdict(samples, fails, names int) string {
	switch {
	case samples > 0 && fails == samples:
		return "client missile table unreadable on every sample: rvaClientUnits is wrong (U stays U)"
	case names == 0:
		return "table readable but no missile seen: fight something that shoots (or cast a skill) and run again"
	}
	return "missile names listed: fix MissileKindByName substrings from classified_other and the names"
}

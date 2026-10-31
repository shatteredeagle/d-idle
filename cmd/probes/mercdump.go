package main

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// mercdump: d2.FindMerc once a second and, while the NPC dialog flag (UI 0x08) is up, what the NPC menu looks like.
// No memory path to the dialog TEXT is known (U), so the "lines" are the expected labels from town.MenuOrder (V only
// for Gheed 147 and Akara 150) next to the raw UI flags, the hovered unit and the NPC position; read the real text
// from tools/shot.sh while the probe runs.

func parseMercdump(args []string) (func(context.Context, *Env) error, error) {
	fs, sec := newFlags("mercdump")
	if err := parseFlags(fs, args, sec); err != nil {
		return nil, err
	}
	n := *sec
	return func(ctx context.Context, e *Env) error { return runMercdump(e, n) }, nil
}

var entryLabel = map[town.Entry]string{town.EntryTalk: "Talk", town.EntryTrade: "Trade", town.EntryHeal: "Heal",
	town.EntryGamble: "Gamble", town.EntryCancel: "Cancel", town.EntryResurrect: "Resurrect"}

// menuLabels is the expected dialog for an NPC class (S/V per town.MenuOrder).
func menuLabels(class int) []string {
	order, ok := town.MenuOrder[class]
	if !ok {
		order = []town.Entry{town.EntryTalk, town.EntryTrade, town.EntryCancel}
	}
	out := make([]string, len(order))
	for i, en := range order {
		out[i] = entryLabel[en]
	}
	return out
}

func openFlags(u d2.UIState) []string {
	var out []string
	for i := range u {
		if u[i] != 0 {
			out = append(out, fmt.Sprintf("0x%02x=%d", i, u[i]))
		}
	}
	return out
}

func mercRecord(m d2.Merc) map[string]any {
	return map[string]any{"present": m.Present, "alive": m.Alive, "class": m.Class, "id": m.ID, "x": m.X, "y": m.Y,
		"life_pct": m.LifePct, "level": m.Level, "name": m.Name, "aura": m.Aura}
}

func runMercdump(e *Env, sec int) error {
	const p = "mercdump"
	err := e.poll(time.Duration(sec)*time.Second, time.Second, func(int) error {
		s, err := e.Snap()
		if err != nil {
			return err
		}
		rec := mercRecord(d2.FindMerc(s))
		if s.Player != nil {
			rec["player_x"], rec["player_y"], rec["area"] = s.Player.X, s.Player.Y, s.Player.Area
		}
		rec["ui_open"] = openFlags(s.UI)
		rec["merc_panel"] = s.UI.Open(d2.UIMerc)
		e.emit(p, "merc", rec)
		if s.UI.Open(d2.UINPCMenu) {
			dl := map[string]any{"hover_active": s.Hovered.Active, "hover_type": s.Hovered.Type, "hover_id": s.Hovered.ID}
			for _, u := range s.Monsters {
				if u.Type == d2.TypeMonster && u.ID == s.Hovered.ID {
					dl["npc_class"], dl["npc_x"], dl["npc_y"] = u.Class, u.X, u.Y
					dl["lines_expected"] = menuLabels(u.Class)
					dl["lines_source"] = "town.MenuOrder labels (text not readable from memory, U)"
					if e.MonsterName != nil {
						dl["npc_name"], _ = e.MonsterName(u.Class)
					}
				}
			}
			e.emit(p, "dialog", dl)
		}
		return nil
	})
	if err != nil {
		return e.fail(p, err)
	}
	return nil
}

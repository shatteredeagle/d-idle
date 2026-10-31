package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"d2lod-bot/internal/d2"
)

// Census of the MonsterData flag byte (+0x16) and enchant array (+0x1C): both U until a live monster read confirms
// them (internal/d2/monflags.go). Everything the classifier says is printed next to the raw bytes so the reader can
// check the bit meanings (2 normal, 4 champion, 8 unique, 16 minion) against what the game shows.

type monOpts struct {
	sec, ms, radius int
}

func parseMonflags(args []string) (func(context.Context, *Env) error, error) {
	fs, sec := newFlags("monflags")
	ms := fs.Int("ms", 1000, "sample interval in ms (100..10000)")
	radius := fs.Int("radius", 40, "only monsters within this many subtiles of the player (1..400)")
	if err := parseFlags(fs, args, sec); err != nil {
		return nil, err
	}
	if *ms < 100 || *ms > 10000 || *radius < 1 || *radius > 400 {
		return nil, fmt.Errorf("monflags: -ms 100..10000 and -radius 1..400")
	}
	o := monOpts{*sec, *ms, *radius}
	return func(ctx context.Context, e *Env) error { return runMonflags(e, o) }, nil
}

func flagBits(f uint8) []int {
	var b []int
	for i := 0; i < 8; i++ {
		if f>>uint(i)&1 == 1 {
			b = append(b, 1<<uint(i))
		}
	}
	return b
}

// monRecord is the JSON shape of one monster; tests pin its keys.
func (e *Env) monRecord(p *d2.Player, u d2.Unit) map[string]any {
	m := u.Monster
	name, npc := "", false
	if e.MonsterName != nil {
		name, npc = e.MonsterName(u.Class)
	}
	r := map[string]any{
		"id": u.ID, "class": u.Class, "name": name, "npc": npc, "mode": u.Mode, "x": u.X, "y": u.Y,
		"dist": math.Round(math.Hypot(float64(u.X-p.X), float64(u.Y-p.Y))*10) / 10,
	}
	if m == nil {
		r["monster_data"] = false
		return r
	}
	ids, ok := m.EnchantIDs()
	raw := make([]string, len(m.Enchants))
	for i, v := range m.Enchants {
		raw[i] = hex8(v)
	}
	r["monster_data"] = true
	r["flags"] = hex8(m.Flags)
	r["flag_bits"] = flagBits(m.Flags)
	r["mon_kind"] = m.Kind().String()
	r["enchants_raw"] = raw
	idl := make([]int, len(ids))
	for i, v := range ids {
		idl[i] = int(v)
	}
	r["enchant_ids"] = idl
	r["enchants_trusted"] = ok
	r["unique_no"] = m.UniqueNo
	r["unique_name"] = m.Name
	r["life"], r["max_life"], r["life_pct"] = m.Life, m.MaxLife, m.LifePct
	r["states"] = m.States
	return r
}

func runMonflags(e *Env, o monOpts) error {
	const p = "monflags"
	seen := map[uint32]string{} // id -> last flags+enchants, to log changes only
	flagHist := map[string]int{}
	kindHist := map[string]int{}
	enchHist := map[string]int{}
	var total, withData int
	err := e.poll(time.Duration(o.sec)*time.Second, time.Duration(o.ms)*time.Millisecond, func(int) error {
		s, err := e.Snap()
		if err != nil {
			return err
		}
		if s.Player == nil || s.State != d2.StateInGame {
			e.emit(p, "skip", map[string]any{"reason": "not in game"})
			return nil
		}
		for _, u := range s.Monsters {
			if !u.Alive() || u.ID == s.Player.ID || d2.IsMercClass(u.Class) {
				continue
			}
			rec := e.monRecord(s.Player, u)
			if rec["npc"] == true || rec["dist"].(float64) > float64(o.radius) {
				continue
			}
			key := fmt.Sprint(rec["flags"], rec["enchants_raw"])
			if seen[u.ID] == key {
				continue
			}
			first := seen[u.ID] == ""
			seen[u.ID] = key
			e.emit(p, "monster", rec)
			if first {
				total++
				if rec["monster_data"] == true {
					withData++
					flagHist[rec["flags"].(string)]++
					kindHist[rec["mon_kind"].(string)]++
					for _, id := range rec["enchant_ids"].([]int) {
						enchHist[fmt.Sprint(id)]++
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return e.fail(p, err)
	}
	e.emit(p, "summary", map[string]any{
		"monsters": total, "with_monster_data": withData, "flags": flagHist, "kinds": kindHist, "enchant_ids": enchHist,
		"verdict": monVerdict(total, flagHist, kindHist),
	})
	return nil
}

// monVerdict is a hint for the reader, not a conclusion: it says what to cross-check.
func monVerdict(total int, flags, kinds map[string]int) string {
	switch {
	case total == 0:
		return "no hostile monsters in range: walk into a pack and run again"
	case len(flags) == 1 && flags["0x00"] == total:
		return "all flags zero: +0x16 is probably the wrong offset (U stays U)"
	case kinds["unknown"] == total:
		return "no known flag bit seen: compare raw flags against what the game shows"
	}
	keys := make([]string, 0, len(flags))
	for k := range flags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return fmt.Sprintf("flag values seen %v: check champion/unique/minion rows against the screen (tools/shot.sh)", keys)
}

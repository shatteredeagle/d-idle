package main

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/senses"
)

// damageDropPct is the life drop between two observations (about 500 ms apart) that is worth a marker on the trail.
const damageDropPct = 15

// activityFor maps what the runner says it is doing (the whyline code of docs/whyline) and whether the fighter has a
// target onto a trail activity code. The codes are the ones of internal/watch/whyline.json; an unknown code is a walk.
func activityFor(whyCode string, attacking, looting bool) uint8 {
	if attacking {
		return senses.ActCombat
	}
	if looting {
		return senses.ActLoot
	}
	switch whyCode {
	case "fight":
		return senses.ActCombat
	case "town_trip", "town_chores", "claim_trip", "to_town", "repair_trip", "repair_wait", "after_death_restock", "after_death_corpse", "player_items", "after_death_gold", "saving_for":
		return senses.ActTown
	case "farm_runs", "farm_gold", "farm_user", "repair_farm", "area_done":
		return senses.ActFarm
	case "guard_exit":
		return senses.ActFlee
	case "heal_trip", "wait_heal", "hold":
		return senses.ActRest
	}
	return senses.ActWalk
}

// historyEvent turns a goal event into a trail event; ok is false for the events the trail does not mark.
func historyEvent(ev goals.Event) (kind, text string, big, ok bool) {
	switch ev.Type {
	case goals.EvKill:
		return senses.EvKill, safeName(ev.Monster, "monster") + " killed", ev.Boss, true
	case goals.EvDeath:
		return senses.EvDeath, "Died to " + safeName(ev.Monster, "something"), true, true
	case goals.EvLevelUp:
		return senses.EvLevelUp, fmt.Sprintf("Level %d", ev.Level), true, true
	case goals.EvPotion:
		return senses.EvPotion, "Drank " + safeName(ev.Potion, "a potion"), false, true
	case goals.EvDrop:
		if ev.Item != nil {
			return senses.EvLoot, "Picked up " + safeName(ev.Item.Name, "an item"), ev.Item.Quality == "unique" || ev.Item.Quality == "set", true
		}
	}
	return "", "", false, false
}

func safeName(s, def string) string {
	if s == "" {
		return def
	}
	if len(s) > 60 {
		return s[:60]
	}
	return s
}

func fightInfo(s combat.FightSummary) *senses.FightInfo {
	return &senses.FightInfo{End: s.End, Kills: s.Kills, Damage: s.Damage, MinLife: s.MinLifePct, Life: s.LifePct, Sec: float64(int(s.Sec*10)) / 10,
		Potions: s.Potions, Casts: s.Casts, Approach: s.Approaches, Left: s.Left, Kites: s.Kites, Dodges: s.Dodges}
}

// useSensesHistory feeds the trail's activity colour, the event markers and the fight summaries (docs/senses-map.md
// "History"). The activity and a life-drop marker ride the OnObserve hook (nav goroutine, 2 Hz, chained after
// useSenses); kills, loot, potions, level-ups and deaths come from the goal manager's event stream; a finished fight
// comes from the fighter's OnSummary. Read-only: nothing here reaches the bot.
// failureSetter is the navigator's OnFailure installer (nav.Navigator.SetOnFailure).
type failureSetter interface {
	SetOnFailure(func(kind, text string, x, y int))
}

func useSensesHistory(ctx context.Context, rig *ladderRig, e *env.Env, f *combat.Fighter, m *goals.Manager, status func() goals.Status, looting func() bool, mover any) {
	if rig == nil || rig.Senses == nil || e == nil || e.OnObserve == nil || status == nil {
		return
	}
	pub := rig.Senses
	prev, lastLife, lastDmg := e.OnObserve, -1, time.Time{}
	e.OnObserve = func(lm *d2map.LevelMap, pos d2map.Point) {
		st := status()
		attacking := f != nil && f.Attacking() != 0
		act := activityFor(st.Why.Code, attacking, looting != nil && looting())
		pub.SetActivity(act)
		prev(lm, pos)
		if c := st.Character; c.MaxLife > 0 {
			pub.AddSample(senses.Sample{Life: c.Life * 100 / c.MaxLife, Mana: pct(c.Mana, c.MaxMana), Act: act, Why: st.Why.Code, Level: c.Level, XP: c.XP})
			pc := c.Life * 100 / c.MaxLife
			if lastLife >= 0 && lastLife-pc >= damageDropPct && time.Since(lastDmg) > 2*time.Second {
				lastDmg = time.Now()
				pub.AddDamage(lastLife-pc, fmt.Sprintf("Life %d%% to %d%%", lastLife, pc))
			}
			lastLife = pc
		}
	}
	if fs, ok := mover.(failureSetter); ok {
		fs.SetOnFailure(func(kind, text string, x, y int) { pub.AddEventAt(kind, text, x, y, 0) })
	}
	if f != nil {
		f.OnNote = func(n combat.Note) {
			switch {
			case n.HasPos:
				pub.AddEventAt(n.Kind, n.Text, n.X, n.Y, n.Secs)
			default:
				pub.AddEvent(n.Kind, n.Text, false, nil)
			}
		}
		f.OnSummary = func(s combat.FightSummary) {
			fi := fightInfo(s)
			pub.AddEvent(senses.EvFight, fmt.Sprintf("Fight: %d kills in %.1f s, end=%s, life min %d%%", s.Kills, s.Sec, s.End, s.MinLifePct), false, fi)
			switch s.Left {
			case "chicken":
				pub.AddEvent(senses.EvChicken, fmt.Sprintf("Chickened at %d%% life", s.LifePct), true, fi)
			case "portal":
				pub.AddEvent(senses.EvPortal, fmt.Sprintf("Town portal at %d%% life", s.LifePct), true, fi)
			}
		}
	}
	if m != nil {
		ch, unsub := m.Subscribe()
		go func() {
			defer unsub()
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-ch:
					if !ok {
						return
					}
					if k, t, big, ok := historyEvent(ev); ok {
						pub.AddEvent(k, t, big, nil)
					}
				}
			}
		}()
	}
}

func pct(v, max int) int {
	if max <= 0 {
		return 0
	}
	return v * 100 / max
}

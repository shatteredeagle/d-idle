package cheat

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

func (e *Env) applyMemory(ctx context.Context, g *game, req Request) (string, error) {
	switch req.Kind {
	case KindHeal:
		n, err := g.heal()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("refilled %d stats (life, mana, stamina) on the server unit", n), nil
	case KindGold, KindStashGold:
		v, ok := req.Int("value")
		if !ok || v < 0 {
			return "", fmt.Errorf("%s needs a non-negative value", req.Kind)
		}
		id := statGold
		if req.Kind == KindStashGold {
			id = statStashGold
		}
		if err := g.setBoth(id, int32(v)); err != nil {
			return "", fmt.Errorf("%w; the character holds none of this yet: use the save-edit cheat (it adds the stat), then retry", err)
		}
		return fmt.Sprintf("%s = %d", req.Kind, v), nil
	case KindXP:
		v, ok := req.Int("value")
		if !ok || v < 0 {
			return "", fmt.Errorf("xp needs a non-negative value")
		}
		if err := g.setBoth(statExp, int32(uint32(v))); err != nil {
			return "", fmt.Errorf("%w; use the save-edit cheat to add it", err)
		}
		return fmt.Sprintf("experience = %d (the game levels up at the next experience gain, not immediately; use level or the save-edit cheat to change the level)", v), nil
	case KindLevel:
		return e.memLevel(g, req)
	case KindStats:
		return e.memStats(g, req)
	case KindSetStat:
		id, ok1 := req.Int("stat")
		val, ok2 := req.Int("value")
		if !ok1 || !ok2 {
			return "", fmt.Errorf("setstat needs stat and value")
		}
		unit, vec := req.Str("unit"), req.Str("vec")
		if unit == "" {
			unit = UnitBoth
		}
		if vec == "" {
			vec = VecBoth
		}
		if ins, _ := req.Int("insert"); ins != 0 {
			var done []string
			for _, un := range []string{UnitServer, UnitClient} {
				if unit != UnitBoth && unit != un {
					continue
				}
				u, err := g.unit(un)
				if err != nil {
					return "", err
				}
				if a, _ := g.slot(u, VecMerged, int(id), 0); a != 0 {
					if err := g.set(un, VecBoth, int(id), int32(val)); err != nil {
						return "", err
					}
					done = append(done, un+" (existing)")
					if b, _ := g.slot(u, VecBase, int(id), 0); b == 0 && g.insertStat(u, VecBase, int(id), int32(val)) == nil {
						done = append(done, un+" base (inserted)")
					}
					continue
				}
				if err := g.insertStat(u, VecMerged, int(id), int32(val)); err != nil {
					return "", err
				}
				done = append(done, un+" merged (inserted)")
				// also the base vector when it has room: the merged vector is rebuilt from base + item lists
				if a, _ := g.slot(u, VecBase, int(id), 0); a == 0 {
					if err := g.insertStat(u, VecBase, int(id), int32(val)); err == nil {
						done = append(done, un+" base (inserted)")
					}
				} else if g.set(un, VecBase, int(id), int32(val)) == nil {
					done = append(done, un+" base (existing)")
				}
			}
			return fmt.Sprintf("stat %d = %d: %s", id, val, strings.Join(done, ", ")), nil
		}
		if err := g.set(unit, vec, int(id), int32(val)); err != nil {
			return "", err
		}
		return fmt.Sprintf("stat %d = %d on %s/%s", id, val, unit, vec), nil
	case KindGod:
		return e.god(ctx, g, req)
	case KindGodRestore:
		return e.godRestore(g, req)
	}
	return "", fmt.Errorf("cheat: %s not implemented in memory", req.Kind)
}

// setBoth writes a stat into the server unit and the client copy, base and merged vectors. The server
// is authoritative (the save is made from it); the client copy is what the UI shows and the server does
// not push attribute changes to it. VERIFIED LIVE (docs/cheats.md).
func (g *game) setBoth(id int, v int32) error { return g.set(UnitBoth, VecBoth, id, v) }

// heal sets life, mana and stamina to their maxima on the server unit (both vectors, so a recalculation
// cannot revert it) and on the client copy.
func (g *game) heal() (int, error) {
	n := 0
	for _, p := range [][2]int{{statLife, statMaxLife}, {statMana, statMaxMana}, {statStamina, statMaxStamina}} {
		max, ok := g.get(UnitServer, VecMerged, p[1])
		if !ok {
			return n, fmt.Errorf("max stat %d not readable", p[1])
		}
		if err := g.set(UnitServer, VecBoth, p[0], max); err != nil {
			return n, err
		}
		_ = g.set(UnitClient, VecBoth, p[0], max)
		n++
	}
	return n, nil
}

// addStat adds delta to a stat that is present (both units/vectors). ok is false when it is absent.
func (g *game) addStat(id int, delta int32) bool {
	cur, ok := g.get(UnitServer, VecMerged, id)
	if !ok {
		return false
	}
	return g.setBoth(id, cur+delta) == nil
}

// memLevel sets the level stat and what follows from it as far as the stats exist in memory: experience
// for the level, max life/mana/stamina by the class gains (refilled). Unspent points are added only when
// the stat is already present (a fresh character has none: then use the save-edit cheat).
func (e *Env) memLevel(g *game, req Request) (string, error) {
	to, ok := req.Int("value")
	if !ok || to < 1 || to > 99 {
		return "", fmt.Errorf("level needs a value 1-99")
	}
	rules, err := e.Rules()
	if err != nil {
		return "", err
	}
	from, _ := g.get(UnitServer, VecMerged, statLevel)
	if err := g.setBoth(statLevel, int32(to)); err != nil {
		return "", err
	}
	var skipped []string
	if exp := rules.ExpToLevel(int(to)); exp > 0 && exp < 1<<31 {
		if g.setBoth(statExp, int32(exp)) != nil {
			skipped = append(skipped, "exp")
		}
	}
	if n := int32(to) - from; n > 0 {
		gn := rules.Gains(int(g.class))
		pts := int32(5)
		if gn.StatPerLevel > 0 {
			pts = int32(gn.StatPerLevel)
		}
		if !g.addStat(statPoints, n*pts) {
			skipped = append(skipped, "statpts")
		}
		if !g.addStat(statSkillPts, n) {
			skipped = append(skipped, "skillpts")
		}
		for _, p := range []struct{ cur, max, per int }{{statLife, statMaxLife, gn.LifePerLevel}, {statMana, statMaxMana, gn.ManaPerLevel}, {statStamina, statMaxStamina, gn.StaminaPerLevel}} {
			if !g.addStat(p.max, n*int32(p.per)*256/4) {
				skipped = append(skipped, "max"+fmt.Sprint(p.max))
			}
		}
		if _, err := g.heal(); err != nil {
			skipped = append(skipped, "heal")
		}
	}
	msg := fmt.Sprintf("level %d -> %d", from, to)
	if len(skipped) > 0 {
		msg += " (not present in memory, skipped: " + strings.Join(skipped, ", ") + ")"
	}
	return msg, nil
}

// memStats sets attributes (str dex vit ene), unspent points (points, skills). Vitality/energy changes
// also move max life/mana/stamina by the class gains. Only stats that exist in memory can be set.
func (e *Env) memStats(g *game, req Request) (string, error) {
	rules, err := e.Rules()
	if err != nil {
		return "", err
	}
	gn := rules.Gains(int(g.class))
	var did []string
	for _, p := range []struct {
		key string
		id  int
	}{{"str", statStrength}, {"dex", statDexterity}, {"vit", statVitality}, {"ene", statEnergy}, {"points", statPoints}, {"skills", statSkillPts}} {
		v, ok := req.Int(p.key)
		if !ok {
			continue
		}
		old, present := g.get(UnitServer, VecMerged, p.id)
		if !present {
			return "", fmt.Errorf("%s: %w; use the save-edit cheat to add it", p.key, errAbsent)
		}
		if err := g.setBoth(p.id, int32(v)); err != nil {
			return "", err
		}
		d := int32(v) - old
		switch p.id {
		case statVitality:
			g.addStat(statMaxLife, d*int32(gn.LifePerVit)*256/4)
			g.addStat(statMaxStamina, d*int32(gn.StaminaPerVit)*256/4)
		case statEnergy:
			g.addStat(statMaxMana, d*int32(gn.ManaPerEnergy)*256/4)
		}
		did = append(did, fmt.Sprintf("%s %d->%d", p.key, old, v))
	}
	if len(did) == 0 {
		return "", fmt.Errorf("stats needs one of str dex vit ene points skills")
	}
	_, _ = g.heal()
	return strings.Join(did, ", "), nil
}

// godBuffer is the largest life a 21-bit stat can hold (8191.99 in 24.8 fixed point).
const godBuffer int32 = 1<<21 - 1

// god makes the character very hard to kill for the duration given in "for" (default 60s). There is no
// invincibility flag we could find (see docs/cheats.md), so it works in two layers, both on the server
// unit: max life and life are raised to the 21-bit stat limit (8191 life, so no normal hit kills in one
// blow) and, every 5 ms, life/mana/stamina are refilled. On exit (timeout or ctx) max life and life are
// restored to their previous values (read back), so a Save and Exit afterwards does not keep the inflated life.
// An undo marker with the originals is written before the first change (godstate.go); a SIGKILLed run is
// repaired by the next god start or `d2cheat restore`.
func (e *Env) god(ctx context.Context, g *game, req Request) (string, error) {
	on, _ := req.Params["on"].(bool)
	if !on {
		n, err := g.heal()
		return fmt.Sprintf("god off: nothing running in this process; refilled %d stats", n), err
	}
	d := 60 * time.Second
	if s := req.Str("for"); s != "" {
		pd, err := time.ParseDuration(s)
		if err != nil {
			return "", err
		}
		d = pd
	}
	pre, err := e.pendingRestore(g)
	if err != nil {
		return "", err
	}
	origMax, ok := g.get(UnitServer, VecMerged, statMaxLife)
	if !ok {
		return "", fmt.Errorf("max life not readable")
	}
	origLife, _ := g.get(UnitServer, VecMerged, statLife)
	buffer := true
	if v, ok := req.Params["buffer"].(bool); ok {
		buffer = v
	}
	if buffer {
		// marker first: if we die between here and the restore, the originals are on disk
		if err := e.writeMarker(godMarker{Time: e.now(), Instance: e.Instance, Character: g.name, MaxLife: origMax, Life: origLife, PID: os.Getpid()}); err != nil {
			return "", fmt.Errorf("cheat: cannot write the undo marker, refusing god mode: %w", err)
		}
		// runs on return, panic, timeout and ctx cancel (SIGINT/SIGTERM via the caller's context)
		defer func() {
			if err := g.restoreLife(origMax); err == nil {
				_ = os.Remove(e.markerPath())
			}
		}()
		if err := g.setBoth(statMaxLife, godBuffer); err != nil {
			return "", err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	ticks, hits := 0, 0
	lowest := int32(1<<31 - 1)
	simhit, _ := req.Int("simhit") // research: pretend a monster hits for this much, every 50 ms
	for ctx.Err() == nil {
		if simhit > 0 && ticks%10 == 5 {
			if life, ok := g.get(UnitServer, VecMerged, statLife); ok {
				_ = g.set(UnitServer, VecBoth, statLife, life-int32(simhit)*256)
			}
		}
		max, _ := g.get(UnitServer, VecMerged, statMaxLife)
		if life, ok := g.get(UnitServer, VecMerged, statLife); ok {
			if life < max {
				hits++
			}
			if life < lowest {
				lowest = life
			}
		}
		if _, err := g.heal(); err != nil {
			return "", fmt.Errorf("god stopped after %d refills: %w", ticks, err)
		}
		ticks++
		e.sleep(5 * time.Millisecond)
	}
	msg := fmt.Sprintf("god mode ran %v: %d refills, life below max on %d samples, lowest life seen %d", d, ticks, hits, lowest/256)
	if pre != "" {
		msg = pre + "; " + msg
	}
	return msg, nil
}

func (e *Env) sleep(d time.Duration) {
	if e.Sleep != nil {
		e.Sleep(d)
		return
	}
	time.Sleep(d)
}

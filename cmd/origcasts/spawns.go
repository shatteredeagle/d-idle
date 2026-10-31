package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

const spawnHeader = "area,visit,normal,minion,champion,unique,superunique,total,names,boss_names,coverage_pct,known_cells,explore_s,end"

// superuniques of the five areas (SuperUniques.txt names): a boss-flagged monster with one of these names is
// "superunique" in the d2d table, any other boss-flagged one is a rare ("unique" column of the d2d CSV).
var superuniques = map[string]bool{"Bishibosh": true, "Rakanishu": true, "Treehead WoodFist": true, "Corpsefire": true, "Coldcrow": true}

type seen struct {
	class      int
	name, boss string
	flags      uint8
	uniqueNo   uint16
	kind       d2.MonKind
}

// census accumulates every monster id seen alive in one area (de-duplicated by unit id).
type census struct {
	r    *rig
	area int
	ids  map[uint32]seen
	last time.Time
}

func (c *census) poll() {
	if time.Since(c.last) < 200*time.Millisecond {
		return
	}
	c.last = time.Now()
	s, err := c.r.e.Snapshot()
	if err != nil || s.Player == nil || s.Player.Area != c.area {
		return
	}
	cat := combat.NewCatalog(c.r.db)
	for i := range s.Monsters {
		u := &s.Monsters[i]
		if _, ok := c.ids[u.ID]; ok || !u.Alive() || u.Monster == nil {
			continue
		}
		m := cat.Monster(u.Class)
		if m != nil && (m.NPC || !m.Killable) {
			continue
		}
		sn := seen{class: u.Class, name: cat.MonsterName(m), flags: u.Monster.Flags, uniqueNo: u.Monster.UniqueNo, kind: u.Monster.Kind(), boss: u.Monster.Name}
		c.ids[u.ID] = sn
		if sn.flags != 0 {
			fmt.Fprintf(os.Stderr, "# area %d non-normal id=%d class=%d %q flags=0x%02x uniqueNo=%d bossname=%q enchants=%v pos=(%d,%d)\n",
				c.area, u.ID, u.Class, sn.name, sn.flags, sn.uniqueNo, sn.boss, u.Monster.Enchants, u.X, u.Y)
		}
	}
}

// spawns walks the legs in order. A leg is "N" (go to area N by nav.ToArea, explore it completely, census), "-N"
// (travel only) or "xN" (first explore the current level until its exit to N is known, then as "N"): nav.ToArea does
// not explore for an exit it has not seen, and V-live it wandered into the Underground Passage when sent from Stony
// Field to Dark Wood (the only way on foot goes through level 10, docs/nav.md).
func (r *rig) spawns(ctx context.Context, visit string, legs []string) {
	stop := r.god(ctx, "spawn count probe: survive while exploring whole levels at clvl 1 to count monsters (throwaway Testy)")
	defer func() { fmt.Fprintln(os.Stderr, "# god stop:", stop()) }()
	fmt.Println(spawnHeader)
	for _, leg := range legs {
		explore := strings.HasPrefix(leg, "x")
		area, err := strconv.Atoi(strings.TrimPrefix(leg, "x"))
		if err != nil {
			die(err)
		}
		if explore {
			r.exploreUntilExit(ctx, area)
		}
		if area < 0 { // travel only (a leg through an area that is not counted this visit, e.g. back out of the Den)
			fmt.Fprintf(os.Stderr, "# travel to %d: %v\n", -area, r.nav.ToArea(ctx, -area))
			continue
		}
		t0 := time.Now()
		c := &census{r: r, area: area, ids: map[uint32]seen{}}
		end := "explored"
		if err := r.nav.ToArea(ctx, area); err != nil {
			fmt.Fprintf(os.Stderr, "# ToArea(%d): %v\n", area, err)
			end = "toarea:" + err.Error()
		} else {
			limit := time.Now().Add(240 * time.Second)
			err := r.nav.Explore(ctx, func() bool { c.poll(); return time.Now().After(limit) })
			if err != nil {
				end = err.Error()
			} else {
				end = "limit"
			}
		}
		c.poll()
		cov, cells := r.coverage(area)
		fmt.Println(c.row(area, visit, cov, cells, time.Since(t0), end))
		if ctx.Err() != nil {
			return
		}
	}
}

func (r *rig) coverage(area int) (pct float64, cells int) {
	if _, err := r.e.World.Update(); err != nil {
		return 0, 0
	}
	lm := r.e.World.Current()
	if lm == nil || lm.No != area || lm.Grid == nil {
		return 0, 0
	}
	cells = lm.Grid.KnownCount()
	if a := lm.Rect.W * lm.Rect.H; a > 0 {
		pct = 100 * float64(cells) / float64(a)
	}
	return
}

func (c *census) row(area int, visit string, cov float64, cells int, dur time.Duration, end string) string {
	var normal, minion, champ, uniq, sup int
	names := map[string]int{}
	var bosses []string
	for _, s := range c.ids {
		// V-live flag bytes (MonsterData+0x16): 0x00 ordinary (Kind() says Unknown), 0x10 pack follower of an elite,
		// 0x0c champion (named "Champion/Possessed/Ghostly/... X"), 0x08 rare monster (own name), 0x0a superunique
		// (Bishibosh); d2.Kind() tests the boss bit first, so every champion would read as unique: classify here.
		switch {
		case s.flags&0x04 != 0:
			champ++
			bosses = append(bosses, "champion:"+s.boss)
		case s.flags&0x08 != 0 && superuniques[s.boss]:
			sup++
			bosses = append(bosses, "superunique:"+s.boss)
		case s.flags&0x08 != 0:
			uniq++
			bosses = append(bosses, "unique:"+s.boss)
		case s.flags&0x10 != 0:
			minion++
			names[s.name]++
		default: // MonKindNormal, or Unknown (flags 0x00: V-live every ordinary Fallen/Zombie/Quill Rat in the Blood Moor read 0)
			normal++
			names[s.name]++
		}
	}
	var ns []string
	for n, k := range names {
		ns = append(ns, fmt.Sprintf("%s x%d", n, k))
	}
	sort.Strings(ns)
	sort.Strings(bosses)
	return fmt.Sprintf("%d,%s,%d,%d,%d,%d,%d,%d,%q,%q,%.1f,%d,%.0f,%q", area, visit, normal, minion, champ, uniq, sup, len(c.ids),
		strings.Join(ns, ";"), strings.Join(bosses, ";"), cov, cells, dur.Seconds(), end)
}

func (r *rig) exploreUntilExit(ctx context.Context, dest int) {
	t0 := time.Now()
	err := r.nav.Explore(ctx, func() bool {
		if _, err := r.e.World.Update(); err != nil {
			return false
		}
		for _, x := range r.e.World.Current().Exits {
			if x.Dest == dest {
				return true
			}
		}
		return time.Since(t0) > 240*time.Second
	})
	fmt.Fprintf(os.Stderr, "# explore until exit to %d is known: %v after %v\n", dest, err, time.Since(t0).Round(time.Second))
}

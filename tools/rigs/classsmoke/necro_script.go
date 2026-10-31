//go:build rigs

package main

// nsummon: one scripted Necromancer session (agent CLASSNEC). Walks to the Blood Moor, then records what memory shows for
// Bone Armor, a curse on a monster, a monster corpse, Raise Skeleton, Clay Golem, Corpse Explosion, the combat Hostiles filter
// with minions around, and the minion count after an area change. Cheats: only `heal` (cheat ledger, actor classnec).

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
)

func init() {
	if len(os.Args) > 1 && os.Args[1] == "nsummon" {
		cheat.Allowed = append(cheat.Allowed, "Necci")
		nsummon()
		os.Exit(0)
	}
}

func dist(ax, ay, bx, by int) float64 { return math.Hypot(float64(ax-bx), float64(ay-by)) }

func stateChain(src mem.Source, addr uint32) string {
	top := rdU32(src, addr+0x5C)
	if top == 0 {
		return "(no stat list)"
	}
	var sb strings.Builder
	child := rdU32(src, top+0x3c)
	for n := 0; child > 0x10000 && n < 40; n++ {
		st := rdU32(src, child+0x14)
		vec, cnt := rdU32(src, child+0x24), rdU32(src, child+0x28)&0xffff
		fmt.Fprintf(&sb, " state%d{", st)
		for i := uint32(0); i < cnt && i < 12; i++ {
			w0, w1 := rdU32(src, vec+i*8), rdU32(src, vec+i*8+4)
			fmt.Fprintf(&sb, "%d=%d ", w0>>16, int32(w1))
		}
		sb.WriteString("}")
		child = rdU32(src, child+0x2c)
	}
	return sb.String()
}

func nsummon() {
	src, err := mem.OpenGame()
	die(err)
	defer src.Close()
	game := input.NewGame(input.NewClient(""), input.Reader{Src: src})
	home, _ := os.UserHomeDir()
	inst := "Games/d2lod-inst/" + os.Getenv(mem.InstanceEnv) + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	db, err := data.Open(home+"/"+inst, home+"/d2lod-live/classnec/data")
	die(err)
	tune := env.DefaultTuning()
	kills := 0
	e := &env.Env{ID: "classnec", Log: slog.New(slog.NewTextHandler(os.Stdout, nil)), Mem: src, Read: d2.New(src), In: game, Data: db,
		Tune: &tune, Clock: env.RealClock{}, Emit: func(ev goals.Event) {
			if ev.Type == goals.EvKill {
				kills++
			}
		}}
	e.World = d2map.NewTracker(src)
	nv := nav.New(e)
	f := combat.New(e, nv)
	rows, err := combat.LoadListRows(home+"/"+inst, db)
	die(err)
	f.SetListRows(rows)
	cat := combat.NewCatalog(db)
	ctx := context.Background()
	t0 := time.Now()
	say := func(format string, a ...any) {
		fmt.Printf("[%5.1fs] %s\n", time.Since(t0).Seconds(), fmt.Sprintf(format, a...))
	}
	name := func(c int) string {
		if m := db.MonsterByClassID(c); m != nil {
			return m.Key
		}
		return "?"
	}
	snap := func() *d2.Snapshot {
		s, err := e.Snapshot()
		die(err)
		return s
	}
	census := func(label string) (owned []d2.Unit) {
		s := snap()
		p := s.Player
		var parts []string
		for _, u := range s.Monsters {
			d := dist(p.X, p.Y, u.X, u.Y)
			if d > 60 {
				continue
			}
			if u.OwnerID == p.ID && u.OwnerType == 0 {
				owned = append(owned, u)
				parts = append(parts, fmt.Sprintf("OWNED(%s id%d cls%d mode%d d%.0f)", name(u.Class), u.ID, u.Class, u.Mode, d))
			} else if u.OwnerID != 0 {
				parts = append(parts, fmt.Sprintf("other-owner(%d/%d %s)", u.OwnerType, u.OwnerID, name(u.Class)))
			} else if !u.Alive() {
				parts = append(parts, fmt.Sprintf("corpse(%s id%d mode%d d%.0f)", name(u.Class), u.ID, u.Mode, d))
			} else {
				parts = append(parts, fmt.Sprintf("mon(%s id%d mode%d d%.0f)", name(u.Class), u.ID, u.Mode, d))
			}
		}
		say("%s: life %d/%d mana %d/%d area %d | %s", label, p.Life, p.MaxLife, p.Mana, p.MaxMana, p.Area, strings.Join(parts, " "))
		return owned
	}
	heal := func(why string) {
		_, err := cheat.DefaultEnv().Do(ctx, cheat.Request{Kind: "heal", Actor: "classnec", Reason: why, Params: map[string]any{}})
		if err != nil {
			say("heal: %v", err)
		}
	}
	hostiles := func() []combat.Target {
		s := snap()
		return combat.Hostiles(s, combat.HostileCtx{Cat: cat, Diff: s.Player.Difficulty, ClassLvl: s.Player.Level})
	}
	castAt := func(key string, w input.Point) { die(game.CastAt(key, w)); time.Sleep(700 * time.Millisecond) }

	if len(os.Args) > 2 && os.Args[2] == "e" {
		nsummonE(ctx, f, nv, game, src, snap, heal, hostiles, name, say, census)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "d" {
		nsummonD(ctx, f, nv, game, src, snap, heal, hostiles, name, say)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "c" {
		nsummonC(ctx, nv, game, src, snap, heal, hostiles, name, say)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "b" {
		nsummonB(ctx, e, f, nv, game, src, cat, say, snap, census, heal, hostiles, name)
		return
	}
	say("EnsureHotkeys: %v %v", f.EnsureHotkeys(ctx), f.Hotkeys())
	heal("CLASSNEC smoke: refill mana before the summon probe (throwaway Necci)")
	if snap().Player.InTown() {
		say("ExitTo(2): %v", nv.ExitTo(ctx, 2))
	}
	xctx, xc := context.WithTimeout(ctx, 60*time.Second)
	say("Explore: %v", nv.Explore(xctx, func() bool {
		for _, h := range hostiles() {
			if h.Dist < 30 {
				return true
			}
		}
		return false
	}))
	xc()
	census("before anything")
	p := snap().Player
	say("player states (stat list children):%s", stateChain(src, p.Addr))

	// 1. Bone Armor
	castAt("f3", input.Point{X: p.X, Y: p.Y})
	p = snap().Player
	say("after Bone Armor (f3 at own feet) mana %d: states:%s", p.Mana, stateChain(src, p.Addr))

	// 2. curse on the nearest monster
	hs := hostiles()
	if len(hs) == 0 {
		say("no hostile found, stop")
		return
	}
	tg := hs[0]
	say("target %s id%d at (%d,%d) d=%.0f states before:%s", tg.Name, tg.Unit.ID, tg.Unit.X, tg.Unit.Y, tg.Dist, stateChain(src, tg.Unit.Addr))
	castAt("f4", input.Point{X: tg.Unit.X, Y: tg.Unit.Y})
	for _, u := range snap().Monsters {
		if u.ID == tg.Unit.ID {
			say("target after Amplify Damage (f4 at monster): mode %d states:%s", u.Mode, stateChain(src, u.Addr))
		}
	}
	// the same curse cast at the player's own feet (what combat.Buff does for the profile's second buff group)
	p = snap().Player
	m0 := p.Mana
	castAt("f4", input.Point{X: p.X, Y: p.Y})
	p = snap().Player
	say("Amplify Damage at own feet: mana %d -> %d (cost spent, target states now:%s)", m0, p.Mana, stateChain(src, tg.Unit.Addr))

	// 3. kill it with Bone Spear and watch the corpse
	deadAt := time.Time{}
	var cid uint32
	for time.Since(t0) < 90*time.Second {
		s := snap()
		var cur *d2.Unit
		for i := range s.Monsters {
			if s.Monsters[i].ID == tg.Unit.ID {
				cur = &s.Monsters[i]
			}
		}
		if cur == nil {
			say("target left the unit table")
			break
		}
		if !cur.Alive() {
			deadAt = time.Now()
			cid = cur.ID
			say("target dead: mode %d class %d (%s) pos (%d,%d)", cur.Mode, cur.Class, name(cur.Class), cur.X, cur.Y)
			break
		}
		if s.Player.Mana < 8 {
			heal("CLASSNEC smoke: mana refill during the kill (throwaway Necci)")
		}
		castAt("f2", input.Point{X: cur.X, Y: cur.Y})
	}
	if cid == 0 {
		return
	}
	var last string
	for time.Since(deadAt) < 6*time.Second {
		s := snap()
		cur := "gone"
		for _, u := range s.Monsters {
			if u.ID == cid {
				cur = fmt.Sprintf("mode %d", u.Mode)
			}
		}
		if cur != last {
			say("corpse id%d %.2fs after death: %s", cid, time.Since(deadAt).Seconds(), cur)
			last = cur
		}
		time.Sleep(100 * time.Millisecond)
	}
	// 4. Raise Skeleton on the corpse
	var cpos input.Point
	for _, u := range snap().Monsters {
		if u.ID == cid {
			cpos = input.Point{X: u.X, Y: u.Y}
		}
	}
	heal("CLASSNEC smoke: mana for the summon probe (throwaway Necci)")
	m0 = snap().Player.Mana
	castAt("f5", cpos)
	time.Sleep(1500 * time.Millisecond)
	owned := census("after Raise Skeleton (f5) at the corpse")
	say("mana %d -> %d, owned %d", m0, snap().Player.Mana, len(owned))
	if len(owned) > 0 {
		say("minion states:%s", stateChain(src, owned[0].Addr))
	}
	// the combat Hostiles filter with a minion around
	hs = hostiles()
	var ids []string
	for _, h := range hs {
		ids = append(ids, fmt.Sprintf("%s/%d", h.Name, h.Unit.ID))
	}
	say("combat.Hostiles now: %v (owned minion ids must not be in it)", ids)

	// 5. Clay Golem through the right popup (not in the F-key layout)
	heal("CLASSNEC smoke: mana for the golem cast (throwaway Necci)")
	cat2 := cat.SkillByName("Clay Golem")
	if cat2 != nil {
		say("SetRight(Clay Golem): %v", f.SetRight(ctx, cat2.ID))
		p = snap().Player
		die(game.Click(input.Right, 400, 300))
		time.Sleep(1800 * time.Millisecond)
		owned = census("after Clay Golem")
	}
	// 6. second target: Corpse Explosion on a fresh corpse
	say("Explore for the next monster: %v", nv.Explore(ctx, func() bool {
		for _, h := range hostiles() {
			if h.Dist < 30 {
				return true
			}
		}
		return false
	}))
	say("SetRight(Bone Spear)...")
	_ = f.EnsureHotkeys(ctx)
	// Clear with minions around (they must be left alone); record the result and the census
	heal("CLASSNEC smoke: mana before the 25 s Clear with minions (throwaway Necci)")
	cctx, cc := context.WithTimeout(ctx, 40*time.Second)
	res, cerr := f.Clear(cctx, combat.ClearOpts{Radius: 50, Timeout: 30 * time.Second})
	cc()
	say("Clear with minions: %v err=%v kills(events)=%d", res, cerr, kills)
	owned = census("after Clear")
	// corpses now? try Corpse Explosion on one
	var corpse *d2.Unit
	s := snap()
	for i := range s.Monsters {
		u := s.Monsters[i]
		if !u.Alive() && u.OwnerID == 0 {
			corpse = &s.Monsters[i]
			break
		}
	}
	if corpse != nil {
		ce := cat.SkillByName("Corpse Explosion")
		say("SetRight(Corpse Explosion): %v", f.SetRight(ctx, ce.ID))
		heal("CLASSNEC smoke: mana for Corpse Explosion (throwaway Necci)")
		s = snap()
		x, y := input.Calibrated.WorldToScreenVP(s.Viewport.X, s.Viewport.Y, float64(corpse.X)+0.5, float64(corpse.Y)+0.5)
		before := len(s.Monsters)
		m0 = s.Player.Mana
		die(game.Click(input.Right, int(x), int(y)))
		time.Sleep(1500 * time.Millisecond)
		s = snap()
		say("Corpse Explosion on corpse id%d: mana %d -> %d, monster table %d -> %d", corpse.ID, m0, s.Player.Mana, before, len(s.Monsters))
		census("after Corpse Explosion")
	} else {
		say("no corpse left after Clear (corpse lifetime or removal)")
	}
	// 7. area change: minions follow? town and back
	say("ExitTo(1): %v", nv.ExitTo(ctx, 1))
	time.Sleep(3 * time.Second)
	census("in town after the area change")
	say("ExitTo(2): %v", nv.ExitTo(ctx, 2))
	time.Sleep(3 * time.Second)
	census("back in the Blood Moor")
	if errors.Is(cerr, context.Canceled) {
		return
	}
}

// rawUnit prints the words 0x80..0xB0 of a UnitAny (owner type/id live at 0x94/0x98) and the flags at 0xC4.
func rawUnit(src mem.Source, u d2.Unit) string {
	var sb strings.Builder
	for off := uint32(0x80); off < 0xB0; off += 4 {
		fmt.Fprintf(&sb, " +%x=%x", off, rdU32(src, u.Addr+off))
	}
	fmt.Fprintf(&sb, " +c4=%x +c8=%x", rdU32(src, u.Addr+0xC4), rdU32(src, u.Addr+0xC8))
	return sb.String()
}

func nsummonB(ctx context.Context, e *env.Env, f *combat.Fighter, nv *nav.Navigator, game *input.Game, src mem.Source, cat combat.Catalog,
	say func(string, ...any), snap func() *d2.Snapshot, census func(string) []d2.Unit, heal func(string), hostiles func() []combat.Target, name func(int) string) {
	say("EnsureHotkeys: %v", f.EnsureHotkeys(ctx))
	heal("CLASSNEC smoke: refill before summon probe B (throwaway Necci)")
	if snap().Player.InTown() {
		say("ExitTo(2): %v", nv.ExitTo(ctx, 2))
	}
	census("start")
	// raw owner/flag words of every minion and of one hostile
	for _, u := range snap().Monsters {
		if name(u.Class) == "claygolem" || name(u.Class) == "necroskeleton" || name(u.Class) == "fallen1" {
			say("raw %s id%d mode%d owner(+94/+98 parsed)=%d/%d:%s", name(u.Class), u.ID, u.Mode, u.OwnerType, u.OwnerID, rawUnit(src, u))
			say("   states:%s", stateChain(src, u.Addr))
		}
	}
	p := snap().Player
	say("player id %d", p.ID)
	near := func(maxd float64) *combat.Target {
		for _, h := range hostiles() {
			if h.Dist < maxd && h.Unit.Alive() && name(h.Unit.Class) != "claygolem" && name(h.Unit.Class) != "necroskeleton" {
				h := h
				return &h
			}
		}
		return nil
	}
	start := time.Now()
	for near(40) == nil && time.Since(start) < 60*time.Second {
		xctx, xc := context.WithTimeout(ctx, 20*time.Second)
		nv.Explore(xctx, func() bool { return near(40) != nil })
		xc()
	}
	tg := near(40)
	if tg == nil {
		say("nothing to fight")
		return
	}
	say("target %s id%d d=%.0f", tg.Name, tg.Unit.ID, tg.Dist)
	say("walk close: %v", game.MoveTo(input.Point{X: tg.Unit.X, Y: tg.Unit.Y}, input.MoveOpts{Tolerance: 9, Timeout: 15 * time.Second}))
	// curse at close range, poll the monster's states
	cur := func(id uint32) (d2.Unit, bool) {
		for _, u := range snap().Monsters {
			if u.ID == id {
				return u, true
			}
		}
		return d2.Unit{}, false
	}
	if u, ok := cur(tg.Unit.ID); ok {
		m0 := snap().Player.Mana
		die(game.CastAt("f4", input.Point{X: u.X, Y: u.Y}))
		for i := 0; i < 8; i++ {
			time.Sleep(250 * time.Millisecond)
			if u2, ok := cur(tg.Unit.ID); ok {
				say("Amplify Damage +%dms d=%.0f mana %d->%d monster states:%s", (i+1)*250, dist(snap().Player.X, snap().Player.Y, u2.X, u2.Y), m0, snap().Player.Mana, stateChain(src, u2.Addr))
			}
		}
	}
	// kill it
	for i := 0; i < 40; i++ {
		u, ok := cur(tg.Unit.ID)
		if !ok || !u.Alive() {
			break
		}
		if snap().Player.Mana < 8 {
			heal("CLASSNEC smoke: mana refill during the kill (throwaway Necci)")
		}
		die(game.CastAt("f2", input.Point{X: u.X, Y: u.Y}))
		time.Sleep(500 * time.Millisecond)
	}
	u, ok := cur(tg.Unit.ID)
	if !ok || u.Alive() {
		say("target did not die (ok=%v)", ok)
		return
	}
	say("corpse %s id%d mode %d at (%d,%d), raw:%s", name(u.Class), u.ID, u.Mode, u.X, u.Y, rawUnit(src, u))
	say("walk to corpse: %v", game.MoveTo(input.Point{X: u.X, Y: u.Y}, input.MoveOpts{Tolerance: 4, Timeout: 10 * time.Second}))
	for n := 1; n <= 3; n++ {
		heal("CLASSNEC smoke: mana for Raise Skeleton casts (throwaway Necci)")
		m0 := snap().Player.Mana
		die(game.CastAt("f5", input.Point{X: u.X, Y: u.Y}))
		time.Sleep(1500 * time.Millisecond)
		c, _ := cur(u.ID)
		say("Raise Skeleton cast %d: mana %d->%d, corpse mode %d raw:%s", n, m0, snap().Player.Mana, c.Mode, rawUnit(src, c))
		ow := census(fmt.Sprintf("after raise %d", n))
		for _, m := range ow {
			say("   owned %s id%d raw:%s", name(m.Class), m.ID, rawUnit(src, m))
		}
	}
	for _, m := range snap().Monsters {
		if name(m.Class) == "necroskeleton" {
			say("skeleton id%d owner parsed %d/%d states:%s", m.ID, m.OwnerType, m.OwnerID, stateChain(src, m.Addr))
		}
	}
	say("Hostiles with minions:")
	for _, h := range hostiles() {
		if name(h.Unit.Class) == "claygolem" || name(h.Unit.Class) == "necroskeleton" {
			say("   MINION LISTED AS HOSTILE: %s id%d", h.Name, h.Unit.ID)
		}
	}
	say("done")
}

// nsummonC: curse retries at 6 to 14 subtiles, polling the monster's states and mana; also the raw words of live hostiles.
func nsummonC(ctx context.Context, nv *nav.Navigator, game *input.Game, src mem.Source, snap func() *d2.Snapshot, heal func(string),
	hostiles func() []combat.Target, name func(int) string, say func(string, ...any)) {
	heal("CLASSNEC smoke: refill before the curse probe (throwaway Necci)")
	if snap().Player.InTown() {
		say("ExitTo(2): %v", nv.ExitTo(ctx, 2))
	}
	pick := func() *combat.Target {
		for _, h := range hostiles() {
			if h.Dist < 40 && h.Unit.Alive() && name(h.Unit.Class) != "claygolem" && name(h.Unit.Class) != "necroskeleton" {
				h := h
				return &h
			}
		}
		return nil
	}
	start := time.Now()
	for pick() == nil && time.Since(start) < 90*time.Second {
		xctx, xc := context.WithTimeout(ctx, 20*time.Second)
		nv.Explore(xctx, func() bool { return pick() != nil })
		xc()
	}
	for try := 1; try <= 3; try++ {
		tg := pick()
		if tg == nil {
			say("no target")
			return
		}
		say("try %d: %s id%d d=%.0f raw(live hostile):%s", try, tg.Name, tg.Unit.ID, tg.Dist, rawUnit(src, tg.Unit))
		game.MoveTo(input.Point{X: tg.Unit.X, Y: tg.Unit.Y}, input.MoveOpts{Tolerance: 10, Timeout: 12 * time.Second})
		time.Sleep(600 * time.Millisecond)
		var u d2.Unit
		for _, m := range snap().Monsters {
			if m.ID == tg.Unit.ID {
				u = m
			}
		}
		if u.ID == 0 {
			continue
		}
		m0 := snap().Player.Mana
		die(game.CastAt("f4", input.Point{X: u.X, Y: u.Y}))
		for i := 0; i < 8; i++ {
			time.Sleep(250 * time.Millisecond)
			for _, m := range snap().Monsters {
				if m.ID == u.ID {
					say("  +%dms mode %d d=%.0f mana %d->%d states:%s", (i+1)*250, m.Mode, dist(snap().Player.X, snap().Player.Y, m.X, m.Y), m0, snap().Player.Mana, stateChain(src, m.Addr))
				}
			}
		}
		for i := 0; i < 20; i++ {
			for _, m := range snap().Monsters {
				if m.ID == u.ID && m.Alive() {
					game.CastAt("f1", input.Point{X: m.X, Y: m.Y})
				}
			}
			time.Sleep(400 * time.Millisecond)
		}
		heal("CLASSNEC smoke: refill between curse tries (throwaway Necci)")
	}
}

// listWords dumps the first 0x40 bytes of every child state list of a unit (owner type/id of a list sit at +8/+0xC).
func listWords(src mem.Source, addr uint32) string {
	top := rdU32(src, addr+0x5C)
	var sb strings.Builder
	fmt.Fprintf(&sb, " top(+8=%x +c=%x +14=%x +34=%x)", rdU32(src, top+8), rdU32(src, top+0xC), rdU32(src, top+0x14), rdU32(src, top+0x34))
	child := rdU32(src, top+0x3c)
	for n := 0; child > 0x10000 && n < 20; n++ {
		fmt.Fprintf(&sb, " child{state %d +4=%x +8=%x +c=%x +10=%x +34=%x}", rdU32(src, child+0x14), rdU32(src, child+4), rdU32(src, child+8), rdU32(src, child+0xC), rdU32(src, child+0x10), rdU32(src, child+0x34))
		child = rdU32(src, child+0x2c)
	}
	return sb.String()
}

// nsummonD: fresh game. Curse a LIVE monster first (no minions yet), then raise minions and compare flags / list owners.
func nsummonD(ctx context.Context, f *combat.Fighter, nv *nav.Navigator, game *input.Game, src mem.Source, snap func() *d2.Snapshot, heal func(string),
	hostiles func() []combat.Target, name func(int) string, say func(string, ...any)) {
	say("EnsureHotkeys: %v", f.EnsureHotkeys(ctx))
	heal("CLASSNEC smoke: refill before probe D (throwaway Necci)")
	if snap().Player.InTown() {
		say("ExitTo(2): %v", nv.ExitTo(ctx, 2))
	}
	pick := func() *combat.Target {
		for _, h := range hostiles() {
			if h.Dist < 45 && h.Unit.Alive() {
				h := h
				return &h
			}
		}
		return nil
	}
	start := time.Now()
	for pick() == nil && time.Since(start) < 60*time.Second {
		xctx, xc := context.WithTimeout(ctx, 20*time.Second)
		nv.Explore(xctx, func() bool { return pick() != nil })
		xc()
	}
	tg := pick()
	if tg == nil {
		say("no target")
		return
	}
	p := snap().Player
	say("player list:%s", listWords(src, p.Addr))
	say("live hostile %s id%d d=%.0f flags %x list:%s", tg.Name, tg.Unit.ID, tg.Dist, rdU32(src, tg.Unit.Addr+0xC4), listWords(src, tg.Unit.Addr))
	say("walk: %v", game.MoveTo(input.Point{X: tg.Unit.X, Y: tg.Unit.Y}, input.MoveOpts{Tolerance: 11, Timeout: 15 * time.Second}))
	time.Sleep(400 * time.Millisecond)
	var u d2.Unit
	for _, m := range snap().Monsters {
		if m.ID == tg.Unit.ID {
			u = m
		}
	}
	m0 := snap().Player.Mana
	say("cast Amplify Damage at id%d (%d,%d), player (%d,%d) mode %d", u.ID, u.X, u.Y, snap().Player.X, snap().Player.Y, snap().Player.Mode)
	die(game.CastAt("f4", input.Point{X: u.X, Y: u.Y}))
	for i := 0; i < 8; i++ {
		time.Sleep(250 * time.Millisecond)
		for _, m := range snap().Monsters {
			if m.ID == u.ID {
				say("  +%dms mode %d mana %d->%d states:%s", (i+1)*250, m.Mode, m0, snap().Player.Mana, stateChain(src, m.Addr))
			}
		}
	}
	// golem and kill
	say("SetRight golem skipped (popup is slow); killing with Bone Spear")
	for i := 0; i < 40; i++ {
		var cur *d2.Unit
		ms := snap().Monsters
		for j := range ms {
			if ms[j].ID == u.ID {
				cur = &ms[j]
			}
		}
		if cur == nil || !cur.Alive() {
			break
		}
		if snap().Player.Mana < 8 {
			heal("CLASSNEC smoke: mana refill during the kill (throwaway Necci)")
		}
		game.CastAt("f2", input.Point{X: cur.X, Y: cur.Y})
		time.Sleep(500 * time.Millisecond)
	}
	for _, m := range snap().Monsters {
		if m.ID == u.ID {
			say("corpse mode %d flags %x list:%s", m.Mode, rdU32(src, m.Addr+0xC4), listWords(src, m.Addr))
			game.MoveTo(input.Point{X: m.X, Y: m.Y}, input.MoveOpts{Tolerance: 4, Timeout: 10 * time.Second})
			heal("CLASSNEC smoke: mana for Raise Skeleton (throwaway Necci)")
			game.CastAt("f5", input.Point{X: m.X, Y: m.Y})
		}
	}
	time.Sleep(1500 * time.Millisecond)
	for _, m := range snap().Monsters {
		d := dist(snap().Player.X, snap().Player.Y, m.X, m.Y)
		if d < 30 {
			say("unit %s id%d mode %d flags %x owner94/98 %d/%d list:%s", name(m.Class), m.ID, m.Mode, rdU32(src, m.Addr+0xC4), m.OwnerType, m.OwnerID, listWords(src, m.Addr))
		}
	}
}

// nsummonE: death with minions. Raise one skeleton, then life := 2 with no potions drunk, wait for the death, respawn, look around.
func nsummonE(ctx context.Context, f *combat.Fighter, nv *nav.Navigator, game *input.Game, src mem.Source, snap func() *d2.Snapshot, heal func(string),
	hostiles func() []combat.Target, name func(int) string, say func(string, ...any), census func(string) []d2.Unit) {
	say("EnsureHotkeys: %v", f.EnsureHotkeys(ctx))
	heal("CLASSNEC smoke: refill before the death probe (throwaway Necci)")
	if snap().Player.InTown() {
		say("ExitTo(2): %v", nv.ExitTo(ctx, 2))
	}
	pick := func() *combat.Target {
		for _, h := range hostiles() {
			if h.Dist < 80 && h.Unit.Alive() && name(h.Unit.Class) != "necroskeleton" && name(h.Unit.Class) != "claygolem" {
				h := h
				return &h
			}
		}
		return nil
	}
	start := time.Now()
	for pick() == nil && time.Since(start) < 90*time.Second {
		xctx, xc := context.WithTimeout(ctx, 20*time.Second)
		nv.Explore(xctx, func() bool { return pick() != nil })
		xc()
	}
	tg := pick()
	if tg == nil {
		say("no target")
		return
	}
	game.MoveTo(input.Point{X: tg.Unit.X, Y: tg.Unit.Y}, input.MoveOpts{Tolerance: 6, Timeout: 15 * time.Second})
	_, err := cheat.DefaultEnv().Do(ctx, cheat.Request{Kind: "setstat", Actor: "classnec", Reason: "CLASSNEC smoke: life 2 for the death-loop probe (throwaway Necci), no potions are drunk", Params: map[string]any{"stat": int64(6), "value": int64(512)}})
	say("life to 2: %v; life now %d", err, snap().Player.Life)
	t1 := time.Now()
	var diedAt time.Time
	for time.Since(t1) < 60*time.Second {
		p := snap().Player
		if combat.IsDead(p) {
			diedAt = time.Now()
			say("DEAD after %.1fs: mode %d life %d area %d", time.Since(t1).Seconds(), p.Mode, p.Life, p.Area)
			break
		}
		if t := pick(); t != nil {
			game.MoveTo(input.Point{X: t.Unit.X, Y: t.Unit.Y}, input.MoveOpts{Tolerance: 1, Timeout: 3 * time.Second})
		}
		time.Sleep(200 * time.Millisecond)
	}
	if diedAt.IsZero() {
		say("no death in 60 s (life %d mode %d)", snap().Player.Life, snap().Player.Mode)
		return
	}
	census("at death")
	say("Respawn: %v", f.Respawn(ctx, 20*time.Second))
	time.Sleep(1500 * time.Millisecond)
	p := snap().Player
	say("after respawn: area %d life %d/%d mana %d/%d left %d right %d mode %d", p.Area, p.Life, p.MaxLife, p.Mana, p.MaxMana, p.LeftSkill, p.RightSkill, p.Mode)
	say("player states:%s", stateChain(src, p.Addr))
	for _, sk := range p.Skills {
		if sk.ID >= 66 {
			say("  skill %d lvl %d+%d", sk.ID, sk.Level, sk.Bonus)
		}
	}
	census("after respawn")
}

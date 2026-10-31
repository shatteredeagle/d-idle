package cheat

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/mem"
)

// Unit selector for stat writes. In single player the process holds two copies of the player:
// the client's (the PlayerUnit global at RVA 0x3A6A70, what internal/d2 reads) and the embedded game
// server's authoritative one, found by signature scan (serverPlayer). VERIFIED LIVE, see docs/cheats.md.
const (
	UnitClient = "client"
	UnitServer = "server"
	UnitBoth   = "both"
	VecBase    = "base"   // StatList +0x24 vector (the unit's own stats)
	VecMerged  = "merged" // StatList +0x48 vector (what GetUnitStat reads for players)
	VecBoth    = "both"
)

// Stat list layout (docs/mem.md Q4).
const (
	uoType, uoClass, uoID, uoData, uoStats = 0x00, 0x04, 0x0C, 0x14, 0x5C
	slFlags                                = 0x10
	slVec, slCount                         = 0x24, 0x28
	slVecEx, slCountEx                     = 0x48, 0x4C
	slFlagEx                               = 0x80000000
)

// Stat ids used here (same as internal/d2).
const (
	statStrength, statEnergy, statDexterity, statVitality = 0, 1, 2, 3
	statPoints, statSkillPts                              = 4, 5
	statLife, statMaxLife, statMana, statMaxMana          = 6, 7, 8, 9
	statStamina, statMaxStamina                           = 10, 11
	statLevel, statExp, statGold, statStashGold           = 12, 13, 14, 15
)

type game struct {
	w      mem.Writer
	r      *d2.Reader
	client uint32
	server uint32 // 0 when not found
	name   string
	class  d2.Class
	id     uint32
}

// attach opens the game and applies the policy: in game, offline, allowlisted character.
func (e *Env) attach(ent *Entry) (*game, func(), error) {
	w, closeFn, err := e.Open()
	if err != nil {
		return nil, nil, err
	}
	fail := func(err error) (*game, func(), error) { closeFn(); return nil, nil, err }
	r := d2.New(w)
	if st := r.State(); st != d2.StateInGame {
		return fail(fmt.Errorf("cheat: game is not in a game (state %v)", st))
	}
	pl, err := r.Player()
	if err != nil {
		return fail(err)
	}
	ent.Character = pl.Name
	if err := CheckCharacter(pl.Name); err != nil {
		return fail(err)
	}
	if err := CheckOffline(w); err != nil {
		return fail(err)
	}
	g := &game{w: w, r: r, client: pl.Addr, name: pl.Name, class: pl.Class, id: pl.ID}
	g.server, _ = serverPlayer(w, g)
	return g, closeFn, nil
}

// serverPlayer finds the game server's copy of the local player: a second UnitAny with type 0, the same
// class and id, a stat list with the extended flag and the same name, at another address than the client's.
// The server unit lives on the heap (no static pointer to it was found; see docs/cheats.md), so this scans
// the private read-write mappings. Fake sources have no process and return 0.
func serverPlayer(src mem.Source, g *game) (uint32, error) {
	p, ok := src.(*mem.Proc)
	if !ok {
		return 0, errors.New("cheat: no process to scan")
	}
	f, err := os.Open(fmt.Sprintf("/proc/%d/maps", p.PID))
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) < 2 || !strings.HasPrefix(fs[1], "rw") {
			continue
		}
		var lo, hi uint64
		fmt.Sscanf(fs[0], "%x-%x", &lo, &hi)
		if lo < 0x10000 || hi > 0x80000000 || hi-lo > 0x10000000 {
			continue
		}
		buf := make([]byte, hi-lo)
		if p.ReadAt(buf, uint32(lo)) != nil {
			continue
		}
		for o := 0; o+0x70 <= len(buf); o += 4 {
			if binary.LittleEndian.Uint32(buf[o+uoID:]) != g.id || binary.LittleEndian.Uint32(buf[o+uoClass:]) != uint32(g.class) ||
				binary.LittleEndian.Uint32(buf[o+uoType:]) != 0 {
				continue
			}
			a := uint32(lo) + uint32(o)
			if a == g.client || !g.isPlayerUnit(a) {
				continue
			}
			return a, nil
		}
	}
	return 0, errors.New("cheat: server-side player not found")
}

func (g *game) isPlayerUnit(a uint32) bool {
	sl, err := mem.U32(g.w, a+uoStats)
	if err != nil || sl < 0x10000 {
		return false
	}
	if fl, err := mem.U32(g.w, sl+slFlags); err != nil || fl&slFlagEx == 0 {
		return false
	}
	pd, err := mem.U32(g.w, a+uoData)
	if err != nil || pd < 0x10000 {
		return false
	}
	n, err := cstr(g.w, pd, 16)
	return err == nil && n == g.name
}

func (g *game) unit(which string) (uint32, error) {
	switch which {
	case UnitClient:
		return g.client, nil
	case UnitServer:
		if g.server == 0 {
			return 0, errors.New("cheat: server-side player not available")
		}
		return g.server, nil
	}
	return 0, fmt.Errorf("cheat: unit %q (want client|server)", which)
}

// slot returns the address of the value of stat (id, sub) in the unit's vector, or 0 when absent.
func (g *game) slot(unit uint32, vec string, id, sub int) (uint32, error) {
	sl, err := mem.U32(g.w, unit+uoStats)
	if err != nil {
		return 0, err
	}
	vAt, cAt := uint32(slVec), uint32(slCount)
	if vec == VecMerged {
		vAt, cAt = slVecEx, slCountEx
	}
	base, err := mem.U32(g.w, sl+vAt)
	if err != nil {
		return 0, err
	}
	cnt, err := mem.U16(g.w, sl+cAt)
	if err != nil {
		return 0, err
	}
	if cnt > 512 {
		return 0, fmt.Errorf("cheat: implausible stat count %d", cnt)
	}
	b := make([]byte, int(cnt)*8)
	if err := g.w.ReadAt(b, base); err != nil {
		return 0, err
	}
	for i := 0; i < int(cnt); i++ {
		if int(binary.LittleEndian.Uint16(b[i*8+2:])) == id && int(binary.LittleEndian.Uint16(b[i*8:])) == sub {
			return base + uint32(i*8) + 4, nil
		}
	}
	return 0, nil
}

// get reads a raw stat value; ok is false when the stat is absent from that vector.
func (g *game) get(which, vec string, id int) (int32, bool) {
	u, err := g.unit(which)
	if err != nil {
		return 0, false
	}
	a, err := g.slot(u, vec, id, 0)
	if err != nil || a == 0 {
		return 0, false
	}
	v, err := mem.I32(g.w, a)
	return v, err == nil
}

var errAbsent = errors.New("stat not present in that stat vector")

// set writes a raw stat value into the selected unit(s) and vector(s). Absent stats are an error
// (never inserted: the vectors are heap arrays whose capacity belongs to the game).
func (g *game) set(which, vec string, id int, val int32) error {
	units := []string{which}
	if which == UnitBoth {
		units = []string{UnitClient, UnitServer}
	}
	vecs := []string{vec}
	if vec == VecBoth {
		vecs = []string{VecBase, VecMerged}
	}
	wrote := 0
	var errs []error
	for _, un := range units {
		u, err := g.unit(un)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, v := range vecs {
			a, err := g.slot(u, v, id, 0)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if a == 0 {
				errs = append(errs, fmt.Errorf("%s/%s stat %d: %w", un, v, id, errAbsent))
				continue
			}
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], uint32(val))
			if err := g.w.WriteAt(b[:], a); err != nil {
				errs = append(errs, err)
				continue
			}
			wrote++
		}
	}
	if wrote == 0 {
		return errors.Join(errs...)
	}
	return nil
}

// snapshot reads the interesting stats of both units for the ledger's before/after.
func (g *game) snapshot() map[string]any {
	m := map[string]any{}
	for _, un := range []string{UnitClient, UnitServer} {
		for _, st := range []struct {
			n  string
			id int
		}{{"life", statLife}, {"maxlife", statMaxLife}, {"mana", statMana}, {"maxmana", statMaxMana}, {"stamina", statStamina},
			{"level", statLevel}, {"exp", statExp}, {"gold", statGold}, {"stashgold", statStashGold},
			{"str", statStrength}, {"dex", statDexterity}, {"vit", statVitality}, {"ene", statEnergy},
			{"statpts", statPoints}, {"skillpts", statSkillPts}, {"mf", 80}, {"speed", 67}, {"goldfind", 79}} {
			if v, ok := g.get(un, VecMerged, st.id); ok {
				m[un+"."+st.n] = v
			}
		}
	}
	return m
}

func (e *Env) doMemory(ctx context.Context, req Request, ent *Entry) (string, error) {
	g, closeFn, err := e.attach(ent)
	if err != nil {
		return "", err
	}
	defer closeFn()
	ent.Before = g.snapshot()
	if on, _ := req.Params["on"].(bool); req.Kind == KindGod && on {
		// long-running: record the start now, so a killed process still leaves a trace
		start := *ent
		start.Params = map[string]any{"phase": "start", "for": req.Params["for"], "buffer": req.Params["buffer"]}
		start.Success = true
		if err := e.Ledger.Append(start); err != nil {
			return "", err
		}
	}
	msg, err := e.applyMemory(ctx, g, req)
	ent.After = g.snapshot()
	return msg, err
}

// Inspect reads the key stats of the client and server player copies without writing anything (no
// ledger entry: reads are free). Used by `d2cheat show` and to verify a cheat took effect.
func (e *Env) Inspect() (name string, stats map[string]any, err error) {
	w, closeFn, err := e.Open()
	if err != nil {
		return "", nil, err
	}
	defer closeFn()
	r := d2.New(w)
	pl, err := r.Player()
	if err != nil {
		return "", nil, err
	}
	g := &game{w: w, r: r, client: pl.Addr, name: pl.Name, class: pl.Class, id: pl.ID}
	g.server, _ = serverPlayer(w, g)
	return pl.Name, g.snapshot(), nil
}

// insertStat adds a stat that is absent from a unit's vector, using a spare slot (count < capacity;
// the game owns the allocation, so a full vector is an error). The vector stays sorted by id, which the
// game's lookup relies on. Experimental: see docs/cheats.md for what was verified.
func (g *game) insertStat(unit uint32, vec string, id int, val int32) error {
	sl, err := mem.U32(g.w, unit+uoStats)
	if err != nil {
		return err
	}
	vAt, cAt := uint32(slVec), uint32(slCount)
	if vec == VecMerged {
		vAt, cAt = slVecEx, slCountEx
	}
	base, err := mem.U32(g.w, sl+vAt)
	if err != nil {
		return err
	}
	cnt, err := mem.U16(g.w, sl+cAt)
	if err != nil {
		return err
	}
	capacity, err := mem.U16(g.w, sl+cAt+2)
	if err != nil {
		return err
	}
	if cnt >= capacity {
		return fmt.Errorf("cheat: %s vector full (%d/%d), cannot add stat %d", vec, cnt, capacity, id)
	}
	b := make([]byte, int(cnt)*8)
	if err := g.w.ReadAt(b, base); err != nil {
		return err
	}
	idx := int(cnt)
	for i := 0; i < int(cnt); i++ {
		eid := int(binary.LittleEndian.Uint16(b[i*8+2:]))
		if eid == id {
			return fmt.Errorf("cheat: stat %d already present", id)
		}
		if eid > id {
			idx = i
			break
		}
	}
	var ent [8]byte
	binary.LittleEndian.PutUint16(ent[2:], uint16(id))
	binary.LittleEndian.PutUint32(ent[4:], uint32(val))
	if idx < int(cnt) {
		if err := g.w.WriteAt(b[idx*8:], base+uint32(idx+1)*8); err != nil {
			return err
		}
	}
	if err := g.w.WriteAt(ent[:], base+uint32(idx)*8); err != nil {
		return err
	}
	var c [2]byte
	binary.LittleEndian.PutUint16(c[:], cnt+1)
	return g.w.WriteAt(c[:], sl+cAt)
}

package d2dcombat

import (
	"context"
	"strconv"
	"sync"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/input"
)

// Input is d2dinput.Controller with the two combat-specific mappings the original keyboard flow needs:
//   - Key("1".."4") (combat's potion hotkeys) becomes "cmd belt <col>", because the channel's key verb does not know the
//     belt hotkeys (U) and the belt verb is the documented way;
//   - CastAt looks up the monster standing at the target and sends its unit id, so a spell hits the unit even after it
//     moved a cell (V verb with unit id from the spike; the lookup costs one "monsters" round trip per cast).
//
// It counts what it sent for the smoke tool.
type Input struct {
	*d2dinput.Controller
	ctx context.Context

	mu    sync.Mutex
	stats Stats
}

// Stats counts input calls by kind.
type Stats struct {
	Casts   map[int]int // skill id -> casts
	Potions int
	Clicks  int // left/right clicks (melee and right-skill fallback)
}

// Stats returns a copy of the counters.
func (i *Input) Stats() Stats {
	i.mu.Lock()
	defer i.mu.Unlock()
	s := i.stats
	s.Casts = map[int]int{}
	for k, v := range i.stats.Casts {
		s.Casts[k] = v
	}
	return s
}

// Key maps the belt keys to the belt verb and passes everything else on.
func (i *Input) Key(name string) error {
	if len(name) == 1 && name[0] >= '1' && name[0] <= '4' {
		col, _ := strconv.Atoi(name)
		err := i.C.Drink(i.ctx, col-1)
		if err == nil {
			i.mu.Lock()
			i.stats.Potions++
			i.mu.Unlock()
		}
		return err
	}
	return i.Controller.Key(name)
}

// CastAt casts the hotkey's skill at w, at the living monster within one cell of w when there is one.
func (i *Input) CastAt(hotkey string, w input.Point) error {
	id, ok := i.Skills[hotkey]
	if !ok {
		return i.Controller.CastAt(hotkey, w)
	}
	var unit uint32
	x, y := (float64(w.X)+0.5)/5, (float64(w.Y)+0.5)/5
	if ml, err := i.C.Monsters(i.ctx); err == nil {
		if m, ok := d2dctl.NearestAlive(ml.Items, x, y, 1); ok {
			unit = uint32(m.Unit)
		}
	}
	err := i.Controller.Cast(id, w, unit)
	if err == nil {
		i.mu.Lock()
		if i.stats.Casts == nil {
			i.stats.Casts = map[int]int{}
		}
		i.stats.Casts[id]++
		i.mu.Unlock()
	}
	return err
}

// Click counts clicks and passes them on.
func (i *Input) Click(b input.Button, x, y int, mods ...input.Mod) error {
	i.mu.Lock()
	i.stats.Clicks++
	i.mu.Unlock()
	return i.Controller.Click(b, x, y, mods...)
}

// Swing is a left-button attack ("cmd skill <id> x y <unit> left"): the monster standing at w gets the blow by unit
// id, so the hero walks up and hits it without a pointer hover (a bare click on d2d acts on the unit under the pointer
// of the previous frame, which under lockstep is never the target: palmelee, docs/palmelee.md). Without a monster
// near w it clicks the point like before. skill 0 is the basic attack.
func (i *Input) Swing(skill int, w input.Point) error {
	x, y := (float64(w.X)+0.5)/5, (float64(w.Y)+0.5)/5
	ml, err := i.C.Monsters(i.ctx)
	if err != nil {
		return err
	}
	m, ok := d2dctl.NearestAlive(ml.Items, x, y, 1.5)
	if !ok {
		return i.Controller.ClickWorld(w)
	}
	err = i.C.CmdSkill(i.ctx, skill, m.CellX, m.CellY, d2dctl.SkillOpts{Unit: m.Unit, Left: true})
	if err == nil {
		i.mu.Lock()
		i.stats.Clicks++
		if i.stats.Casts == nil {
			i.stats.Casts = map[int]int{}
		}
		i.stats.Casts[skill]++
		i.mu.Unlock()
	}
	return err
}

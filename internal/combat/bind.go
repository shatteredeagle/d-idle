package combat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/data/mpq"
	"d2lod-bot/internal/input"
)

// Hotkey binding goes through the right-skill popup: click the right skill button, hover the skill icon, press the
// F-key. VERIFIED LIVE (2026-10-28): hover + F8 in the popup bound Fire Bolt (checked by picking another skill from
// the popup and pressing F8: the right skill came back as 36). DEAD END, VERIFIED LIVE: hovering the icon in the skill
// tree (T) and pressing F5-F8 bound nothing (key never selected the skill); the tooltip there says "You have not
// learned this skill yet" for an item-granted skill, which may be why.

// PopupGeometry places the icons of the right-skill popup on the 800x600 client. Measured from game-window
// screenshots: the grid is anchored at the right-skill button and grows up and to the left; rows are the skills.txt
// skilldesc ListRow values that the character has skills in (empty rows are skipped), row 0 at the bottom.
type PopupGeometry struct {
	BtnX, BtnY int // right skill button
	Col0X      int // x of the rightmost icon column
	ColDX      int // column pitch
	Row0Y      int // y of the bottom row
	RowDY      int // row pitch
}

// Popup is the geometry Bind uses.
var Popup = PopupGeometry{BtnX: 660, BtnY: 578, Col0X: 698, ColDX: 49, Row0Y: 487, RowDY: 47}

// Want is one skill-to-key binding the character should have.
type Want struct {
	Skill SkillSpec
	Key   string
}

// WantedBindings lists the bindings the build's layout asks for, for the skills the character knows, ordered by key.
func WantedBindings(p *d2.Player, cat Catalog, b Build) []Want {
	return wantedFor(p, cat, SelectProfile(p, cat, b).Layout)
}

func wantedFor(p *d2.Player, cat Catalog, layout map[string]string) []Want {
	known := KnownSkills(p, cat, nil)
	var out []Want
	for name, key := range layout {
		if sp, ok := known[name]; ok {
			out = append(out, Want{Skill: sp, Key: key})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// LoadListRows reads skilldesc.txt ListRow per skill id from the install's MPQs (the data cache does not keep that
// column). Install is the game directory. The result is for SetListRows.
func LoadListRows(install string, db *data.DB) (map[int]int, error) {
	var raw []byte
	for _, name := range []string{"Patch_D2.mpq", "d2exp.mpq", "d2data.mpq"} {
		path := filepath.Join(install, name)
		if _, err := os.Stat(path); err != nil {
			entries, _ := os.ReadDir(install)
			for _, e := range entries {
				if strings.EqualFold(e.Name(), name) {
					path = filepath.Join(install, e.Name())
				}
			}
		}
		a, err := mpq.Open(path)
		if err != nil {
			continue
		}
		if b, err := a.ReadFile(`data\global\excel\skilldesc.txt`); err == nil {
			raw = b
			break
		}
	}
	if raw == nil {
		return nil, fmt.Errorf("combat: skilldesc.txt not found in %s", install)
	}
	t := data.ParseTable(raw)
	byDesc := map[string]int{}
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		byDesc[strings.ToLower(r.Str("skilldesc"))] = r.Int("ListRow")
	}
	out := map[int]int{}
	for _, s := range db.Skills {
		if row, ok := byDesc[strings.ToLower(s.DescKey)]; ok {
			out[s.ID] = row
		}
	}
	return out, nil
}

// SetListRows installs the skill id to popup ListRow mapping that EnsureHotkeys needs to find icons.
func (f *Fighter) SetListRows(m map[int]int) { f.listRows = m }

// popupCells returns, for the character's skills, the screen cell of every popup slot in the row of skill id, as
// candidates in right-to-left order, plus the count of skills in that row.
func (f *Fighter) popupCells(p *d2.Player, id int) []input.Screen {
	row, ok := f.listRows[id]
	if !ok || row < 0 {
		return nil
	}
	rows := map[int]int{} // ListRow -> skills in it
	for _, sk := range p.Skills {
		if r, ok := f.listRows[sk.ID]; ok && r >= 0 {
			rows[r]++
		}
	}
	var order []int
	for r := range rows {
		order = append(order, r)
	}
	sort.Ints(order)
	idx := sort.SearchInts(order, row)
	g := Popup
	var out []input.Screen
	for c := 0; c < rows[row]; c++ {
		out = append(out, input.Screen{X: g.Col0X - c*g.ColDX, Y: g.Row0Y - idx*g.RowDY})
	}
	return out
}

func (f *Fighter) openPopup(ctx context.Context) error {
	if err := f.e.In.Click(input.Left, Popup.BtnX, Popup.BtnY); err != nil {
		return err
	}
	return f.sleep(ctx, 400*time.Millisecond)
}

func (f *Fighter) rightSkill(ctx context.Context) (int, error) {
	if err := f.sleep(ctx, 200*time.Millisecond); err != nil {
		return 0, err
	}
	s, err := f.read(ctx)
	if err != nil {
		return 0, err
	}
	return s.Player.RightSkill, nil
}

// selectOther makes sure the right-hand skill is not id, by picking the first popup cell that gives another skill.
func (f *Fighter) selectOther(ctx context.Context, p *d2.Player, id int) error {
	cur, err := f.rightSkill(ctx)
	if err != nil || cur != id {
		return err
	}
	for _, sk := range p.Skills {
		if sk.ID == id {
			continue
		}
		for _, c := range f.popupCells(p, sk.ID) {
			if err := f.openPopup(ctx); err != nil {
				return err
			}
			if err := f.e.In.Click(input.Left, c.X, c.Y); err != nil {
				return err
			}
			if now, err := f.rightSkill(ctx); err != nil || now != id {
				return err
			}
		}
	}
	return fmt.Errorf("combat: cannot select another right skill")
}

// checkKey reports whether pressing key selects skill id. The right skill is first moved off id, otherwise a key that
// is not bound at all would look right.
func (f *Fighter) checkKey(ctx context.Context, p *d2.Player, id int, key string) (bool, error) {
	if err := f.selectOther(ctx, p, id); err != nil {
		return false, err
	}
	if err := f.e.In.Key(key); err != nil {
		return false, err
	}
	cur, err := f.rightSkill(ctx)
	return cur == id, err
}

// EnsureHotkeys makes sure every skill the layout wants is on its F-key. A key that already selects the skill is
// left alone; the rest are bound through the right-skill popup and verified again. Verified bindings go into the
// Fighter's hotkey map. Needs SetListRows. Errors for single skills are collected, the others still bound.
func (f *Fighter) EnsureHotkeys(ctx context.Context) error {
	snap, err := f.read(ctx)
	if err != nil {
		return err
	}
	p := snap.Player
	if len(f.listRows) == 0 {
		f.info("combat bind", "class", p.Class.String(), "level", p.Level, "result", "popup rows not loaded")
		return fmt.Errorf("%w: popup rows not loaded (SetListRows)", ErrBindFailed)
	}
	var bad, bound []string
	wanted := wantedFor(p, f.cat, f.profileFor(p).Layout)
	// One summary line per call and class so a first run shows what was wanted and what ended up on a key (BUILDV2).
	defer func() {
		f.info("combat bind", "class", p.Class.String(), "level", p.Level, "wanted", len(wanted), "bound", strings.Join(bound, ","), "failed", len(bad))
	}()
	for _, w := range wanted {
		ok, err := f.checkKey(ctx, p, w.Skill.ID, w.Key)
		if err != nil {
			return err
		}
		if !ok {
			if err := f.bind(ctx, p, w); err != nil {
				bad = append(bad, fmt.Sprintf("%s: %v", w.Skill.Name, err))
				continue
			}
			if ok, err = f.checkKey(ctx, p, w.Skill.ID, w.Key); err != nil {
				return err
			}
			if !ok {
				bad = append(bad, w.Skill.Name+": key does not select the skill after binding")
				continue
			}
		}
		f.keys[w.Skill.ID] = w.Key
		bound = append(bound, w.Skill.Name+"="+w.Key)
		f.info("hotkey ok", "skill", w.Skill.Name, "key", w.Key)
	}
	if len(bad) > 0 {
		return fmt.Errorf("%w: %s", ErrBindFailed, strings.Join(bad, "; "))
	}
	return nil
}

// bind finds the skill's icon by picking each cell of its popup row until the right skill reads back as the wanted
// one, then opens the popup again, hovers that cell and presses the key.
func (f *Fighter) bind(ctx context.Context, p *d2.Player, w Want) error {
	in := f.e.In
	cells := f.popupCells(p, w.Skill.ID)
	if len(cells) == 0 {
		return fmt.Errorf("no popup row for skill %d", w.Skill.ID)
	}
	if err := f.selectOther(ctx, p, w.Skill.ID); err != nil {
		return err
	}
	for _, c := range cells {
		if err := f.openPopup(ctx); err != nil {
			return err
		}
		if err := in.Click(input.Left, c.X, c.Y); err != nil {
			return err
		}
		cur, err := f.rightSkill(ctx)
		if err != nil {
			return err
		}
		if cur != w.Skill.ID {
			continue
		}
		if err := f.openPopup(ctx); err != nil {
			return err
		}
		if err := in.MoveMouse(c.X, c.Y); err != nil {
			return err
		}
		if err := f.sleep(ctx, 300*time.Millisecond); err != nil {
			return err
		}
		if err := in.Key(w.Key); err != nil {
			return err
		}
		if err := f.sleep(ctx, 200*time.Millisecond); err != nil {
			return err
		}
		return in.Click(input.Left, c.X, c.Y) // picking the icon closes the popup (Esc would open the game menu)
	}
	return fmt.Errorf("icon not found in %d popup cells", len(cells))
}

// BindSkill binds one skill by name to a key (verified, recorded). EnsureHotkeys is the normal entry point.
func (f *Fighter) BindSkill(ctx context.Context, name, key string) error {
	snap, err := f.read(ctx)
	if err != nil {
		return err
	}
	sp, ok := KnownSkills(snap.Player, f.cat, nil)[name]
	if !ok {
		return fmt.Errorf("combat: skill %q not known", name)
	}
	w := Want{Skill: sp, Key: key}
	if err := f.bind(ctx, snap.Player, w); err != nil {
		return err
	}
	ok, err = f.checkKey(ctx, snap.Player, sp.ID, key)
	if err != nil {
		return err
	}
	if !ok {
		return ErrBindFailed
	}
	f.keys[sp.ID] = key
	return nil
}

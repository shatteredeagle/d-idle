package cheat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"d2lod-bot/internal/cheat/d2s"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/mem"
)

// Rules is the game data a consistent level-up needs: experience.txt and charstats.txt per class.
type Rules interface {
	ExpToLevel(level int) int64
	// Gains returns the charstats gains in quarter points for class id 0..6.
	Gains(class int) Gains
}

// Gains are per-level and per-point gains in quarter points (4 = 1 per level), straight from charstats.txt.
type Gains struct {
	LifePerLevel, ManaPerLevel, StaminaPerLevel int
	LifePerVit, StaminaPerVit, ManaPerEnergy    int
	StatPerLevel                                int
}

type dbRules struct{ db *data.DB }

func (r dbRules) ExpToLevel(level int) int64 { return r.db.ExpToLevel(level) }
func (r dbRules) Gains(class int) Gains {
	for _, c := range r.db.Classes {
		if c.ID == class {
			return Gains{c.LifePerLevel, c.ManaPerLevel, c.StaminaPerLevel, c.LifePerVit, c.StaminaPerVit, c.ManaPerEnergy, c.StatPerLevel}
		}
	}
	return Gains{}
}

func defaultRules() (Rules, error) {
	l := layout()
	root := os.Getenv("D2_ROOT")
	if root == "" {
		root = l.GameRoot("i0")
	}
	install := filepath.Join(root, "proton", "pfx", "drive_c", "Program Files (x86)", "Diablo II")
	db, err := data.OpenCached(install, l, filepath.Join(l.HomeDir(), ".cache", "d2lod-cheat-data"))
	if err != nil {
		return nil, fmt.Errorf("cheat: game tables: %w", err)
	}
	return dbRules{db}, nil
}

// gameInGame: a character is loaded when the instance's Game.exe is in a game (not at a menu).
func gameInGame(inst string) (bool, error) {
	if len(mem.FindGamesFor(inst)) == 0 {
		return false, nil
	}
	p, err := mem.OpenInstance(inst)
	if err != nil {
		return true, err // cannot tell: assume in game
	}
	defer p.Close()
	return d2.New(p).State() != d2.StateMenu, nil
}

// SaveSnapshot summarizes a save for the ledger.
func saveSnapshot(f *d2s.File) map[string]any {
	m := map[string]any{"level": f.Level(), "progression": f.Progression()}
	for _, st := range f.Stats {
		m[statNames[st.ID]] = st.Value
	}
	for d := 0; d < 3; d++ {
		m[fmt.Sprintf("difficulty%d", d)] = f.Difficulty(d)
		m[fmt.Sprintf("waypoints%d", d)] = f.WaypointMask(d)
		m[fmt.Sprintf("quests%d", d)] = questSummary(f, d)
	}
	return m
}

var statNames = [16]string{"str", "ene", "dex", "vit", "statpts", "skillpts", "life", "maxlife", "mana", "maxmana", "stamina", "maxstamina", "level", "exp", "gold", "stashgold"}

// resolveSave finds the .d2s: params "file" (path or bare name) or "character".
func (e *Env) resolveSave(req Request) (string, error) {
	name := req.Str("file")
	if name == "" {
		name = req.Str("character")
	}
	if name == "" {
		return "", errors.New("cheat: saveedit needs file= or character=")
	}
	p := name
	if !strings.ContainsAny(name, `/\`) {
		if !strings.HasSuffix(strings.ToLower(name), ".d2s") {
			name += ".d2s"
		}
		p = filepath.Join(e.SaveDir, name)
	}
	return p, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// doSave edits a .d2s: allowlisted character, instance not in a game, backup first, atomic replace.
func (e *Env) doSave(ctx context.Context, req Request, ent *Entry) (string, error) {
	path, err := e.resolveSave(req)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	f, err := d2s.Parse(b)
	if err != nil {
		return "", err
	}
	ent.Character = f.Name()
	if err := CheckCharacter(f.Name()); err != nil {
		return "", err
	}
	if !strings.EqualFold(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), f.Name()) {
		return "", fmt.Errorf("cheat: file %s does not hold character %s", filepath.Base(path), f.Name())
	}
	if in, err := e.InGame(); err != nil || in {
		return "", fmt.Errorf("%w (%v)", ErrGameRunning, err)
	}
	ent.Before = saveSnapshot(f)
	msg, err := e.applySave(f, req)
	if err != nil {
		return "", err
	}
	// Backup only once we know the edit is valid, but before touching the file.
	if err := os.MkdirAll(e.BackupDir, 0o755); err != nil {
		return "", fmt.Errorf("cheat: backup dir: %w", err)
	}
	// Two edits in one second must not collide (a chained d2cheat script): -2, -3, ... keeps every backup.
	stamp := fmt.Sprintf("%s-%s", f.Name(), e.now().Format("20060102T150405Z"))
	ent.Backup = filepath.Join(e.BackupDir, stamp+".d2s")
	for n := 2; n < 100; n++ {
		if _, err := os.Stat(ent.Backup); err != nil {
			break
		}
		ent.Backup = filepath.Join(e.BackupDir, fmt.Sprintf("%s-%d.d2s", stamp, n))
	}
	if err := copyFile(path, ent.Backup); err != nil {
		return "", fmt.Errorf("cheat: backup failed, save not edited: %w", err)
	}
	out := f.Bytes()
	if _, err := d2s.Parse(out); err != nil {
		return "", fmt.Errorf("cheat: edited save does not parse back: %w", err)
	}
	tmp := path + ".cheat.tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	ent.After = saveSnapshot(f)
	return msg, nil
}

// applySave applies the parameters to f. Level raises are done as the game would: experience for the
// level, +5 stat points and +1 skill point per level gained, max life/mana/stamina grown by the class gains,
// current values refilled. Explicit params override the derived ones.
func (e *Env) applySave(f *d2s.File, req Request) (string, error) {
	var did []string
	get := func(k string) (int64, bool) { return req.Int(k) }
	if lv, ok := get("level"); ok {
		rules, err := e.Rules()
		if err != nil {
			return "", err
		}
		if err := levelTo(f, int(lv), rules); err != nil {
			return "", err
		}
		did = append(did, fmt.Sprintf("level %d", lv))
	}
	for _, p := range []struct {
		key string
		id  int
	}{{"exp", d2s.StatExp}, {"gold", d2s.StatGold}, {"stashgold", d2s.StatStashGold}, {"statpoints", d2s.StatPoints},
		{"skillpoints", d2s.StatSkillPoints}, {"str", d2s.StatStrength}, {"dex", d2s.StatDexterity}, {"vit", d2s.StatVitality},
		{"ene", d2s.StatEnergy}, {"maxlife", d2s.StatMaxLife}, {"maxmana", d2s.StatMaxMana}, {"maxstamina", d2s.StatMaxStamina}} {
		v, ok := get(p.key)
		if !ok {
			continue
		}
		if v < 0 {
			return "", fmt.Errorf("cheat: %s must not be negative", p.key)
		}
		raw := uint32(v)
		if p.id >= d2s.StatLife && p.id <= d2s.StatMaxStamina {
			raw <<= 8
			cur := p.id - 1
			if err := f.SetStat(cur, raw); err != nil {
				return "", err
			}
		}
		if err := f.SetStat(p.id, raw); err != nil {
			return "", err
		}
		did = append(did, fmt.Sprintf("%s %d", p.key, v))
	}
	if v, ok := get("waypoints"); ok {
		for d := 0; d < 3; d++ {
			if v&(1<<d) != 0 {
				if err := f.SetWaypoints(d, d2s.AllWaypoints); err != nil {
					return "", err
				}
			}
		}
		did = append(did, fmt.Sprintf("waypoints (difficulty mask %d)", v))
	}
	if d, ok := get("difficulty"); ok {
		act, _ := get("act")
		if err := f.SetActive(int(d), int(act)); err != nil {
			return "", err
		}
		did = append(did, fmt.Sprintf("difficulty %d act %d", d, act))
	}
	if v, ok := get("progression"); ok {
		f.SetProgression(byte(v))
		did = append(did, fmt.Sprintf("progression %d", v))
	}
	if s := req.Str("skills"); s != "" { // "36:20,37:20"
		for _, kv := range strings.Split(s, ",") {
			var id, lvl int
			if _, err := fmt.Sscanf(kv, "%d:%d", &id, &lvl); err != nil {
				return "", fmt.Errorf("cheat: skills entry %q (want id:level)", kv)
			}
			if err := f.SetSkillLevel(id, lvl); err != nil {
				return "", err
			}
		}
		did = append(did, "skills "+s)
	}
	pg, err := e.applyProgress(f, req)
	if err != nil {
		return "", err
	}
	did = append(did, pg...)
	if len(did) == 0 {
		return "", errors.New("cheat: saveedit changed nothing (no known parameter)")
	}
	return "save edited: " + strings.Join(did, ", "), nil
}

// levelTo sets the level and the stats that follow from it (see applySave). Going down only sets the level and exp.
func levelTo(f *d2s.File, to int, r Rules) error {
	from := f.Level()
	if err := f.SetLevel(to); err != nil {
		return err
	}
	exp := r.ExpToLevel(to)
	if exp < 0 || exp > 1<<32-1 {
		return fmt.Errorf("cheat: no experience for level %d", to)
	}
	if err := f.SetStat(d2s.StatExp, uint32(exp)); err != nil {
		return err
	}
	if to <= from {
		return nil
	}
	n := to - from
	g := r.Gains(f.Class())
	add := func(id int, delta int) error {
		v, _ := f.Stat(id)
		return f.SetStat(id, v+uint32(delta))
	}
	pts := n * 5
	if g.StatPerLevel > 0 {
		pts = n * g.StatPerLevel
	}
	if err := add(d2s.StatPoints, pts); err != nil {
		return err
	}
	if err := add(d2s.StatSkillPoints, n); err != nil {
		return err
	}
	for _, p := range []struct{ cur, max, perLevel int }{
		{d2s.StatLife, d2s.StatMaxLife, g.LifePerLevel}, {d2s.StatMana, d2s.StatMaxMana, g.ManaPerLevel}, {d2s.StatStamina, d2s.StatMaxStamina, g.StaminaPerLevel}} {
		max, _ := f.Stat(p.max)
		max += uint32(n * p.perLevel * 256 / 4)
		if err := f.SetStat(p.max, max); err != nil {
			return err
		}
		if err := f.SetStat(p.cur, max); err != nil {
			return err
		}
	}
	return nil
}

// questSummary lists the non-zero quest words of difficulty d as "index=0xWORD" (for the ledger diff).
func questSummary(f *d2s.File, d int) string {
	var parts []string
	for i := 0; i < d2s.QuestWords; i++ {
		if w, _ := f.QuestWord(d, i); w != 0 {
			parts = append(parts, fmt.Sprintf("%d=0x%04x", i, w))
		}
	}
	return strings.Join(parts, " ")
}

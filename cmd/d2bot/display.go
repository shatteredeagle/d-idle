package main

import (
	"context"
	"d2lod-bot/internal/combat"
	"fmt"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/event"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/status"
	"d2lod-bot/internal/telemetry"
)

// display wires the live display (internal/status hub at /live/) and the telemetry tracker to the runner.
type display struct {
	hub *status.Hub
	tr  *telemetry.Tracker
	cat bot.Catalog
	run *bot.Runner // set after the runner exists
}

func newDisplay(cat bot.Catalog) *display {
	cfg := telemetry.Config{AreaName: status.AreaName}
	if cat != nil {
		cfg.ExpForLevel = func(l int) (uint32, bool) {
			v := cat.ExpToReach(l)
			return uint32(v), v > 0 || l <= 1
		}
	}
	d := &display{hub: status.NewHub(), tr: telemetry.New(telemetry.RealClock{}, cfg), cat: cat}
	d.hub.RegisterTelemetry(d.tr, status.AreaName)
	d.hub.Register(status.NewCharacterPanel(charSource{d}))
	d.hub.Register(status.NewInventoryPanel(invSource{d}, nil))
	return d
}

// OnSnapshot is bot.Deps.OnSnapshot: it feeds the tracker (must stay quick, must not call the Runner).
func (d *display) OnSnapshot(s *d2.Snapshot) { d.tr.Observe(d.frame(s)) }

func (d *display) frame(s *d2.Snapshot) telemetry.Frame {
	p := s.Player
	f := telemetry.Frame{InGame: true, InTown: p.InTown(), Area: p.Area, Level: p.Level, Exp: p.Exp, Gold: p.Gold, Dead: p.Life <= 0 || p.Mode == 0 || p.Mode == 17}
	friends := combat.Friendlies(s, bot.FriendCatalog(d.cat))
	for i := range s.Monsters {
		u := &s.Monsters[i]
		m := telemetry.Monster{ID: u.ID, TxtNo: u.Class, Mode: u.Mode, X: u.X, Y: u.Y}
		if d.cat != nil {
			if mm, ok := d.cat.Monster(u.Class); ok {
				m.Name = mm.Name
				m.Friendly = !mm.Killable
			}
		}
		m.Friendly = m.Friendly || friends[u.ID] != combat.NotFriendly
		f.Monsters = append(f.Monsters, m)
	}
	for _, u := range s.Ground {
		g := telemetry.GroundItem{ID: u.ID, Name: d.itemName(u), X: u.X, Y: u.Y}
		if u.Item != nil {
			g.Quality = qualityName(u.Item.Quality)
		}
		f.Ground = append(f.Ground, g)
	}
	if s.Inventory != nil && d.cat != nil {
		for _, u := range s.Inventory.Container(d2.ContBelt) {
			if m, ok := d.cat.Item(u.Class); ok {
				switch m.Potion {
				case bot.PotionHealing:
					f.Belt.Health++
				case bot.PotionMana:
					f.Belt.Mana++
				case bot.PotionRejuv:
					f.Belt.Rejuv++
				}
			}
		}
	}
	return f
}

func (d *display) itemName(u d2.Unit) string {
	if d.cat != nil {
		if m, ok := d.cat.Item(u.Class); ok {
			return m.Name
		}
	}
	return fmt.Sprintf("item %d", u.Class)
}

func qualityName(q int) string {
	switch q {
	case 1:
		return status.QLow
	case 3:
		return status.QSuperior
	case 4:
		return status.QMagic
	case 5:
		return status.QSet
	case 6:
		return status.QRare
	case 7:
		return status.QUnique
	case 8:
		return status.QCrafted
	}
	return status.QNormal
}

var activityOf = map[bot.State]string{
	bot.StateInTown: "town", bot.StateTraveling: "exploring", bot.StateClearing: "fighting", bot.StateResupplying: "town",
	bot.StateRecovering: "dead", bot.StatePaused: "idle", bot.StateError: "idle", bot.StateIdle: "idle",
}

// feed keeps the activity bar, goal and error log current and polls the panels until ctx ends. Errors and infos come
// from the instance's event bus (telemetry consumes the bus, internal/event/MIGRATION.md step 3).
func (d *display) feed(ctx context.Context, m *goals.Manager, bus *event.Bus) {
	go d.hub.Run(ctx)
	sub := bus.Subscribe("telemetry", 1024)
	defer sub.Close()
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-sub.C():
			if !ok {
				return
			}
			d.consume(e)
		case <-t.C:
			if d.run != nil {
				st := d.run.Status()
				d.tr.SetActivity(activityOf[d.run.State()], st.Activity)
			}
			for _, g := range m.Goals() {
				if g.State == goals.StateActive {
					d.tr.SetGoal(telemetry.Goal{Name: g.Title, Progress: g.Progress.Percent / 100})
				}
			}
		}
	}
}

// consume maps the bus events the tracker shows in its timeline.
func (d *display) consume(e event.Event) {
	switch p := e.Payload().(type) {
	case event.Error:
		d.tr.Error(p.Message)
	}
}

// ---- panel sources over the runner's newest snapshot ----

type charSource struct{ d *display }

func (c charSource) Character() status.CharacterData {
	var out status.CharacterData
	if c.d.run == nil {
		return out
	}
	s := c.d.run.LastSnapshot()
	if s == nil {
		return out
	}
	p := s.Player
	out = status.CharacterData{Name: p.Name, Class: p.Class.String(), Level: p.Level, Exp: p.Exp, Life: p.Life, MaxLife: p.MaxLife,
		Mana: p.Mana, MaxMana: p.MaxMana, Stamina: p.Stamina, MaxStamina: p.MaxStamina, Str: p.Strength, Dex: p.Dexterity, Vit: p.Vitality,
		Ene: p.Energy, StatPoints: p.StatPoints, SkillPts: p.SkillPoints, LeftSkill: p.LeftSkill, RightSkill: p.RightSkill,
		Difficulty: [...]string{"normal", "nightmare", "hell"}[min(max(int(p.Difficulty), 0), 2)],
		Resists:    status.Resists{Fire: p.Resist.Fire, Cold: p.Resist.Cold, Light: p.Resist.Lightning, Poison: p.Resist.Poison}}
	if c.d.cat != nil {
		out.ExpPrev = uint32(c.d.cat.ExpToReach(p.Level))
		out.ExpNext = uint32(c.d.cat.ExpToReach(p.Level + 1))
	}
	for _, sk := range p.Skills {
		out.Skills = append(out.Skills, status.Skill{ID: sk.ID, Level: sk.Level + sk.Bonus})
	}
	return out
}

type invSource struct{ d *display }

var bodySlot = map[int]string{1: "head", 2: "neck", 3: "torso", 4: "rhand", 5: "lhand", 6: "rring", 7: "lring", 8: "belt", 9: "feet", 10: "gloves"}

func (i invSource) Inventory() status.InventoryData {
	out := status.InventoryData{Equipment: map[string]*status.Item{}, BeltRows: 1}
	if i.d.run == nil {
		return out
	}
	s := i.d.run.LastSnapshot()
	if s == nil || s.Inventory == nil {
		return out
	}
	out.Gold, out.StashGold = s.Player.Gold, s.Player.StashGold
	for _, u := range s.Inventory.Items {
		if u.Item == nil {
			continue
		}
		it := status.Item{ID: u.ID, Code: fmt.Sprint(u.Class), Name: i.d.itemName(u), Quality: qualityName(u.Item.Quality), Ethereal: u.Item.Ethereal(),
			Level: u.Item.ILvl, Identified: u.Item.Identified(), X: u.Item.GX, Y: u.Item.GY, W: 1, H: 1}
		if i.d.cat != nil {
			if m, ok := i.d.cat.Item(u.Class); ok {
				it.W, it.H = max(m.W, 1), max(m.H, 1)
			}
		}
		switch u.Item.Where {
		case d2.ContEquipment:
			if k, ok := bodySlot[u.Item.Body]; ok {
				c := it
				out.Equipment[k] = &c
			}
		case d2.ContInventory:
			out.Inventory = append(out.Inventory, it)
		case d2.ContStash:
			out.Stash = append(out.Stash, it)
		case d2.ContCube:
			out.Cube = append(out.Cube, it)
		case d2.ContBelt:
			out.Belt = append(out.Belt, it)
		}
	}
	return out
}

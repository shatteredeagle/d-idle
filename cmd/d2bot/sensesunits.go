package main

import (
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/senses"
)

// unitSource builds the unit list of the side panel (docs/senses-units.md) from one d2.Snapshot. It runs on the nav
// goroutine (like sensesFeed) and returns fresh values; the panel gets them only through Publisher.SetUnits.
type unitSource struct {
	Read    d2.StateReader
	Cat     combat.Catalog // may be nil: names stay empty, friendly tests use the class tables only
	ModName func(id int) string
	// Attacking returns the id of the monster the fighter is attacking now (0 = none). d2d never sets Snapshot.Hovered
	// (TASK-186), so the bot's own target is the source; nil = fall back to Hovered.
	Attacking func() uint32
}

// recentReader is implemented by the d2d reader (d2denv.Reader.Recent).
type recentReader interface {
	Recent(maxAge time.Duration) (*d2.Snapshot, bool)
}

// recentSnapAge: the age up to which the minimap feed reuses the walker's snapshot.
const recentSnapAge = time.Second

// superFlag is the 0x02 bit next to the boss bit of a superunique (0x0a, docs/orig-spawns.md). U for the live game:
// the census saw it, no drill told a superunique from a rare by it.
const superFlag = 0x02

// Units reads the monsters and classifies them: hostile monster, mercenary (blue), NPC (green). Dead units, summons and
// other friendlies are left out. The target is the monster the fighter attacks (Attacking) or, on Game.exe, the one under the
// cursor (Snapshot.Hovered; U: not checked live).
func (s unitSource) Units() []senses.Unit {
	if s.Read == nil {
		return nil
	}
	// The nav walk that calls this has just read a snapshot: reuse it instead of queueing a second full read behind it
	// (WALKSMOOTH2, V live: that read was a third of the walk's time).
	var snap *d2.Snapshot
	var err error
	if rr, ok := s.Read.(recentReader); ok {
		snap, _ = rr.Recent(recentSnapAge)
	}
	if snap == nil {
		snap, err = s.Read.Snapshot()
	}
	if err != nil || snap == nil || snap.State != d2.StateInGame {
		return nil
	}
	out := make([]senses.Unit, 0, len(snap.Monsters))
	for _, m := range snap.Monsters {
		if !m.Alive() {
			continue
		}
		u := senses.Unit{ID: m.ID, X: m.X, Y: m.Y, Class: m.Class}
		why, friendly := combat.FriendlyWhyOf(snap, m, s.Cat)
		switch {
		case friendly && (why == combat.WhyNPC):
			u.Kind = senses.KindNPC
		case friendly && (why == combat.WhyHirelingAI || d2.IsMercClass(m.Class)):
			u.Kind = senses.KindMerc
		case friendly:
			continue
		default:
			u.Kind = senses.KindMonster
		}
		if s.Cat != nil {
			if cm := s.Cat.Monster(m.Class); cm != nil {
				u.Name = s.Cat.MonsterName(cm)
			}
		}
		if u.Kind == senses.KindMonster {
			u.Target = snap.Hovered.Active && snap.Hovered.Type == d2.TypeMonster && snap.Hovered.ID == m.ID
			if s.Attacking != nil {
				if id := s.Attacking(); id != 0 && id == m.ID {
					u.Target = true
				}
			}
			if mi := m.Monster; mi != nil {
				if mi.MaxLife > 0 {
					u.LifePct = max(1, min(100, mi.LifePct))
				}
				s.elite(&u, mi)
			}
		}
		out = append(out, u)
	}
	return out
}

// elite fills the elite kind, own name and mod names. Mod ids come from MonsterInfo.EnchantIDs (the one place that reads
// the enchant array); no ModName: no mod names.
func (s unitSource) elite(u *senses.Unit, mi *d2.MonsterInfo) {
	switch mi.Kind() {
	case d2.MonKindChampion:
		u.Elite = senses.EliteChampion
	case d2.MonKindUnique:
		u.Elite = senses.EliteUnique
		if mi.Flags&superFlag != 0 {
			u.Elite = senses.EliteSuperunique
		}
		u.Title = mi.Name
	default:
		return
	}
	if s.ModName == nil {
		return
	}
	ids, _ := mi.EnchantIDs()
	for _, id := range ids {
		u.Mods = append(u.Mods, s.ModName(int(id)))
	}
}

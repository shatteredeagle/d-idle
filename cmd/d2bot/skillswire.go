package main

import (
	"strings"

	"d2lod-bot/internal/art"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/skilltree"
	"d2lod-bot/internal/watch"
)

// The Skills page on the real bot (docs/skilltrees.md): the hero's class trees from the game tables and the newest
// snapshot. Read only; the allocation route stays a 501 stub.

type skillsSource struct {
	db  *data.DB
	ref *runnerRef
}

// skillsAPI is what newFun mounts: nil without tables.
func skillsAPI(db *data.DB, ref *runnerRef) *watch.SkillsAPI {
	if db == nil || len(db.Skills) == 0 {
		return nil
	}
	a := watch.NewSkillsAPI()
	a.Bind(skillsSource{db, ref})
	return a
}

func (s skillsSource) View() skilltree.View {
	t := skilltree.Tables{Skills: s.db.Skills, Text: s.db.Text}
	var snap *d2.Snapshot
	if s.ref != nil {
		if r := s.ref.p.Load(); r != nil {
			snap = r.LastSnapshot()
		}
	}
	if snap == nil || snap.Player == nil {
		return skilltree.Build(t, skilltree.Hero{}, nil)
	}
	return skilltree.Build(t, heroSkills(s.db, snap.Player), func(sk *data.Skill) string {
		return art.SkillKey(strings.ToLower(sk.Class), sk.IconCel)
	})
}

// heroSkills reads the hero's points: base points are the hard points (V live, d2.Skill.Level); the bonus is what
// Player.Effective adds on top (single-skill stat V live; all, class, tab and shrine sources are U, and the d2d
// backend has no effective stat vector, so there it shows only what the snapshot carries).
func heroSkills(db *data.DB, p *d2.Player) skilltree.Hero {
	h := skilltree.Hero{Level: p.Level, Unspent: p.SkillPoints, Points: map[int]skilltree.Points{}}
	if i := int(p.Class); i >= 0 && i < len(db.Classes) {
		h.ClassCode, h.ClassName = db.Classes[i].Code, db.Classes[i].Name
	}
	for _, sk := range p.Skills {
		ref := d2.SkillRef{}
		if d := db.SkillByID(sk.ID); d != nil {
			ref = d2.SkillRef{Own: h.ClassCode != "" && strings.EqualFold(d.Class, h.ClassCode), Tab: d.Page}
		}
		h.Points[sk.ID] = skilltree.Points{Base: sk.Level, Bonus: p.Effective(sk.ID, ref) - sk.Level}
	}
	return h
}

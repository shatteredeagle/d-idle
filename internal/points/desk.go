package points

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/skilltree"
)

// Reply is what an action answers with. The page reads state back from GET, never from a reply; the read-back numbers
// are only the proof that the game took the point.
type Reply struct {
	OK           bool   `json:"ok"`
	Already      bool   `json:"already,omitempty"`
	Code         string `json:"code,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Status       int    `json:"-"`
	NeedsConfirm bool   `json:"needsConfirm,omitempty"`
	Tier         string `json:"tier,omitempty"`
	// Before and After are the value that moved: the attribute (stat) or the skill's hard points (skill).
	Before int `json:"before,omitempty"`
	After  int `json:"after,omitempty"`
	// Left is the unspent count of the kind that was spent, after the action.
	Left int `json:"left,omitempty"`
}

// Config wires a Desk to the bot. Every function may be nil: the desk then refuses with a reason instead of guessing.
type Config struct {
	Hero string
	T    Tables
	// Dir is where the settings and the audit ring live (the quest store directory is the convention); "" keeps both in memory.
	Dir string
	Now func() time.Time

	// Enabled is resource.pointsApi.
	Enabled func() bool

	Player func() *d2.Player                             // the newest snapshot's player, nil before the first
	Fresh  func(ctx context.Context) (*d2.Player, error) // a re-read for the read-back
	Base   func(ctx context.Context) (*[4]int, bool)     // base attributes without items
	Gear   func() []GearReq
	Busy   func() bool // a fight is on: points are never spent then (the leveler's rule)
	Verbs  func() Verbs

	// SpendStat and SpendSkill send the verb; statID is StatDef.ID, index is ClassSkillIndex.
	SpendStat  func(ctx context.Context, statID int) error
	SpendSkill func(ctx context.Context, index int) error

	Playable func(skillName string) bool
	Attack   func(skillName string) bool

	// Automatic allocator (autoalloc.go). AutoOn is the knob agency.autoAlloc; Builds lists every build file with its
	// class and tier; Next is the build's remaining steps for this hero, in order; OffBuild names skills the hero owns
	// that the build does not call for; Respec is the price model; Logf writes the trace lines.
	AutoOn   func() bool
	Builds   func() []BuildInfo
	Next     func(build string, p *d2.Player) ([]Step, error)
	OffBuild func(build string, p *d2.Player) []string
	Respec   func() RespecParams
	Logf     func(msg string, kv ...any)
}

// Desk is the points service of one hero.
type Desk struct {
	c     Config
	store *Store
	audit *Audit
	auto  autoState
}

// NewDesk builds the desk. It reads no game state until asked.
func NewDesk(c Config) *Desk {
	if c.Now == nil {
		c.Now = time.Now
	}
	return &Desk{c: c, store: NewStore(c.Dir, c.Hero), audit: NewAudit(c.Dir, c.Hero)}
}

func (d *Desk) enabled() bool { return d.c.Enabled == nil || d.c.Enabled() }

func (d *Desk) verbs() Verbs {
	if d.c.Verbs == nil {
		return Verbs{Via: "none", Why: "no backend is wired"}
	}
	return d.c.Verbs()
}

// CanSpendSkills says whether a skill point can be spent by verb on this backend now.
func (d *Desk) CanSpendSkills() bool { v := d.verbs(); return v.Skill && d.c.SpendSkill != nil }

// Autopilot is the per-hero switch (default off).
func (d *Desk) Autopilot() bool { return d.store.Get().Autopilot }

// Waiting returns the banked counts and the sentence for the Needs-you row; ok is false with nothing banked.
func (d *Desk) Waiting() (text string, stat, skill int, ok bool) {
	if d.c.Player == nil {
		return "", 0, 0, false
	}
	p := d.c.Player()
	if p == nil {
		return "", 0, 0, false
	}
	stat, skill = max(p.StatPoints, 0), max(p.SkillPoints, 0)
	text = Waiting(stat, skill)
	return text, stat, skill, text != ""
}

// View is the sheet for GET /api/hero/points.
func (d *Desk) View(ctx context.Context) Sheet {
	in := Input{Hero: d.c.Hero, Verbs: d.verbs(), Autopilot: d.Autopilot(), Enabled: d.enabled()}
	if d.c.Player != nil {
		in.Player = d.c.Player()
	}
	if in.Player != nil && len(d.c.T.Classes) > 0 {
		in.Class = d.c.T.Class(in.Player.Class.String())
	}
	if d.c.Base != nil && in.Player != nil {
		if b, ok := d.c.Base(ctx); ok {
			in.Base = b
		}
	}
	if d.c.Gear != nil {
		in.Gear = d.c.Gear()
	}
	return BuildSheet(in)
}

// Recent is the trail for the page's "who did this" line.
func (d *Desk) Recent(n int) []Event { return d.audit.Recent(n) }

// Record appends one event to the audit ring.
func (d *Desk) Record(e Event) {
	if e.At.IsZero() {
		e.At = d.c.Now()
	}
	_ = d.audit.Append(e)
}

func refuse(status int, code, reason string) Reply {
	return Reply{Status: status, Code: code, Reason: reason}
}

// precheck is what every spend needs: the API on, a player, no fight.
func (d *Desk) precheck() (*d2.Player, *Reply) {
	if !d.enabled() {
		r := refuse(http.StatusServiceUnavailable, "off", "the points desk is switched off (resource.pointsApi)")
		return nil, &r
	}
	var p *d2.Player
	if d.c.Player != nil {
		p = d.c.Player()
	}
	if p == nil {
		r := refuse(http.StatusServiceUnavailable, "no_hero", "the hero is not in a game yet")
		return nil, &r
	}
	if d.c.Busy != nil && d.c.Busy() {
		r := refuse(http.StatusConflict, "busy", "a fight is on; points are spent at a calm moment, try again in a few seconds")
		return nil, &r
	}
	return p, nil
}

// SpendStat puts one unspent stat point into an attribute by verb and reads the game back.
func (d *Desk) SpendStat(ctx context.Context, key string) Reply {
	def, ok := StatByKey(key)
	if !ok {
		return refuse(http.StatusBadRequest, "bad_stat", `stat must be "str", "dex", "vit" or "ene"`)
	}
	p, bad := d.precheck()
	if bad != nil {
		return *bad
	}
	if ok, why := StatLegal(p.StatPoints, d.verbs()); !ok {
		code, st := "no_points", http.StatusConflict
		if p.StatPoints >= 1 {
			code, st = "no_verb", http.StatusNotImplemented
		}
		return refuse(st, code, why)
	}
	if d.c.SpendStat == nil {
		return refuse(http.StatusNotImplemented, "no_verb", "this backend has no stat verb")
	}
	before := attr(p, def.Key)
	if err := d.c.SpendStat(ctx, def.ID); err != nil {
		return refuse(http.StatusBadGateway, "verb_failed", "the game did not take the verb: "+err.Error())
	}
	q, err := d.fresh(ctx, func(q *d2.Player) bool { return q.StatPoints < p.StatPoints })
	if err != nil {
		return refuse(http.StatusBadGateway, "read_failed", "sent, but the read-back failed: "+err.Error())
	}
	if q.StatPoints >= p.StatPoints {
		return refuse(http.StatusBadGateway, "not_taken", fmt.Sprintf("the game still shows %d stat points: the point was not taken", q.StatPoints))
	}
	return Reply{OK: true, Before: before, After: attr(q, def.Key), Left: q.StatPoints}
}

func attr(p *d2.Player, key string) int {
	switch key {
	case "str":
		return p.Strength
	case "dex":
		return p.Dexterity
	case "vit":
		return p.Vitality
	}
	return p.Energy
}

// fresh re-reads the player until cond holds or a few tries pass; the verb is applied at the game's next tick.
func (d *Desk) fresh(ctx context.Context, cond func(*d2.Player) bool) (*d2.Player, error) {
	if d.c.Fresh == nil {
		return nil, fmt.Errorf("no re-read wired")
	}
	var last *d2.Player
	for i := 0; i < 8; i++ {
		q, err := d.c.Fresh(ctx)
		if err != nil {
			return nil, err
		}
		last = q
		if cond(q) {
			return q, nil
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	return last, nil
}

// hero builds the skilltree view of the hero for the legality check.
func (d *Desk) hero(p *d2.Player) skilltree.Hero {
	h := skilltree.Hero{Level: p.Level, Unspent: p.SkillPoints, Points: map[int]skilltree.Points{}}
	if len(d.c.T.Classes) > 0 {
		if c := d.c.T.Class(p.Class.String()); c != nil {
			h.ClassCode, h.ClassName = c.Code, c.Name
		}
	}
	for _, s := range p.Skills {
		h.Points[s.ID] = skilltree.Points{Base: s.Level}
	}
	return h
}

func (d *Desk) legal() Legal {
	return Legal{T: d.c.T, Playable: d.c.Playable, Attack: d.c.Attack}
}

// CheckSkill is the legality of one more point, for the page's per-skill buttons and the POST.
func (d *Desk) CheckSkill(id int, confirm bool) SkillVerdict {
	var p *d2.Player
	if d.c.Player != nil {
		p = d.c.Player()
	}
	if p == nil {
		return SkillVerdict{Code: "no_hero", Reason: "the hero is not in a game yet"}
	}
	return d.legal().CheckSkill(d.hero(p), id, confirm)
}

// SpendSkill puts one unspent skill point into a skill by verb after the legality check and reads the game back.
// An unsupported attack skill needs confirm; any other unsupported skill is allowed and labelled.
func (d *Desk) SpendSkill(ctx context.Context, id int, confirm bool) Reply {
	p, bad := d.precheck()
	if bad != nil {
		return *bad
	}
	v := d.legal().CheckSkill(d.hero(p), id, confirm)
	if !v.OK {
		st := http.StatusConflict
		if v.Code == CodeUnknownSkill {
			st = http.StatusBadRequest
		}
		r := refuse(st, v.Code, v.Reason)
		r.NeedsConfirm, r.Tier = v.NeedsConfirm, v.Tier
		return r
	}
	vb := d.verbs()
	if !vb.Skill || d.c.SpendSkill == nil {
		return refuse(http.StatusNotImplemented, "no_verb", "this backend has no skill verb: "+vb.Why)
	}
	c := d.c.T.Class(p.Class.String())
	if c == nil {
		return refuse(http.StatusBadRequest, CodeUnknownSkill, "no class table for "+p.Class.String())
	}
	idx, ok := ClassSkillIndex(d.c.T, c.Code, id)
	if !ok {
		return refuse(http.StatusBadRequest, CodeUnknownSkill, fmt.Sprintf("skill %d has no tree index for %s", id, c.Name))
	}
	before := d.hero(p).Points[id].Base
	if err := d.c.SpendSkill(ctx, idx); err != nil {
		return refuse(http.StatusBadGateway, "verb_failed", "the game did not take the verb: "+err.Error())
	}
	q, err := d.fresh(ctx, func(q *d2.Player) bool { return q.SkillPoints < p.SkillPoints })
	if err != nil {
		return refuse(http.StatusBadGateway, "read_failed", "sent, but the read-back failed: "+err.Error())
	}
	after := d.hero(q).Points[id].Base
	if q.SkillPoints >= p.SkillPoints || after <= before {
		return refuse(http.StatusBadGateway, "not_taken", fmt.Sprintf("the game shows skill level %d (was %d): the point was not taken", after, before))
	}
	return Reply{OK: true, Before: before, After: after, Left: q.SkillPoints, Tier: v.Tier}
}

// SetAutopilot flips the per-hero switch. A repeated press is ok with already set.
func (d *Desk) SetAutopilot(on bool) Reply {
	changed, err := d.store.SetAutopilot(on, d.c.Now())
	if err != nil {
		return refuse(http.StatusInternalServerError, "store", "could not save the switch: "+err.Error())
	}
	return Reply{OK: true, Already: !changed}
}

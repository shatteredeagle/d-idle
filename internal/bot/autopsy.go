package bot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

// Death post-mortem (docs/deathautopsy.md). Every DeathRecord carries an Autopsy: what the character was doing, the last
// minute of life and mana, the kit it had, and a one-sentence cause in plain words. It rides in deaths.json (survives
// restarts, shown by the /deaths API) and is appended to deaths.jsonl next to it (one JSON object per death, bounded).

const (
	trailWindow   = 60 * time.Second
	trailEvery    = time.Second
	autopsyKeep   = 200       // lines kept in deaths.jsonl after a trim
	autopsyMaxLen = 256 << 10 // bytes of deaths.jsonl that trigger a trim
)

// TrailPoint is one sample of the minute before a death.
type TrailPoint struct {
	SecBefore int    `json:"secBefore"`
	Life      int    `json:"life"` // percent
	Mana      int    `json:"mana"` // percent
	Hostiles  int    `json:"hostiles"`
	State     string `json:"state"` // traveling, clearing, resupplying, ...
}

// Kit is what the character carried when it died.
type Kit struct {
	BeltHP, BeltMP, BeltRV int
	PackHP                 int
	TownPortals            int
	Gold                   int
	Equipped               int
}

// Autopsy is the structured post-mortem of one death. Fields marked "best effort" come from the last living snapshot.
type Autopsy struct {
	Class         string       `json:"class"`
	Build         string       `json:"build,omitempty"`
	KillerType    string       `json:"killerType,omitempty"` // melee or ranged (monstats flag), best effort
	KillerLevel   int          `json:"killerLevel,omitempty"`
	KillerAffixes []string     `json:"killerAffixes,omitempty"`
	HostilesNear  int          `json:"hostilesNear"` // hostile monsters within the fight radius at the last living snapshot
	State         string       `json:"state"`        // what the bot was doing: traveling, clearing, ...
	Activity      string       `json:"activity,omitempty"`
	Goal          string       `json:"goal,omitempty"`
	LastSafety    string       `json:"lastSafety,omitempty"`  // last survival action decided in the minute before (none = never)
	PotionsDrunk  int          `json:"potionsDrunk"`          // in the trail window
	DamagePer5s   []int        `json:"damagePer5s,omitempty"` // life percent lost per 5 s, oldest first
	Trail         []TrailPoint `json:"trail,omitempty"`
	Kit           Kit          `json:"kit"`
	LostXP        int64        `json:"lostXP,omitempty"`
	Reaction      string       `json:"reaction,omitempty"` // what a rule did about it: ban, pause

	// Added for M0 (docs/deathautopsy.md "M0 fields"). Nullable fields are null when unknown or not yet known.
	BeltHP          int        `json:"beltHP"`          // healing potions in the belt slots of the last living snapshot (memory, not inferred)
	BeltMP          int        `json:"beltMP"`          // mana potions, same source
	PotionsDrunk60s int        `json:"potionsDrunk60s"` // potions that left belt or pack in the last 60 s (same count as potionsDrunk)
	GoldCarried     int        `json:"goldCarried"`
	StashGold       int        `json:"stashGold"`
	GoldNextMinute  *int       `json:"goldNextMinute"` // gold on the first minute line after the death (null until it arrives)
	MercAlive       *bool      `json:"mercAlive"`      // null: no merc unit was listed
	BackInFieldTs   *time.Time `json:"backInFieldTs"`  // first living snapshot outside town after the death
	SecondsToField  *int       `json:"secondsToField"` // backInFieldTs minus death time
	GoldRecovered   *int       `json:"goldRecovered"`  // gold the post-death sweep picked up (null: no sweep ran)
	RouteID         string     `json:"routeId"`        // area ids of the last distinct areas entered before the death, "2>3>8"
	FieldSecs       int        `json:"fieldSecs"`      // trail seconds spent outside town before the death (potion bug triage, docs/potion-bug.md)
	LowLifeSecs     int        `json:"lowLifeSecs"`    // of those, seconds under the drink line (combat.drinkLifePct) with a trail sample
	Bug             bool       `json:"bug"`            // beltHP >= 1 and no potion drunk in the last 60 s

	// Added for ROUTEGUARD (docs/deathautopsy.md "Route guard fields"): what a repeat of the Soaky death spot needs in one grep.
	Nearest    []NearHostile  `json:"nearest,omitempty"`    // the 5 nearest hostiles at the last living snapshot
	PortalCast *PortalNote    `json:"portalCast,omitempty"` // the last Town Portal cast the fighter made (nil: none ever)
	LastLife   []LifeSample   `json:"lastLife,omitempty"`   // last 6 life samples, at least 100 ms apart, newest last
	Casts5s    int            `json:"casts5s"`              // skill casts in the 5 s before the death
	CastSkills map[string]int `json:"castSkills,omitempty"` // the same per skill id
	Mode       string         `json:"mode,omitempty"`       // moving (a walk) or fighting (a clear/kill), from the last bot state
	MercState  string         `json:"mercState,omitempty"`  // none (no unit listed), alive or dead
	RouteGuard string         `json:"routeGuard,omitempty"` // the route guard verdict line at the time
	Summary    string         `json:"summary"`
}

// NearHostile is one monster of the nearest-five list: class id, name and distance in subtiles (max of dx, dy).
type NearHostile struct {
	Class int    `json:"class"`
	Name  string `json:"name,omitempty"`
	Dist  int    `json:"dist"`
}

// PortalNote says whether a Town Portal cast started shortly before the death and how long it ran.
type PortalNote struct {
	SecBefore int  `json:"secBefore"` // start of the cast, seconds before the death
	DurMs     int  `json:"durMs"`     // 0 while the cast was still running
	OK        bool `json:"ok"`
}

// fighterDiag is what the combat fighter offers the autopsy (the adapter forwards it; stubs and fakes do not need it).
type fighterDiag interface {
	LastPortalCast() combat.PortalCast
	RecentCasts(window time.Duration) (int, map[int]int)
}

type trailStamp struct {
	at              time.Time
	life, mana, hos int
	state           string
	potions         int
	safety          SafetyAction
	field           bool // outside town
}

// noteTrail records one trail sample per second while alive. Called with obMu held, from trackAlive.
func (r *Runner) noteTrail(s *d2.Snapshot, now time.Time) {
	o := &r.ob
	if n := len(o.trail); n > 0 && now.Sub(o.trail[n-1].at) < trailEvery {
		return
	}
	p := s.Player
	r.mu.Lock() // lock order obMu -> mu, as observe() does
	st, act, goal := string(r.st), r.activity, ""
	if r.goal != nil {
		goal = r.goal.ID
	}
	r.mu.Unlock()
	o.activity, o.goalID = act, goal
	mana := 0
	if p.MaxMana > 0 {
		mana = p.Mana * 100 / p.MaxMana
	}
	o.trail = append(o.trail, trailStamp{at: now, life: p.LifePct(), mana: mana, hos: r.hostileNear(s, r.e.Tune.Combat.FightRadius), state: st, potions: int(r.drank.Load()), safety: o.lastSafety, field: !p.InTown()})
	for len(o.trail) > 0 && now.Sub(o.trail[0].at) > trailWindow {
		o.trail = o.trail[1:]
	}
}

// kitOf counts potions, portal scrolls and gear in a snapshot.
func (r *Runner) kitOf(s *d2.Snapshot) Kit {
	k := Kit{Gold: s.Player.Gold}
	if s.Inventory == nil || r.d.Catalog == nil {
		return k
	}
	for _, u := range s.Inventory.Container(d2.ContEquipment) {
		_ = u
		k.Equipped++
	}
	for i := range s.Inventory.Items {
		u := &s.Inventory.Items[i]
		if u.Item == nil {
			continue
		}
		m, ok := r.d.Catalog.Item(u.Class)
		if !ok {
			continue
		}
		switch u.Item.Where {
		case d2.ContBelt:
			switch m.Potion {
			case PotionHealing:
				k.BeltHP++
			case PotionMana:
				k.BeltMP++
			case PotionRejuv:
				k.BeltRV++
			}
		case d2.ContInventory:
			if strings.Contains(m.Name, "Town Portal") {
				k.TownPortals++
			}
			if m.Potion == PotionHealing {
				k.PackHP++
			}
		}
	}
	return k
}

// buildAutopsy assembles the post-mortem from the trail and the last living snapshot. Called with obMu held.
func (r *Runner) buildAutopsy(p *d2.Player, a aliveInfo, now time.Time) *Autopsy {
	o := &r.ob
	au := &Autopsy{Class: p.Class.String(), Build: r.buildName(p), KillerLevel: a.killerLevel, Kit: a.kit, State: "unknown",
		Activity: o.activity, Goal: o.goalID, LostXP: o.xpLost}
	if a.killer != nil {
		if m, ok := r.monsterMeta(a.killer.Class); ok {
			au.KillerType = "melee"
			if m.Ranged {
				au.KillerType = "ranged"
			}
		}
	}
	if a.killerAffix != 0 {
		au.KillerAffixes = strings.Split(a.killerAffix.String(), ",")
	}
	au.HostilesNear = a.hostiles
	au.BeltHP, au.BeltMP = a.kit.BeltHP, a.kit.BeltMP
	au.GoldCarried, au.StashGold, au.MercAlive = a.gold, a.stashGold, a.merc
	au.RouteID = o.routeID()
	var last SafetyAction
	first := -1
	for i, t := range o.trail {
		last = t.safety
		if first < 0 {
			first = i
		}
		if t.field {
			au.FieldSecs++
			if t.life < r.e.Tune.Combat.DrinkLifePct {
				au.LowLifeSecs++
			}
		}
		au.Trail = append(au.Trail, TrailPoint{SecBefore: int(now.Sub(t.at).Round(time.Second) / time.Second), Life: t.life, Mana: t.mana, Hostiles: t.hos, State: t.state})
		au.State = t.state
	}
	if first >= 0 {
		au.PotionsDrunk = int(r.drank.Load()) - o.trail[first].potions
	}
	au.PotionsDrunk60s = au.PotionsDrunk
	r.fillRouteFacts(au, a, now)
	au.Bug = au.BeltHP >= 1 && au.PotionsDrunk60s == 0
	// damage per 5 s window, oldest first
	for lo := 0; lo < len(o.trail); {
		hi := lo
		for hi+1 < len(o.trail) && o.trail[hi+1].at.Sub(o.trail[lo].at) < 5*time.Second {
			hi++
		}
		drop := 0
		for i := lo + 1; i <= hi; i++ {
			if d := o.trail[i-1].life - o.trail[i].life; d > 0 {
				drop += d
			}
		}
		au.DamagePer5s = append(au.DamagePer5s, drop)
		lo = hi + 1
	}
	if last != SafetyNone {
		au.LastSafety = last.String()
	} else {
		au.LastSafety = "none"
	}
	au.Summary = au.summarize(a.killerName, r.areaName(o.lastGood.area))
	return au
}

func (au *Autopsy) summarize(killer, area string) string {
	how := map[string]string{"traveling": "walking through", "clearing": "fighting in", "resupplying": "on a town trip in", "recovering": "recovering in"}[au.State]
	if how == "" {
		how = "in"
	}
	if killer == "" {
		killer = "an unknown monster"
	}
	parts := []string{fmt.Sprintf("%s %s, killed by %s", how, area, killer)}
	if au.HostilesNear > 1 {
		parts = append(parts, fmt.Sprintf("%d hostiles near", au.HostilesNear))
	}
	if n := len(au.Trail); n > 0 {
		parts = append(parts, fmt.Sprintf("mana %d%%", au.Trail[n-1].Mana))
	}
	parts = append(parts, fmt.Sprintf("belt %d hp %d mp, %d portal scrolls, %d potions drunk in the last minute", au.Kit.BeltHP, au.Kit.BeltMP, au.Kit.TownPortals, au.PotionsDrunk))
	if au.Bug {
		parts = append(parts, fmt.Sprintf("BUG: %d healing potions in the belt and none drunk in the last minute (%d s in the field, %d s under the drink line)", au.BeltHP, au.FieldSecs, au.LowLifeSecs))
	}
	if au.LastSafety == "none" {
		parts = append(parts, "no survival action was taken")
	}
	if au.Activity != "" {
		parts = append(parts, "doing: "+au.Activity)
	}
	if len(au.Nearest) > 0 {
		var ns []string
		for _, n := range au.Nearest {
			ns = append(ns, fmt.Sprintf("%s(%d)@%d", orUnknown(n.Name), n.Class, n.Dist))
		}
		parts = append(parts, "nearest: "+strings.Join(ns, " "))
	}
	if pc := au.PortalCast; pc != nil {
		parts = append(parts, fmt.Sprintf("town portal cast started %ds before, %d ms, ok=%v", pc.SecBefore, pc.DurMs, pc.OK))
	} else {
		parts = append(parts, "no town portal cast")
	}
	parts = append(parts, fmt.Sprintf("mode %s, %d casts in 5 s, merc %s", au.Mode, au.Casts5s, au.MercState))
	if au.RouteGuard != "" {
		parts = append(parts, au.RouteGuard)
	}
	return strings.Join(parts, "; ")
}

// appendAutopsy writes one JSON line next to the death log and trims the file when it grows. A failure is only logged.
func (r *Runner) appendAutopsy(rec DeathRecord) {
	path := r.cfg.DeathLogPath
	if path == "" || rec.Autopsy == nil {
		return
	}
	path = strings.TrimSuffix(path, ".json") + ".jsonl"
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		r.e.Log.Warn("autopsy not saved", "err", err)
		return
	}
	_, werr := f.Write(append(b, '\n'))
	f.Close()
	if werr != nil {
		r.e.Log.Warn("autopsy not saved", "err", werr)
		return
	}
	if st, err := os.Stat(path); err == nil && st.Size() > autopsyMaxLen {
		trimLines(path, autopsyKeep)
	}
}

func trimLines(path string, keep int) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	f.Close()
	if len(lines) > keep {
		lines = lines[len(lines)-keep:]
	}
	_ = os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// setReaction notes what a rule did about the death stamped at, so the card can say what the bot will do differently.
// Caller holds obMu.
func (r *Runner) setReaction(at time.Time, text string) {
	for i := range r.dlog.Records {
		if rc := &r.dlog.Records[i]; rc.Time.Equal(at) && rc.Autopsy != nil {
			if rc.Autopsy.Reaction != "" {
				rc.Autopsy.Reaction += "; "
			}
			rc.Autopsy.Reaction += text
			return
		}
	}
}

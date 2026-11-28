package bot

import (
	"errors"
	"fmt"
	"os"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/diffready"
	"d2lod-bot/internal/gear"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/task"
)

// The opt-in difficulty hop (bot.diffReadyV2, docs/diffready.md): the bot computes whether the hero is ready for the
// next difficulty, publishes the line (ops why, the Pact/Return screen) and hops only after the owner's go. It never
// hops on its own and never goes back down (user decisions 2026-11-27).

// GrantDifficultyGo records the owner's go for target ("nightmare" or "hell"); the next readiness check that finds the
// hero ready consumes it. The player-facing door is RequestDifficultyGo (operator verb go-difficulty, Return screen button).
func (r *Runner) GrantDifficultyGo(target string) {
	r.diffGo.Grant(target, r.goHero(r.latest()), r.e.Clock.Now())
}

// PersistDifficultyGo keeps an unconsumed go in path across a restart of the bot. The file carries the hero name and
// the end of the go (diffready.GoTTL); a go for another hero or an ended one is dropped with a log line.
func (r *Runner) PersistDifficultyGo(path string) error {
	r.diffGo.Logf = func(msg string, args ...any) { r.e.Log.Warn(msg, args...) }
	return r.diffGo.Persist(path)
}

// goHero is the name a go is bound to ("" without a snapshot: such a go matches nobody).
func (r *Runner) goHero(s *d2.Snapshot) string {
	if s == nil || s.Player == nil {
		return ""
	}
	return s.Player.Name
}

// DifficultyOffer is the next difficulty with the owner-facing line and the unmet requirements.
type DifficultyOffer struct {
	Target  string   `json:"target,omitempty"` // "" when the hero is in Hell or the knob is 0
	Ready   bool     `json:"ready"`
	Pending bool     `json:"pending,omitempty"` // the owner already said go and the hop has not happened yet
	Line    string   `json:"line,omitempty"`
	Missing []string `json:"missing,omitempty"`
}

func nextDifficulty(s *d2.Snapshot) string {
	return [...]string{"nightmare", "hell", ""}[min(max(int(s.Player.Difficulty), 0), 2)]
}

// DifficultyOffer evaluates the next difficulty now (no side effect but the published line and dropping a stale go).
func (r *Runner) DifficultyOffer() DifficultyOffer {
	if r.e.Tune.Bot.DiffReadyV2 == 0 {
		return DifficultyOffer{}
	}
	s := r.latest()
	if s == nil || s.Player == nil {
		return DifficultyOffer{Line: "no snapshot yet"}
	}
	next := nextDifficulty(s)
	r.diffGo.Keep(next)
	if next == "" {
		return DifficultyOffer{}
	}
	v, err := r.diffVerdict(s, next)
	if err != nil {
		return DifficultyOffer{Target: next, Line: err.Error()}
	}
	return DifficultyOffer{Target: next, Ready: v.Ready, Pending: r.diffGo.Granted(next, r.goHero(s), r.e.Clock.Now()), Line: v.Why(), Missing: v.Missing}
}

// RequestDifficultyGo is the owner's go: it is refused with the reason unless bot.diffReadyV2 is on and the hero is
// ready for target, the next difficulty. The hop itself follows at the next check and consumes the go. Both doors
// (operator verb, watch page) end here, so the measured-run refusal is here: a go inside a gate or soak would change
// the difficulty mid-run.
func (r *Runner) RequestDifficultyGo(target string) (string, error) {
	if os.Getenv("D2BOT_RUN") != "" {
		return "", errors.New("a measured run is in progress (D2BOT_RUN is set): no difficulty go")
	}
	if r.e.Tune.Bot.DiffReadyV2 == 0 {
		return "", errors.New("the difficulty hop is off (bot.diffReadyV2 is 0)")
	}
	o := r.DifficultyOffer()
	switch {
	case o.Target == "":
		return "", errors.New("no next difficulty to go to: " + o.Line)
	case target != o.Target:
		return "", fmt.Errorf("the next difficulty is %s, not %q", o.Target, target)
	case !o.Ready:
		return "", errors.New(o.Line)
	}
	r.GrantDifficultyGo(target)
	return "go recorded for " + target + "; the hero hops at its next safe check", nil
}

// DifficultyReadiness is the last readiness line ("" while bot.diffReadyV2 is 0 or nothing was checked).
func (r *Runner) DifficultyReadiness() string {
	r.diffMu.Lock()
	defer r.diffMu.Unlock()
	return r.diffWhy
}

// DifficultyGate is the chain's view (cmd/d2bot taskChain): active is false while the knob is 0 and the old rule
// stands. Active and ok mean ready AND the owner said go. The go is not consumed here: ensureDifficulty takes it when
// the hop's new game has started (diffHopStarted), so a hop that cannot start yet keeps it (fable review 2 finding 4).
func (r *Runner) DifficultyGate(next string, clvl int) (ok bool, why string, active bool) {
	if r.e.Tune.Bot.DiffReadyV2 == 0 {
		return false, "", false
	}
	s := r.latest()
	if s == nil || s.Player == nil {
		return false, "no snapshot yet", true
	}
	r.diffGo.Keep(next)
	v, err := r.diffVerdict(s, next)
	if err != nil {
		return false, err.Error(), true
	}
	if !v.Ready {
		return false, v.Why(), true
	}
	if !r.diffGo.Granted(next, r.goHero(s), r.e.Clock.Now()) {
		return false, v.Why(), true
	}
	return true, "", true
}

// diffVerdict builds the hero from the snapshot and evaluates target; the line is published for ops why.
func (r *Runner) diffVerdict(s *d2.Snapshot, target string) (diffready.Verdict, error) {
	set := func(line string) {
		r.diffMu.Lock()
		r.diffWhy = line
		r.diffMu.Unlock()
	}
	t, err := diffready.Shipped()
	if err != nil {
		set("difficulty readiness: " + err.Error())
		return diffready.Verdict{}, err
	}
	p := s.Player
	h := diffready.Hero{Class: p.Class, Level: p.Level, MaxLife: p.MaxLife, Res: leveler.RawResists(p), MaxRes: p.MaxResist,
		KeySkillLevel: -1, Healing: -1, ManaOrRejuv: -1, Portals: -1}
	h.KeySkillLevel = r.keySkillLevel(p)
	if s.Merc != nil {
		h.Merc = diffready.Yes // U: a read MercSheet is taken as a living merc
	}
	if c, known := r.supplyCount(s); known {
		h.Healing = c.Belt[supply.HP] + c.Pack[supply.HP]
		h.ManaOrRejuv = c.Belt[supply.MP] + c.Pack[supply.MP] + c.Belt[supply.RV] + c.Pack[supply.RV]
	}
	if td := r.d.Task; td != nil && td.Quests != nil {
		if v, ok := td.Quests.Read(int(p.Difficulty)); ok { // U flag position: the Eve of Destruction word (quest.BaalQuest)
			h.Baal = diffready.No
			if st, has := v.Quests[task.QBaal]; has && st.Stage >= task.StageDone {
				h.Baal = diffready.Yes
			}
		}
	}
	r.obMu.Lock()
	h.DeathsLastHour = r.dlog.since(r.e.Clock.Now().Add(-time.Hour), 0)
	r.obMu.Unlock()
	v, err := t.Check(target, h)
	if err != nil {
		set("difficulty readiness: " + err.Error())
		return v, err
	}
	set(v.Why())
	return v, nil
}

// wantedDifficultyV2 is wantedDifficulty under bot.diffReadyV2: the next difficulty only, only when ready, only with
// the owner's go, honoring the ban of a failed switch. Never goes down, never hops alone. It only reads the go: a hero
// in the field, a backoff or a new game that fails must not eat it (diffHopStarted consumes it).
func (r *Runner) wantedDifficultyV2(s *d2.Snapshot) string {
	next := nextDifficulty(s)
	r.diffGo.Keep(next)
	if next == "" {
		return ""
	}
	if until, ok := r.loop.diffBan[next]; ok && r.e.Clock.Now().Before(until) {
		return ""
	}
	v, err := r.diffVerdict(s, next)
	if err != nil {
		r.e.Log.Warn("difficulty readiness failed", "err", fmt.Sprint(err))
		return ""
	}
	if !v.Ready || !r.diffGo.Granted(next, r.goHero(s), r.e.Clock.Now()) {
		return ""
	}
	return next
}

// keySkillLevel is the base level of the build's main attack at the hero's level (the table's main_attack, data), for
// the readiness row's key_skill_min. -1 (not read, never blocks) when the skill table is not loaded, the table has no
// main_attack or it names a weapon attack that is not a skill. U: base points only, +skills from gear do not count.
func (r *Runner) keySkillLevel(p *d2.Player) int {
	if r.e.Data == nil {
		return -1
	}
	return keySkillLevelOf(r.buildName(p), p.Level, leveler.OwnedSkills(r.e.Data, p), func(n string) bool { return r.e.Data.SkillByName(n) != nil })
}

func keySkillLevelOf(build string, clvl int, owned map[string]int, isSkill func(string) bool) int {
	ma, err := gear.MainAttackAt(build, clvl)
	if err != nil || ma.Skill == "" || !isSkill(ma.Skill) {
		return -1
	}
	return owned[ma.Skill]
}

// diffHopStarted consumes the owner's go for want: the new game of the hop was made (ensureDifficulty). One go, one hop.
// s is the snapshot the hop was decided on: the go is taken for that hero only.
func (r *Runner) diffHopStarted(want string, s *d2.Snapshot) {
	if r.e.Tune.Bot.DiffReadyV2 != 0 {
		r.diffGo.Take(want, r.goHero(s), r.e.Clock.Now())
	}
}

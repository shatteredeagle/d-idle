package bot

import (
	"d2lod-bot/internal/d2"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Repeat-site rule (docs/plan/plan-v2.md 10.2, approved): two deaths in one area within the repeat-site window, or a
// death within the repeat-site gap of the previous one, ban that area for RepeatSiteBanMin and write a planner dump.
// It changes where the bot farms after a repeat death, nothing about how it fights. The ban reason is whyRepeat, which
// idleBreak treats like a bot fault, not a death streak: with the area as the only farm area the no-idle fallback
// lifts it after the short area wait (bot.areaIdleSec), so the bot never stands in town for the full ban.
const whyRepeat = "repeat-site"

const decisionKeep = 40

type decisionNote struct {
	At   time.Time `json:"at"`
	Text string    `json:"text"`
}

// noteDecision keeps the last decisionKeep planner/runner decisions for the planner dump.
func (r *Runner) noteDecision(text string) {
	r.decMu.Lock()
	r.decisions = append(r.decisions, decisionNote{r.e.Clock.Now(), text})
	if n := len(r.decisions) - decisionKeep; n > 0 {
		r.decisions = append([]decisionNote(nil), r.decisions[n:]...)
	}
	r.decMu.Unlock()
}

// repeatSite reports whether the death rec is a repeat site and how many deaths in its area fall in the window.
// Caller holds obMu. Town deaths never count.
func (r *Runner) repeatSite(rec DeathRecord) (repeat bool, inWindow int, why string) {
	t := r.e.Tune.Bot
	if t.RepeatSiteBanMin <= 0 || rec.Area == 0 || r.d.Plan.IsTown(rec.Area) {
		return false, 0, ""
	}
	win := time.Duration(max(t.RepeatSiteWindowMin, 1)) * time.Minute
	inWindow = r.dlog.since(rec.Time.Add(-win), rec.Area)
	var prev *DeathRecord
	for i := range r.dlog.Records {
		if rc := &r.dlog.Records[i]; rc.Time.Before(rec.Time) && (prev == nil || rc.Time.After(prev.Time)) {
			prev = rc
		}
	}
	gap := time.Duration(max(t.RepeatSiteGapSec, 1)) * time.Second
	switch {
	case inWindow >= 2:
		return true, inWindow, fmt.Sprintf("%d deaths in %d min", inWindow, int(win.Minutes()))
	case prev != nil && rec.Time.Sub(prev.Time) <= gap:
		return true, inWindow, fmt.Sprintf("death %ds after the previous one", int(rec.Time.Sub(prev.Time)/time.Second))
	}
	return false, inWindow, ""
}

// repeatEscalateWindow is how far back earlier repeat-site bans of the same area count for the escalation.
const repeatEscalateWindow = 12 * time.Hour

// repeatBanCap is the longest an escalated repeat-site ban gets.
const repeatBanCap = 6 * time.Hour

// repeatBanLen is the ban length for the repeat death rec: the base length, x4 for the second ban of the area within
// 12 h and x12 for the third and later (cap 6 h), when bot.repeatSiteEscalate is on. The count comes from the death log
// (RepeatBanMin on earlier records), so it survives a restart. Caller holds obMu.
func (r *Runner) repeatBanLen(rec DeathRecord) time.Duration {
	t := r.e.Tune.Bot
	base := time.Duration(t.RepeatSiteBanMin) * time.Minute
	if t.RepeatSiteEscalate <= 0 {
		return base
	}
	prior := 0
	for i := range r.dlog.Records {
		if rc := &r.dlog.Records[i]; rc.Area == rec.Area && rc.RepeatBanMin > 0 && rc.Time.Before(rec.Time) && rec.Time.Sub(rc.Time) <= repeatEscalateWindow {
			prior++
		}
	}
	mult := 1
	switch {
	case prior == 1:
		mult = 4
	case prior >= 2:
		mult = 12
	}
	return min(base*time.Duration(mult), max(repeatBanCap, base))
}

// markRepeatBan records the ban minutes on the death record. Caller holds obMu.
func (r *Runner) markRepeatBan(at time.Time, min int) {
	for i := len(r.dlog.Records) - 1; i >= 0; i-- {
		if r.dlog.Records[i].Time.Equal(at) {
			r.dlog.Records[i].RepeatBanMin = min
			return
		}
	}
}

// applyRepeatSite bans the area and writes the dump when rec is a repeat site. It runs after the spiral breaker, so a
// spiral and a repeat death on the same area end with the longer repeat-site ban.
func (r *Runner) applyRepeatSite(rec *recovery) {
	r.obMu.Lock()
	repeat, k, why := r.repeatSite(rec.rec)
	if !repeat {
		r.obMu.Unlock()
		return
	}
	ban := r.repeatBanLen(rec.rec)
	r.markRepeatBan(rec.rec.Time, int(ban/time.Minute))
	if err := r.dlog.save(r.cfg.DeathLogPath); err != nil {
		r.e.Log.Warn("death log not saved", "err", err)
	}
	r.skipAreaLocked(rec.area, ban, whyRepeat)
	r.setReaction(rec.rec.Time, fmt.Sprintf("repeat site (%s): area banned for %d min", why, int(ban.Minutes())))
	r.obMu.Unlock()
	r.loop.stepArea = 0
	path := r.writePlannerDump(rec, why)
	if path == "" {
		path = "none"
	}
	r.e.Log.Warn(fmt.Sprintf("death: repeat-site ban area=%d name=%s deaths=%d window=%dmin planner dump=%s",
		rec.area, rec.rec.AreaName, k, r.e.Tune.Bot.RepeatSiteWindowMin, path), "why", why, "banMin", int(ban.Minutes()))
}

type areaState struct {
	Area      int    `json:"area"`
	Name      string `json:"name"`
	BanLeft   int    `json:"banLeftSec,omitempty"`
	BanReason string `json:"banReason,omitempty"`
	Allowed   string `json:"notAllowed,omitempty"` // areaAllowed text, empty when the planner may enter
}

type plannerDump struct {
	Time        time.Time      `json:"time"`
	Trigger     string         `json:"trigger"`
	Death       DeathRecord    `json:"death"`
	Goal        any            `json:"goal,omitempty"`
	Runner      map[string]any `json:"runner"`
	Kit         *Kit           `json:"kit,omitempty"`
	Areas       []areaState    `json:"areas"`
	Decisions   []decisionNote `json:"decisions"`
	RecentDeath []DeathRecord  `json:"recentDeaths"`
}

// writePlannerDump writes plannerdump-<ts>.json next to the death log (the instance data dir) and returns its path,
// or "" when there is no data dir or the write failed.
func (r *Runner) writePlannerDump(rec *recovery, trigger string) string {
	if r.cfg.DeathLogPath == "" {
		return ""
	}
	now := r.e.Clock.Now()
	d := plannerDump{Time: now, Trigger: trigger, Death: rec.rec, Runner: map[string]any{"stepArea": r.loop.stepArea, "stepAreaName": r.areaName(r.loop.stepArea), "stepLevel": r.loop.stepLevel}}
	r.mu.Lock()
	if r.goal != nil {
		d.Goal = *r.goal
	}
	d.Runner["state"], d.Runner["activity"], d.Runner["message"] = string(r.st), r.activity, r.message
	r.mu.Unlock()
	s := r.latest()
	if s != nil && s.Player != nil {
		k := r.kitOf(s)
		d.Kit = &k
		d.Runner["clvl"], d.Runner["area"] = s.Player.Level, s.Player.Area
	}
	seen := map[int]bool{}
	var ids []int
	for _, st := range r.d.Plan.Steps[difficultyName(planDiff(s))] {
		if st.Area != 0 && !seen[st.Area] {
			seen[st.Area] = true
			ids = append(ids, st.Area)
		}
	}
	r.obMu.Lock()
	for a := range r.avoid {
		if !seen[a] {
			seen[a] = true
			ids = append(ids, a)
		}
	}
	sort.Ints(ids)
	for _, a := range ids {
		as := areaState{Area: a, Name: r.areaName(a), BanReason: r.avoidWhy[a]}
		if until, ok := r.avoid[a]; ok && until.After(now) {
			as.BanLeft = int(until.Sub(now) / time.Second)
		} else {
			as.BanReason = ""
		}
		d.Areas = append(d.Areas, as)
	}
	n := len(r.dlog.Records)
	d.RecentDeath = append(d.RecentDeath, r.dlog.Records[max(n-5, 0):]...)
	r.obMu.Unlock()
	if s != nil && s.Player != nil {
		for i := range d.Areas {
			d.Areas[i].Allowed = r.areaAllowed(s, d.Areas[i].Area)
		}
	}
	r.decMu.Lock()
	d.Decisions = append(d.Decisions, r.decisions...)
	r.decMu.Unlock()
	b, err := json.MarshalIndent(d, "", " ")
	if err != nil {
		return ""
	}
	path := filepath.Join(filepath.Dir(r.cfg.DeathLogPath), "plannerdump-"+now.UTC().Format("20060102T150405Z")+".json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		r.e.Log.Warn("planner dump not saved", "err", err)
		return ""
	}
	return path
}

func planDiff(s *d2.Snapshot) d2.Difficulty {
	if s == nil || s.Player == nil {
		return d2.Normal
	}
	return s.Player.Difficulty
}

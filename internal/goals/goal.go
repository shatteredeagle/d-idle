// Package goals is the goal model and manager of d2lod-bot: the user sets goals, a Runner
// (the bot, or the simulator in goals/sim) plays them. The manager owns the queue, the
// scheduler, persistence, statistics and the event log. It never touches the game.
package goals

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Type string

const (
	ReachLevel  Type = "reach_level"
	FarmArea    Type = "farm_area"
	KillBoss    Type = "kill_boss"
	CollectItem Type = "collect_item"
	Idle        Type = "idle"
	// Campaign plays a quest campaign (internal/task): Params.Template names it. Progress comes from the runner.
	Campaign Type = "campaign"
	// Runs plays N farm runs of a named boss run (internal/boss farm loop): Params.Area is the run name or "auto" (the
	// scheduler picks), Params.Count the number of runs, Params.UntilMins an optional time limit. The runner reports
	// one boss kill per completed run, which is the headline counter.
	Runs Type = "runs"
)

var allTypes = []Type{ReachLevel, FarmArea, KillBoss, CollectItem, Idle, Campaign, Runs}

type State string

const (
	StateQueued  State = "queued"
	StateActive  State = "active"
	StatePaused  State = "paused" // held by the user; the scheduler skips it
	StateDone    State = "done"
	StateFailed  State = "failed"
	StateSkipped State = "skipped"
)

// Finished reports whether the goal will not run again unless reset.
func (s State) Finished() bool { return s == StateDone || s == StateFailed || s == StateSkipped }

// ItemFilter selects items. Empty fields match anything; Name is a case-insensitive substring.
type ItemFilter struct {
	Name    string `json:"name,omitempty"`
	Quality string `json:"quality,omitempty"` // normal, magic, rare, set, unique, crafted, any
	Type    string `json:"type,omitempty"`    // rune, gem, charm, ring, ...
}

func (f ItemFilter) Empty() bool {
	return f.Name == "" && (f.Quality == "" || f.Quality == "any") && f.Type == ""
}

func (f ItemFilter) Match(it Item) bool {
	if f.Name != "" && !strings.Contains(strings.ToLower(it.Name), strings.ToLower(f.Name)) {
		return false
	}
	if f.Quality != "" && f.Quality != "any" && !strings.EqualFold(f.Quality, it.Quality) {
		return false
	}
	if f.Type != "" && !strings.EqualFold(f.Type, it.Type) {
		return false
	}
	return true
}

func (f ItemFilter) String() string {
	var p []string
	if f.Quality != "" && f.Quality != "any" {
		p = append(p, f.Quality)
	}
	if f.Name != "" {
		p = append(p, f.Name)
	} else if f.Type != "" {
		p = append(p, f.Type)
	}
	if len(p) == 0 {
		return "any item"
	}
	return strings.Join(p, " ")
}

// Params holds the parameters of every goal type; each type reads only its own fields.
type Params struct {
	Level      int        `json:"level,omitempty"`      // reach_level: target character level
	Area       string     `json:"area,omitempty"`       // farm_area: area name or run key (see Catalog)
	AreaID     int        `json:"areaId,omitempty"`     // farm_area: optional numeric area id
	Boss       string     `json:"boss,omitempty"`       // kill_boss
	Count      int        `json:"count,omitempty"`      // kill_boss: kills; collect_item: items
	Item       ItemFilter `json:"item,omitempty"`       // collect_item (required), farm_area (counts drops)
	UntilKills int        `json:"untilKills,omitempty"` // farm_area limits; all 0 = endless
	UntilMins  int        `json:"untilMinutes,omitempty"`
	UntilDrops int        `json:"untilDrops,omitempty"`
	Minutes    int        `json:"minutes,omitempty"`    // idle: length; 0 with a schedule = rest in every window
	Difficulty string     `json:"difficulty,omitempty"` // normal, nightmare, hell; empty = bot decides
	Template   string     `json:"template,omitempty"`   // campaign: "normal_act1" (default)
	Cain       bool       `json:"cain,omitempty"`       // campaign: include the optional Search for Cain chain
	Chain      bool       `json:"chain,omitempty"`      // campaign (*_all): on completion queue the next difficulty's campaign
}

// Window is a recurring weekly time window in the manager clock's local time.
// Start > End wraps past midnight; the part after midnight belongs to the day the window began.
type Window struct {
	Days  []int  `json:"days,omitempty"` // 0 = Sunday .. 6; empty = every day
	Start string `json:"start"`          // "HH:MM"
	End   string `json:"end"`
}

func parseHM(s string) (int, error) {
	h, m, ok := strings.Cut(s, ":")
	if !ok {
		return 0, fmt.Errorf("bad time %q, want HH:MM", s)
	}
	hh, e1 := strconv.Atoi(h)
	mm, e2 := strconv.Atoi(m)
	if e1 != nil || e2 != nil || hh < 0 || hh > 23 || mm < 0 || mm > 59 || len(m) != 2 || len(h) != 2 {
		return 0, fmt.Errorf("bad time %q, want HH:MM", s)
	}
	return hh*60 + mm, nil
}

func (w Window) validate() error {
	a, err := parseHM(w.Start)
	if err != nil {
		return err
	}
	b, err := parseHM(w.End)
	if err != nil {
		return err
	}
	if a == b {
		return fmt.Errorf("window %s-%s is empty", w.Start, w.End)
	}
	for _, d := range w.Days {
		if d < 0 || d > 6 {
			return fmt.Errorf("bad weekday %d", d)
		}
	}
	return nil
}

func (w Window) contains(t time.Time) bool {
	a, e1 := parseHM(w.Start)
	b, e2 := parseHM(w.End)
	if e1 != nil || e2 != nil {
		return false
	}
	day := func(d time.Weekday) bool {
		if len(w.Days) == 0 {
			return true
		}
		for _, x := range w.Days {
			if x == int(d) {
				return true
			}
		}
		return false
	}
	now := t.Hour()*60 + t.Minute()
	if a < b {
		return now >= a && now < b && day(t.Weekday())
	}
	if now >= a {
		return day(t.Weekday())
	}
	return now < b && day((t.Weekday()+6)%7)
}

// Progress is maintained by the manager from runner events and status.
type Progress struct {
	Current       int     `json:"current"` // headline metric, meaning depends on the type
	Target        int     `json:"target"`  // 0 when endless
	Percent       float64 `json:"percent"` // 0..100, meaningful when Target > 0
	StartLevel    int     `json:"startLevel,omitempty"`
	Kills         int     `json:"kills"`
	BossKills     int     `json:"bossKills"`
	Drops         int     `json:"drops"` // drops matching the goal's filter (any drop when no filter)
	Deaths        int     `json:"deaths"`
	ActiveSeconds int64   `json:"activeSeconds"`

	// Optional tracking fields (docs/goals/tracking.md section 3). All are omitted when empty, so version 1 files and
	// old clients are unaffected; the manager does not compute them yet, a Runner-side or state-goal evaluator fills them.
	Steps      []Step `json:"steps,omitempty"`
	Phase      string `json:"phase,omitempty"`      // "travel", "fight", "turn-in", "cube"
	ETASeconds int64  `json:"etaSeconds,omitempty"` // 0 unknown
	Source     string `json:"source,omitempty"`     // how Current was obtained: "memory", "events" or "mixed"
	Verified   bool   `json:"verified,omitempty"`   // confirmed by a memory read (true) or inferred from events (false)
}

// Step is one sub-step of a goal (3 of 5 seals); State is queued, active, done, failed or skipped.
type Step struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	State   State  `json:"state"`
	Current int    `json:"current,omitempty"`
	Target  int    `json:"target,omitempty"`
}

type Goal struct {
	ID          string     `json:"id"`
	Type        Type       `json:"type"`
	Title       string     `json:"title"`
	Params      Params     `json:"params"`
	Priority    int        `json:"priority"` // higher runs first; ties keep queue order
	Schedule    []Window   `json:"schedule,omitempty"`
	State       State      `json:"state"`
	Endless     bool       `json:"endless"`
	Progress    Progress   `json:"progress"`
	Seed        string     `json:"seed,omitempty"`       // key of the --goals entry that created it (AddSeeded); never set by dashboard adds
	Note        string     `json:"note,omitempty"`       // last reason, e.g. why it failed or paused
	HeldAt      int        `json:"heldAt,omitempty"`     // legacy: read once by restore to migrate into Manager.locked, never written
	ResumeAt    *time.Time `json:"resumeAt,omitempty"`   // a goal paused after deaths or a failed resist gate resumes by itself at this time
	DeathHolds  int        `json:"deathHolds,omitempty"` // death pauses so far; the rest before the next resume doubles with it
	DeathBase   int        `json:"deathBase,omitempty"`  // Progress.Deaths at the last resume: MaxDeaths counts from here
	NextIDs     []string   `json:"nextIds,omitempty"`    // goals a completed chained campaign enqueued (idempotence marker)
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

func (g *Goal) clone() Goal {
	c := *g
	c.Schedule = append([]Window(nil), g.Schedule...)
	c.NextIDs = append([]string(nil), g.NextIDs...)
	for i := range c.Schedule {
		c.Schedule[i].Days = append([]int(nil), g.Schedule[i].Days...)
	}
	if g.StartedAt != nil {
		t := *g.StartedAt
		c.StartedAt = &t
	}
	if g.CompletedAt != nil {
		t := *g.CompletedAt
		c.CompletedAt = &t
	}
	return c
}

func (g *Goal) inWindow(t time.Time) bool {
	if len(g.Schedule) == 0 {
		return true
	}
	for _, w := range g.Schedule {
		if w.contains(t) {
			return true
		}
	}
	return false
}

func validDifficulty(d string) bool {
	return d == "" || d == "normal" || d == "nightmare" || d == "hell"
}

// Normalize fills defaults and derived fields and validates. It is called on every add/update.
func (g *Goal) Normalize() error {
	p := &g.Params
	p.Area = strings.TrimSpace(p.Area)
	p.Boss = strings.TrimSpace(p.Boss)
	p.Item.Name = strings.TrimSpace(p.Item.Name)
	p.Item.Quality = strings.ToLower(strings.TrimSpace(p.Item.Quality))
	p.Item.Type = strings.ToLower(strings.TrimSpace(p.Item.Type))
	p.Difficulty = strings.ToLower(strings.TrimSpace(p.Difficulty))
	if !validDifficulty(p.Difficulty) {
		return fmt.Errorf("difficulty must be normal, nightmare or hell")
	}
	if g.Priority < -10 || g.Priority > 10 {
		return fmt.Errorf("priority must be between -10 and 10")
	}
	for _, w := range g.Schedule {
		if err := w.validate(); err != nil {
			return err
		}
	}
	for _, n := range []int{p.Count, p.UntilKills, p.UntilMins, p.UntilDrops, p.Minutes, p.Level, p.AreaID} {
		if n < 0 {
			return fmt.Errorf("numbers must not be negative")
		}
	}
	switch g.Type {
	case ReachLevel:
		if p.Level < 1 || p.Level > 99 {
			return fmt.Errorf("level must be between 1 and 99")
		}
	case FarmArea:
		if p.Area == "" && p.AreaID == 0 {
			return fmt.Errorf("farm_area needs an area")
		}
	case KillBoss:
		if p.Boss == "" {
			return fmt.Errorf("kill_boss needs a boss")
		}
		if p.Count == 0 {
			p.Count = 1
		}
	case CollectItem:
		if p.Item.Empty() {
			return fmt.Errorf("collect_item needs an item filter (name, quality or type)")
		}
		if p.Count == 0 {
			p.Count = 1
		}
	case Runs:
		if p.Area == "" {
			p.Area = "auto"
		}
		if p.Count == 0 {
			p.Count = 1
		}
	case Idle:
		if p.Minutes == 0 && len(g.Schedule) == 0 {
			return fmt.Errorf("idle needs minutes or a schedule window")
		}
	case Campaign:
		if p.Template == "" {
			p.Template = CampaignNormalAct1
		}
		if CampaignQuests(p.Template, p.Cain) == 0 {
			return fmt.Errorf("unknown campaign template %q", p.Template)
		}
		if d := templateDifficulty(p.Template); d != "" {
			if p.Difficulty != "" && p.Difficulty != d {
				return fmt.Errorf("template %s plays in %s, not %s", p.Template, d, p.Difficulty)
			}
			p.Difficulty = d
		}
	default:
		return fmt.Errorf("unknown goal type %q", g.Type)
	}
	if len(g.Title) > 120 {
		return fmt.Errorf("title too long")
	}
	g.Title = strings.TrimSpace(g.Title)
	if g.Title == "" {
		g.Title = g.describe()
	}
	g.Endless = g.target() == 0
	if g.State == "" {
		g.State = StateQueued
	}
	return nil
}

func (g *Goal) describe() string {
	p := g.Params
	switch g.Type {
	case ReachLevel:
		return fmt.Sprintf("Reach level %d", p.Level)
	case FarmArea:
		area := p.Area
		if area == "" {
			area = "area " + strconv.Itoa(p.AreaID)
		}
		var lim []string
		if p.UntilKills > 0 {
			lim = append(lim, fmt.Sprintf("%d kills", p.UntilKills))
		}
		if p.UntilMins > 0 {
			lim = append(lim, fmt.Sprintf("%d min", p.UntilMins))
		}
		if p.UntilDrops > 0 {
			lim = append(lim, fmt.Sprintf("%d drops", p.UntilDrops))
		}
		if len(lim) == 0 {
			return "Farm " + area
		}
		return "Farm " + area + " until " + strings.Join(lim, " or ")
	case KillBoss:
		if p.Count > 1 {
			return fmt.Sprintf("Kill %s x%d", p.Boss, p.Count)
		}
		return "Kill " + p.Boss
	case CollectItem:
		return fmt.Sprintf("Collect %d x %s", p.Count, p.Item)
	case Runs:
		return fmt.Sprintf("Run %s x%d", p.Area, p.Count)
	case Idle:
		if p.Minutes > 0 {
			return fmt.Sprintf("Rest for %d min", p.Minutes)
		}
		return "Rest during scheduled windows"
	case Campaign:
		switch g.Params.Template {
		case CampaignNightmareAll:
			return "Campaign: Nightmare"
		case CampaignHellAll:
			return "Campaign: Hell"
		}
		return "Campaign: Normal Act 1"
	}
	return string(g.Type)
}

// target is the headline target, 0 when the goal has no end condition.
func (g *Goal) target() int {
	p := g.Params
	switch g.Type {
	case ReachLevel:
		return p.Level
	case KillBoss, CollectItem, Runs:
		return p.Count
	case FarmArea:
		switch {
		case p.UntilKills > 0:
			return p.UntilKills
		case p.UntilDrops > 0:
			return p.UntilDrops
		case p.UntilMins > 0:
			return p.UntilMins
		}
	case Idle:
		return p.Minutes
	case Campaign:
		return CampaignQuests(p.Template, p.Cain)
	}
	return 0
}

// refreshProgress recomputes Current/Target/Percent and reports whether the goal is complete.
func (g *Goal) refreshProgress() (done bool) {
	p, pr := g.Params, &g.Progress
	frac := func(cur, tgt int) float64 {
		if tgt <= 0 {
			return 0
		}
		f := float64(cur) / float64(tgt)
		if f > 1 {
			f = 1
		}
		return f
	}
	mins := int(pr.ActiveSeconds / 60)
	best := 0.0
	switch g.Type {
	case ReachLevel:
		pr.Target = p.Level
		span := p.Level - pr.StartLevel
		if pr.Current >= p.Level {
			done, best = true, 1
		} else if span > 0 {
			best = float64(pr.Current-pr.StartLevel) / float64(span)
		}
	case KillBoss:
		pr.Current, pr.Target = pr.BossKills, p.Count
		best = frac(pr.Current, pr.Target)
		done = pr.Current >= pr.Target
	case Runs:
		pr.Current, pr.Target = pr.BossKills, p.Count
		best = frac(pr.Current, pr.Target)
		done = pr.Current >= pr.Target
		if p.UntilMins > 0 {
			best = max(best, frac(int(pr.ActiveSeconds), p.UntilMins*60))
			done = done || pr.ActiveSeconds >= int64(p.UntilMins)*60
		}
	case CollectItem:
		pr.Target = p.Count
		pr.Current = pr.Drops
		best = frac(pr.Current, pr.Target)
		done = pr.Current >= pr.Target
	case FarmArea:
		pr.Target = g.target()
		switch {
		case p.UntilKills > 0:
			pr.Current = pr.Kills
		case p.UntilDrops > 0:
			pr.Current = pr.Drops
		default:
			pr.Current = mins
		}
		if p.UntilKills > 0 {
			best = max(best, frac(pr.Kills, p.UntilKills))
			done = done || pr.Kills >= p.UntilKills
		}
		if p.UntilDrops > 0 {
			best = max(best, frac(pr.Drops, p.UntilDrops))
			done = done || pr.Drops >= p.UntilDrops
		}
		if p.UntilMins > 0 {
			best = max(best, frac(int(pr.ActiveSeconds), p.UntilMins*60))
			done = done || pr.ActiveSeconds >= int64(p.UntilMins)*60
		}
	case Campaign:
		// Current/Target/Percent are the runner's quest count (MergeProgress); done when every quest is.
		if pr.Target == 0 {
			pr.Target = g.target()
		}
		best = frac(pr.Current, pr.Target)
		done = pr.Target > 0 && pr.Current >= pr.Target
	case Idle:
		pr.Target = p.Minutes
		pr.Current = mins
		if p.Minutes > 0 {
			best = frac(int(pr.ActiveSeconds), p.Minutes*60)
			done = pr.ActiveSeconds >= int64(p.Minutes)*60
		}
	}
	if g.Endless {
		pr.Target, pr.Percent = 0, 0
	} else {
		pr.Percent = float64(int(best*1000)) / 10
	}
	return done
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// Campaign templates (every act has nodes now; a count of 0 would reject the goal as unknown).
const (
	CampaignNormalAct1 = "normal_act1"
	CampaignNormalAct2 = "normal_act2" // Act 2 alone (a character already in Act 2)
	CampaignNormalAct3 = "normal_act3"
	CampaignNormalAct4 = "normal_act4"
	CampaignNormalAct5 = "normal_act5"
	CampaignNormalAll  = "normal_all" // Acts 1 to 5 in order
	// The same quests replayed per difficulty (quests reset per difficulty); the goal's difficulty defaults to it.
	CampaignNightmareAll = "nightmare_all"
	CampaignHellAll      = "hell_all"
)

// CampaignTemplates lists every template name, in act order.
var CampaignTemplates = []string{CampaignNormalAct1, CampaignNormalAct2, CampaignNormalAct3, CampaignNormalAct4, CampaignNormalAct5, CampaignNormalAll, CampaignNightmareAll, CampaignHellAll}

// Per-act quest counts. internal/task has a test that keeps them equal to len(Entries) of the matching builders
// (task.Template); a count of 0 means the act has no nodes yet.
func actQuests(act int, cain bool) int {
	switch act {
	case 1:
		if cain {
			return 7
		}
		return 6
	case 2:
		return 8 // task.Act2Options{Merc: true}
	case 3:
		return 8 // task.Act3Options{Merc: true}
	case 4:
		return 3 // Izual, Diablo, hop (task.Act4Options{})
	case 5:
		return 3 // Siege, Ancients, Baal (task.Act5Options{})
	}
	return 0
}

// templateDifficulty is the difficulty a template plays in.
func templateDifficulty(template string) string {
	switch template {
	case CampaignNormalAct1, CampaignNormalAct2, CampaignNormalAct3, CampaignNormalAct4, CampaignNormalAct5, CampaignNormalAll:
		return "normal"
	case CampaignNightmareAll:
		return "nightmare"
	case CampaignHellAll:
		return "hell"
	}
	return ""
}

// CampaignQuests is the number of quests of a campaign template (0 for an unknown one).
func CampaignQuests(template string, cain bool) int {
	switch template {
	case "", CampaignNormalAct1:
		return actQuests(1, cain)
	case CampaignNormalAct2:
		return actQuests(2, cain)
	case CampaignNormalAct3:
		return actQuests(3, cain)
	case CampaignNormalAct4:
		return actQuests(4, cain)
	case CampaignNormalAct5:
		return actQuests(5, cain)
	case CampaignNormalAll, CampaignNightmareAll, CampaignHellAll:
		n := 0
		for act := 1; act <= 5; act++ {
			n += actQuests(act, cain)
		}
		return n
	}
	return 0
}

// MergeProgress copies the task layer's tracking fields over g's (Current, Target, Percent, Steps, Phase, ETA, Source,
// Verified); the counters the manager keeps itself (kills, deaths, active seconds) stay.
func (p *Progress) MergeProgress(from Progress) {
	p.Current, p.Target, p.Percent = from.Current, from.Target, from.Percent
	p.Steps, p.Phase, p.ETASeconds, p.Source, p.Verified = from.Steps, from.Phase, from.ETASeconds, from.Source, from.Verified
}

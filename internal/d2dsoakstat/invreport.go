package d2dsoakstat

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// DefaultHoldout is the holdout seed convention (docs/geninv.md): seeds 16 to 23 are never used for tuning, fitting or
// diagnosis; the gate run reports them as their own group beside the tuning seeds (1 to 8 in baselines-early).
const DefaultHoldout = "16-23"

// Group names of the report.
const (
	GroupTuning  = "tuning"
	GroupHoldout = "holdout"
)

var seedInName = regexp.MustCompile(`(?:seed[-_]?|base)(\d+)`)

// SeedOf returns the seed of a run: its Seed field, else the number in its name (seed-N, baseN), else 0 (unknown).
func SeedOf(r Run) int {
	if r.Seed > 0 {
		return r.Seed
	}
	if m := seedInName.FindStringSubmatch(r.Name); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

// ParseSeedSet parses "16-23,30" into a set; "" is the empty set.
func ParseSeedSet(s string) (map[int]bool, error) {
	set := map[int]bool{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		a, b, isRange := strings.Cut(p, "-")
		lo, e1 := strconv.Atoi(a)
		hi := lo
		var e2 error
		if isRange {
			hi, e2 = strconv.Atoi(b)
		}
		if e1 != nil || e2 != nil || hi < lo || hi-lo > 999 {
			return nil, fmt.Errorf("bad seed set %q", p)
		}
		for i := lo; i <= hi; i++ {
			set[i] = true
		}
	}
	return set, nil
}

// InvGroup is one seed group of the report.
type InvGroup struct {
	Name     string   `json:"name"`
	Seeds    []int    `json:"seeds"`
	Hours    float64  `json:"hours"`
	Kills    int      `json:"kills"`
	Deaths   int      `json:"deaths"`
	XPPerH   float64  `json:"xpPerHour"`
	Hang     int      `json:"hang"`
	Stalls   int      `json:"stalls"`
	WedgeMax float64  `json:"wedgeMaxMin"`
	ProgMax  float64  `json:"noProgressMaxMin"`
	Restarts int      `json:"botRestarts"`
	Fresh    int      `json:"freshGames"`
	Chickens int      `json:"chickens"`
	Verdict  string   `json:"verdict"`
	Violated []string `json:"violated,omitempty"`
}

// InvRow is one seed of the report.
type InvRow struct {
	Run   string `json:"run"`
	Seed  int    `json:"seed"`
	Group string `json:"group"`
	Invariants
}

// Coverage is the "proven on" data of a report (axes of docs/research/bot-generality.md; the README line is owned by gen-claims).
type Coverage struct {
	TuningSeeds  []int   `json:"tuningSeeds"`
	HoldoutSeeds []int   `json:"holdoutSeeds"`
	Class        string  `json:"class"`
	Acts         []int   `json:"acts"`
	Difficulty   string  `json:"difficulty"`
	Engine       string  `json:"engine"`
	ClvlMin      int     `json:"clvlMin"`
	ClvlMax      int     `json:"clvlMax"`
	Hours        float64 `json:"virtualHours"`
}

// InvReport is the invariants section of a gate or extract report.
type InvReport struct {
	Limits   InvLimits  `json:"limits"`
	Rows     []InvRow   `json:"rows"`
	Groups   []InvGroup `json:"groups"`
	Coverage Coverage   `json:"coverage"`
	Verdict  string     `json:"verdict"`
	Violated []string   `json:"violated,omitempty"` // "seed N: CODE"
}

// BuildInvReport groups the runs (holdout seeds against the rest), judges each group and the whole set. A run without
// a counter block is NO-DATA. The set verdict is VIOLATED when any seed is, PASS only when every seed is PASS, else NO-DATA.
func BuildInvReport(runs []Run, holdout map[int]bool) InvReport {
	rep := InvReport{Limits: DefaultInvLimits()}
	byGroup := map[string][]Run{}
	acts := map[int]bool{}
	cov := &rep.Coverage
	engines := map[string]bool{}
	first := true
	for _, r := range runs {
		seed := SeedOf(r)
		g := GroupTuning
		if holdout[seed] {
			g = GroupHoldout
		}
		inv := Invariants{Verdict: InvNoData, Exit: "open", ExitNote: "no invariant data for this run"}
		if r.Inv != nil {
			inv = *r.Inv
		}
		rep.Rows = append(rep.Rows, InvRow{Run: r.Name, Seed: seed, Group: g, Invariants: inv})
		byGroup[g] = append(byGroup[g], r)
		if inv.Verdict == InvViolated {
			for _, v := range inv.Violated {
				rep.Violated = append(rep.Violated, fmt.Sprintf("%s: %s", runLabel(r.Name, seed), v))
			}
		}
		if inv.Verdict == InvNoData {
			continue
		}
		for _, a := range inv.Acts {
			acts[a] = true
		}
		if inv.Engine != "" {
			engines[inv.Engine] = true
		}
		if first {
			cov.ClvlMin, cov.ClvlMax, first = inv.ClvlMin, inv.ClvlMax, false
		}
		cov.ClvlMin, cov.ClvlMax = min(cov.ClvlMin, inv.ClvlMin), max(cov.ClvlMax, inv.ClvlMax)
		cov.Hours += inv.VirtMin / 60
		if r.Cover != nil {
			cov.Class, cov.Difficulty = r.Cover.Class, r.Cover.Difficulty
		}
		if seed > 0 {
			if g == GroupHoldout {
				cov.HoldoutSeeds = append(cov.HoldoutSeeds, seed)
			} else {
				cov.TuningSeeds = append(cov.TuningSeeds, seed)
			}
		}
	}
	sort.Ints(cov.TuningSeeds)
	sort.Ints(cov.HoldoutSeeds)
	for a := range acts {
		cov.Acts = append(cov.Acts, a)
	}
	sort.Ints(cov.Acts)
	cov.Engine = "unknown"
	if len(engines) == 1 {
		for e := range engines {
			cov.Engine = e
		}
	}
	if cov.Class == "" {
		cov.Class = "unknown"
	}
	if cov.Difficulty == "" {
		cov.Difficulty = "unknown"
	}
	for _, name := range []string{GroupTuning, GroupHoldout} {
		if rs := byGroup[name]; len(rs) > 0 {
			rep.Groups = append(rep.Groups, groupOf(name, rs))
		}
	}
	rep.Verdict = setVerdict(rep.Rows)
	return rep
}

func runLabel(name string, seed int) string {
	if seed > 0 {
		return "seed " + strconv.Itoa(seed)
	}
	return name
}

func setVerdict(rows []InvRow) string {
	if len(rows) == 0 {
		return InvNoData
	}
	v := InvPass
	for _, r := range rows {
		switch r.Verdict {
		case InvViolated:
			return InvViolated
		case InvPass:
		default:
			v = InvNoData
		}
	}
	return v
}

func groupOf(name string, rs []Run) InvGroup {
	g := InvGroup{Name: name}
	var rows []InvRow
	set := map[string]bool{}
	var xp float64
	for _, r := range rs {
		inv := Invariants{Verdict: InvNoData}
		if r.Inv != nil {
			inv = *r.Inv
		}
		rows = append(rows, InvRow{Invariants: inv})
		if s := SeedOf(r); s > 0 {
			g.Seeds = append(g.Seeds, s)
		}
		if inv.Verdict == InvNoData {
			continue
		}
		g.Hours += inv.VirtMin / 60
		g.Kills += inv.Kills
		g.Deaths += inv.Deaths
		xp += float64(r.XP)
		g.Hang += inv.PaceFrozen + inv.CombatHolding
		g.Stalls += inv.StallWarns + inv.StallExits
		g.WedgeMax = max(g.WedgeMax, inv.WedgeMin)
		g.ProgMax = max(g.ProgMax, inv.NoProgressMin)
		g.Restarts += inv.BotRestarts
		g.Fresh += inv.FreshGames
		g.Chickens += inv.Chickens
		for _, v := range inv.Violated {
			set[v] = true
		}
	}
	sort.Ints(g.Seeds)
	if g.Hours > 0 {
		g.XPPerH = xp / g.Hours
	}
	for v := range set {
		g.Violated = append(g.Violated, v)
	}
	sort.Strings(g.Violated)
	g.Verdict = setVerdict(rows)
	return g
}

// SeedList formats seeds as ranges ("1-8,12").
func SeedList(s []int) string {
	if len(s) == 0 {
		return "none"
	}
	var parts []string
	for i := 0; i < len(s); {
		j := i
		for j+1 < len(s) && s[j+1] == s[j]+1 {
			j++
		}
		if j > i {
			parts = append(parts, fmt.Sprintf("%d-%d", s[i], s[j]))
		} else {
			parts = append(parts, strconv.Itoa(s[i]))
		}
		i = j + 1
	}
	return strings.Join(parts, ",")
}

func verdictText(v string, violated []string) string {
	if v == InvViolated {
		return v + "(" + strings.Join(violated, ",") + ")"
	}
	return v
}

// Text renders the report: one line per seed, one per group, the coverage line and the INVARIANTS verdict.
func (rep InvReport) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "invariants (docs/geninv.md): is the run sound, apart from is it good\n")
	fmt.Fprintf(&b, "%-26s %-7s %5s %4s %5s %-9s %-9s %-11s %-10s %-9s %-7s %s\n", "run", "group", "vmin", "hang", "stall", "wedge", "noprog", "deaths/1000k", "restarts", "exit", "unrec", "verdict")
	for _, r := range rep.Rows {
		if r.Verdict == InvNoData {
			fmt.Fprintf(&b, "%-26s %-7s %s  %s\n", r.Run, r.Group, InvNoData, r.ExitNote)
			continue
		}
		fmt.Fprintf(&b, "%-26s %-7s %5.0f %4d %5d %3.0fm/%-4d %6.0fm   %4.1f (%d/%dk) fresh%d/chk%d/bot%d %-9s %-7d %s\n",
			r.Run, r.Group, r.VirtMin, r.PaceFrozen+r.CombatHolding, r.StallWarns+r.StallExits, r.WedgeMin, min(r.WedgeLines, 9999),
			r.NoProgressMin, r.DeathsPer1000, r.Deaths, r.Kills, r.FreshGames, r.Chickens, r.BotRestarts, r.Exit, r.Unrecovered, verdictText(r.Verdict, r.Violated))
		if r.ExitNote != "" && (r.Exit == "early" || r.Exit == "crash") {
			fmt.Fprintf(&b, "    exit: %s\n", r.ExitNote)
		}
	}
	for _, g := range rep.Groups {
		fmt.Fprintf(&b, "group %-8s seeds %s: %.1f virtual h, %d kills, %d deaths, %.0f XP/h, hang %d, stall rungs %d, wedge max %.0f min, no-progress max %.0f min, bot restarts %d, fresh games %d, chickens %d: %s\n",
			g.Name, SeedList(g.Seeds), g.Hours, g.Kills, g.Deaths, g.XPPerH, g.Hang, g.Stalls, g.WedgeMax, g.ProgMax, g.Restarts, g.Fresh, g.Chickens, verdictText(g.Verdict, g.Violated))
	}
	if !hasGroup(rep.Groups, GroupHoldout) {
		fmt.Fprintf(&b, "group holdout: none ran (convention: seeds %s; docs/geninv.md)\n", DefaultHoldout)
	}
	c := rep.Coverage
	acts := make([]string, len(c.Acts))
	for i, a := range c.Acts {
		acts[i] = strconv.Itoa(a)
	}
	if len(acts) == 0 {
		acts = []string{"unknown"}
	}
	fmt.Fprintf(&b, "COVERAGE (proven on): seeds %s (tuning), %s (holdout); class %s; act %s; difficulty %s; engine %s; clvl %d-%d; %.1f virtual h. Outside these axes the claim is UNPROVEN.\n",
		SeedList(c.TuningSeeds), SeedList(c.HoldoutSeeds), c.Class, strings.Join(acts, ","), c.Difficulty, c.Engine, c.ClvlMin, c.ClvlMax, c.Hours)
	fmt.Fprintf(&b, "INVARIANTS: %s\n", verdictText(rep.Verdict, rep.Violated))
	return b.String()
}

func hasGroup(gs []InvGroup, name string) bool {
	for _, g := range gs {
		if g.Name == name {
			return true
		}
	}
	return false
}

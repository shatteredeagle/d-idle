package drill

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/ladder"
)

// T0Lab is the instance as the T0 drill uses it: the town code of TownLab (the wired points live there) plus the run
// bracket of the CRUTCH rule (docs/t0-slice.md).
type T0Lab interface {
	TownLab
	// StartRun writes the run-start marker (d2cheat runstart): setup cheats go before it, none after.
	StartRun(ctx context.Context, run, reason string) error
	// WritesAfterStart counts the ledger writes after the marker of run ("writes after start: N").
	WritesAfterStart(run string) (int, error)
	// VirtualNow is the virtual time since the lab started stepping.
	VirtualNow() time.Duration
}

// T0FightLab is optional: a lab that can put the T0 fight-time points (t0.retreat.drink, t0.unique.fight,
// t0.area.toohard, t2.mercenary) through their real call sites implements it, and the drill then runs one pass of each
// per round. The offline fake asks the ladder with realistic states; the live lab has no implementation yet
// (docs/probes/t0drill.md, "Live run needs").
type T0FightLab interface {
	FightPoints(ctx context.Context, point string) ChoreInfo
}

// T0FightPoints are the plans a T0FightLab is asked for, one per wired fight-time point.
var T0FightPoints = []string{ladder.PointDrink, ladder.PointUnique, ladder.PointTooHard, ladder.PointMerc}

// T0Config selects the drill: the answer profiles, an optional scripted answer file, the virtual minutes and the dir of
// the persisted answers ("" keeps them in memory).
type T0Config struct {
	Profiles []string // cautious, bold, inconsistent, holdout (seeded); "file" reads AnswersFile
	Seeds    []int64  // one per holdout, in order
	Answers  string   // scripted answer file for the "file" profile
	Minutes  int      // virtual minutes of play after the start (default 10)
	Dir      string
	Round    time.Duration // virtual time between two bot passes (default 30 s)
}

// DefaultT0 is the drill of TASK-21: three known profiles and two holdouts from fixed seeds.
func DefaultT0() T0Config {
	return T0Config{Profiles: []string{"cautious", "bold", "inconsistent", "holdout", "holdout"}, Seeds: []int64{101, 202}}
}

const ticksPerSecond = 25 // 40 ms a tick (the lab's Pause)

// T0 returns the T0 group (card t0-slice): from a fresh character, with the ladder on and scripted answers, the wired
// town points run for a bounded virtual time. Setup (worn gear, gold, empty pack) is written BEFORE the run start; the
// bot acts only through its real code paths after it. The verdict checks the invariants, not the answers.
func T0(l T0Lab, c T0Config) []Drill {
	if c.Minutes <= 0 {
		c.Minutes = 10
	}
	if c.Round <= 0 {
		c.Round = 30 * time.Second
	}
	seed := 0
	var out []Drill
	for i, p := range c.Profiles {
		name, sd := p, int64(i+1)
		if p == "holdout" {
			if seed < len(c.Seeds) {
				sd = c.Seeds[seed]
			}
			name = fmt.Sprintf("holdout%d", sd)
			seed++
		}
		prof, sd, id := p, sd, "T0."+name
		out = append(out, Drill{ID: id, Primitive: "T0", Limit: time.Duration(c.Minutes)*time.Minute + 30*time.Minute,
			Title: fmt.Sprintf("ladder on, %s answers, %d virtual minutes of the wired town points from a fresh character, writes after start 0", name, c.Minutes),
			Do:    func(ctx context.Context, r *Run) Outcome { return t0Run(ctx, r, l, c, prof, sd, id) }})
	}
	return out
}

var t0Runs int

func t0Run(ctx context.Context, r *Run, l T0Lab, c T0Config, profile string, seed int64, id string) Outcome {
	var src ladder.AnswerSource
	switch profile {
	case "file":
		fs, err := ladder.LoadScript(c.Answers)
		if err != nil {
			return Blocks(err.Error())
		}
		src = fs
	default:
		ps, err := ladder.NewProfileSource(profile, seed)
		if err != nil {
			return Blocks(err.Error())
		}
		src = ps
	}
	// setup, all before the start
	if err := l.ClearPack(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.SetGold(ctx, r, 1000); err != nil {
		return setupFail(err)
	}
	if err := l.Wear(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.Tick(ctx, 20); err != nil {
		return setupFail(err)
	}
	t0Runs++
	run := fmt.Sprintf("%s-%d", id, t0Runs)
	vstart := l.VirtualNow()
	sink := &ladder.RecordSink{}
	lad, err := ladder.Open(c.Dir, strings.ToLower(strings.ReplaceAll(id, ".", "-")), ladder.ClockFunc(func() time.Duration { return l.VirtualNow() - vstart }), sink, src, nil)
	if err != nil {
		return Blocks(err.Error())
	}
	ladder.Install(lad)
	defer ladder.Install(nil)
	if err := l.StartRun(ctx, run, "T0 drill: setup written, the bot plays through its own code paths from here"); err != nil {
		return setupFail(err)
	}
	var errs []string
	rounds, fightPasses := 0, 0
	for l.VirtualNow()-vstart < time.Duration(c.Minutes)*time.Minute {
		if ctx.Err() != nil {
			return Blocks("cancelled: " + ctx.Err().Error())
		}
		for _, plan := range []string{"repair", "scrolls"} {
			if ci := l.Chores(ctx, plan); ci.Err != "" {
				errs = append(errs, plan+": "+ci.Err)
			}
		}
		if fl, ok := l.(T0FightLab); ok {
			for _, pt := range T0FightPoints {
				if ci := fl.FightPoints(ctx, pt); ci.Err != "" {
					errs = append(errs, pt+": "+ci.Err)
				}
				fightPasses++
			}
		}
		lad.Pump()
		if err := l.Tick(ctx, int(c.Round.Seconds())*ticksPerSecond); err != nil {
			return Blocks("tick: " + err.Error())
		}
		rounds++
	}
	lad.Pump()
	v, err := l.TView(ctx)
	if err != nil {
		return Blocks("read: " + err.Error())
	}
	writes, werr := l.WritesAfterStart(run)
	st, rules := lad.Stats(), lad.Store()
	nrules, nans, nfaded := rules.Stats()
	rec := t0Record{Profile: profile, Seed: seed, Rounds: rounds, RuleFile: ruleFileHash(), RulesLearned: nrules, AnswersGiven: nans,
		ThoughtsFaded: nfaded, ThoughtsOpened: st.Opened, Faults: st.Faults, Asked: st.Asked, FightPasses: fightPasses, WritesAfterStart: writes}
	line, _ := json.Marshal(rec)
	obs := fmt.Sprintf("%s; writes after start: %d; durability %d/%d gold %d; errors %v", line, writes, v.DurCur, v.DurMax, v.Gold, errs)
	o := Outcome{Intended: "no loop, no crash, no stall for any answer; every pass returned; no ladder fault; no chore error; writes after start: 0", Observed: obs,
		Sig: fmt.Sprintf("rounds=%d rules=%d answers=%d faded=%d", rounds, nrules, nans, nfaded)}
	switch {
	case werr != nil:
		o.Verdict, o.Reason = Blocked, "ledger: "+werr.Error()
	case writes != 0:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("CHEATED: %d writes after the run start", writes)
	case st.Faults != 0:
		o.Verdict, o.Reason = Fail, "ladder fault: "+st.LastFault
	case len(errs) != 0:
		o.Verdict, o.Reason = Fail, "chore error: "+errs[0]
	case len(rules.OpenThoughts()) > 60 || len(rules.Log()) > 8*(rounds+fightPasses)+16:
		o.Verdict, o.Reason = Fail, "the answer log or the open thoughts grew without bound"
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("%d passes, %d thoughts, %d answers, %d rules, %d faded", rounds, st.Opened, nans, nrules, nfaded)
	}
	return o
}

// t0Record is the run record of acceptance item 3: the rule-file hash, rules learned, answers given, thoughts faded.
type t0Record struct {
	Profile          string `json:"profile"`
	Seed             int64  `json:"seed"`
	Rounds           int    `json:"rounds"`
	RuleFile         string `json:"ruleFileHash"`
	RulesLearned     int    `json:"rulesLearned"`
	AnswersGiven     int    `json:"answersGiven"`
	ThoughtsFaded    int    `json:"thoughtsFaded"`
	ThoughtsOpened   int    `json:"thoughtsOpened"`
	Asked            int    `json:"asked"`
	FightPasses      int    `json:"fightPasses"`
	Faults           int    `json:"faults"`
	WritesAfterStart int    `json:"writesAfterStart"`
}

func ruleFileHash() string {
	f, err := decide.ShippedRuleFile()
	if err != nil {
		return "unreadable"
	}
	b, _ := json.Marshal(f)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:6])
}

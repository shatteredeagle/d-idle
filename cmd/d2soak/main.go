// Command d2soak runs a long soak against a bot's HTTP API and fails with a clear report on leaks, stalls,
// too many deaths or no progress.
//
//	d2soak run --target http://127.0.0.1:8420 --for 24h --out soak-1 \
//	    --proc game=match:Game.exe --proc wineserver=match:wineserver --proc bot=PID
//	d2soak report --in soak-1/samples.jsonl      re-analyze a finished or partial run
//
// run samples health, stats, events, process stats (from /proc), the bot's /metrics and the sentinel log every
// --every (default 30 s) into OUT/samples.jsonl, writes OUT/report.{json,txt} at the end (or on Ctrl-C) and exits
// 0 on pass, 1 on fail, 2 on usage errors. docs: tools/soak/CHECKLIST.md.
package main

import (
	"bufio"
	"context"
	"d2lod-bot/internal/buildinfo"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

func main() {
	buildinfo.Exit("d2soak")
	os.Exit(realMain(os.Args[1:]))
}

func usage() int {
	fmt.Fprintln(os.Stderr, "usage: d2soak run [flags] | report|status|alerts --in samples.jsonl [limit flags]")
	return 2
}

func realMain(args []string) int {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "procs":
		return cmdProcs(args[1:])
	case "run":
		return cmdRun(args[1:])
	case "report":
		return cmdReport(args[1:])
	case "status":
		return cmdStatus(args[1:])
	case "alerts":
		return cmdAlerts(args[1:])
	case "sidecar":
		return cmdSidecar(args[1:])
	}
	return usage()
}

// profiles are the exit criteria of docs/archive/review/roadmap.md. The RSS criterion there is relative (< 5 %/h after
// 15 min); the harness checks an absolute MB/h slope, so pick -max-rss-mb-per-hour from the first run's RSS.
//
// Profiles also set the leak thresholds (UNVERIFIED against a long run, see docs/soak.md). The slopes are judged
// after the warm-up (default 10 min) and fd steps at area_change events are ignored, so the fd limits only have to
// cover genuine growth: the wineserver and d2input gain about +30/h in the first hours (LEAKHUNT), which is why
// the 2 h profile is loose and the long ones tighter (more data, and a recycle can reset the processes).
var profiles = map[string]func(*Limits){
	"m1": func(l *Limits) { // 2 h run
		l.MaxDeathsPerHour, l.MaxNoProgress, l.MaxSentinelPerHr, l.MaxLadderPerHour = 1, 5*time.Minute, 2, 1
		l.MinXPPerHour = 15000
		l.Progress.NoGain = 45 * time.Minute
		l.MaxDeathsPerHour, l.Death = 0, DefaultDeathLimits() // the death rule replaces the deaths-per-hour gate
		l.MaxRSSMBPerHour, l.MaxFDsPerHour, l.MaxThreadsPerHour, l.MaxGoroutinesPerH = 60, 15, 10, 10
	},
	"m3": func(l *Limits) { // 24 h run: no fatal, no manual restart
		l.MaxDeathsPerHour = 1
		l.MaxRSSMBPerHour, l.MaxFDsPerHour, l.MaxThreadsPerHour, l.MaxGoroutinesPerH = 40, 8, 5, 5
	},
	// m24: the milestone-alpha 24 h unattended campaign soak (docs/soak24.md). Stricter than m3: helper/game/bot fds at
	// 5/h (the wineserver keeps its own 40/h and 2000-per-day cap), ladder and sentinel rates capped, a 10 min stall
	// limit, and a low XP floor so a bot that only idles cannot pass (UNVERIFIED numbers until the first 24 h run).
	"m24": func(l *Limits) {
		l.MaxDeathsPerHour, l.MaxNoProgress, l.MaxSentinelPerHr, l.MaxLadderPerHour = 1, 10*time.Minute, 2, 2
		l.MinXPPerHour = 1000
		l.MaxDeathsPerHour, l.Death = 0, DefaultDeathLimits()
		l.MaxWineOrphans = 2
		l.MaxRSSMBPerHour, l.MaxFDsPerHour, l.MaxThreadsPerHour, l.MaxGoroutinesPerH = 40, 5, 5, 5
	},
	"v1": func(l *Limits) { // 72 h run
		l.MaxDeathsPerHour = 1
		l.MaxRSSMBPerHour, l.MaxFDsPerHour, l.MaxThreadsPerHour, l.MaxGoroutinesPerH = 30, 5, 3, 3
	},
}

// limitFlags registers the limit flags and -profile; call the returned func after Parse to apply the profile
// to every limit not set explicitly.
func limitFlags(fs *flag.FlagSet) (*Limits, func() error) {
	l := DefaultLimits()
	profile := fs.String("profile", "", "preset limits from roadmap.md: m1 (2 h), m3 (24 h), m24 (24 h alpha), v1 (72 h); explicit flags win")
	fs.Float64Var(&l.MaxDeathsPerHour, "max-deaths-per-hour", l.MaxDeathsPerHour, "fail above this death rate (0 = off)")
	fs.DurationVar(&l.MaxNoProgress, "max-no-progress", l.MaxNoProgress, "fail when no progress for longer (idle/paused exempt)")
	fs.DurationVar(&l.MaxUnhealthy, "max-unhealthy", l.MaxUnhealthy, "fail when health is down for longer")
	fs.Float64Var(&l.MinXPPerHour, "min-xp-per-hour", l.MinXPPerHour, "fail below this XP rate (0 = off)")
	fs.Float64Var(&l.MaxLadderPerHour, "max-ladder-per-hour", l.MaxLadderPerHour, "failsafe ladder rungs per hour")
	fs.Float64Var(&l.MaxSentinelPerHr, "max-sentinel-per-hour", l.MaxSentinelPerHr, "sentinel actions per hour")
	fs.Float64Var(&l.MaxRSSMBPerHour, "max-rss-mb-per-hour", l.MaxRSSMBPerHour, "RSS slope limit per process")
	fs.Float64Var(&l.MaxFDsPerHour, "max-fds-per-hour", l.MaxFDsPerHour, "fd count slope limit")
	fs.Float64Var(&l.MaxGoroutinesPerH, "max-goroutines-per-hour", l.MaxGoroutinesPerH, "bot goroutine slope limit")
	fs.Float64Var(&l.MaxThreadsPerHour, "max-threads-per-hour", l.MaxThreadsPerHour, "thread count slope limit")
	fs.IntVar(&l.MaxZombieSamples, "max-zombie-samples", l.MaxZombieSamples, "consecutive zombie samples allowed")
	fs.DurationVar(&l.WarmUp, "warmup", l.WarmUp, "dropped from the start of every slope series and after each process restart (0 = none)")
	fs.Float64Var(&l.AreaStepFDs, "area-step-fds", l.AreaStepFDs, "fd jump at an area_change event ignored as a first-visit step (0 = count all)")
	fs.DurationVar(&l.WineMinSpan, "wine-min-span", l.WineMinSpan, "post-warm-up data needed before the wineserver fd slope is judged")
	fs.Float64Var(&l.WineMaxFDsPerHour, "wine-max-fds-per-hour", l.WineMaxFDsPerHour, "wineserver fd slope limit (0 = off)")
	fs.IntVar(&l.MaxWineOrphans, "max-wine-orphans", l.MaxWineOrphans, "warn when more orphaned winedevice.exe stand at the end (needs -wine-prefix; 0 = off)")
	fs.Float64Var(&l.WineMaxGrowth24h, "wine-max-growth-24h", l.WineMaxGrowth24h, "fail when the wineserver fd slope projects more than this many fds in 24 h (0 = off)")
	fs.DurationVar(&l.MinSlopeSpan, "min-slope-span", l.MinSlopeSpan, "data needed before a slope is judged")
	fs.DurationVar(&l.Progress.NoGain, "no-gain", l.Progress.NoGain, "fail when no clvl gain, quest-step change, gold gain or xp gain for this long, from the minute lines (0 = off; profile m1 uses 45m)")
	fs.IntVar(&l.Progress.NoGainMinXP, "no-gain-min-xp", l.Progress.NoGainMinXP, "xp gain that counts as progress for no_gain, at least 1% of the level's xp span (0 = 500, -1 = xp does not count)")
	fs.IntVar(&l.Progress.LoopRepeats, "loop-repeats", l.Progress.LoopRepeats, "fail when one step in one area repeats this many times without a gain (0 = off)")
	fs.DurationVar(&l.Progress.IdleInTown, "idle-in-town", l.Progress.IdleInTown, "fail when one unbroken Waiting state lasts this long (0 = off)")
	fs.Float64Var(&l.Progress.StateShare, "state-share", l.Progress.StateShare, "warn when a state group takes more than this share of an hour (0 = off)")
	l.Death = DefaultDeathLimits()
	l.Death.Enabled = false
	fs.BoolVar(&l.Death.Enabled, "death-rule", false, "judge the death rule of plan-v2 10.2 from the bot log (profiles m1 and m24 turn it on)")
	fs.DurationVar(&l.Death.MaxBack, "death-max-back", l.Death.MaxBack, "death_recovery_stuck: one death to back in the field above this fails")
	fs.IntVar(&l.Death.MinKills, "death-min-kills", l.Death.MinKills, "death_rate needs this many kills in the 60 min window")
	fs.BoolVar(&l.Progress.FailBelowFloor, "fail-below-floor", l.Progress.FailBelowFloor, "below_floor_clvl fails instead of warning")
	return &l, func() error {
		if *profile == "" {
			return nil
		}
		apply, ok := profiles[*profile]
		if !ok {
			return fmt.Errorf("unknown -profile %q (m1, m3, m24, v1)", *profile)
		}
		set := map[string]bool{}
		fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
		p := l
		apply(&p)
		for flagName, dst := range map[string][2]any{
			"max-deaths-per-hour":     {&l.MaxDeathsPerHour, p.MaxDeathsPerHour},
			"max-sentinel-per-hour":   {&l.MaxSentinelPerHr, p.MaxSentinelPerHr},
			"max-ladder-per-hour":     {&l.MaxLadderPerHour, p.MaxLadderPerHour},
			"min-xp-per-hour":         {&l.MinXPPerHour, p.MinXPPerHour},
			"max-rss-mb-per-hour":     {&l.MaxRSSMBPerHour, p.MaxRSSMBPerHour},
			"max-fds-per-hour":        {&l.MaxFDsPerHour, p.MaxFDsPerHour},
			"max-threads-per-hour":    {&l.MaxThreadsPerHour, p.MaxThreadsPerHour},
			"max-goroutines-per-hour": {&l.MaxGoroutinesPerH, p.MaxGoroutinesPerH},
		} {
			if !set[flagName] {
				*dst[0].(*float64) = dst[1].(float64)
			}
		}
		if !set["death-rule"] {
			l.Death.Enabled = p.Death.Enabled
		}
		if !set["no-gain"] {
			l.Progress.NoGain = p.Progress.NoGain
		}
		if !set["max-no-progress"] {
			l.MaxNoProgress = p.MaxNoProgress
		}
		return nil
	}
}

func alertFlags(fs *flag.FlagSet) *AlertRules {
	r := DefaultAlertRules()
	fs.DurationVar(&r.StallXP, "alert-stall", r.StallXP, "alert when XP stands still this long (0 = off)")
	fs.Float64Var(&r.XPRiskLevels, "alert-xp-risk-levels", r.XPRiskLevels, "warn when unsaved XP exceeds this many levels worth, needs -save-file (0 = off)")
	fs.Float64Var(&r.RestartsPerHour, "alert-restarts-per-hour", r.RestartsPerHour, "alert above this many process restarts in the last hour (0 = off)")
	fs.Float64Var(&r.PanicsPerHour, "alert-panics-per-hour", r.PanicsPerHour, "alert above this many recovered bot panics in the last hour (0 = off)")
	fs.Float64Var(&r.HeapRunawayMB, "alert-heap-runaway-mb", r.HeapRunawayMB, "crit when the bot's RSS exceeds this many MB (heap_runaway; 0 = off)")
	fs.Float64Var(&r.HeapGrowthMB, "alert-heap-growth-mb", r.HeapGrowthMB, "crit when the bot's RSS grows by more than this many MB within -alert-heap-growth-window (0 = off)")
	fs.DurationVar(&r.HeapGrowthWindow, "alert-heap-growth-window", r.HeapGrowthWindow, "window of the RSS growth alert")
	fs.DurationVar(&r.FDWindow, "alert-fd-window", r.FDWindow, "trailing window for the fd slope alert (0 = off)")
	fs.DurationVar(&r.TickAge, "alert-tick-age", r.TickAge, "alert when the bot log has no minute line for this long (needs -bot-log; 0 = off)")
	fs.Float64Var(&r.GameRestartsPerHour, "alert-game-restarts-per-hour", r.GameRestartsPerHour, "warn above this many game restarts in the last hour, from the minute lines (needs -bot-log; 0 = off)")
	fs.IntVar(&r.WineOrphans, "alert-wine-orphans", r.WineOrphans, "warn above this many orphaned winedevice.exe (needs -wine-prefix; 0 = off)")
	fs.StringVar(&r.BotLog, "bot-log", "", "bot log file, or a directory (newest *.log): minute lines for the tick-age and game-restart alerts; default the one the run recorded")
	return &r
}

type runCfg struct {
	Collector *Collector
	Every     time.Duration
	For       time.Duration
	Out       string
	Limits    Limits
	Rules     AlertRules
	FailFast  bool
	AlertCmd  string // run on every new fail alert (D2SOAK_ALERT_CMD), "" = none
	Now       func() time.Time
	Log       func(string, ...any)
}

func cmdRun(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	target := fs.String("target", "http://127.0.0.1:8420", "server root (or .../api/v1) of the bot")
	instance := fs.String("instance", "", "instance id (default: the first one, or the flat legacy routes)")
	token := fs.String("token", os.Getenv("D2BOT_TOKEN"), "API token (default $D2BOT_TOKEN)")
	dur := fs.Duration("for", 24*time.Hour, "soak length")
	every := fs.Duration("every", 30*time.Second, "sample interval")
	out := fs.String("out", "", "output directory (default soak-YYYYmmdd-HHMMSS)")
	metricsURL := fs.String("metrics-url", "", "bot Prometheus URL (default TARGET/metrics, 'off' disables)")
	saveFile := fs.String("save-file", "", "the character's .d2s: the report then measures XP from saved experience and alerts on xp_at_risk")
	sentinel := fs.String("sentinel-log", "", "sentinel JSONL file to count actions from")
	failFast := fs.Bool("fail-fast", false, "stop at the first failing check (checked every 10 samples)")
	var procs, winePfx multi
	fs.Var(&winePfx, "wine-prefix", "WINEPREFIX whose winedevice.exe are counted per sample (repeatable; none = no sampling)")
	fs.Var(&procs, "proc", "watched process name=PID|self|match:SUBSTR (repeatable; e.g. game=match:Game.exe)")
	waitTarget := fs.Duration("wait-target", 2*time.Minute, "keep retrying while the bot is not reachable yet (0 = fail at once)")
	lim, finish := limitFlags(fs)
	rules := alertFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := finish(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	botLog := &rules.BotLog
	ps, err := parseProcSpecs(procs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cl := &http.Client{Timeout: 10 * time.Second}
	c, err := resolveWait(ctx, cl, *token, *target, *instance, *waitTarget)
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2soak:", err)
		return 2
	}
	c.Procs, c.SentinelLog, c.WinePrefixes, c.SaveFile = ps, *sentinel, winePfx, *saveFile
	switch *metricsURL {
	case "":
		c.MetricsURL = strings.TrimSuffix(strings.TrimRight(*target, "/"), "/api/v1") + "/metrics"
	case "off":
	default:
		c.MetricsURL = *metricsURL
	}
	dir := *out
	if dir == "" {
		dir = "soak-" + time.Now().Format("20060102-150405")
	}
	cfg := runCfg{Collector: c, Every: *every, For: *dur, Out: dir, Limits: *lim, Rules: *rules, FailFast: *failFast, AlertCmd: os.Getenv(alertCmdEnv), Now: time.Now,
		Log: func(f string, a ...any) { fmt.Fprintf(os.Stderr, f+"\n", a...) }}
	r, err := runSoak(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2soak:", err)
		return 2
	}
	fmt.Print(r.Text())
	fmt.Print(logSummary(*botLog))
	return exitCode(r)
}

// logSummary renders the fight and death summaries of a bot log ("" for no path); problems are one line, not fatal.
func logSummary(path string) string {
	if path == "" {
		return ""
	}
	var out strings.Builder
	for _, part := range []func(io.Reader) (string, error){
		func(r io.Reader) (string, error) { h, err := ParseFightLines(r); return FightText(h), err },
		func(r io.Reader) (string, error) { d, err := ParseDeathLines(r); return DeathText(d), err },
		func(r io.Reader) (string, error) { m, err := ParseMinuteLines(r); return MinuteText(m), err },
	} {
		f, err := os.Open(resolveBotLog(path))
		if err != nil {
			return fmt.Sprintf("bot log: %v\n", err)
		}
		t, err := part(f)
		f.Close()
		if err != nil {
			return fmt.Sprintf("bot log: %v\n", err)
		}
		out.WriteString(t)
	}
	return out.String()
}

// progressFromLog reads the bot log's minute lines and judges them; a missing log or none judged is an empty report.
func progressFromLog(path string, lim ProgressLimits) ProgressReport {
	if path == "" {
		return ProgressReport{}
	}
	f, err := os.Open(resolveBotLog(path))
	if err != nil {
		return ProgressReport{}
	}
	defer f.Close()
	ms, _ := ParseMinuteLines(f)
	return AnalyzeMinutes(ms, lim)
}

// exitCode: 0 pass, 1 fail, 3 nothing to judge (fewer than 2 samples).
func exitCode(r Report) int {
	switch {
	case r.NoData:
		return 3
	case !r.Pass:
		return 1
	}
	return 0
}

// runSoak samples until ctx ends or For has elapsed, then writes and returns the report.
func runSoak(ctx context.Context, cfg runCfg) (Report, error) {
	if err := os.MkdirAll(cfg.Out, 0o755); err != nil {
		return Report{}, err
	}
	f, err := os.OpenFile(filepath.Join(cfg.Out, "samples.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return Report{}, err
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	if err := writeJSONAtomic(filepath.Join(cfg.Out, "limits.json"), cfg.Limits); err != nil {
		return Report{}, err
	}
	if cfg.Rules.BotLog != "" { // so `d2soak status|alerts -in RUN` finds the log without a flag
		os.WriteFile(filepath.Join(cfg.Out, "botlog.txt"), []byte(cfg.Rules.BotLog+"\n"), 0o644)
	}
	botVersion := cfg.Collector.Version(ctx)
	os.WriteFile(filepath.Join(cfg.Out, versionFile), []byte(botVersion+"\n"), 0o644)
	cfg.Log("d2soak: bot version %s", botVersion)
	lw := newLiveWriter(cfg.Out, cfg.AlertCmd, cfg.Log)
	defer lw.Close()
	var ss []Sample
	start := cfg.Now()
	cfg.Log("d2soak: sampling %s every %s for %s into %s", cfg.Collector.Base, cfg.Every, cfg.For, cfg.Out)
	tk := time.NewTicker(cfg.Every)
	defer tk.Stop()
	for {
		s := cfg.Collector.Sample(ctx, cfg.Now())
		if ctx.Err() != nil {
			break
		}
		ss = append(ss, s)
		if err := enc.Encode(s); err != nil {
			return Report{}, err
		}
		if len(ss)%4 == 1 { // every ~2 min at the default interval; a dashboard failure must not stop the soak
			if err := lw.write(ss, cfg.Limits, cfg.Rules, cfg.Now()); err != nil {
				cfg.Log("d2soak: alerts: %v", err)
			}
		}
		if cfg.FailFast && len(ss)%10 == 0 {
			if r := Analyze(ss, cfg.Limits); !r.Pass {
				cfg.Log("d2soak: failing, stopping early (-fail-fast)")
				break
			}
		}
		if cfg.Now().Sub(start) >= cfg.For {
			break
		}
		select {
		case <-ctx.Done():
		case <-tk.C:
			continue
		}
		break
	}
	r := Analyze(ss, cfg.Limits)
	r.BotVersion = botVersion
	lw.write(ss, cfg.Limits, cfg.Rules, cfg.Now())
	return r, writeReport(cfg.Out, r)
}

func writeReport(dir string, r Report) error {
	b, _ := json.MarshalIndent(r, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "report.json"), b, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "report.txt"), []byte(r.Text()), 0o644)
}

func readSamples(path string) ([]Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var ss []Sample
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		if len(strings.TrimSpace(sc.Text())) == 0 {
			continue
		}
		var s Sample
		if err := json.Unmarshal(sc.Bytes(), &s); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, n, err)
		}
		ss = append(ss, s)
	}
	return ss, sc.Err()
}

func cmdReport(args []string) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	in := fs.String("in", "", "samples.jsonl, a run directory, or a directory of runs (the newest run is used)")
	asJSON := fs.Bool("json", false, "print JSON")
	botLog := fs.String("bot-log", "", "bot log file: append per-hour fight and death summaries")
	deathsFile := fs.String("deaths", "", "deaths.jsonl of the bot (secondsToField, belt and gold per death); default deaths.jsonl in the run directory")
	lim, finish := limitFlags(fs)
	if err := fs.Parse(args); err != nil || *in == "" {
		return 2
	}
	if err := finish(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	path, dir := samplesPath(*in)
	ss, err := readSamples(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2soak:", err)
		return 2
	}
	// Without any limit flag the run is judged with the limits it was recorded with (limits.json, the sidecar's m24).
	limitSet := false
	fs.Visit(func(f *flag.Flag) {
		limitSet = limitSet || (f.Name != "in" && f.Name != "json" && f.Name != "bot-log" && f.Name != "deaths")
	})
	l := *lim
	if saved, ok := loadLimits(dir); ok && !limitSet {
		l = saved
	}
	if *botLog == "" {
		*botLog = loadBotLog(dir)
	}
	r := Analyze(ss, l)
	r.BotVersion = loadVersion(dir)
	if *deathsFile == "" {
		if p := filepath.Join(dir, "deaths.jsonl"); fileExists(p) {
			*deathsFile = p
		}
	}
	// The same judge as the live alerts (live.go): report and sidecar cannot disagree.
	j := judgeLog(*botLog, *deathsFile, l)
	pr, dr := j.Progress, j.Death
	if dr.Evaluated {
		r.DeathRule = &dr.Summary
	}
	pr.Findings = append(pr.Findings, dr.Findings...)
	for _, f := range pr.Findings {
		r.Findings = append(r.Findings, f)
		if f.Level == "fail" {
			r.Pass = false
			if !r.NoData {
				r.Verdict = "FAIL"
			}
		}
	}
	if *asJSON {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Print(r.Text())
		fmt.Print(pr.Text())
		fmt.Print(dr.Text())
		fmt.Print(logSummary(*botLog))
	}
	return exitCode(r)
}

// resolveWait retries Resolve while the bot is still starting (a soak started right after the bot would
// otherwise exit and systemd would show it inactive).
func resolveWait(ctx context.Context, cl *http.Client, token, target, instance string, wait time.Duration) (*Collector, error) {
	deadline := time.Now().Add(wait)
	for {
		c, err := Resolve(ctx, cl, token, target, instance)
		if err == nil || time.Now().After(deadline) || ctx.Err() != nil {
			return c, err
		}
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
		}
	}
}

// samplesPath accepts a samples file, a run directory, or a directory of runs (then the run whose samples.jsonl
// changed last) and returns the file and the directory.
func samplesPath(in string) (string, string) {
	if st, err := os.Stat(in); err == nil && st.IsDir() {
		if _, err := os.Stat(filepath.Join(in, "samples.jsonl")); err != nil {
			if run := newestRunDir(in); run != "" {
				in = run
			}
		}
		return filepath.Join(in, "samples.jsonl"), in
	}
	return in, filepath.Dir(in)
}

// newestRunDir is the subdirectory of root with the most recently modified samples.jsonl ("" = none).
func newestRunDir(root string) string {
	ents, _ := os.ReadDir(root)
	var best string
	var bt time.Time
	for _, e := range ents {
		st, err := os.Stat(filepath.Join(root, e.Name(), "samples.jsonl"))
		if err == nil && (best == "" || st.ModTime().After(bt)) {
			best, bt = filepath.Join(root, e.Name()), st.ModTime()
		}
	}
	return best
}

func cmdStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	in := fs.String("in", "", "samples.jsonl or the run directory")
	rules := alertFlags(fs)
	if err := fs.Parse(args); err != nil || *in == "" {
		return 2
	}
	path, dir := samplesPath(*in)
	if rules.BotLog == "" {
		rules.BotLog = loadBotLog(dir)
	}
	ss, err := readSamples(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2soak:", err)
		return 2
	}
	fmt.Println(StatusLine(ss, time.Now()))
	if x := ExtraStatus(ss, *rules, time.Now()); x != "" {
		fmt.Println(x)
	}
	return 0
}

func cmdAlerts(args []string) int {
	fs := flag.NewFlagSet("alerts", flag.ContinueOnError)
	in := fs.String("in", "", "samples.jsonl or the run directory")
	write := fs.Bool("write", false, "also write alerts.json and status.txt next to the samples")
	lim, finish := limitFlags(fs)
	rules := alertFlags(fs)
	if err := fs.Parse(args); err != nil || *in == "" {
		return 2
	}
	if err := finish(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	path, dir := samplesPath(*in)
	ss, err := readSamples(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2soak:", err)
		return 2
	}
	l := *lim
	if saved, ok := loadLimits(dir); ok {
		l = saved
	}
	if rules.BotLog == "" {
		rules.BotLog = loadBotLog(dir)
	}
	rules.DeathsFile = filepath.Join(dir, "deaths.jsonl")
	a := EvalAlerts(ss, l, *rules, time.Now())
	newTracker(dir).apply(&a, time.Now())
	if *write {
		if err := writeLive(dir, ss, l, *rules, time.Now()); err != nil {
			fmt.Fprintln(os.Stderr, "d2soak:", err)
			return 2
		}
	}
	b, _ := json.MarshalIndent(a, "", "  ")
	fmt.Println(string(b))
	if a.Level != "ok" {
		return 1
	}
	return 0
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

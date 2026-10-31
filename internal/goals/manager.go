package goals

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Mode string

const (
	ModeStopped Mode = "stopped" // autopilot off, runner idle
	ModeRunning Mode = "running"
	ModePaused  Mode = "paused" // runner paused, active goal kept
)

var (
	ErrNotFound = errors.New("goals: no such goal")
	ErrInvalid  = errors.New("goals: invalid request")
	ErrState    = errors.New("goals: not allowed in the current state")
)

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, a...))
}

const (
	maxFinished  = 100
	maxCreditGap = 5 * time.Minute // longer gaps between ticks (suspend, clock jump) are not credited as play time
	saveEvery    = 5 * time.Second
)

type Options struct {
	Runner       Runner
	Path         string           // state file; empty = in memory only
	Clock        func() time.Time // default time.Now
	EventLogSize int              // default 1000
	Logf         func(format string, args ...any)
	Chain        *Chain // optional: enqueue the next difficulty's campaign when a chained one completes
}

// Manager owns the goal queue and drives a Runner. All methods are safe for concurrent use.
// Run starts the event pump and the 1 s scheduler tick; tests drive Tick and Handle directly.
type Manager struct {
	opt Options

	mu        sync.Mutex
	goals     []*Goal // live goals first in scheduling order, finished ones are kept for history
	nextID    int64
	mode      Mode
	activeID  string
	settings  Settings
	stats     *Stats
	log       *ring[Event]
	seq       uint64
	rev       uint64
	subs      map[int]chan Event
	subSeq    int
	dirty     bool
	level     int // highest character level seen from LevelUp events or the runner status
	lastSave  time.Time
	lastTick  time.Time
	activeRem time.Duration
	lastError string
	warning   string
	started   time.Time
	locked    map[string]lock // goal id -> why a Hold error paused it; a higher level or its quest unlock resumes it
	questSeen map[string]int  // "diff/quest" -> highest stage seen
	pruned    []string        // seed keys of pruned goals, oldest first (see maxPrunedSeeds)
}

// New loads the state file if there is one. A corrupt file is recovered from the backup or replaced by an
// empty state (see Snapshot.LoadWarning); ErrFutureVersion is returned as an error.
func New(opt Options) (*Manager, error) {
	if opt.Runner == nil {
		return nil, errors.New("goals: Options.Runner is required")
	}
	if opt.Clock == nil {
		opt.Clock = time.Now
	}
	if opt.EventLogSize <= 0 {
		opt.EventLogSize = 1000
	}
	if opt.Logf == nil {
		opt.Logf = func(string, ...any) {}
	}
	now := opt.Clock()
	m := &Manager{
		opt: opt, mode: ModeStopped, settings: DefaultSettings(), stats: newStats(now),
		log: newRing[Event](opt.EventLogSize), subs: map[int]chan Event{},
		started: now, lastTick: now, seq: uint64(now.UnixMilli()) * 1000,
	}
	if opt.Path != "" {
		st, warn, err := loadState(opt.Path, now)
		if err != nil {
			return nil, err
		}
		m.warning = warn
		if st != nil {
			m.restore(st)
		}
	}
	if m.warning != "" {
		m.emitLocked(Event{Type: EvInfo, Message: "State file: " + m.warning})
	}
	if c, ok := opt.Runner.(Configurer); ok {
		c.Configure(m.settings)
	}
	if m.settings.AutoResume && m.mode != ModeRunning {
		m.mode = ModeRunning
	}
	if m.mode == ModePaused { // a pause does not survive a restart: there is nothing to resume
		m.mode = ModeStopped
	}
	m.resumeChains(now)
	m.scheduleLocked(now)
	return m, nil
}

// cleanHoldsLocked makes m.locked the only hold record after a load. It migrates the legacy Goal.HeldAt of a state file
// written by an NMSIM-only build, and drops holds that no longer mean anything: no goal, a goal that is not paused, or a
// pause made by hand (an older build left the hold behind and the next level-up would restart the goal).
func (m *Manager) cleanHoldsLocked(st *fileState) {
	for _, g := range m.goals {
		_, held := m.locked[g.ID]
		if !held && g.HeldAt > 0 && g.State == StatePaused {
			m.lockAtLocked(g, g.HeldAt)
		}
		g.HeldAt = 0
		if g.State != StatePaused || g.Note == userPauseNote {
			delete(m.locked, g.ID)
		}
	}
	for id := range m.locked {
		if m.find(id) == nil {
			delete(m.locked, id)
		}
	}
}

func (m *Manager) restore(st *fileState) {
	m.locked, m.questSeen = st.Locks, st.QuestSeen
	m.pruned = trimSeeds(st.PrunedSeeds)
	m.nextID = st.NextID
	if st.Settings.Validate() == nil {
		m.settings = st.Settings
	}
	if st.Stats != nil {
		m.stats = st.Stats
		if m.stats.Qualities == nil {
			m.stats.Qualities = map[string]int{}
		}
	}
	dropped := 0
	for _, g := range st.Goals {
		if g == nil || g.ID == "" || g.Normalize() != nil {
			dropped++
			continue
		}
		if g.State == StateActive {
			g.State = StateQueued // the runner does not survive a restart
		}
		m.goals = append(m.goals, g)
		if n, err := strconv.ParseInt(g.ID[1:], 10, 64); err == nil && n > m.nextID {
			m.nextID = n
		}
	}
	m.cleanHoldsLocked(st)
	if dropped > 0 {
		m.warning = fmt.Sprintf("%s%d invalid goals dropped", sepIf(m.warning), dropped)
	}
	m.sortLocked()
}

func sepIf(s string) string {
	if s == "" {
		return ""
	}
	return s + "; "
}

// Run pumps runner events and ticks the scheduler until ctx is done, then saves.
func (m *Manager) Run(ctx context.Context) error {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	evs := m.opt.Runner.Events()
	for {
		select {
		case <-ctx.Done():
			return m.Close()
		case ev := <-evs:
			m.Handle(ev)
		case <-t.C:
			m.Tick(m.opt.Clock())
		}
	}
}

// Close stops nothing but saves the state.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked(m.opt.Clock())
}

// ---- persistence --------------------------------------------------------------------------------

func (m *Manager) saveLocked(now time.Time) error {
	if m.opt.Path == "" {
		return nil
	}
	st := fileState{
		Version: FileVersion, SavedAt: now, NextID: m.nextID, Mode: m.mode,
		Goals: m.goals, Settings: m.settings, Stats: m.stats, Locks: m.locked, QuestSeen: m.questSeen, PrunedSeeds: m.pruned,
	}
	data, err := marshal(&st)
	if err == nil {
		err = writeAtomic(m.opt.Path, data)
	}
	if err != nil {
		m.opt.Logf("goals: save failed: %v", err)
		m.lastError = "saving state failed: " + err.Error()
		return err
	}
	m.dirty = false
	m.lastSave = now
	return nil
}

// ---- events -------------------------------------------------------------------------------------

func (m *Manager) emitLocked(ev Event) Event {
	m.seq++
	ev.Seq = m.seq
	if ev.Time.IsZero() {
		ev.Time = m.opt.Clock()
	}
	if ev.Message == "" {
		ev.Message = ev.describe()
	}
	m.log.push(ev)
	m.rev++
	m.dirty = true
	for _, c := range m.subs {
		select {
		case c <- ev:
		default: // slow consumer: it can catch up from the ring with Events(since)
		}
	}
	return ev
}

// Events returns logged events with Seq > since, oldest first, at most limit (0 = all kept).
func (m *Manager) Events(since uint64, limit int) []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Event
	m.log.each(func(e Event) bool {
		if e.Seq > since {
			out = append(out, e)
		}
		return true
	})
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

// Subscribe returns a channel of new events and a cancel function.
func (m *Manager) Subscribe() (<-chan Event, func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subSeq++
	id := m.subSeq
	c := make(chan Event, 256)
	m.subs[id] = c
	return c, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if _, ok := m.subs[id]; ok {
			delete(m.subs, id)
			close(c)
		}
	}
}

// Handle processes one event from the runner: statistics, goal progress, completion.
func (m *Manager) Handle(ev Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.opt.Clock()
	ev.Time = now
	g := m.activeLocked()
	if g != nil {
		ev.GoalID = g.ID
	}
	m.stats.record(&ev, ev.GoalID)
	if ev.Type == EvQuest && m.questSeenLocked(ev) {
		m.questLocked(ev, now) // a repeat is not logged again, but may still release a goal (idempotent)
		return
	}
	m.emitLocked(ev)

	if ev.Type == EvError && ev.Fatal {
		m.lastError = ev.Message
		m.haltLocked("stopped after a fatal error: " + ev.Message)
		m.saveLocked(now)
		return
	}
	if ev.Type == EvLevelUp {
		m.level = max(m.level, ev.Level)
		m.unlockLocked(ev.Level, now)
	}
	if ev.Type == EvQuest {
		m.questLocked(ev, now)
		return
	}
	if g == nil {
		return
	}
	if ev.Type == EvError && ev.Hold {
		if m.locked == nil {
			m.locked = map[string]lock{}
		}
		m.locked[g.ID] = lock{Level: ev.Level, Unlock: ev.Unlock, Diff: ev.Diff}
		m.holdLocked(g, ev.Message, now)
		// The runner read the level when its loop started; a level-up may have been handled since. Do not wait
		// for yet another one.
		if cur := m.knownLevelLocked(); ev.Level > 0 && cur > ev.Level {
			m.unlockLocked(cur, now)
		}
		return
	}
	pr := &g.Progress
	switch ev.Type {
	case EvKill:
		pr.Kills++
		if (g.Type == KillBoss && sameName(ev.Monster, g.Params.Boss)) || (g.Type != KillBoss && ev.Boss) {
			pr.BossKills++
		}
	case EvDrop:
		if ev.Item != nil && g.Params.Item.Match(*ev.Item) {
			pr.Drops += ev.Item.qty()
		}
	case EvDeath:
		pr.Deaths++
	case EvLevelUp:
		if ev.Level > 0 && g.Type == ReachLevel { // other types keep their own headline metric (a campaign counts quests)
			pr.Current = ev.Level
		}
	}
	if g.refreshProgress() {
		m.finishLocked(g, StateDone, "", now)
		return
	}
	if ev.Type == EvDeath && m.settings.MaxDeaths > 0 && pr.Deaths-g.DeathBase >= m.settings.MaxDeaths {
		m.deathHoldLocked(g, now)
	}
}

const (
	deathRestBase = 15 * time.Minute
	deathRestMax  = 2 * time.Hour
)

// deathHoldLocked pauses a goal that died MaxDeaths times. It is not a user pause: nobody may be there, so the goal
// resumes by itself after a rest (15 min, doubling per pause up to 2 h) or at the next level up, whichever is first.
// A goal that is not itself a level goal gets a reach_level goal first, so the retry is stronger than the try.
func (m *Manager) deathHoldLocked(g *Goal, now time.Time) {
	lvl := m.opt.Runner.Status().Character.Level
	rest := min(deathRestBase<<min(g.DeathHolds, 4), deathRestMax)
	g.DeathHolds++
	at := now.Add(rest)
	g.ResumeAt = &at
	m.lockAtLocked(g, max(lvl, 1))
	why := fmt.Sprintf("paused after %d deaths; resumes at %s or on the next level up", g.Progress.Deaths-g.DeathBase, at.Format("15:04"))
	if g.Type != ReachLevel && lvl > 0 {
		if id := m.farmFirstLocked(g, lvl+1, "", now); id != "" {
			why += "; farming level " + fmt.Sprint(lvl+1) + " first (" + id + ")"
		}
	}
	m.holdLocked(g, why, now)
}

// lockAtLocked records that g is held until the character is above level (see unlockLocked).
func (m *Manager) lockAtLocked(g *Goal, level int) {
	if m.locked == nil {
		m.locked = map[string]lock{}
	}
	m.locked[g.ID] = lock{Level: level}
}

// farmFirstLocked queues a reach_level goal for level unless one that reaches it is already waiting. It returns the id
// of the new goal, or "" if nothing was added.
func (m *Manager) farmFirstLocked(g *Goal, level int, difficulty string, now time.Time) string {
	for _, o := range m.goals {
		if o.Type == ReachLevel && (o.State == StateQueued || o.State == StateActive) && o.Params.Level >= level {
			return ""
		}
	}
	lg, err := m.addLocked(Goal{Type: ReachLevel, Priority: g.Priority,
		Params: Params{Level: level, Difficulty: difficulty}}, now)
	if err != nil {
		m.opt.Logf("farm goal for %s: %v", g.ID, err)
		return ""
	}
	return lg.ID
}

// gateLocked re-checks the resist gate when a chained campaign of the next difficulty is about to start for the
// first time: the chain only evaluated it once, when the campaign was created. On refusal the campaign is held,
// a level is farmed in the difficulty below, and the next level up asks again.
func (m *Manager) gateLocked(g *Goal, now time.Time) bool {
	if g.Type != Campaign || !g.Params.Chain || g.StartedAt != nil || !m.opt.Chain.usable() {
		return false
	}
	var diff string
	switch g.Params.Template {
	case CampaignNightmareAll:
		diff = "nightmare"
	case CampaignHellAll:
		diff = "hell"
	default:
		return false
	}
	lvl := m.opt.Runner.Status().Character.Level
	if lvl == 0 {
		return false // level unknown: nothing to judge
	}
	ok, why := m.opt.Chain.Ready(diff, lvl)
	if ok {
		return false
	}
	m.lockAtLocked(g, lvl)
	note := fmt.Sprintf("%s not entered at clvl %d: %s", diff, lvl, why)
	if id := m.farmFirstLocked(g, max(m.opt.Chain.Floor(diff), lvl+gateFarmLevels), prevDifficulty(diff), now); id != "" {
		note += "; farming first (" + id + ")"
	}
	m.holdLocked(g, note, now)
	return true
}

// gateFarmLevels is how many levels one gate refusal farms: resists come slowly, and a goal per level is churn.
const gateFarmLevels = 3

func prevDifficulty(d string) string {
	if d == "hell" {
		return "nightmare"
	}
	return "normal"
}

func sameName(a, b string) bool { return len(a) == len(b) && lower(a) == lower(b) }

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// ---- scheduler ----------------------------------------------------------------------------------

func (m *Manager) activeLocked() *Goal {
	if m.activeID == "" {
		return nil
	}
	return m.find(m.activeID)
}

func (m *Manager) find(id string) *Goal {
	for _, g := range m.goals {
		if g.ID == id {
			return g
		}
	}
	return nil
}

// pickLocked returns the first runnable goal in queue order that is inside its schedule window.
func (m *Manager) pickLocked(now time.Time) *Goal {
	for _, g := range m.goals {
		if (g.State == StateQueued || g.State == StateActive) && g.inWindow(now) {
			return g
		}
	}
	return nil
}

// scheduleLocked makes the runner work on the best goal. It does nothing unless the mode is running.
func (m *Manager) scheduleLocked(now time.Time) {
	if m.mode != ModeRunning {
		return
	}
	for tries := 0; tries < len(m.goals)+1; tries++ {
		want, cur := m.pickLocked(now), m.activeLocked()
		if want == cur && (cur != nil || m.activeID == "") {
			return
		}
		if cur != nil {
			m.opt.Runner.Stop()
			cur.State = StateQueued
			m.activeID = ""
			m.emitLocked(Event{Type: EvGoalPreempted, GoalID: cur.ID, Message: "Switching away from: " + cur.Title})
		}
		if want == nil {
			return
		}
		if m.gateLocked(want, now) {
			continue
		}
		if m.startLocked(want, now) {
			return
		}
	}
}

func (m *Manager) startLocked(g *Goal, now time.Time) bool {
	if lvl := m.opt.Runner.Status().Character.Level; lvl > 0 {
		if g.Progress.StartLevel == 0 {
			g.Progress.StartLevel = lvl
		}
		if g.Type == ReachLevel {
			g.Progress.Current = lvl
		}
	}
	if err := m.opt.Runner.Start(g.clone()); err != nil {
		g.State, g.Note = StateFailed, "could not start: "+err.Error()
		t := now
		g.CompletedAt = &t
		m.emitLocked(Event{Type: EvGoalFailed, GoalID: g.ID, Message: "Could not start " + g.Title + ": " + err.Error()})
		m.pruneLocked()
		m.saveLocked(now)
		return false
	}
	g.State, g.Note = StateActive, ""
	if g.StartedAt == nil {
		t := now
		g.StartedAt = &t
	}
	m.activeID = g.ID
	m.lastTick = now
	m.activeRem = 0
	m.emitLocked(Event{Type: EvGoalStarted, GoalID: g.ID, Message: "Started: " + g.Title})
	m.saveLocked(now)
	return true
}

// finishLocked ends the active goal (done or failed) and schedules the next one.
func (m *Manager) finishLocked(g *Goal, st State, note string, now time.Time) {
	if m.activeID == g.ID {
		m.opt.Runner.Stop()
		m.activeID = ""
	}
	g.State, g.Note = st, note
	delete(m.locked, g.ID)
	t := now
	g.CompletedAt = &t
	switch st {
	case StateDone:
		g.Progress.Percent = 100
		m.emitLocked(Event{Type: EvGoalDone, GoalID: g.ID, Message: "Completed: " + g.Title})
		m.chainLocked(g, now)
	case StateFailed:
		m.emitLocked(Event{Type: EvGoalFailed, GoalID: g.ID, Message: "Failed: " + g.Title + ": " + note})
	case StateSkipped:
		m.emitLocked(Event{Type: EvGoalSkipped, GoalID: g.ID, Message: "Skipped: " + g.Title})
	}
	m.pruneLocked()
	m.scheduleLocked(now)
	m.saveLocked(now)
}

// holdLocked puts a goal on user-style hold (state paused) with a reason, and schedules the next one.
func (m *Manager) holdLocked(g *Goal, why string, now time.Time) {
	if m.activeID == g.ID {
		m.opt.Runner.Stop()
		m.activeID = ""
	}
	g.State, g.Note = StatePaused, why
	m.emitLocked(Event{Type: EvGoalPaused, GoalID: g.ID, Message: g.Title + ": " + why})
	m.scheduleLocked(now)
	m.saveLocked(now)
}

// knownLevelLocked is the newest character level the manager has seen.
func (m *Manager) knownLevelLocked() int {
	if l := m.opt.Runner.Status().Character.Level; l > m.level {
		m.level = l
	}
	return m.level
}

// unlockLocked resumes goals held by a Hold error once the character is above the level it was held at.
func (m *Manager) unlockLocked(level int, now time.Time) {
	for _, g := range m.goals {
		l, ok := m.locked[g.ID]
		if !ok || g.State != StatePaused || level <= l.Level {
			continue
		}
		delete(m.locked, g.ID)
		g.State, g.Note, g.ResumeAt, g.DeathHolds = StateQueued, "", nil, 0
		g.DeathBase = g.Progress.Deaths
		m.emitLocked(Event{Type: EvGoalResumed, GoalID: g.ID, Message: "Resumed: " + g.Title + " (level " + fmt.Sprint(level) + ")"})
	}
	m.scheduleLocked(now)
	m.saveLocked(now)
}

// resumeDueLocked puts goals whose rest is over back in the queue.
func (m *Manager) resumeDueLocked(now time.Time) {
	for _, g := range m.goals {
		if g.State != StatePaused || g.ResumeAt == nil || now.Before(*g.ResumeAt) {
			continue
		}
		delete(m.locked, g.ID)
		g.State, g.Note, g.ResumeAt = StateQueued, "", nil
		g.DeathBase = g.Progress.Deaths
		m.emitLocked(Event{Type: EvGoalResumed, GoalID: g.ID, Message: "Resumed after a rest: " + g.Title})
		m.saveLocked(now)
	}
}

// questSeenLocked records the event's stage and reports whether this stage (or a higher one) was already seen.
func (m *Manager) questSeenLocked(ev Event) bool {
	key := fmt.Sprintf("%d/%s", ev.Diff, ev.Quest)
	if m.questSeen == nil {
		m.questSeen = map[string]int{}
	}
	if m.questSeen[key] >= ev.Stage {
		return true
	}
	m.questSeen[key] = ev.Stage
	m.dirty = true
	return false
}

// questLocked resumes goals held for the unlock this finished quest provides.
func (m *Manager) questLocked(ev Event, now time.Time) {
	if !ev.Done || ev.Unlock == "" {
		return
	}
	resumed := false
	for _, g := range m.goals {
		l, ok := m.locked[g.ID]
		if !ok || g.State != StatePaused || l.Unlock != ev.Unlock || l.Diff != ev.Diff {
			continue
		}
		delete(m.locked, g.ID)
		g.State, g.Note = StateQueued, ""
		resumed = true
		m.emitLocked(Event{Type: EvGoalResumed, GoalID: g.ID, Message: "Resumed: " + g.Title + " (" + ev.Unlock + " unlocked)"})
	}
	if resumed {
		m.scheduleLocked(now)
		m.saveLocked(now)
	}
	// A new questSeen entry is dirty and Tick saves it; repeats change nothing.
}

// haltLocked stops everything (mode stopped); the active goal returns to the queue.
func (m *Manager) haltLocked(why string) {
	if g := m.activeLocked(); g != nil {
		g.State = StateQueued
	}
	m.activeID = ""
	m.opt.Runner.Stop()
	m.mode = ModeStopped
	m.emitLocked(Event{Type: EvControl, Message: why})
}

// userPauseNote marks a pause made by hand; restore uses it to drop a stale hold left by an older build.
const userPauseNote = "paused by you"

func (m *Manager) pruneLocked() {
	var fin []*Goal
	for _, g := range m.goals {
		if g.State.Finished() {
			fin = append(fin, g)
		}
	}
	if len(fin) <= maxFinished {
		return
	}
	sort.SliceStable(fin, func(i, j int) bool { return fin[i].CompletedAt.Before(*fin[j].CompletedAt) })
	drop := map[string]bool{}
	for _, g := range fin[:len(fin)-maxFinished] {
		drop[g.ID] = true
		delete(m.locked, g.ID)
		if g.Seed != "" {
			m.pruned = append(m.pruned, g.Seed)
		}
	}
	kept := m.goals[:0]
	for _, g := range m.goals {
		if !drop[g.ID] {
			kept = append(kept, g)
		}
	}
	m.goals = kept
	m.pruned = trimSeeds(m.pruned)
}

// maxPrunedSeeds bounds the remembered seed keys of pruned goals.
const maxPrunedSeeds = 200

// trimSeeds keeps the newest maxPrunedSeeds keys.
func trimSeeds(s []string) []string {
	if len(s) > maxPrunedSeeds {
		return append([]string(nil), s[len(s)-maxPrunedSeeds:]...)
	}
	return s
}

// Tick advances time: credits play time, enforces time limits and schedule windows, saves when due.
func (m *Manager) Tick(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dt := now.Sub(m.lastTick)
	m.lastTick = now
	if g := m.activeLocked(); g != nil && m.mode == ModeRunning && dt > 0 && dt <= maxCreditGap {
		st := m.opt.Runner.Status()
		if st.State == RunnerRunning {
			m.activeRem += dt
			sec := int64(m.activeRem / time.Second)
			m.activeRem -= time.Duration(sec) * time.Second
			g.Progress.ActiveSeconds += sec
			m.stats.Totals.ActiveSeconds += sec
			m.rev++
		}
		if g.Type == ReachLevel && st.Character.Level > 0 {
			g.Progress.Current = st.Character.Level
		}
		if pr, ok := m.opt.Runner.(ProgressReporter); ok && g.Type == Campaign {
			if p, have := pr.Progress(); have {
				g.Progress.MergeProgress(p)
				m.rev++
			}
		}
		if g.refreshProgress() {
			m.finishLocked(g, StateDone, "", now)
		}
	}
	m.resumeDueLocked(now)
	m.scheduleLocked(now)
	if m.dirty && now.Sub(m.lastSave) >= saveEvery {
		m.saveLocked(now)
	}
}

// ---- control ------------------------------------------------------------------------------------

// Start begins playing the queue, or resumes after Pause.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.opt.Clock()
	switch m.mode {
	case ModeRunning:
		return nil
	case ModePaused:
		if m.activeID != "" {
			if err := m.opt.Runner.Resume(); err != nil {
				return err
			}
		}
		m.lastTick = now
	}
	m.mode = ModeRunning
	m.lastError = ""
	m.emitLocked(Event{Type: EvControl, Message: "Bot started"})
	m.scheduleLocked(now)
	m.saveLocked(now)
	return nil
}

// Pause freezes play; the active goal stays active.
func (m *Manager) Pause() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mode != ModeRunning {
		return fmt.Errorf("%w: bot is %s", ErrState, m.mode)
	}
	if m.activeID != "" {
		if err := m.opt.Runner.Pause(); err != nil {
			return err
		}
	}
	m.mode = ModePaused
	m.emitLocked(Event{Type: EvControl, Message: "Bot paused"})
	m.saveLocked(m.opt.Clock())
	return nil
}

// Stop abandons the current goal's run (it returns to the queue with its progress) and idles the bot.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mode == ModeStopped {
		return nil
	}
	m.haltLocked("Bot stopped")
	m.saveLocked(m.opt.Clock())
	return nil
}

// ---- goals --------------------------------------------------------------------------------------

// sortLocked keeps live goals ordered by priority (high first), stable in queue order, ahead of finished ones.
func (m *Manager) sortLocked() {
	sort.SliceStable(m.goals, func(i, j int) bool {
		a, b := m.goals[i], m.goals[j]
		if a.State.Finished() != b.State.Finished() {
			return !a.State.Finished()
		}
		if a.State.Finished() {
			return false
		}
		return a.Priority > b.Priority
	})
}

func (m *Manager) liveIdx() []int {
	var idx []int
	for i, g := range m.goals {
		if !g.State.Finished() {
			idx = append(idx, i)
		}
	}
	return idx
}

// Goals returns copies: live goals in scheduling order, then finished ones (newest first).
func (m *Manager) Goals() []Goal {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Goal, 0, len(m.goals))
	var fin []Goal
	for _, g := range m.goals {
		if g.State.Finished() {
			fin = append(fin, g.clone())
		} else {
			out = append(out, g.clone())
		}
	}
	sort.SliceStable(fin, func(i, j int) bool { return fin[i].CompletedAt.After(*fin[j].CompletedAt) })
	return append(out, fin...)
}

func (m *Manager) Goal(id string) (Goal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	g := m.find(id)
	if g == nil {
		return Goal{}, ErrNotFound
	}
	return g.clone(), nil
}

// Add validates and appends a goal. pos is the index among live goals; -1 appends after goals of equal
// or higher priority.
func (m *Manager) Add(g Goal, pos int) (Goal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addAtLocked(g, pos, m.opt.Clock())
}

// AddSeeded is the idempotent add used for --goals on the command line. It skips (added=false) when a goal with
// the same seed key exists in any state, or when an identical goal (type and params) is still unfinished, so a
// restart never queues a campaign twice. Dashboard adds use Add and may duplicate deliberately.
func (m *Manager) AddSeeded(g Goal, seed string) (out Goal, added bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	probe := g
	if err := probe.Normalize(); err != nil {
		return Goal{}, false, invalid("%v", err)
	}
	for _, e := range m.goals {
		if seed != "" && e.Seed == seed {
			return e.clone(), false, nil
		}
		if !e.State.Finished() && e.Type == probe.Type && reflect.DeepEqual(e.Params, probe.Params) {
			return e.clone(), false, nil
		}
	}
	if seed != "" && slices.Contains(m.pruned, seed) {
		return Goal{}, false, nil
	}
	g.Seed = seed
	out, err = m.addAtLocked(g, -1, m.opt.Clock())
	return out, err == nil, err
}

// addLocked appends a goal at its priority position.
func (m *Manager) addLocked(g Goal, now time.Time) (Goal, error) { return m.addAtLocked(g, -1, now) }

func (m *Manager) addAtLocked(g Goal, pos int, now time.Time) (Goal, error) {
	g.State, g.Progress, g.Note, g.StartedAt, g.CompletedAt, g.NextIDs = StateQueued, Progress{}, "", nil, nil, nil
	delete(m.locked, g.ID)
	g.ResumeAt, g.DeathHolds, g.DeathBase = nil, 0, 0
	if err := g.Normalize(); err != nil {
		return Goal{}, invalid("%v", err)
	}
	m.nextID++
	g.ID = "g" + strconv.FormatInt(m.nextID, 10)
	g.CreatedAt = now
	ng := &g
	g.refreshProgress()
	live := m.liveIdx()
	at := len(live)
	if pos >= 0 && pos < len(live) {
		at = pos
	} else {
		for i, gi := range live {
			if m.goals[gi].Priority < g.Priority {
				at = i
				break
			}
		}
	}
	gi := len(m.goals)
	if at < len(live) {
		gi = live[at]
	} else if len(live) > 0 {
		gi = live[len(live)-1] + 1
	}
	m.goals = append(m.goals, nil)
	copy(m.goals[gi+1:], m.goals[gi:])
	m.goals[gi] = ng
	m.emitLocked(Event{Type: EvGoalAdded, GoalID: g.ID, Message: "Added: " + g.Title})
	m.scheduleLocked(now)
	m.saveLocked(now)
	return ng.clone(), nil
}

// Update applies f to a copy of the goal, validates, and stores it. Editing the active goal restarts it
// (progress is kept) so the runner sees the new parameters.
func (m *Manager) Update(id string, f func(*Goal)) (Goal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	g := m.find(id)
	if g == nil {
		return Goal{}, ErrNotFound
	}
	c := g.clone()
	auto := g.Title == g.describe()
	f(&c)
	if auto && c.Title == g.Title {
		c.Title = "" // an automatic title follows the parameters
	}
	c.ID, c.CreatedAt, c.State, c.Progress, c.StartedAt, c.CompletedAt = g.ID, g.CreatedAt, g.State, g.Progress, g.StartedAt, g.CompletedAt
	if err := c.Normalize(); err != nil {
		return Goal{}, invalid("%v", err)
	}
	if g.Type != c.Type {
		return Goal{}, invalid("a goal's type cannot change; add a new goal")
	}
	now := m.opt.Clock()
	wasActive := m.activeID == id
	if wasActive {
		m.opt.Runner.Stop()
		m.activeID = ""
		c.State = StateQueued
	}
	*g = c
	g.Note = ""
	if !g.refreshProgress() && g.State == StateDone {
		g.State, g.CompletedAt = StateQueued, nil // a raised target reopens a finished goal
	}
	m.sortLocked()
	m.emitLocked(Event{Type: EvGoalEdited, GoalID: id, Message: "Edited: " + g.Title})
	m.scheduleLocked(now)
	m.saveLocked(now)
	return g.clone(), nil
}

func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g := m.find(id)
	if g == nil {
		return ErrNotFound
	}
	now := m.opt.Clock()
	if m.activeID == id {
		m.opt.Runner.Stop()
		m.activeID = ""
	}
	kept := m.goals[:0]
	for _, x := range m.goals {
		if x != g {
			kept = append(kept, x)
		}
	}
	m.goals = kept
	delete(m.locked, id)
	m.emitLocked(Event{Type: EvGoalRemoved, GoalID: id, Message: "Removed: " + g.Title})
	m.scheduleLocked(now)
	m.saveLocked(now)
	return nil
}

// Move puts a live goal at index `to` among live goals. Its priority is clamped between its new neighbours
// so the visible order is the order the scheduler uses.
func (m *Manager) Move(id string, to int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g := m.find(id)
	if g == nil {
		return ErrNotFound
	}
	if g.State.Finished() {
		return fmt.Errorf("%w: finished goals cannot be reordered", ErrState)
	}
	var live []*Goal
	for _, x := range m.goals {
		if !x.State.Finished() && x != g {
			live = append(live, x)
		}
	}
	to = max(0, min(to, len(live)))
	live = append(live[:to], append([]*Goal{g}, live[to:]...)...)
	if to > 0 && g.Priority > live[to-1].Priority {
		g.Priority = live[to-1].Priority
	}
	if to+1 < len(live) && g.Priority < live[to+1].Priority {
		g.Priority = live[to+1].Priority
	}
	m.commitOrderLocked(live)
	return nil
}

// Reorder sets the order of all live goals. ids must list every live goal exactly once. Priorities are
// raised so that the order is non-increasing in priority.
func (m *Manager) Reorder(ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var live []*Goal
	for _, x := range m.goals {
		if !x.State.Finished() {
			live = append(live, x)
		}
	}
	if len(ids) != len(live) {
		return invalid("reorder must list all %d live goals", len(live))
	}
	by := map[string]*Goal{}
	for _, x := range live {
		by[x.ID] = x
	}
	order := make([]*Goal, 0, len(ids))
	for _, id := range ids {
		x := by[id]
		if x == nil {
			return invalid("unknown or duplicate goal %q", id)
		}
		delete(by, id)
		order = append(order, x)
	}
	for i := len(order) - 2; i >= 0; i-- {
		order[i].Priority = max(order[i].Priority, order[i+1].Priority)
	}
	m.commitOrderLocked(order)
	return nil
}

func (m *Manager) commitOrderLocked(live []*Goal) {
	var fin []*Goal
	for _, x := range m.goals {
		if x.State.Finished() {
			fin = append(fin, x)
		}
	}
	m.goals = append(live, fin...)
	m.rev++
	now := m.opt.Clock()
	m.scheduleLocked(now)
	m.saveLocked(now)
}

// PauseGoal holds a goal: the scheduler skips it until ResumeGoal.
func (m *Manager) PauseGoal(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g := m.find(id)
	if g == nil {
		return ErrNotFound
	}
	if g.State != StateQueued && g.State != StateActive {
		return fmt.Errorf("%w: goal is %s", ErrState, g.State)
	}
	delete(m.locked, id) // a user pause replaces any hold
	m.holdLocked(g, userPauseNote, m.opt.Clock())
	return nil
}

func (m *Manager) ResumeGoal(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g := m.find(id)
	if g == nil {
		return ErrNotFound
	}
	if g.State != StatePaused {
		return fmt.Errorf("%w: goal is %s", ErrState, g.State)
	}
	delete(m.locked, id)
	g.State, g.Note, g.ResumeAt, g.DeathBase = StateQueued, "", nil, g.Progress.Deaths
	now := m.opt.Clock()
	m.emitLocked(Event{Type: EvGoalResumed, GoalID: id, Message: "Resumed: " + g.Title})
	m.scheduleLocked(now)
	m.saveLocked(now)
	return nil
}

// Skip ends a goal without completing it.
func (m *Manager) Skip(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g := m.find(id)
	if g == nil {
		return ErrNotFound
	}
	if g.State.Finished() {
		return fmt.Errorf("%w: goal is %s", ErrState, g.State)
	}
	m.finishLocked(g, StateSkipped, "skipped by you", m.opt.Clock())
	return nil
}

// Reset puts a finished or paused goal back in the queue with progress cleared.
func (m *Manager) Reset(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g := m.find(id)
	if g == nil {
		return ErrNotFound
	}
	if g.State == StateActive {
		return fmt.Errorf("%w: goal is running", ErrState)
	}
	g.State, g.Note, g.Progress, g.StartedAt, g.CompletedAt = StateQueued, "", Progress{}, nil, nil
	delete(m.locked, id)
	g.ResumeAt, g.DeathHolds, g.DeathBase = nil, 0, 0
	g.refreshProgress()
	m.sortLocked()
	now := m.opt.Clock()
	m.emitLocked(Event{Type: EvGoalResumed, GoalID: id, Message: "Restarted: " + g.Title})
	m.scheduleLocked(now)
	m.saveLocked(now)
	return nil
}

// ---- settings, status, statistics ---------------------------------------------------------------

func (m *Manager) Settings() Settings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.settings
}

func (m *Manager) SetSettings(s Settings) error {
	if err := s.Validate(); err != nil {
		return invalid("%v", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings = s
	if c, ok := m.opt.Runner.(Configurer); ok {
		c.Configure(s)
	}
	m.rev++
	m.saveLocked(m.opt.Clock())
	return nil
}

type Snapshot struct {
	Mode         Mode      `json:"mode"`
	ActiveGoalID string    `json:"activeGoalId,omitempty"`
	Runner       Status    `json:"runner"`
	Now          time.Time `json:"now"`
	Started      time.Time `json:"started"`
	XPPerHour    float64   `json:"xpPerHour"`
	Totals       Totals    `json:"totals"`
	Rev          uint64    `json:"rev"`
	LastSeq      uint64    `json:"lastSeq"`
	LastError    string    `json:"lastError,omitempty"`
	LoadWarning  string    `json:"loadWarning,omitempty"`
}

func (m *Manager) Snapshot() Snapshot {
	rs := m.opt.Runner.Status() // outside the lock: it is the runner's own state
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.opt.Clock()
	return Snapshot{
		Mode: m.mode, ActiveGoalID: m.activeID, Runner: rs, Now: now, Started: m.started,
		XPPerHour: m.stats.XPPerHour(now, 15*time.Minute), Totals: m.stats.Totals,
		Rev: m.rev, LastSeq: m.seq, LastError: m.lastError, LoadWarning: m.warning,
	}
}

type StatsSummary struct {
	Since     time.Time      `json:"since"`
	Totals    Totals         `json:"totals"`
	XPPerHour float64        `json:"xpPerHour"`
	Qualities map[string]int `json:"qualities"`
	Drops     []DropRecord   `json:"drops"` // newest first
	Areas     []AreaVisit    `json:"areas"` // newest first; the current visit has Seconds = time so far
}

func (m *Manager) StatsSummary() StatsSummary {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.opt.Clock()
	s := m.stats
	out := StatsSummary{Since: s.Since, Totals: s.Totals, XPPerHour: s.XPPerHour(now, 15*time.Minute), Qualities: map[string]int{}}
	for k, v := range s.Qualities {
		out.Qualities[k] = v
	}
	for i := len(s.Drops) - 1; i >= 0; i-- {
		out.Drops = append(out.Drops, s.Drops[i])
	}
	for i := len(s.Areas) - 1; i >= 0; i-- {
		v := s.Areas[i]
		if v.Seconds == 0 {
			v.Seconds = int64(now.Sub(v.Enter).Seconds())
		}
		out.Areas = append(out.Areas, v)
	}
	return out
}

func (m *Manager) Series(window, step time.Duration) []SeriesPoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stats.Series(m.opt.Clock(), window, step)
}

// Rev changes whenever anything observable changes (goals, progress, events).
func (m *Manager) Rev() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rev
}
